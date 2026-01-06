package main

import (
	"context"
	"encoding/base64"
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

// executeAction dispatches to the appropriate action handler based on config.Action.
// Supported actions are: getevents, sendmail, sendinvite, and getinbox.
//
// For sendmail action, if no recipients are specified, the email is sent to the
// mailbox owner (self). All actions log their operations to the provided CSV logger.
//
// Returns an error if the action fails or if the action name is unknown.
func executeAction(ctx context.Context, client *msgraphsdk.GraphServiceClient, config *Config, logger *CSVLogger) error {
	switch config.Action {
	case ActionGetEvents:
		if err := listEvents(ctx, client, config.Mailbox, config.Count, config, logger); err != nil {
			return fmt.Errorf("failed to list events: %w", err)
		}
	case ActionSendMail:
		// If no recipients specified at all, default 'to' to the sender mailbox
		if len(config.To) == 0 && len(config.Cc) == 0 && len(config.Bcc) == 0 {
			config.To = []string{config.Mailbox}
		}

		sendEmail(ctx, client, config.Mailbox, config.To, config.Cc, config.Bcc, config.Subject, config.Body, config.BodyHTML, config.AttachmentFiles, config, logger)
	case ActionSendInvite:
		// Use Subject for calendar invite
		// For backward compatibility, if InviteSubject is set, use it instead
		inviteSubject := config.Subject
		if config.InviteSubject != "" {
			inviteSubject = config.InviteSubject
		}
		// If using default email subject, change to default calendar invite subject
		if inviteSubject == "Automated Tool Notification" {
			inviteSubject = "It's testing event"
		}
		createInvite(ctx, client, config.Mailbox, inviteSubject, config.StartTime, config.EndTime, config, logger)
	case ActionGetInbox:
		if err := listInbox(ctx, client, config.Mailbox, config.Count, config, logger); err != nil {
			return fmt.Errorf("failed to list inbox: %w", err)
		}
	case ActionGetSchedule:
		if err := checkAvailability(ctx, client, config.Mailbox, config.To[0], config, logger); err != nil {
			return fmt.Errorf("failed to check availability: %w", err)
		}
	default:
		return fmt.Errorf("unknown action: %s", config.Action)
	}

	return nil
}

func listEvents(ctx context.Context, client *msgraphsdk.GraphServiceClient, mailbox string, count int, config *Config, logger *CSVLogger) error {
	// Configure request to get top N events
	requestConfig := &users.ItemEventsRequestBuilderGetRequestConfiguration{
		QueryParameters: &users.ItemEventsRequestBuilderGetQueryParameters{
			Top: Int32Ptr(int32(count)),
		},
	}

	logVerbose(config.VerboseMode, "Calling Graph API: GET /users/%s/events?$top=%d", mailbox, count)

	// Execute API call with retry logic
	var getValueFunc func() []models.Eventable
	err := retryWithBackoff(ctx, config.MaxRetries, config.RetryDelay, func() error {
		apiResult, apiErr := client.Users().ByUserId(mailbox).Events().Get(ctx, requestConfig)
		if apiErr == nil {
			getValueFunc = apiResult.GetValue
		}
		return apiErr
	})

	if err != nil {
		// Enrich error with rate limit and service error details
		enrichedErr := enrichGraphAPIError(err, logger, "listEvents")
		return fmt.Errorf("error fetching calendar for %s: %w", mailbox, enrichedErr)
	}

	events := getValueFunc()
	eventCount := len(events)

	logVerbose(config.VerboseMode, "API response received: %d events", eventCount)
	fmt.Printf("Upcoming events for %s:\n", mailbox)

	if eventCount == 0 {
		fmt.Println("No events found.")
		// Log summary entry when no events found
		if logger != nil {
			logger.WriteRow([]string{ActionGetEvents, StatusSuccess, mailbox, "No events found (0 events)", "N/A"})
		}
	} else {
		for _, event := range events {
			subject := "N/A"
			if event.GetSubject() != nil {
				subject = *event.GetSubject()
			}

			id := "N/A"
			if event.GetId() != nil {
				id = *event.GetId()
			}

			fmt.Printf("- %s (ID: %s)\n", subject, id)

			// Write to CSV
			if logger != nil {
				logger.WriteRow([]string{ActionGetEvents, StatusSuccess, mailbox, subject, id})
			}
		}
		// Log summary entry after all events
		fmt.Printf("\nTotal events retrieved: %d\n", eventCount)
		if logger != nil {
			logger.WriteRow([]string{ActionGetEvents, StatusSuccess, mailbox, fmt.Sprintf("Retrieved %d event(s)", eventCount), "SUMMARY"})
		}
	}

	return nil
}

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

	logVerbose(config.VerboseMode, "Calling Graph API: POST /users/%s/sendMail", senderMailbox)
	logVerbose(config.VerboseMode, "Email details - To: %v, CC: %v, BCC: %v", to, cc, bcc)
	err := client.Users().ByUserId(senderMailbox).SendMail().Post(ctx, requestBody, nil)

	status := StatusSuccess
	attachmentCount := len(attachmentPaths)
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

func createInvite(ctx context.Context, client *msgraphsdk.GraphServiceClient, mailbox, subject, startTimeStr, endTimeStr string, config *Config, logger *CSVLogger) {
	event := models.NewEvent()
	event.SetSubject(&subject)

	// Parse start time, default to now if not provided
	var startTime time.Time
	var err error
	if startTimeStr == "" {
		startTime = time.Now()
	} else {
		startTime, err = parseFlexibleTime(startTimeStr)
		if err != nil {
			log.Printf("Error parsing start time: %v. Using current time instead.", err)
			startTime = time.Now()
		}
	}

	// Parse end time, default to 1 hour after start if not provided
	var endTime time.Time
	if endTimeStr == "" {
		endTime = startTime.Add(1 * time.Hour)
	} else {
		endTime, err = parseFlexibleTime(endTimeStr)
		if err != nil {
			log.Printf("Error parsing end time: %v. Using start + 1 hour instead.", err)
			endTime = startTime.Add(1 * time.Hour)
		}
	}

	// Set start time
	startDateTime := models.NewDateTimeTimeZone()
	startTimeFormatted := startTime.Format(time.RFC3339)
	startDateTime.SetDateTime(&startTimeFormatted)
	timezone := "UTC"
	startDateTime.SetTimeZone(&timezone)
	event.SetStart(startDateTime)

	// Set end time
	endDateTime := models.NewDateTimeTimeZone()
	endTimeFormatted := endTime.Format(time.RFC3339)
	endDateTime.SetDateTime(&endTimeFormatted)
	endDateTime.SetTimeZone(&timezone)
	event.SetEnd(endDateTime)

	// Create the event
	logVerbose(config.VerboseMode, "Calling Graph API: POST /users/%s/events", mailbox)
	logVerbose(config.VerboseMode, "Calendar invite - Subject: %s, Start: %s, End: %s", subject, startTime.Format(time.RFC3339), endTime.Format(time.RFC3339))
	createdEvent, err := client.Users().ByUserId(mailbox).Events().Post(ctx, event, nil)

	status := StatusSuccess
	eventID := "N/A"
	if err != nil {
		// Enrich error with rate limit and service error details
		enrichedErr := enrichGraphAPIError(err, logger, "createInvite")
		log.Printf("Error creating invite: %v", enrichedErr)
		status = fmt.Sprintf("%s: %v", StatusError, enrichedErr)
	} else {
		if createdEvent.GetId() != nil {
			eventID = *createdEvent.GetId()
		}
		logVerbose(config.VerboseMode, "Calendar event created successfully via Graph API")
		logVerbose(config.VerboseMode, "Event ID: %s", eventID)
		fmt.Printf("Calendar invitation created in mailbox: %s\n", mailbox)
		fmt.Printf("Subject: %s\n", subject)
		fmt.Printf("Start: %s\n", startTime.Format("2006-01-02 15:04:05 MST"))
		fmt.Printf("End: %s\n", endTime.Format("2006-01-02 15:04:05 MST"))
		fmt.Printf("Event ID: %s\n", eventID)
	}

	// Write to CSV
	if logger != nil {
		logger.WriteRow([]string{ActionSendInvite, status, mailbox, subject, startTime.Format(time.RFC3339), endTime.Format(time.RFC3339), eventID})
	}
}

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
	fmt.Printf("Newest %d messages in inbox for %s:\n\n", count, mailbox)

	if messageCount == 0 {
		fmt.Println("No messages found.")
		// Log summary entry when no messages found
		if logger != nil {
			logger.WriteRow([]string{ActionGetInbox, StatusSuccess, mailbox, "No messages found (0 messages)", "N/A", "N/A", "N/A"})
		}
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

			// Write to CSV
			if logger != nil {
				logger.WriteRow([]string{ActionGetInbox, StatusSuccess, mailbox, subject, sender, recipientStr, receivedDate})
			}
		}
		// Log summary entry after all messages
		fmt.Printf("Total messages retrieved: %d\n", messageCount)
		if logger != nil {
			logger.WriteRow([]string{ActionGetInbox, StatusSuccess, mailbox, fmt.Sprintf("Retrieved %d message(s)", messageCount), "SUMMARY", "SUMMARY", "SUMMARY"})
		}
	}

	return nil
}

// checkAvailability checks the recipient's availability for the next working day at 12:00 UTC.
func checkAvailability(ctx context.Context, client *msgraphsdk.GraphServiceClient, mailbox string, recipient string, config *Config, logger *CSVLogger) error {
	// Calculate next working day
	now := time.Now().UTC()
	nextWorkingDay := addWorkingDays(now, 1)

	// Set time to 12:00 UTC (noon)
	checkDateTime := time.Date(
		nextWorkingDay.Year(),
		nextWorkingDay.Month(),
		nextWorkingDay.Day(),
		12, 0, 0, 0,
		time.UTC,
	)

	// End time is 1 hour later (13:00 UTC)
	endDateTime := checkDateTime.Add(1 * time.Hour)

	logVerbose(config.VerboseMode, "Checking availability for %s on %s (12:00-13:00 UTC)", recipient, checkDateTime.Format("2006-01-02"))

	// Create DateTimeTimeZone objects for Graph API
	startTimeZone := models.NewDateTimeTimeZone()
	startTimeZone.SetDateTime(pointerTo(checkDateTime.Format(time.RFC3339)))
	startTimeZone.SetTimeZone(pointerTo("UTC"))

	endTimeZone := models.NewDateTimeTimeZone()
	endTimeZone.SetDateTime(pointerTo(endDateTime.Format(time.RFC3339)))
	endTimeZone.SetTimeZone(pointerTo("UTC"))

	// Create request body
	requestBody := users.NewItemCalendarGetSchedulePostRequestBody()
	requestBody.SetSchedules([]string{recipient})
	requestBody.SetStartTime(startTimeZone)
	requestBody.SetEndTime(endTimeZone)
	interval := int32(60) // 60-minute intervals
	requestBody.SetAvailabilityViewInterval(&interval)

	logVerbose(config.VerboseMode, "Calling Graph API: POST /users/%s/calendar/getSchedule", mailbox)

	// Execute API call with retry logic
	var scheduleInfo []models.ScheduleInformationable
	err := retryWithBackoff(ctx, config.MaxRetries, config.RetryDelay, func() error {
		response, apiErr := client.Users().ByUserId(mailbox).Calendar().GetSchedule().Post(ctx, requestBody, nil)
		if apiErr == nil && response != nil {
			scheduleInfo = response.GetValue()
		}
		return apiErr
	})

	if err != nil {
		// Enrich error with rate limit and service error details
		enrichedErr := enrichGraphAPIError(err, logger, "checkAvailability")
		csvRow := []string{ActionGetSchedule, fmt.Sprintf("Error: %v", enrichedErr), mailbox, recipient, checkDateTime.Format(time.RFC3339), "N/A"}
		if logger != nil {
			logger.WriteRow(csvRow)
		}
		return fmt.Errorf("error checking availability for %s: %w", recipient, enrichedErr)
	}

	logVerbose(config.VerboseMode, "API response received: %d schedule(s)", len(scheduleInfo))

	// Parse availability view
	if len(scheduleInfo) == 0 {
		errMsg := "no schedule information returned"
		csvRow := []string{ActionGetSchedule, fmt.Sprintf("Error: %s", errMsg), mailbox, recipient, checkDateTime.Format(time.RFC3339), "N/A"}
		if logger != nil {
			logger.WriteRow(csvRow)
		}
		return fmt.Errorf("no schedule information returned")
	}

	// Get availability view from first schedule
	info := scheduleInfo[0]
	availabilityView := ""
	if info.GetAvailabilityView() != nil {
		availabilityView = *info.GetAvailabilityView()
	}

	if availabilityView == "" {
		errMsg := "empty availability view returned"
		csvRow := []string{ActionGetSchedule, fmt.Sprintf("Error: %s", errMsg), mailbox, recipient, checkDateTime.Format(time.RFC3339), "N/A"}
		if logger != nil {
			logger.WriteRow(csvRow)
		}
		return fmt.Errorf("empty availability view returned")
	}

	// Interpret availability
	status := interpretAvailability(availabilityView)

	// Display results
	fmt.Printf("Availability Check Results:\n")
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("Organizer:     %s\n", mailbox)
	fmt.Printf("Recipient:     %s\n", recipient)
	fmt.Printf("Check Date:    %s\n", checkDateTime.Format("2006-01-02"))
	fmt.Printf("Check Time:    12:00-13:00 UTC\n")
	fmt.Printf("Status:        %s\n", status)
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

	logVerbose(config.VerboseMode, "Availability view: %s → %s", availabilityView, status)

	// Log to CSV
	if logger != nil {
		csvRow := []string{ActionGetSchedule, StatusSuccess, mailbox, recipient, checkDateTime.Format(time.RFC3339), availabilityView}
		logger.WriteRow(csvRow)
	}

	return nil
}

// interpretAvailability converts Microsoft Graph availability view codes to human-readable status.
func interpretAvailability(view string) string {
	if len(view) == 0 {
		return "Unknown (empty response)"
	}

	// Get the first character (representing the time slot status)
	code := string(view[0])

	switch code {
	case "0":
		return "Free"
	case "1":
		return "Tentative"
	case "2":
		return "Busy"
	case "3":
		return "Out of Office"
	case "4":
		return "Working Elsewhere"
	default:
		return fmt.Sprintf("Unknown (%s)", code)
	}
}

// createFileAttachments reads files and creates Graph API attachment objects
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

// getAttachmentContentBase64 returns base64 encoded file content (for debugging/verbose)
func getAttachmentContentBase64(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

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

// parseFlexibleTime parses a time string accepting multiple formats
func parseFlexibleTime(timeStr string) (time.Time, error) {
	if timeStr == "" {
		return time.Time{}, fmt.Errorf("time string is empty")
	}

	// Try RFC3339 first (with timezone)
	t, err := time.Parse(time.RFC3339, timeStr)
	if err == nil {
		return t, nil
	}

	// Try PowerShell sortable format (without timezone) - assume UTC
	t, err = time.Parse("2006-01-02T15:04:05", timeStr)
	if err == nil {
		return t.UTC(), nil
	}

	return time.Time{}, fmt.Errorf("invalid time format (expected RFC3339 like '2026-01-15T14:00:00Z' or PowerShell sortable like '2026-01-15T14:00:00')")
}

// addWorkingDays adds a specified number of working days (Monday-Friday) to the given time.
func addWorkingDays(t time.Time, days int) time.Time {
	if days <= 0 {
		return t
	}

	result := t
	daysAdded := 0

	for daysAdded < days {
		result = result.Add(24 * time.Hour)

		// Check if this is a working day (Monday=1, Friday=5)
		weekday := result.Weekday()
		if weekday != time.Saturday && weekday != time.Sunday {
			daysAdded++
		}
	}

	return result
}

// Status constants
const (
	StatusSuccess = "Success"
	StatusError   = "Error"
)
