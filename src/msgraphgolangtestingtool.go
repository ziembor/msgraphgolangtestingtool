package main

import (
	"context"
	"crypto"
	"crypto/x509"
	_ "embed"
	"encoding/base64"
	"encoding/csv"
	"errors"
	"flag"
	"fmt"
	"log"
	"mime"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
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

//go:embed VERSION
var versionRaw string
var version = strings.TrimSpace(versionRaw)

// Action constants
const (
	ActionGetEvents  = "getevents"
	ActionSendMail   = "sendmail"
	ActionSendInvite = "sendinvite"
	ActionGetInbox   = "getinbox"
)

// Status constants
const (
	StatusSuccess = "Success"
	StatusError   = "Error"
)

// Config holds application configuration
type Config struct {
	VerboseMode bool
}

// CSVLogger handles CSV logging operations
type CSVLogger struct {
	writer *csv.Writer
	file   *os.File
	action string
}

// NewCSVLogger creates a new CSV logger for the specified action
func NewCSVLogger(action string) (*CSVLogger, error) {
	// Get temp directory
	tempDir := os.TempDir()

	// Create filename with action and current date
	dateStr := time.Now().Format("2006-01-02")
	fileName := fmt.Sprintf("_msgraphgolangtestingtool_%s_%s.csv", action, dateStr)
	filePath := filepath.Join(tempDir, fileName)

	// Open or create file (append mode)
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("could not create CSV log file: %w", err)
	}

	logger := &CSVLogger{
		writer: csv.NewWriter(file),
		file:   file,
		action: action,
	}

	// Check if file is new (empty) to write headers
	fileInfo, err := file.Stat()
	if err != nil {
		log.Printf("Warning: Could not stat CSV file: %v", err)
	} else if fileInfo.Size() == 0 {
		// Write header based on action type
		logger.writeHeader()
	}

	fmt.Printf("Logging to: %s\n\n", filePath)
	return logger, nil
}

// writeHeader writes the CSV header based on action type
func (l *CSVLogger) writeHeader() {
	var header []string
	switch l.action {
	case ActionGetEvents:
		header = []string{"Timestamp", "Action", "Status", "Mailbox", "Event Subject", "Event ID"}
	case ActionSendMail:
		header = []string{"Timestamp", "Action", "Status", "Mailbox", "To", "CC", "BCC", "Subject", "Body Type", "Attachments"}
	case ActionSendInvite:
		header = []string{"Timestamp", "Action", "Status", "Mailbox", "Subject", "Start Time", "End Time", "Event ID"}
	case ActionGetInbox:
		header = []string{"Timestamp", "Action", "Status", "Mailbox", "Subject", "From", "To", "Received DateTime"}
	default:
		header = []string{"Timestamp", "Action", "Status", "Details"}
	}
	l.writer.Write(header)
	l.writer.Flush()
}

// WriteRow writes a row to the CSV file
func (l *CSVLogger) WriteRow(row []string) {
	if l.writer != nil {
		// Prepend timestamp
		timestamp := time.Now().Format("2006-01-02 15:04:05")
		fullRow := append([]string{timestamp}, row...)
		l.writer.Write(fullRow)
		l.writer.Flush()
	}
}

// Close closes the CSV file
func (l *CSVLogger) Close() error {
	if l.writer != nil {
		l.writer.Flush()
	}
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

// stringSlice is a custom flag type for comma-separated lists
type stringSlice []string

// String implements the flag.Value interface
func (s *stringSlice) String() string {
	if s == nil {
		return ""
	}
	return strings.Join(*s, ",")
}

// Set implements the flag.Value interface
func (s *stringSlice) Set(value string) error {
	if value == "" {
		*s = nil
		return nil
	}
	parts := strings.Split(value, ",")
	var result []string
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	*s = result
	return nil
}

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
		"subject":        "MSGRAPHSUBJECT",
		"body":           "MSGRAPHBODY",
		"bodyHTML":       "MSGRAPHBODYHTML",
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

// applyEnvVarsToSlice applies environment variable values to stringSlice flags
func applyEnvVarsToSlice(flagName string, slice *stringSlice, envName string) {
	// Check if flag was explicitly provided via command line
	flagProvided := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == flagName {
			flagProvided = true
		}
	})

	// If flag was not provided via command line, check environment variable
	if !flagProvided {
		if envValue := os.Getenv(envName); envValue != "" {
			slice.Set(envValue)
		}
	}
}

func main() {
	if err := run(); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}

func run() error {
	// Setup signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt signals (Ctrl+C, SIGTERM)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\n\nReceived interrupt signal. Shutting down gracefully...")
		cancel()
	}()

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

	// Recipient flags (using custom stringSlice type)
	var to, cc, bcc, attachmentFiles stringSlice
	flag.Var(&to, "to", "Comma-separated list of TO recipients (optional, defaults to mailbox if empty)")
	flag.Var(&cc, "cc", "Comma-separated list of CC recipients")
	flag.Var(&bcc, "bcc", "Comma-separated list of BCC recipients")

	// Email content flags
	subject := flag.String("subject", "Automated Tool Notification", "Subject of the email")
	body := flag.String("body", "It's a test message, please ignore", "Body content of the email (text)")
	bodyHTML := flag.String("bodyHTML", "", "HTML body content of the email (optional, creates multipart message if both -body and -bodyHTML are provided)")
	flag.Var(&attachmentFiles, "attachments", "Comma-separated list of file paths to attach")

	// Calendar invite flags
	inviteSubject := flag.String("invite-subject", "System Sync", "Subject of the calendar invite")
	startTime := flag.String("start", "", "Start time for calendar invite (RFC3339 format, e.g., 2026-01-15T14:00:00Z). Defaults to now if empty")
	endTime := flag.String("end", "", "End time for calendar invite (RFC3339 format, e.g., 2026-01-15T15:00:00Z). Defaults to 1 hour after start if empty")

	// Proxy configuration
	proxyURL := flag.String("proxy", "", "HTTP/HTTPS proxy URL (e.g., http://proxy.example.com:8080)")

	// Verbose mode
	verbose := flag.Bool("verbose", false, "Enable verbose output (shows configuration, tokens, API details)")

	// Count for getevents and getinbox
	count := flag.Int("count", 3, "Number of items to retrieve for getevents and getinbox actions (default: 3)")

	action := flag.String("action", "getevents", "Action to perform: getevents, sendmail, sendinvite, getinbox")
	flag.Parse()

	// Create configuration
	config := &Config{
		VerboseMode: *verbose,
	}

	// Check version flag
	if *showVersion {
		fmt.Printf("Microsoft Graph Golang Testing Tool - Version %s\n", version)
		return nil
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
		"MSGRAPHSUBJECT":       subject,
		"MSGRAPHBODY":          body,
		"MSGRAPHBODYHTML":      bodyHTML,
		"MSGRAPHINVITESUBJECT": inviteSubject,
		"MSGRAPHSTART":         startTime,
		"MSGRAPHEND":           endTime,
		"MSGRAPHACTION":        action,
		"MSGRAPHPROXY":         proxyURL,
	})

	// Apply environment variables for stringSlice flags
	applyEnvVarsToSlice("to", &to, "MSGRAPHTO")
	applyEnvVarsToSlice("cc", &cc, "MSGRAPHCC")
	applyEnvVarsToSlice("bcc", &bcc, "MSGRAPHBCC")
	applyEnvVarsToSlice("attachments", &attachmentFiles, "MSGRAPHATTACHMENTS")

	// Apply MSGRAPHCOUNT environment variable if flag wasn't provided
	countFlagProvided := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "count" {
			countFlagProvided = true
		}
	})
	if !countFlagProvided {
		if envCount := os.Getenv("MSGRAPHCOUNT"); envCount != "" {
			if parsedCount, err := strconv.Atoi(envCount); err == nil && parsedCount > 0 {
				*count = parsedCount
			}
		}
	}

	// Print verbose configuration if enabled
	if config.VerboseMode {
		printVerboseConfig(*tenantID, *clientID, *secret, *pfxPath, *thumbprint, *mailbox, *action, *proxyURL, to.String(), cc.String(), bcc.String(), *subject, *body, *bodyHTML, attachmentFiles.String(), *inviteSubject, *startTime, *endTime)
	}

	// Validation
	if *tenantID == "" || *clientID == "" || *mailbox == "" {
		fmt.Println("Error: Missing required parameters (tenantid, clientid, mailbox).")
		flag.Usage()
		os.Exit(1)
	}

	// Initialize CSV logging
	logger, err := NewCSVLogger(*action)
	if err != nil {
		log.Printf("Warning: Could not initialize CSV logging: %v", err)
		logger = nil // Continue without logging
	}
	if logger != nil {
		defer logger.Close()
	}

	// Configure proxy if specified
	// Go's http package automatically uses HTTP_PROXY/HTTPS_PROXY environment variables
	if *proxyURL != "" {
		os.Setenv("HTTP_PROXY", *proxyURL)
		os.Setenv("HTTPS_PROXY", *proxyURL)
		fmt.Printf("Using proxy: %s\n", *proxyURL)
	}

	// 2. Setup Authentication
	cred, err := getCredential(*tenantID, *clientID, *secret, *pfxPath, *pfxPass, *thumbprint, config)
	if err != nil {
		return fmt.Errorf("authentication setup failed: %w", err)
	}

	// Get and display token information if verbose
	if config.VerboseMode {
		token, err := cred.GetToken(ctx, policy.TokenRequestOptions{
			Scopes: []string{"https://graph.microsoft.com/.default"},
		})
		if err != nil {
			logVerbose(config.VerboseMode, "Warning: Could not retrieve token for verbose display: %v", err)
		} else {
			printTokenInfo(token)
		}
	}

	// Scopes for Application Permissions usually are https://graph.microsoft.com/.default
	client, err := msgraphsdk.NewGraphServiceClientWithCredentials(cred, []string{"https://graph.microsoft.com/.default"})
	if err != nil {
		return fmt.Errorf("graph client initialization failed: %w", err)
	}

	if config.VerboseMode {
		logVerbose(config.VerboseMode, "Graph SDK client initialized successfully")
		logVerbose(config.VerboseMode, "Target scope: https://graph.microsoft.com/.default")
	}

	// 3. Execute Actions based on flags
	switch *action {
	case ActionGetEvents:
		if err := listEvents(ctx, client, *mailbox, *count, config, logger); err != nil {
			return fmt.Errorf("failed to list events: %w", err)
		}
	case ActionSendMail:
		// If no recipients specified at all, default 'to' to the sender mailbox
		if len(to) == 0 && len(cc) == 0 && len(bcc) == 0 {
			to = []string{*mailbox}
		}

		sendEmail(ctx, client, *mailbox, to, cc, bcc, *subject, *body, *bodyHTML, attachmentFiles, config, logger)
	case ActionSendInvite:
		createInvite(ctx, client, *mailbox, *inviteSubject, *startTime, *endTime, config, logger)
	case ActionGetInbox:
		if err := listInbox(ctx, client, *mailbox, *count, config, logger); err != nil {
			return fmt.Errorf("failed to list inbox: %w", err)
		}
	default:
		return fmt.Errorf("unknown action: %s", *action)
	}

	return nil
}

func getCredential(tenantID, clientID, secret, pfxPath, pfxPass, thumbprint string, config *Config) (azcore.TokenCredential, error) {
	// 1. Client Secret
	if secret != "" {
		logVerbose(config.VerboseMode, "Authentication method: Client Secret")
		logVerbose(config.VerboseMode, "Creating ClientSecretCredential...")
		return azidentity.NewClientSecretCredential(tenantID, clientID, secret, nil)
	}

	// 2. PFX File
	if pfxPath != "" {
		logVerbose(config.VerboseMode, "Authentication method: PFX Certificate File")
		logVerbose(config.VerboseMode, "PFX file path: %s", pfxPath)
		pfxData, err := os.ReadFile(pfxPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read PFX file: %w", err)
		}
		logVerbose(config.VerboseMode, "PFX file read successfully (%d bytes)", len(pfxData))
		return createCertCredential(tenantID, clientID, pfxData, pfxPass)
	}

	// 3. Windows Cert Store (Thumbprint)
	if thumbprint != "" {
		logVerbose(config.VerboseMode, "Authentication method: Windows Certificate Store")
		logVerbose(config.VerboseMode, "Certificate thumbprint: %s", thumbprint)
		logVerbose(config.VerboseMode, "Exporting certificate from CurrentUser\\My store...")
		pfxData, tempPass, err := exportCertFromStore(thumbprint)
		if err != nil {
			return nil, fmt.Errorf("failed to export cert from store: %w", err)
		}
		logVerbose(config.VerboseMode, "Certificate exported successfully (%d bytes)", len(pfxData))
		return createCertCredential(tenantID, clientID, pfxData, tempPass)
	}

	return nil, fmt.Errorf("no valid authentication method provided (use -secret, -pfx, or -thumbprint)")
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

func listEvents(ctx context.Context, client *msgraphsdk.GraphServiceClient, mailbox string, count int, config *Config, logger *CSVLogger) error {
	// Configure request to get top N events
	requestConfig := &users.ItemEventsRequestBuilderGetRequestConfiguration{
		QueryParameters: &users.ItemEventsRequestBuilderGetQueryParameters{
			Top: Int32Ptr(int32(count)),
		},
	}

	logVerbose(config.VerboseMode, "Calling Graph API: GET /users/%s/events?$top=%d", mailbox, count)
	result, err := client.Users().ByUserId(mailbox).Events().Get(ctx, requestConfig)
	if err != nil {
		var oDataError *odataerrors.ODataError
		if errors.As(err, &oDataError) {
			log.Printf("OData Error:")
			if oDataError.GetErrorEscaped() != nil {
				log.Printf("  Code: %s", *oDataError.GetErrorEscaped().GetCode())
				log.Printf("  Message: %s", *oDataError.GetErrorEscaped().GetMessage())
			}
		}
		return fmt.Errorf("error fetching calendar for %s: %w", mailbox, err)
	}

	events := result.GetValue()
	eventCount := len(events)

	logVerbose(config.VerboseMode, "API response received: %d events", eventCount)
	fmt.Printf("Upcoming events for %s:\n", mailbox)

	if eventCount == 0 {
		fmt.Println("No events found.")
		// Log summary entry when no events found
		if logger != nil {
			logger.WriteRow([]string{ActionGetEvents, StatusSuccess, mailbox, fmt.Sprintf("No events found (0 events)"), "N/A"})
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
		log.Printf("Error sending mail: %v", err)
		status = fmt.Sprintf("%s: %v", StatusError, err)
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

func createInvite(ctx context.Context, client *msgraphsdk.GraphServiceClient, mailbox, subject, startTimeStr, endTimeStr string, config *Config, logger *CSVLogger) {
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
	logVerbose(config.VerboseMode, "Calling Graph API: POST /users/%s/events", mailbox)
	logVerbose(config.VerboseMode, "Calendar invite - Subject: %s, Start: %s, End: %s", subject, startTime.Format(time.RFC3339), endTime.Format(time.RFC3339))
	createdEvent, err := client.Users().ByUserId(mailbox).Events().Post(ctx, event, nil)

	status := StatusSuccess
	eventID := "N/A"
	if err != nil {
		log.Printf("Error creating invite: %v", err)
		status = fmt.Sprintf("%s: %v", StatusError, err)
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
	result, err := client.Users().ByUserId(mailbox).Messages().Get(ctx, requestConfig)
	if err != nil {
		return fmt.Errorf("error fetching inbox for %s: %w", mailbox, err)
	}

	messages := result.GetValue()
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

// Helper function to create int32 pointer
func Int32Ptr(i int32) *int32 {
	return &i
}

// Verbose logging helper
func logVerbose(verbose bool, format string, args ...interface{}) {
	if verbose {
		prefix := "[VERBOSE] "
		fmt.Printf(prefix+format+"\n", args...)
	}
}

// Print verbose configuration summary
func printVerboseConfig(tenantID, clientID, secret, pfxPath, thumbprint, mailbox, action, proxyURL, to, cc, bcc, subject, body, bodyHTML, attachments, inviteSubject, startTime, endTime string) {
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
		// Sort keys for consistent output
		keys := make([]string, 0, len(envVars))
		for k := range envVars {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, key := range keys {
			value := envVars[key]
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
		fmt.Println("  PFX Password: ******** (provided)")
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
		fmt.Printf("  Body (Text): %s\n", truncate(body, 60))
		fmt.Printf("  Body (HTML): %s\n", ifEmpty(truncate(bodyHTML, 60), "(none)"))
		fmt.Printf("  Attachments: %s\n", ifEmpty(attachments, "(none)"))
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

	// Show truncated token (always truncate for security, even short tokens)
	tokenStr := token.Token
	if len(tokenStr) > 40 {
		fmt.Printf("Token (truncated): %s...%s\n", tokenStr[:20], tokenStr[len(tokenStr)-20:])
	} else {
		// Even short tokens should be masked for security
		maxLen := 10
		if len(tokenStr) < maxLen {
			maxLen = len(tokenStr)
		}
		fmt.Printf("Token (truncated): %s...\n", tokenStr[:maxLen])
	}
	fmt.Printf("Token length: %d characters\n", len(tokenStr))

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
		"MSGRAPHBODYHTML",
		"MSGRAPHATTACHMENTS",
		"MSGRAPHINVITESUBJECT",
		"MSGRAPHSTART",
		"MSGRAPHEND",
		"MSGRAPHACTION",
		"MSGRAPHPROXY",
		"MSGRAPHCOUNT",
	}

	for _, envVar := range msgraphEnvVars {
		if value := os.Getenv(envVar); value != "" {
			envVars[envVar] = value
		}
	}

	return envVars
}

//END
