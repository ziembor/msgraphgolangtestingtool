package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"time"

	msgraphsdk "github.com/microsoftgraph/msgraph-sdk-go"
	"github.com/microsoftgraph/msgraph-sdk-go/models"
	"github.com/microsoftgraph/msgraph-sdk-go/users"
)

// sendEmail sends an email via Microsoft Graph API.
// Supports HTML and text bodies, multiple recipients (To, CC, BCC), and file attachments.
// In WhatIf mode, displays a preview of the email without actually sending it.
func sendEmail(ctx context.Context, client *msgraphsdk.GraphServiceClient, senderMailbox string, to, cc, bcc []string, subject, textContent, htmlContent string, attachmentPaths []string, config *Config, logger *CSVLogger) {
	message := models.NewMessage()

	// Set Subject
	message.SetSubject(&subject)

	// Set body - prefer HTML if provided, otherwise use text
	body := models.NewItemBody()
	if htmlContent != "" {
		body.SetContent(&htmlContent)
		contentType := models.HTML_BODYTYPE
		body.SetContentType(&contentType)
		logVerbose(config.VerboseMode, "Email body type: HTML")
	} else {
		body.SetContent(&textContent)
		contentType := models.TEXT_BODYTYPE
		body.SetContentType(&contentType)
		logVerbose(config.VerboseMode, "Email body type: Text")
	}
	message.SetBody(body)

	// Add Recipients
	if len(to) > 0 {
		message.SetToRecipients(createRecipients(to))
	}
	if len(cc) > 0 {
		message.SetCcRecipients(createRecipients(cc))
	}
	if len(bcc) > 0 {
		message.SetBccRecipients(createRecipients(bcc))
	}

	// Add Attachments
	if len(attachmentPaths) > 0 {
		fileAttachments, err := createFileAttachments(attachmentPaths, config)
		if err != nil {
			log.Printf("Error creating attachments: %v", err)
		} else if len(fileAttachments) > 0 {
			message.SetAttachments(fileAttachments)
			logVerbose(config.VerboseMode, "Attachments added: %d file(s)", len(fileAttachments))
		}
	}

	requestBody := users.NewItemSendMailPostRequestBody()
	requestBody.SetMessage(message)

	status := StatusSuccess
	attachmentCount := len(attachmentPaths)

	// WhatIf / Dry Run Mode
	if config.WhatIf {
		fmt.Println("========================================")
		fmt.Println("WHATIF MODE - DRY RUN (Email NOT sent)")
		fmt.Println("========================================")
		fmt.Printf("From: %s\n", senderMailbox)
		fmt.Printf("To: %v\n", to)
		if len(cc) > 0 {
			fmt.Printf("Cc: %v\n", cc)
		}
		if len(bcc) > 0 {
			fmt.Printf("Bcc: %v\n", bcc)
		}
		fmt.Printf("Subject: %s\n", subject)

		// Show body type and preview
		if htmlContent != "" {
			fmt.Println("Body Type: HTML")
			bodyPreview := htmlContent
			if len(bodyPreview) > 200 {
				bodyPreview = bodyPreview[:200] + "..."
			}
			fmt.Printf("Body Preview: %s\n", bodyPreview)
		} else {
			fmt.Println("Body Type: Text")
			bodyPreview := textContent
			if len(bodyPreview) > 200 {
				bodyPreview = bodyPreview[:200] + "..."
			}
			fmt.Printf("Body Preview: %s\n", bodyPreview)
		}

		if attachmentCount > 0 {
			fmt.Printf("Attachments: %d file(s)\n", attachmentCount)
			for i, path := range attachmentPaths {
				if fileInfo, err := os.Stat(path); err == nil {
					fmt.Printf("  [%d] %s (%d bytes)\n", i+1, filepath.Base(path), fileInfo.Size())
				} else {
					fmt.Printf("  [%d] %s (error reading file)\n", i+1, filepath.Base(path))
				}
			}
		}
		fmt.Println("========================================")
		status = "DRY RUN"
		logVerbose(config.VerboseMode, "WhatIf mode enabled - email preview displayed, API call skipped")
	} else {
		// Normal execution - actually send the email
		logVerbose(config.VerboseMode, "Calling Graph API: POST /users/%s/sendMail", senderMailbox)
		logVerbose(config.VerboseMode, "Email details - To: %v, CC: %v, BCC: %v", to, cc, bcc)
		err := client.Users().ByUserId(senderMailbox).SendMail().Post(ctx, requestBody, nil)

		if err != nil {
			// Enrich error with rate limit and service error details
			enrichedErr := enrichGraphAPIError(err, logger, "sendEmail")
			log.Printf("Error sending mail: %v", enrichedErr)
			status = fmt.Sprintf("%s: %v", StatusError, enrichedErr)
		} else {
			logVerbose(config.VerboseMode, "Email sent successfully via Graph API")
			fmt.Printf("Email sent successfully from %s.\n", senderMailbox)
			fmt.Printf("To: %v\n", to)
			fmt.Printf("Cc: %v\n", cc)
			fmt.Printf("Bcc: %v\n", bcc)
			fmt.Printf("Subject: %s\n", subject)
			if htmlContent != "" {
				fmt.Println("Body Type: HTML")
			} else {
				fmt.Println("Body Type: Text")
			}
			if attachmentCount > 0 {
				fmt.Printf("Attachments: %d file(s)\n", attachmentCount)
			}
		}
	}

	// Write to CSV
	if logger != nil {
		toStr := strings.Join(to, "; ")
		ccStr := strings.Join(cc, "; ")
		bccStr := strings.Join(bcc, "; ")
		bodyType := "Text"
		if htmlContent != "" {
			bodyType = "HTML"
		}
		logger.WriteRow([]string{ActionSendMail, status, senderMailbox, toStr, ccStr, bccStr, subject, bodyType, fmt.Sprintf("%d", attachmentCount)})
	}
}

// listInbox retrieves and displays the newest messages from a user's inbox.
// Supports both text and JSON output formats.
func listInbox(ctx context.Context, client *msgraphsdk.GraphServiceClient, mailbox string, count int, config *Config, logger *CSVLogger) error {
	// Configure request to get top N messages ordered by received date
	requestConfig := &users.ItemMessagesRequestBuilderGetRequestConfiguration{
		QueryParameters: &users.ItemMessagesRequestBuilderGetQueryParameters{
			Top:     Int32Ptr(int32(count)),
			Orderby: []string{"receivedDateTime DESC"},
			Select:  []string{"subject", "receivedDateTime", "from", "toRecipients"},
		},
	}

	logVerbose(config.VerboseMode, "Calling Graph API: GET /users/%s/messages?$top=%d&$orderby=receivedDateTime DESC", mailbox, count)

	// Execute API call with retry logic
	var getValueFunc func() []models.Messageable
	err := retryWithBackoff(ctx, config.MaxRetries, config.RetryDelay, func() error {
		apiResult, apiErr := client.Users().ByUserId(mailbox).Messages().Get(ctx, requestConfig)
		if apiErr == nil {
			getValueFunc = apiResult.GetValue
		}
		return apiErr
	})

	if err != nil {
		// Enrich error with rate limit and service error details
		enrichedErr := enrichGraphAPIError(err, logger, "listInbox")
		return fmt.Errorf("error fetching inbox for %s: %w", mailbox, enrichedErr)
	}

	messages := getValueFunc()
	messageCount := len(messages)

	logVerbose(config.VerboseMode, "API response received: %d messages", messageCount)

	if config.OutputFormat == "json" {
		printJSON(formatMessagesOutput(messages))
	} else {
		fmt.Printf("Newest %d messages in inbox for %s:\n\n", count, mailbox)

		if messageCount == 0 {
			fmt.Println("No messages found.")
		} else {
			for i, message := range messages {
				// Extract sender
				sender := "N/A"
				if message.GetFrom() != nil && message.GetFrom().GetEmailAddress() != nil {
					if message.GetFrom().GetEmailAddress().GetAddress() != nil {
						sender = *message.GetFrom().GetEmailAddress().GetAddress()
					}
				}

				// Extract recipients
				recipients := []string{}
				for _, recipient := range message.GetToRecipients() {
					if recipient.GetEmailAddress() != nil && recipient.GetEmailAddress().GetAddress() != nil {
						recipients = append(recipients, *recipient.GetEmailAddress().GetAddress())
					}
				}
				recipientStr := "N/A"
				if len(recipients) > 0 {
					recipientStr = strings.Join(recipients, "; ")
				}

				// Extract subject
				subject := "N/A"
				if message.GetSubject() != nil {
					subject = *message.GetSubject()
				}

				// Extract received date
				receivedDate := "N/A"
				if message.GetReceivedDateTime() != nil {
					receivedDate = message.GetReceivedDateTime().Format("2006-01-02 15:04:05")
				}

				fmt.Printf("%d. Subject: %s\n", i+1, subject)
				fmt.Printf("   From: %s\n", sender)
				fmt.Printf("   To: %s\n", recipientStr)
				fmt.Printf("   Received: %s\n\n", receivedDate)
			}
			// Log summary entry after all messages
			fmt.Printf("Total messages retrieved: %d\n", messageCount)
		}
	}

	// Always write to CSV logger
	if logger != nil {
		if messageCount == 0 {
			logger.WriteRow([]string{ActionGetInbox, StatusSuccess, mailbox, "No messages found (0 messages)", "N/A", "N/A", "N/A"})
		} else {
			for _, message := range messages {
				sender := "N/A"
				if message.GetFrom() != nil && message.GetFrom().GetEmailAddress() != nil {
					if message.GetFrom().GetEmailAddress().GetAddress() != nil {
						sender = *message.GetFrom().GetEmailAddress().GetAddress()
					}
				}
				recipients := []string{}
				for _, recipient := range message.GetToRecipients() {
					if recipient.GetEmailAddress() != nil && recipient.GetEmailAddress().GetAddress() != nil {
						recipients = append(recipients, *recipient.GetEmailAddress().GetAddress())
					}
				}
				recipientStr := "N/A"
				if len(recipients) > 0 {
					recipientStr = strings.Join(recipients, "; ")
				}
				subject := "N/A"
				if message.GetSubject() != nil {
					subject = *message.GetSubject()
				}
				receivedDate := "N/A"
				if message.GetReceivedDateTime() != nil {
					receivedDate = message.GetReceivedDateTime().Format("2006-01-02 15:04:05")
				}

				logger.WriteRow([]string{ActionGetInbox, StatusSuccess, mailbox, subject, sender, recipientStr, receivedDate})
			}
			logger.WriteRow([]string{ActionGetInbox, StatusSuccess, mailbox, fmt.Sprintf("Retrieved %d message(s)", messageCount), "SUMMARY", "SUMMARY", "SUMMARY"})
		}
	}

	return nil
}

// exportInbox exports inbox messages to individual JSON files in the temp directory.
// Creates a dated export directory structure and saves each message as a separate JSON file.
func exportInbox(ctx context.Context, client *msgraphsdk.GraphServiceClient, mailbox string, count int, config *Config, logger *CSVLogger) error {
	// Configure request to get top N messages
	requestConfig := &users.ItemMailFoldersItemMessagesRequestBuilderGetRequestConfiguration{
		QueryParameters: &users.ItemMailFoldersItemMessagesRequestBuilderGetQueryParameters{
			Top:     Int32Ptr(int32(count)),
			Orderby: []string{"receivedDateTime DESC"},
			Select:  []string{"id", "internetMessageId", "subject", "receivedDateTime", "from", "toRecipients", "ccRecipients", "bccRecipients", "body", "hasAttachments"},
		},
	}

	logVerbose(config.VerboseMode, "Calling Graph API: GET /users/%s/mailFolders/Inbox/messages?$top=%d&$orderby=receivedDateTime DESC", mailbox, count)

	// Execute API call with retry logic
	var getValueFunc func() []models.Messageable
	err := retryWithBackoff(ctx, config.MaxRetries, config.RetryDelay, func() error {
		// Specifically target Inbox folder
		apiResult, apiErr := client.Users().ByUserId(mailbox).MailFolders().ByMailFolderId("Inbox").Messages().Get(ctx, requestConfig)
		if apiErr == nil {
			getValueFunc = apiResult.GetValue
		}
		return apiErr
	})

	if err != nil {
		enrichedErr := enrichGraphAPIError(err, logger, "exportInbox")
		return fmt.Errorf("error fetching inbox for %s: %w", mailbox, enrichedErr)
	}

	messages := getValueFunc()
	messageCount := len(messages)

	logVerbose(config.VerboseMode, "API response received: %d messages", messageCount)

	if config.OutputFormat != "json" {
		fmt.Printf("Exporting %d messages from inbox for %s...\n", messageCount, mailbox)
	}

	if messageCount == 0 {
		if config.OutputFormat == "json" {
			printJSON([]interface{}{})
		} else {
			fmt.Println("No messages found.")
		}
		if logger != nil {
			logger.WriteRow([]string{ActionExportInbox, StatusSuccess, mailbox, "No messages found (0 messages)", "N/A"})
		}
		return nil
	}

	// Print JSON output if requested
	if config.OutputFormat == "json" {
		printJSON(formatMessagesOutput(messages))
	}

	// Create export directory
	exportDir, err := createExportDir()
	if err != nil {
		return err
	}

	if config.OutputFormat != "json" {
		fmt.Printf("Export directory: %s\n", exportDir)
	}

	successCount := 0
	for _, message := range messages {
		if err := exportMessageToJSON(message, exportDir, config); err != nil {
			log.Printf("Error exporting message ID %s: %v", *message.GetId(), err)
			continue
		}
		successCount++
	}

	if config.OutputFormat != "json" {
		fmt.Printf("Successfully exported %d/%d messages.\n", successCount, messageCount)
	}
	if logger != nil {
		logger.WriteRow([]string{ActionExportInbox, StatusSuccess, mailbox, fmt.Sprintf("Exported %d/%d messages", successCount, messageCount), exportDir})
	}

	return nil
}

// exportMessageToJSON exports a single message to a JSON file.
// Extracts key fields and saves them in a clean, predictable format.
func exportMessageToJSON(message models.Messageable, dir string, config *Config) error {
	// Extract basic info for filename
	id := "unknown_id"
	if message.GetId() != nil {
		id = *message.GetId()
	}

	// Create a simplified structure for export to ensure clean JSON
	// We could use the model directly but it might be verbose or have circular refs depending on serialization
	// Extracting fields explicitly gives us control.
	exportData := make(map[string]interface{})

	if message.GetId() != nil { exportData["id"] = *message.GetId() }
	if message.GetInternetMessageId() != nil { exportData["internetMessageId"] = *message.GetInternetMessageId() }
	if message.GetSubject() != nil { exportData["subject"] = *message.GetSubject() }
	if message.GetReceivedDateTime() != nil { exportData["receivedDateTime"] = message.GetReceivedDateTime().Format(time.RFC3339) }

	// From
	if message.GetFrom() != nil && message.GetFrom().GetEmailAddress() != nil {
		exportData["from"] = extractEmailAddress(message.GetFrom().GetEmailAddress())
	}

	// Recipients
	if message.GetToRecipients() != nil {
		exportData["to"] = extractRecipients(message.GetToRecipients())
	}
	if message.GetCcRecipients() != nil {
		exportData["cc"] = extractRecipients(message.GetCcRecipients())
	}
	if message.GetBccRecipients() != nil {
		exportData["bcc"] = extractRecipients(message.GetBccRecipients())
	}

	// Body
	if message.GetBody() != nil {
		bodyData := make(map[string]string)
		if message.GetBody().GetContentType() != nil {
			bodyData["contentType"] = message.GetBody().GetContentType().String()
		}
		if message.GetBody().GetContent() != nil {
			bodyData["content"] = *message.GetBody().GetContent()
		}
		exportData["body"] = bodyData
	}

	// Marshal to JSON
	jsonData, err := json.MarshalIndent(exportData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal message to JSON: %w", err)
	}

	// Sanitize filename
	filename := fmt.Sprintf("msg_%s.json", sanitizeFilename(id))
	filePath := filepath.Join(dir, filename)

	if err := os.WriteFile(filePath, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write JSON file: %w", err)
	}

	logVerbose(config.VerboseMode, "Exported message to %s", filePath)
	return nil
}

// createExportDir creates the export directory structure: $TEMP/export/YYYY-MM-DD
func createExportDir() (string, error) {
	tempDir := os.TempDir()
	dateStr := time.Now().Format("2006-01-02")
	exportDir := filepath.Join(tempDir, "export", dateStr)

	if err := os.MkdirAll(exportDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create export directory %s: %w", exportDir, err)
	}
	return exportDir, nil
}

// extractEmailAddress extracts name and address from an EmailAddressable object.
func extractEmailAddress(addr models.EmailAddressable) map[string]string {
	res := make(map[string]string)
	if addr.GetName() != nil { res["name"] = *addr.GetName() }
	if addr.GetAddress() != nil { res["address"] = *addr.GetAddress() }
	return res
}

// extractRecipients extracts email addresses from a list of recipients.
func extractRecipients(recipients []models.Recipientable) []map[string]string {
	var res []map[string]string
	for _, r := range recipients {
		if r.GetEmailAddress() != nil {
			res = append(res, extractEmailAddress(r.GetEmailAddress()))
		}
	}
	return res
}

// sanitizeFilename replaces invalid filesystem characters with underscores.
func sanitizeFilename(name string) string {
	invalid := []string{"<", ">", ":", "\"", "/", "\\", "|", "?", "*", "="}
	for _, char := range invalid {
		name = strings.ReplaceAll(name, char, "_")
	}
	return name
}

// createFileAttachments reads files and creates Graph API attachment objects.
// Returns error if no valid attachments could be processed.
func createFileAttachments(filePaths []string, config *Config) ([]models.Attachmentable, error) {
	var attachments []models.Attachmentable

	for _, filePath := range filePaths {
		// Read file content
		fileData, err := os.ReadFile(filePath)
		if err != nil {
			log.Printf("Warning: Could not read attachment file %s: %v", filePath, err)
			continue
		}

		// Create file attachment
		attachment := models.NewFileAttachment()

		// Set the OData type for file attachment
		odataType := "#microsoft.graph.fileAttachment"
		attachment.SetOdataType(&odataType)

		// Set file name (just the base name, not full path)
		fileName := filepath.Base(filePath)
		attachment.SetName(&fileName)

		// Detect content type from file extension
		contentType := mime.TypeByExtension(filepath.Ext(filePath))
		if contentType == "" {
			contentType = "application/octet-stream"
		}
		attachment.SetContentType(&contentType)

		// Set content as base64 encoded bytes
		attachment.SetContentBytes(fileData)

		logVerbose(config.VerboseMode, "Attachment: %s (%s, %d bytes)", fileName, contentType, len(fileData))
		attachments = append(attachments, attachment)
	}

	if len(attachments) == 0 && len(filePaths) > 0 {
		return nil, fmt.Errorf("no valid attachments could be processed")
	}

	return attachments, nil
}

// getAttachmentContentBase64 returns base64 encoded file content (for debugging/verbose).
func getAttachmentContentBase64(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

// createRecipients creates a list of recipient objects from email addresses.
func createRecipients(emails []string) []models.Recipientable {
	recipients := make([]models.Recipientable, len(emails))
	for i, email := range emails {
		recipient := models.NewRecipient()
		emailAddress := models.NewEmailAddress()
		// Need to create a new variable for the address pointer
		address := email
		emailAddress.SetAddress(&address)
		recipient.SetEmailAddress(emailAddress)
		recipients[i] = recipient
	}
	return recipients
}
