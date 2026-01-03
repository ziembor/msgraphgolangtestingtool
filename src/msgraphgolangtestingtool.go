package main

import (
	"context"
	"crypto"
	"crypto/x509"
	"encoding/csv"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	msgraphsdk "github.com/microsoftgraph/msgraph-sdk-go"
	"github.com/microsoftgraph/msgraph-sdk-go/models"
	"github.com/microsoftgraph/msgraph-sdk-go/models/odataerrors"
	"github.com/microsoftgraph/msgraph-sdk-go/users"
	"golang.org/x/crypto/pkcs12"
)

const version = "1.12.4"

var csvWriter *csv.Writer
var csvFile *os.File
var verboseMode bool

// applyEnvVars applies environment variable values to flags that weren't explicitly set via command line
func applyEnvVars(envMap map[string]*string) {
	// Track which flags were explicitly set via command line
	providedFlags := make(map[string]bool)
	flag.Visit(func(f *flag.Flag) {
		providedFlags[f.Name] = true
	})

	// Map flag names to environment variable names
	flagToEnv := map[string]string{
		"tenantid":       "MSGRAPHTENANTID",
		"clientid":       "MSGRAPHCLIENTID",
		"secret":         "MSGRAPHSECRET",
		"pfx":            "MSGRAPHPFX",
		"pfxpass":        "MSGRAPHPFXPASS",
		"thumbprint":     "MSGRAPHTHUMBPRINT",
		"mailbox":        "MSGRAPHMAILBOX",
		"to":             "MSGRAPHTO",
		"cc":             "MSGRAPHCC",
		"bcc":            "MSGRAPHBCC",
		"subject":        "MSGRAPHSUBJECT",
		"body":           "MSGRAPHBODY",
		"invite-subject": "MSGRAPHINVITESUBJECT",
		"start":          "MSGRAPHSTART",
		"end":            "MSGRAPHEND",
		"action":         "MSGRAPHACTION",
		"proxy":          "MSGRAPHPROXY",
	}

	// For each environment variable, if flag wasn't provided, use env value
	for envName, flagPtr := range envMap {
		// Find the flag name for this env var
		var flagName string
		for fn, en := range flagToEnv {
			if en == envName {
				flagName = fn
				break
			}
		}

		// If flag was not provided via command line, check environment variable
		if !providedFlags[flagName] {
			if envValue := os.Getenv(envName); envValue != "" {
				*flagPtr = envValue
			}
		}
	}
}

func main() {
	// 1. Define Command Line Parameters

	showVersion := flag.Bool("version", false, "Show version information")
	tenantID := flag.String("tenantid", "", "The Azure Tenant ID")
	clientID := flag.String("clientid", "", "The Application (Client) ID")
	secret := flag.String("secret", "", "The Client Secret")
	pfxPath := flag.String("pfx", "", "Path to the .pfx certificate file")
	pfxPass := flag.String("pfxpass", "", "Password for the .pfx file")
	// Double backslash for string literal, needs to be careful.
	thumbprint := flag.String("thumbprint", "", "Thumbprint of the certificate in the CurrentUser\\My store")
	mailbox := flag.String("mailbox", "", "The target EXO mailbox email address")

	// Recipient flags
	toRaw := flag.String("to", "", "Comma-separated list of TO recipients (optional, defaults to mailbox if empty)")
	ccRaw := flag.String("cc", "", "Comma-separated list of CC recipients")
	bccRaw := flag.String("bcc", "", "Comma-separated list of BCC recipients")

	// Email content flags
	subject := flag.String("subject", "Automated Tool Notification", "Subject of the email")
	body := flag.String("body", "It's a test message, please ignore", "Body content of the email (text)")

	// Calendar invite flags
	inviteSubject := flag.String("invite-subject", "System Sync", "Subject of the calendar invite")
	startTime := flag.String("start", "", "Start time for calendar invite (RFC3339 format, e.g., 2026-01-15T14:00:00Z). Defaults to now if empty")
	endTime := flag.String("end", "", "End time for calendar invite (RFC3339 format, e.g., 2026-01-15T15:00:00Z). Defaults to 1 hour after start if empty")

	// Proxy configuration
	proxyURL := flag.String("proxy", "", "HTTP/HTTPS proxy URL (e.g., http://proxy.example.com:8080)")

	// Verbose mode
	verbose := flag.Bool("verbose", false, "Enable verbose output (shows configuration, tokens, API details)")

	action := flag.String("action", "getevents", "Action to perform: getevents, sendmail, sendinvite, getinbox")
	flag.Parse()

	// Set global verbose flag
	verboseMode = *verbose

	// Check version flag
	if *showVersion {
		fmt.Printf("Microsoft Graph Golang Testing Tool - Version %s\n", version)
		os.Exit(0)
	}

	// Apply environment variables if flags not set via command line
	applyEnvVars(map[string]*string{
		"MSGRAPHTENANTID":      tenantID,
		"MSGRAPHCLIENTID":      clientID,
		"MSGRAPHSECRET":        secret,
		"MSGRAPHPFX":           pfxPath,
		"MSGRAPHPFXPASS":       pfxPass,
		"MSGRAPHTHUMBPRINT":    thumbprint,
		"MSGRAPHMAILBOX":       mailbox,
		"MSGRAPHTO":            toRaw,
		"MSGRAPHCC":            ccRaw,
		"MSGRAPHBCC":           bccRaw,
		"MSGRAPHSUBJECT":       subject,
		"MSGRAPHBODY":          body,
		"MSGRAPHINVITESUBJECT": inviteSubject,
		"MSGRAPHSTART":         startTime,
		"MSGRAPHEND":           endTime,
		"MSGRAPHACTION":        action,
		"MSGRAPHPROXY":         proxyURL,
	})

	// Print verbose configuration if enabled
	if verboseMode {
		printVerboseConfig(*tenantID, *clientID, *secret, *pfxPath, *thumbprint, *mailbox, *action, *proxyURL, *toRaw, *ccRaw, *bccRaw, *subject, *body, *inviteSubject, *startTime, *endTime)
	}

	// Validation
	if *tenantID == "" || *clientID == "" || *mailbox == "" {
		fmt.Println("Error: Missing required parameters (tenantid, clientid, mailbox).")
		flag.Usage()
		os.Exit(1)
	}

	// Initialize CSV logging
	initCSVLog(*action)
	defer closeCSVLog()

	// Configure proxy if specified
	// Go's http package automatically uses HTTP_PROXY/HTTPS_PROXY environment variables
	if *proxyURL != "" {
		os.Setenv("HTTP_PROXY", *proxyURL)
		os.Setenv("HTTPS_PROXY", *proxyURL)
		fmt.Printf("Using proxy: %s\n", *proxyURL)
	}

	// 2. Setup Authentication
	cred, err := getCredential(*tenantID, *clientID, *secret, *pfxPath, *pfxPass, *thumbprint)
	if err != nil {
		log.Fatalf("Authentication setup failed: %v", err)
	}

	// Get and display token information if verbose
	if verboseMode {
		ctx := context.Background()
		token, err := cred.GetToken(ctx, policy.TokenRequestOptions{
			Scopes: []string{"https://graph.microsoft.com/.default"},
		})
		if err != nil {
			logVerbose("Warning: Could not retrieve token for verbose display: %v", err)
		} else {
			printTokenInfo(token)
		}
	}

	// Scopes for Application Permissions usually are https://graph.microsoft.com/.default
	client, err := msgraphsdk.NewGraphServiceClientWithCredentials(cred, []string{"https://graph.microsoft.com/.default"})
	if err != nil {
		log.Fatalf("Graph client initialization failed: %v", err)
	}

	if verboseMode {
		logVerbose("Graph SDK client initialized successfully")
		logVerbose("Target scope: https://graph.microsoft.com/.default")
	}

	ctx := context.Background()

	// 3. Execute Actions based on flags
	switch *action {
	case "getevents":
		listEvents(ctx, client, *mailbox)
	case "sendmail":
		to := parseList(*toRaw)
		cc := parseList(*ccRaw)
		bcc := parseList(*bccRaw)

		// If no recipients specified at all, default 'to' to the sender mailbox
		if len(to) == 0 && len(cc) == 0 && len(bcc) == 0 {
			to = []string{*mailbox}
		}

		sendEmail(ctx, client, *mailbox, to, cc, bcc, *subject, *body)
	case "sendinvite":
		createInvite(ctx, client, *mailbox, *inviteSubject, *startTime, *endTime)
	case "getinbox":
		listInbox(ctx, client, *mailbox)
	default:
		fmt.Printf("Unknown action: %s\n", *action)
	}
}

func parseList(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	var result []string
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func getCredential(tenantID, clientID, secret, pfxPath, pfxPass, thumbprint string) (azcore.TokenCredential, error) {
	// 1. Client Secret
	if secret != "" {
		logVerbose("Authentication method: Client Secret")
		logVerbose("Creating ClientSecretCredential...")
		return azidentity.NewClientSecretCredential(tenantID, clientID, secret, nil)
	}

	// 2. PFX File
	if pfxPath != "" {
		logVerbose("Authentication method: PFX Certificate File")
		logVerbose("PFX file path: %s", pfxPath)
		pfxData, err := os.ReadFile(pfxPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read PFX file: %w", err)
		}
		logVerbose("PFX file read successfully (%d bytes)", len(pfxData))
		return createCertCredential(tenantID, clientID, pfxData, pfxPass)
	}

	// 3. Windows Cert Store (Thumbprint)
	if thumbprint != "" {
		logVerbose("Authentication method: Windows Certificate Store")
		logVerbose("Certificate thumbprint: %s", thumbprint)
		logVerbose("Exporting certificate from CurrentUser\\My store...")
		pfxData, tempPass, err := exportCertFromStore(thumbprint)
		if err != nil {
			return nil, fmt.Errorf("failed to export cert from store: %w", err)
		}
		logVerbose("Certificate exported successfully (%d bytes)", len(pfxData))
		return createCertCredential(tenantID, clientID, pfxData, tempPass)
	}

	return nil, fmt.Errorf("no valid authentication method provided (use -secret, -pfx, or -thumbprint")
}

func createCertCredential(tenantID, clientID string, pfxData []byte, password string) (*azidentity.ClientCertificateCredential, error) {
	// Decode PFX using pkcs12
	// pkcs12.Decode returns the first private key and certificate.
	key, cert, err := pkcs12.Decode(pfxData, password)
	if err != nil {
		// Fallback: Sometimes pkcs12.Decode fails if the PFX has complex structure.
		// We could try ToPEM logic here if needed, but Decode is usually sufficient for standard exports.
		return nil, fmt.Errorf("failed to decode PFX: %w", err)
	}

	// Ensure key is a crypto.PrivateKey (it should be)
	privKey, ok := key.(crypto.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("decoded key is not a valid crypto.PrivateKey")
	}

	// Options
	opts := &azidentity.ClientCertificateCredentialOptions{
		SendCertificateChain: true,
	}

	// Create Credential
	// azidentity expects a slice of certs.
	certs := []*x509.Certificate{cert}

	return azidentity.NewClientCertificateCredential(tenantID, clientID, certs, privKey, opts)
}

// ... Rest of the functions (listEvents, sendEmail, createInvite) ...

func listEvents(ctx context.Context, client *msgraphsdk.GraphServiceClient, mailbox string) {
	logVerbose("Calling Graph API: GET /users/%s/events", mailbox)
	result, err := client.Users().ByUserId(mailbox).Events().Get(ctx, nil)
	if err != nil {
		var oDataError *odataerrors.ODataError
		if errors.As(err, &oDataError) {
			log.Printf("OData Error:")
			if oDataError.GetErrorEscaped() != nil {
				log.Printf("  Code: %s", *oDataError.GetErrorEscaped().GetCode())
				log.Printf("  Message: %s", *oDataError.GetErrorEscaped().GetMessage())
			}
		}
		log.Fatalf("Error fetching calendar for %s: %+v", mailbox, err)
	}

	events := result.GetValue()
	eventCount := len(events)

	logVerbose("API response received: %d events", eventCount)
	fmt.Printf("Upcoming events for %s:\n", mailbox)

	if eventCount == 0 {
		fmt.Println("No events found.")
		// Log summary entry when no events found
		writeCSVRow([]string{"getevents", "Success", mailbox, fmt.Sprintf("No events found (0 events)"), "N/A"})
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
			writeCSVRow([]string{"getevents", "Success", mailbox, subject, id})
		}
		// Log summary entry after all events
		fmt.Printf("\nTotal events retrieved: %d\n", eventCount)
		writeCSVRow([]string{"getevents", "Success", mailbox, fmt.Sprintf("Retrieved %d event(s)", eventCount), "SUMMARY"})
	}
}

func sendEmail(ctx context.Context, client *msgraphsdk.GraphServiceClient, senderMailbox string, to, cc, bcc []string, subject, content string) {
	message := models.NewMessage()

	// Set Subject
	message.SetSubject(&subject)

	body := models.NewItemBody()
	body.SetContent(&content)

	// Set body type to Text
	contentType := models.TEXT_BODYTYPE
	body.SetContentType(&contentType)

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

	requestBody := users.NewItemSendMailPostRequestBody()
	requestBody.SetMessage(message)

	logVerbose("Calling Graph API: POST /users/%s/sendMail", senderMailbox)
	logVerbose("Email details - To: %v, CC: %v, BCC: %v", to, cc, bcc)
	err := client.Users().ByUserId(senderMailbox).SendMail().Post(ctx, requestBody, nil)

	status := "Success"
	if err != nil {
		log.Printf("Error sending mail: %v", err)
		status = fmt.Sprintf("Error: %v", err)
	} else {
		logVerbose("Email sent successfully via Graph API")
		fmt.Printf("Email sent successfully from %s.\nTo: %v\nCc: %v\nBcc: %v\nSubject: %s\n", senderMailbox, to, cc, bcc, subject)
	}

	// Write to CSV
	toStr := strings.Join(to, "; ")
	ccStr := strings.Join(cc, "; ")
	bccStr := strings.Join(bcc, "; ")
	writeCSVRow([]string{"sendmail", status, senderMailbox, toStr, ccStr, bccStr, subject})
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

func createInvite(ctx context.Context, client *msgraphsdk.GraphServiceClient, mailbox, subject, startTimeStr, endTimeStr string) {
	event := models.NewEvent()
	event.SetSubject(&subject)

	// Parse start time, default to now if not provided
	var startTime time.Time
	var err error
	if startTimeStr == "" {
		startTime = time.Now()
	} else {
		startTime, err = time.Parse(time.RFC3339, startTimeStr)
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
		endTime, err = time.Parse(time.RFC3339, endTimeStr)
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
	logVerbose("Calling Graph API: POST /users/%s/events", mailbox)
	logVerbose("Calendar invite - Subject: %s, Start: %s, End: %s", subject, startTime.Format(time.RFC3339), endTime.Format(time.RFC3339))
	createdEvent, err := client.Users().ByUserId(mailbox).Events().Post(ctx, event, nil)

	status := "Success"
	eventID := "N/A"
	if err != nil {
		log.Printf("Error creating invite: %v", err)
		status = fmt.Sprintf("Error: %v", err)
	} else {
		if createdEvent.GetId() != nil {
			eventID = *createdEvent.GetId()
		}
		logVerbose("Calendar event created successfully via Graph API")
		logVerbose("Event ID: %s", eventID)
		fmt.Printf("Calendar invitation created in mailbox: %s\n", mailbox)
		fmt.Printf("Subject: %s\n", subject)
		fmt.Printf("Start: %s\n", startTime.Format("2006-01-02 15:04:05 MST"))
		fmt.Printf("End: %s\n", endTime.Format("2006-01-02 15:04:05 MST"))
		fmt.Printf("Event ID: %s\n", eventID)
	}

	// Write to CSV
	writeCSVRow([]string{"sendinvite", status, mailbox, subject, startTime.Format(time.RFC3339), endTime.Format(time.RFC3339), eventID})
}

func listInbox(ctx context.Context, client *msgraphsdk.GraphServiceClient, mailbox string) {
	// Configure request to get top 10 messages ordered by received date
	requestConfig := &users.ItemMessagesRequestBuilderGetRequestConfiguration{
		QueryParameters: &users.ItemMessagesRequestBuilderGetQueryParameters{
			Top:     Int32Ptr(10),
			Orderby: []string{"receivedDateTime DESC"},
			Select:  []string{"subject", "receivedDateTime", "from", "toRecipients"},
		},
	}

	logVerbose("Calling Graph API: GET /users/%s/messages?$top=10&$orderby=receivedDateTime DESC", mailbox)
	result, err := client.Users().ByUserId(mailbox).Messages().Get(ctx, requestConfig)
	if err != nil {
		log.Fatalf("Error fetching inbox for %s: %v", mailbox, err)
	}

	messages := result.GetValue()
	messageCount := len(messages)

	logVerbose("API response received: %d messages", messageCount)
	fmt.Printf("Newest 10 messages in inbox for %s:\n\n", mailbox)

	if messageCount == 0 {
		fmt.Println("No messages found.")
		// Log summary entry when no messages found
		writeCSVRow([]string{"getinbox", "Success", mailbox, "No messages found (0 messages)", "N/A", "N/A", "N/A"})
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
			writeCSVRow([]string{"getinbox", "Success", mailbox, subject, sender, recipientStr, receivedDate})
		}
		// Log summary entry after all messages
		fmt.Printf("Total messages retrieved: %d\n", messageCount)
		writeCSVRow([]string{"getinbox", "Success", mailbox, fmt.Sprintf("Retrieved %d message(s)", messageCount), "SUMMARY", "SUMMARY", "SUMMARY"})
	}
}

// Helper function to create int32 pointer
func Int32Ptr(i int32) *int32 {
	return &i
}

// Initialize CSV log file
func initCSVLog(action string) {
	// Get temp directory
	tempDir := os.TempDir()

	// Create filename with current date
	dateStr := time.Now().Format("2006-01-02")
	fileName := fmt.Sprintf("_msgraphgolangtestingtool_%s.csv", dateStr)
	filePath := filepath.Join(tempDir, fileName)

	// Open or create file (append mode)
	var err error
	csvFile, err = os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("Warning: Could not create CSV log file: %v", err)
		return
	}

	csvWriter = csv.NewWriter(csvFile)

	// Check if file is new (empty) to write headers
	fileInfo, _ := csvFile.Stat()
	if fileInfo.Size() == 0 {
		// Write header based on action type
		writeCSVHeader(action)
	}

	fmt.Printf("Logging to: %s\n\n", filePath)
}

// Write CSV header based on action type
func writeCSVHeader(action string) {
	var header []string
	switch action {
	case "getevents":
		header = []string{"Timestamp", "Action", "Status", "Mailbox", "Event Subject", "Event ID"}
	case "sendmail":
		header = []string{"Timestamp", "Action", "Status", "Mailbox", "To", "CC", "BCC", "Subject"}
	case "sendinvite":
		header = []string{"Timestamp", "Action", "Status", "Mailbox", "Subject", "Start Time", "End Time", "Event ID"}
	case "getinbox":
		header = []string{"Timestamp", "Action", "Status", "Mailbox", "Subject", "From", "To", "Received DateTime"}
	default:
		header = []string{"Timestamp", "Action", "Status", "Details"}
	}
	csvWriter.Write(header)
	csvWriter.Flush()
}

// Close CSV log file
func closeCSVLog() {
	if csvWriter != nil {
		csvWriter.Flush()
	}
	if csvFile != nil {
		csvFile.Close()
	}
}

// Write a row to CSV
func writeCSVRow(row []string) {
	if csvWriter != nil {
		// Prepend timestamp
		timestamp := time.Now().Format("2006-01-02 15:04:05")
		fullRow := append([]string{timestamp}, row...)
		csvWriter.Write(fullRow)
		csvWriter.Flush()
	}
}

// Verbose logging helper
func logVerbose(format string, args ...interface{}) {
	if verboseMode {
		prefix := "[VERBOSE] "
		fmt.Printf(prefix+format+"\n", args...)
	}
}

// Print verbose configuration summary
func printVerboseConfig(tenantID, clientID, secret, pfxPath, thumbprint, mailbox, action, proxyURL, to, cc, bcc, subject, body, inviteSubject, startTime, endTime string) {
	fmt.Println("========================================")
	fmt.Println("VERBOSE MODE ENABLED")
	fmt.Println("========================================")
	fmt.Println()

	// Display environment variables
	fmt.Println("Environment Variables (MSGRAPH*):")
	fmt.Println("----------------------------------")
	envVars := getEnvVariables()
	if len(envVars) == 0 {
		fmt.Println("  (no MSGRAPH environment variables set)")
	} else {
		for key, value := range envVars {
			// Mask sensitive values
			displayValue := value
			if key == "MSGRAPHSECRET" || key == "MSGRAPHPFXPASS" {
				displayValue = maskSecret(value)
			}
			fmt.Printf("  %s = %s\n", key, displayValue)
		}
	}
	fmt.Println()

	fmt.Println("Final Configuration (after env vars + flags):")
	fmt.Println("----------------------------------------------")
	fmt.Printf("Version: %s\n", version)
	fmt.Printf("Tenant ID: %s\n", tenantID)
	fmt.Printf("Client ID: %s\n", clientID)
	fmt.Printf("Mailbox: %s\n", mailbox)
	fmt.Printf("Action: %s\n", action)

	// Authentication method
	fmt.Println()
	fmt.Println("Authentication:")
	if secret != "" {
		fmt.Println("  Method: Client Secret")
		// Mask the secret but show length
		fmt.Printf("  Secret: %s (length: %d)\n", maskSecret(secret), len(secret))
	} else if pfxPath != "" {
		fmt.Println("  Method: PFX Certificate")
		fmt.Printf("  PFX Path: %s\n", pfxPath)
		if pfxPath != "" {
			fmt.Println("  PFX Password: ******** (provided)")
		}
	} else if thumbprint != "" {
		fmt.Println("  Method: Windows Certificate Store")
		fmt.Printf("  Thumbprint: %s\n", thumbprint)
	}

	// Network configuration
	if proxyURL != "" {
		fmt.Println()
		fmt.Println("Network Configuration:")
		fmt.Printf("  Proxy: %s\n", proxyURL)
	}

	// Action-specific parameters
	fmt.Println()
	fmt.Println("Action Parameters:")
	switch action {
	case "sendmail":
		fmt.Printf("  To: %s\n", ifEmpty(to, "(defaults to mailbox)"))
		fmt.Printf("  CC: %s\n", ifEmpty(cc, "(none)"))
		fmt.Printf("  BCC: %s\n", ifEmpty(bcc, "(none)"))
		fmt.Printf("  Subject: %s\n", subject)
		fmt.Printf("  Body: %s\n", truncate(body, 60))
	case "sendinvite":
		fmt.Printf("  Invite Subject: %s\n", inviteSubject)
		fmt.Printf("  Start Time: %s\n", ifEmpty(startTime, "(now)"))
		fmt.Printf("  End Time: %s\n", ifEmpty(endTime, "(start + 1 hour)"))
	case "getevents", "getinbox":
		fmt.Println("  (no additional parameters)")
	}

	fmt.Println()
	fmt.Println("========================================")
	fmt.Println()
}

// Print token information
func printTokenInfo(token azcore.AccessToken) {
	fmt.Println()
	fmt.Println("Token Information:")
	fmt.Println("------------------")
	fmt.Printf("Token acquired successfully\n")
	fmt.Printf("Expires at: %s\n", token.ExpiresOn.Format("2006-01-02 15:04:05 MST"))

	// Calculate time until expiration
	timeUntilExpiry := time.Until(token.ExpiresOn)
	fmt.Printf("Valid for: %s\n", timeUntilExpiry.Round(time.Second))

	// Show truncated token (first and last 10 characters for verification)
	tokenStr := token.Token
	if len(tokenStr) > 40 {
		fmt.Printf("Token (truncated): %s...%s\n", tokenStr[:20], tokenStr[len(tokenStr)-20:])
		fmt.Printf("Token length: %d characters\n", len(tokenStr))
	} else {
		fmt.Printf("Token: %s\n", tokenStr)
	}

	fmt.Println()
}

// Helper: Mask secret for display
func maskSecret(secret string) string {
	if len(secret) <= 8 {
		return "********"
	}
	// Show first 4 and last 4 characters
	return secret[:4] + "********" + secret[len(secret)-4:]
}

// Helper: Return default string if empty
func ifEmpty(s, defaultVal string) string {
	if s == "" {
		return defaultVal
	}
	return s
}

// Helper: Truncate string with ellipsis
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// Get all MSGRAPH environment variables
func getEnvVariables() map[string]string {
	envVars := make(map[string]string)

	// List of all MSGRAPH environment variables
	msgraphEnvVars := []string{
		"MSGRAPHTENANTID",
		"MSGRAPHCLIENTID",
		"MSGRAPHSECRET",
		"MSGRAPHPFX",
		"MSGRAPHPFXPASS",
		"MSGRAPHTHUMBPRINT",
		"MSGRAPHMAILBOX",
		"MSGRAPHTO",
		"MSGRAPHCC",
		"MSGRAPHBCC",
		"MSGRAPHSUBJECT",
		"MSGRAPHBODY",
		"MSGRAPHINVITESUBJECT",
		"MSGRAPHSTART",
		"MSGRAPHEND",
		"MSGRAPHACTION",
		"MSGRAPHPROXY",
	}

	for _, envVar := range msgraphEnvVars {
		if value := os.Getenv(envVar); value != "" {
			envVars[envVar] = value
		}
	}

	return envVars
}

//END
