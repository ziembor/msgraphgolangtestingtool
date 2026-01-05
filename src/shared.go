// Package main provides shared business logic for the Microsoft Graph EXO Mails/Calendar Golang Testing Tool.
// This file contains all code shared between the main CLI application and integration tests.
//
// NO BUILD TAGS - This file is compiled in all build modes.
package main

import (
	"context"
	"crypto"
	"crypto/x509"
	_ "embed"
	"encoding/base64"
	"encoding/csv"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"mime"
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
	"software.sslmate.com/src/go-pkcs12"
)

//go:embed VERSION
var versionRaw string
var version = strings.TrimSpace(versionRaw)

// Action constants
const (
	ActionGetEvents    = "getevents"
	ActionSendMail     = "sendmail"
	ActionSendInvite   = "sendinvite"
	ActionGetInbox     = "getinbox"
	ActionGetSchedule  = "getschedule"
)

// Status constants
const (
	StatusSuccess = "Success"
	StatusError   = "Error"
)

// Config holds all application configuration including command-line flags,
// environment variables, and runtime state. This centralized configuration
// structure simplifies passing configuration between functions and improves
// testability.
type Config struct {
	// Core configuration
	ShowVersion bool   // Display version information and exit
	TenantID    string // Azure AD Tenant ID (GUID format)
	ClientID    string // Application (Client) ID (GUID format)
	Mailbox     string // Target user email address
	Action      string // Operation to perform (getevents, sendmail, sendinvite, getinbox, getschedule)

	// Authentication configuration (mutually exclusive)
	Secret     string // Client Secret for authentication
	PfxPath    string // Path to .pfx certificate file
	PfxPass    string // Password for .pfx certificate file
	Thumbprint string // SHA1 thumbprint of certificate in Windows Certificate Store

	// Email recipients (using stringSlice type for comma-separated lists)
	To              stringSlice // To recipients for email
	Cc              stringSlice // CC recipients for email
	Bcc             stringSlice // BCC recipients for email
	AttachmentFiles stringSlice // File paths to attach to email

	// Email content
	Subject  string // Email subject line
	Body     string // Email body text content
	BodyHTML string // Email body HTML content (future use)

	// Calendar invite configuration
	InviteSubject string // Subject of calendar meeting invitation
	StartTime     string // Start time in RFC3339 format (e.g., 2026-01-15T14:00:00Z)
	EndTime       string // End time in RFC3339 format

	// Network configuration
	ProxyURL   string        // HTTP/HTTPS proxy URL (e.g., http://proxy.example.com:8080)
	MaxRetries int           // Maximum retry attempts for transient failures (default: 3)
	RetryDelay time.Duration // Base delay between retries in milliseconds (default: 2000ms)

	// Runtime configuration
	VerboseMode bool   // Enable verbose diagnostic output (maps to DEBUG log level)
	LogLevel    string // Logging level: DEBUG, INFO, WARN, ERROR (default: INFO)
	Count       int    // Number of items to retrieve (for getevents and getinbox actions)
}

// NewConfig creates a new Config with sensible default values.
// Command-line flags and environment variables will override these defaults.
func NewConfig() *Config {
	return &Config{
		// Default values for optional fields
		Subject:       "Automated Tool Notification",
		Body:          "It's a test message, please ignore",
		InviteSubject: "System Sync",
		Action:        ActionGetInbox,
		Count:         3,
		VerboseMode:   false,
		LogLevel:      "INFO",                  // Default: INFO level logging
		ShowVersion:   false,
		MaxRetries:    3,                       // Default: 3 retry attempts
		RetryDelay:    2000 * time.Millisecond, // Default: 2 second base delay
	}
}

// setupLogger configures the global logger based on the provided log level.
// Valid levels are: DEBUG, INFO, WARN, ERROR
// If VerboseMode is true, it overrides LogLevel to DEBUG.
// Returns a configured *slog.Logger that can be used throughout the application.
func setupLogger(config *Config) *slog.Logger {
	// Determine log level
	level := parseLogLevel(config.LogLevel)

	// Verbose mode overrides log level to DEBUG
	if config.VerboseMode {
		level = slog.LevelDebug
	}

	// Create handler options with the determined level
	opts := &slog.HandlerOptions{
		Level: level,
	}

	// Create a text handler that writes to stdout
	handler := slog.NewTextHandler(os.Stdout, opts)

	// Create and return the logger
	return slog.New(handler)
}

// parseLogLevel converts a string log level to slog.Level.
// Defaults to INFO if an invalid level is provided.
func parseLogLevel(levelStr string) slog.Level {
	switch strings.ToUpper(levelStr) {
	case "DEBUG":
		return slog.LevelDebug
	case "INFO":
		return slog.LevelInfo
	case "WARN", "WARNING":
		return slog.LevelWarn
	case "ERROR":
		return slog.LevelError
	default:
		// Default to INFO if invalid level provided
		return slog.LevelInfo
	}
}

// logDebug logs a debug message if debug level is enabled
func logDebug(logger *slog.Logger, msg string, args ...any) {
	if logger != nil {
		logger.Debug(msg, args...)
	}
}

// logInfo logs an informational message
func logInfo(logger *slog.Logger, msg string, args ...any) {
	if logger != nil {
		logger.Info(msg, args...)
	}
}

// logWarn logs a warning message
func logWarn(logger *slog.Logger, msg string, args ...any) {
	if logger != nil {
		logger.Warn(msg, args...)
	}
}

// logError logs an error message
func logError(logger *slog.Logger, msg string, args ...any) {
	if logger != nil {
		logger.Error(msg, args...)
	}
}

// CSVLogger handles CSV logging operations with periodic buffering
type CSVLogger struct {
	writer     *csv.Writer
	file       *os.File
	action     string
	rowCount   int       // Number of rows written since last flush
	lastFlush  time.Time // Time of last flush
	flushEvery int       // Flush every N rows
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
		writer:     csv.NewWriter(file),
		file:       file,
		action:     action,
		rowCount:   0,
		lastFlush:  time.Now(),
		flushEvery: 10, // Flush every 10 rows or on close
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
	case ActionGetSchedule:
		header = []string{"Timestamp", "Action", "Status", "Mailbox", "Recipient", "Check DateTime", "Availability View"}
	default:
		header = []string{"Timestamp", "Action", "Status", "Details"}
	}
	l.writer.Write(header)
	l.writer.Flush()
}

// WriteRow writes a row to the CSV file with periodic buffering
func (l *CSVLogger) WriteRow(row []string) {
	if l.writer != nil {
		// Prepend timestamp
		timestamp := time.Now().Format("2006-01-02 15:04:05")
		fullRow := append([]string{timestamp}, row...)
		l.writer.Write(fullRow)
		l.rowCount++

		// Flush every N rows or every 5 seconds
		if l.rowCount%l.flushEvery == 0 || time.Since(l.lastFlush) > 5*time.Second {
			l.writer.Flush()
			l.lastFlush = time.Now()
		}
	}
}

// Close closes the CSV file, ensuring all buffered data is flushed
func (l *CSVLogger) Close() error {
	if l.writer != nil {
		l.writer.Flush() // Always flush remaining rows on close
	}
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

// stringSlice implements the flag.Value interface for comma-separated string lists.
// This allows natural command-line syntax for lists:
//
//	-to "user1@example.com,user2@example.com"
//
// Values are automatically split on commas and trimmed of whitespace.
// Empty values and extra whitespace are automatically filtered out.
type stringSlice []string

// String returns the comma-separated string representation of the slice.
// This implements the flag.Value interface's String method.
// Returns an empty string if the slice is nil.
func (s *stringSlice) String() string {
	if s == nil {
		return ""
	}
	return strings.Join(*s, ",")
}

// Set parses a comma-separated string into a slice of trimmed strings.
// This implements the flag.Value interface's Set method.
//
// Empty strings are treated as nil slices. Comma-separated values are split,
// trimmed of whitespace, and empty items are filtered out.
//
// Example: "a, b,  , c" becomes []string{"a", "b", "c"}
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

// setupGraphClient creates credentials and initializes the Microsoft Graph SDK client
// using the authentication method specified in the configuration (client secret, PFX
// certificate, or Windows Certificate Store thumbprint).
//
// The function also retrieves and displays token information in verbose mode, including
// token expiration time and validity period.
//
// Returns the initialized GraphServiceClient and any error encountered during setup.
func setupGraphClient(ctx context.Context, config *Config, logger *slog.Logger) (*msgraphsdk.GraphServiceClient, error) {
	// Setup Authentication
	logDebug(logger, "Setting up Microsoft Graph client", "tenantID", maskGUID(config.TenantID), "clientID", maskGUID(config.ClientID))

	cred, err := getCredential(config.TenantID, config.ClientID, config.Secret, config.PfxPath, config.PfxPass, config.Thumbprint, config, logger)
	if err != nil {
		return nil, fmt.Errorf("authentication setup failed: %w", err)
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
		return nil, fmt.Errorf("graph client initialization failed: %w", err)
	}

	if config.VerboseMode {
		logVerbose(config.VerboseMode, "Graph SDK client initialized successfully")
		logVerbose(config.VerboseMode, "Target scope: https://graph.microsoft.com/.default")
	}

	return client, nil
}

func getCredential(tenantID, clientID, secret, pfxPath, pfxPass, thumbprint string, config *Config, logger *slog.Logger) (azcore.TokenCredential, error) {
	// 1. Client Secret
	if secret != "" {
		logDebug(logger, "Authentication method: Client Secret")
		logDebug(logger, "Creating ClientSecretCredential")
		return azidentity.NewClientSecretCredential(tenantID, clientID, secret, nil)
	}

	// 2. PFX File
	if pfxPath != "" {
		logDebug(logger, "Authentication method: PFX Certificate File", "path", pfxPath)
		pfxData, err := os.ReadFile(pfxPath)
		if err != nil {
			logError(logger, "Failed to read PFX file", "path", pfxPath, "error", err)
			return nil, fmt.Errorf("failed to read PFX file: %w", err)
		}
		logDebug(logger, "PFX file read successfully", "bytes", len(pfxData))
		return createCertCredential(tenantID, clientID, pfxData, pfxPass, logger)
	}

	// 3. Windows Cert Store (Thumbprint)
	if thumbprint != "" {
		logDebug(logger, "Authentication method: Windows Certificate Store", "thumbprint", thumbprint)
		logDebug(logger, "Exporting certificate from CurrentUser\\My store")
		pfxData, tempPass, err := exportCertFromStore(thumbprint)
		if err != nil {
			return nil, fmt.Errorf("failed to export cert from store: %w", err)
		}
		logDebug(logger, "Certificate exported successfully", "bytes", len(pfxData))
		return createCertCredential(tenantID, clientID, pfxData, tempPass, logger)
	}

	return nil, fmt.Errorf("no valid authentication method provided (use -secret, -pfx, or -thumbprint)")
}

func createCertCredential(tenantID, clientID string, pfxData []byte, password string, logger *slog.Logger) (*azidentity.ClientCertificateCredential, error) {
	// Decode PFX using go-pkcs12 library (supports SHA-256 and other modern algorithms)
	// pkcs12.DecodeChain returns private key and full certificate chain
	key, cert, caCerts, err := pkcs12.DecodeChain(pfxData, password)
	if err != nil {
		return nil, fmt.Errorf("failed to decode PFX: %w", err)
	}

	// Ensure key is a crypto.PrivateKey (it should be)
	privKey, ok := key.(crypto.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("decoded key is not a valid crypto.PrivateKey")
	}

	// Build certificate chain: primary cert + CA certs
	// azidentity expects a slice of certs with the leaf certificate first
	certs := []*x509.Certificate{cert}
	if len(caCerts) > 0 {
		certs = append(certs, caCerts...)
	}

	// Options - send full certificate chain for better compatibility
	opts := &azidentity.ClientCertificateCredentialOptions{
		SendCertificateChain: true,
	}

	// Create Credential
	return azidentity.NewClientCertificateCredential(tenantID, clientID, certs, privKey, opts)
}

// isRetryableError determines if an error is transient and worth retrying.
// Returns true for network timeouts, Graph API throttling (429), and service
// unavailability (503) errors.
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Check for context cancellation - never retry these
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	// Check for Azure SDK response errors
	var respErr *azcore.ResponseError
	if errors.As(err, &respErr) {
		if respErr.StatusCode == 429 || respErr.StatusCode == 503 || respErr.StatusCode == 504 {
			return true
		}
	}

	// Check error message for common transient patterns
	errMsg := strings.ToLower(err.Error())
	transientPatterns := []string{
		"timeout",
		"connection reset",
		"connection refused",
		"temporary failure",
		"try again",
		"i/o timeout",
		"no such host",
		"network is unreachable",
	}

	for _, pattern := range transientPatterns {
		if strings.Contains(errMsg, pattern) {
			return true
		}
	}

	return false
}

// enrichGraphAPIError enriches Graph API errors with additional context,
// particularly for rate limiting scenarios. It detects rate limit errors (429)
// and extracts the Retry-After header if available.
//
// For rate limit errors, it returns an enriched error message with:
// - Clear indication that rate limit was exceeded
// - Retry-After duration if provided by the API
// - Guidance on remediation (reduce request frequency or implement retry logic)
//
// For other OData errors, it logs the error code and message for debugging.
// For non-OData errors, it returns the original error unchanged.
func enrichGraphAPIError(err error, logger *CSVLogger, operation string) error {
	if err == nil {
		return nil
	}

	// Check if this is an OData error from Microsoft Graph
	var odataErr *odataerrors.ODataError
	if !errors.As(err, &odataErr) {
		// Not an OData error, return as-is
		return err
	}

	// Extract error details if available
	if odataErr.GetErrorEscaped() == nil {
		return err
	}

	errorInfo := odataErr.GetErrorEscaped()
	code := ""
	message := ""

	if errorInfo.GetCode() != nil {
		code = *errorInfo.GetCode()
	}
	if errorInfo.GetMessage() != nil {
		message = *errorInfo.GetMessage()
	}

	// Handle rate limiting (429 TooManyRequests)
	if code == "TooManyRequests" || code == "activityLimitReached" {
		log.Printf("[WARN] Graph API rate limit exceeded during %s (code: %s)", operation, code)

		// Try to extract Retry-After header
		retryAfter := ""
		if odataErr.GetResponseHeaders() != nil {
			if retryHeaders := odataErr.GetResponseHeaders().Get("Retry-After"); len(retryHeaders) > 0 {
				retryAfter = retryHeaders[0] // Get first value
				log.Printf("[INFO] Rate limit retry guidance available: retry after %s seconds", retryAfter)
			}
		}

		// Build enriched error message
		enrichedMsg := fmt.Sprintf("rate limit exceeded during %s", operation)
		if retryAfter != "" {
			enrichedMsg += fmt.Sprintf(" (retry after %s seconds)", retryAfter)
		}
		enrichedMsg += ". Consider: 1) Reducing request frequency, 2) Implementing exponential backoff, 3) Reviewing API throttling limits"

		return fmt.Errorf("%s: %w", enrichedMsg, err)
	}

	// Handle other service errors (503, 504)
	if code == "ServiceUnavailable" || code == "GatewayTimeout" {
		log.Printf("[WARN] Graph API service error during %s (code: %s, message: %s)", operation, code, message)
		return fmt.Errorf("service temporarily unavailable during %s (code: %s): %w", operation, code, err)
	}

	// For other OData errors, log details for debugging
	if code != "" {
		log.Printf("[DEBUG] Graph API error during %s (code: %s, message: %s)", operation, code, message)
	}

	return err
}

// retryWithBackoff wraps an operation with exponential backoff retry logic.
// It attempts the operation up to maxRetries times, waiting with exponential
// backoff between attempts. Only retries errors identified by isRetryableError.
//
// The backoff delay follows the pattern: baseDelay * (2^attempt), capped at 30 seconds.
// For example, with baseDelay=2s: 2s, 4s, 8s, 16s, 30s, 30s...
//
// Returns the last error encountered if all retries are exhausted, or nil on success.
func retryWithBackoff(ctx context.Context, maxRetries int, baseDelay time.Duration, operation func() error) error {
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		// Execute the operation
		lastErr = operation()

		// Success - return immediately
		if lastErr == nil {
			if attempt > 0 {
				log.Printf("Operation succeeded after %d retries", attempt)
			}
			return nil
		}

		// Check if error is retryable
		if !isRetryableError(lastErr) {
			// Non-retryable error - fail immediately
			return lastErr
		}

		// Last attempt failed - return error
		if attempt == maxRetries {
			return fmt.Errorf("operation failed after %d retries: %w", maxRetries, lastErr)
		}

		// Calculate exponential backoff delay (cap at 30 seconds)
		delay := baseDelay * time.Duration(1<<uint(attempt))
		if delay > 30*time.Second {
			delay = 30 * time.Second
		}

		log.Printf("Retryable error encountered (attempt %d/%d): %v. Retrying in %v...",
			attempt+1, maxRetries, lastErr, delay)

		// Wait with context cancellation support
		select {
		case <-ctx.Done():
			return fmt.Errorf("retry cancelled: %w", ctx.Err())
		case <-time.After(delay):
			// Continue to next retry attempt
		}
	}

	return lastErr
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

// interpretAvailability converts Microsoft Graph availability view codes to human-readable status.
// Availability view codes:
//   - "0" = Free
//   - "1" = Tentative
//   - "2" = Busy
//   - "3" = Out of Office
//   - "4" = Working Elsewhere
//
// The function returns the interpreted status string or "Unknown" if the code is unrecognized.
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

// checkAvailability checks the recipient's availability for the next working day at 12:00 UTC.
// It uses the Microsoft Graph getSchedule API to query availability for a 1-hour window.
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

// pointerTo is a generic helper function to create pointers to values
func pointerTo[T any](v T) *T {
	return &v
}

// validateEmail performs basic email format validation by checking for the presence
// of an @ symbol and ensuring both local-part and domain are non-empty.
func validateEmail(email string) error {
	email = strings.TrimSpace(email)
	if email == "" {
		return fmt.Errorf("email cannot be empty")
	}
	if !strings.Contains(email, "@") {
		return fmt.Errorf("invalid email format: %s (missing @)", email)
	}
	parts := strings.Split(email, "@")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return fmt.Errorf("invalid email format: %s", email)
	}
	return nil
}

// validateEmails validates a slice of email addresses
func validateEmails(emails []string, fieldName string) error {
	for _, email := range emails {
		if err := validateEmail(email); err != nil {
			return fmt.Errorf("%s contains invalid email: %w", fieldName, err)
		}
	}
	return nil
}

// validateGUID validates that a string matches standard GUID format
func validateGUID(guid, fieldName string) error {
	guid = strings.TrimSpace(guid)
	if guid == "" {
		return fmt.Errorf("%s cannot be empty", fieldName)
	}
	// Basic GUID format: 8-4-4-4-12 hex characters
	if len(guid) != 36 {
		return fmt.Errorf("%s should be a GUID (36 characters, format: 12345678-1234-1234-1234-123456789012)", fieldName)
	}
	// Check for proper dash positions
	if guid[8] != '-' || guid[13] != '-' || guid[18] != '-' || guid[23] != '-' {
		return fmt.Errorf("%s has invalid GUID format (dashes at wrong positions)", fieldName)
	}
	return nil
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
// It skips weekends (Saturday and Sunday) and returns the resulting time with the same
// time-of-day preserved. This function is used for scheduling operations that must fall
// on business days.
//
// Example: If t is Friday 2026-01-02 at 14:00, addWorkingDays(t, 1) returns Monday 2026-01-05 at 14:00
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

// validateRFC3339Time validates that a string matches RFC3339 or PowerShell sortable timestamp format
func validateRFC3339Time(timeStr, fieldName string) error {
	if timeStr == "" {
		return nil // Empty is allowed (defaults are used)
	}
	_, err := parseFlexibleTime(timeStr)
	if err != nil {
		return fmt.Errorf("%s: %w", fieldName, err)
	}
	return nil
}

// validateFilePath validates and sanitizes a file path for security and usability.
// It checks for path traversal attempts, normalizes the path, and verifies file existence.
// Returns nil if the path is empty (allows optional file paths).
func validateFilePath(path, fieldName string) error {
	if path == "" {
		return nil // Empty is allowed for optional fields
	}

	// Clean and normalize path (resolves . and .. elements)
	cleanPath := filepath.Clean(path)

	// Check for path traversal attempts
	// After cleaning, ".." should not remain in the path unless it's at the start (relative path going up)
	// We need to check if the cleaned path tries to escape the current directory context
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return fmt.Errorf("%s: invalid path: %w", fieldName, err)
	}

	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		// If we can't get cwd, just verify the file exists
		cwd = ""
	}

	// If we have a cwd, check if the absolute path tries to go outside reasonable bounds
	// For absolute paths (like C:\file.pfx or /etc/file.pfx), this is allowed
	// For relative paths, we verify they don't traverse outside the working directory tree
	if cwd != "" && !filepath.IsAbs(path) {
		// Check if cleaned path still contains ".." which indicates traversal
		if strings.Contains(cleanPath, "..") {
			return fmt.Errorf("%s: path contains directory traversal (..) which is not allowed", fieldName)
		}
	}

	// Verify file exists and is accessible
	fileInfo, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%s: file not found: %s", fieldName, path)
		}
		if os.IsPermission(err) {
			return fmt.Errorf("%s: permission denied: %s", fieldName, path)
		}
		return fmt.Errorf("%s: cannot access file: %w", fieldName, err)
	}

	// Verify it's a regular file (not a directory or special file)
	if !fileInfo.Mode().IsRegular() {
		return fmt.Errorf("%s: not a regular file (is it a directory?): %s", fieldName, path)
	}

	return nil
}

// validateConfiguration validates all required configuration fields
func validateConfiguration(config *Config) error {
	// Validate required fields with format checking
	if err := validateGUID(config.TenantID, "Tenant ID"); err != nil {
		return err
	}
	if err := validateGUID(config.ClientID, "Client ID"); err != nil {
		return err
	}
	if err := validateEmail(config.Mailbox); err != nil {
		return fmt.Errorf("invalid mailbox: %w", err)
	}

	// Check that at least one authentication method is provided
	authMethodCount := 0
	if config.Secret != "" {
		authMethodCount++
	}
	if config.PfxPath != "" {
		authMethodCount++
	}
	if config.Thumbprint != "" {
		authMethodCount++
	}

	if authMethodCount == 0 {
		return fmt.Errorf("missing authentication: must provide one of -secret, -pfx, or -thumbprint")
	}
	if authMethodCount > 1 {
		return fmt.Errorf("multiple authentication methods provided: use only one of -secret, -pfx, or -thumbprint")
	}

	// Validate PFX file path if provided
	if config.PfxPath != "" {
		if err := validateFilePath(config.PfxPath, "PFX certificate file"); err != nil {
			return err
		}
	}

	// Validate attachment file paths
	for i, attachmentPath := range config.AttachmentFiles {
		fieldName := fmt.Sprintf("Attachment file #%d", i+1)
		if err := validateFilePath(attachmentPath, fieldName); err != nil {
			return err
		}
	}

	// Validate email lists if provided
	if len(config.To) > 0 {
		if err := validateEmails(config.To, "To recipients"); err != nil {
			return err
		}
	}
	if len(config.Cc) > 0 {
		if err := validateEmails(config.Cc, "CC recipients"); err != nil {
			return err
		}
	}
	if len(config.Bcc) > 0 {
		if err := validateEmails(config.Bcc, "BCC recipients"); err != nil {
			return err
		}
	}

	// Validate RFC3339 times if provided
	if err := validateRFC3339Time(config.StartTime, "Start time"); err != nil {
		return err
	}
	if err := validateRFC3339Time(config.EndTime, "End time"); err != nil {
		return err
	}

	// Validate action
	validActions := map[string]bool{
		ActionGetEvents:   true,
		ActionSendMail:    true,
		ActionSendInvite:  true,
		ActionGetInbox:    true,
		ActionGetSchedule: true,
	}
	if !validActions[config.Action] {
		return fmt.Errorf("invalid action: %s (use: getevents, sendmail, sendinvite, getinbox, getschedule)", config.Action)
	}

	// Validate getschedule-specific requirements
	if config.Action == ActionGetSchedule {
		if len(config.To) == 0 {
			return fmt.Errorf("getschedule action requires -to parameter (recipient email address)")
		}
		if len(config.To) > 1 {
			return fmt.Errorf("getschedule action only supports checking one recipient at a time (got %d recipients)", len(config.To))
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

// maskSecret masks a secret for display
func maskSecret(secret string) string {
	if len(secret) <= 8 {
		return "********"
	}
	// Show first 4 and last 4 characters
	return secret[:4] + "********" + secret[len(secret)-4:]
}

// maskGUID masks a GUID showing only first and last 4 characters
func maskGUID(guid string) string {
	if len(guid) <= 8 {
		return "****"
	}
	return guid[:4] + "****-****-****-****" + guid[len(guid)-4:]
}

// generateBashCompletion generates a bash completion script for the tool
func generateBashCompletion() string {
	return `# msgraphgolangtestingtool bash completion script
# Installation:
#   Linux: Copy to /etc/bash_completion.d/msgraphgolangtestingtool
#   macOS: Copy to /usr/local/etc/bash_completion.d/msgraphgolangtestingtool
#   Manual: source this file in your ~/.bashrc

_msgraphgolangtestingtool_completions() {
    local cur prev opts
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"

    # All available flags
    opts="-action -tenantid -clientid -secret -pfx -pfxpass -thumbprint -mailbox
          -to -cc -bcc -subject -body -bodyHTML -attachments
          -invite-subject -start -end -proxy -count -verbose -version -help
          -maxretries -retrydelay -loglevel -completion"

    # Flag-specific completions
    case "${prev}" in
        -action)
            # Suggest valid actions
            COMPREPLY=( $(compgen -W "getevents sendmail sendinvite getinbox" -- ${cur}) )
            return 0
            ;;
        -pfx|-attachments)
            # File path completion
            COMPREPLY=( $(compgen -f -- ${cur}) )
            return 0
            ;;
        -loglevel)
            # Suggest log levels
            COMPREPLY=( $(compgen -W "DEBUG INFO WARN ERROR" -- ${cur}) )
            return 0
            ;;
        -completion)
            # Suggest shell types
            COMPREPLY=( $(compgen -W "bash powershell" -- ${cur}) )
            return 0
            ;;
        -version|-verbose|-help)
            # No completion after boolean flags
            return 0
            ;;
        -maxretries|-retrydelay|-count)
            # Numeric values - no completion
            return 0
            ;;
        -tenantid|-clientid|-secret|-pfxpass|-thumbprint|-mailbox|-to|-cc|-bcc|-subject|-body|-bodyHTML|-invite-subject|-start|-end|-proxy)
            # String values - no completion
            return 0
            ;;
    esac

    # Default: complete with flag names
    COMPREPLY=( $(compgen -W "${opts}" -- ${cur}) )
    return 0
}

# Register the completion function for the tool
complete -F _msgraphgolangtestingtool_completions msgraphgolangtestingtool.exe
complete -F _msgraphgolangtestingtool_completions msgraphgolangtestingtool
complete -F _msgraphgolangtestingtool_completions ./msgraphgolangtestingtool.exe
complete -F _msgraphgolangtestingtool_completions ./msgraphgolangtestingtool
`
}

// generatePowerShellCompletion generates a PowerShell completion script for the tool
func generatePowerShellCompletion() string {
	return `# msgraphgolangtestingtool PowerShell completion script
# Installation:
#   Add to your PowerShell profile: notepad $PROFILE
#   Or run manually: . .\msgraphgolangtestingtool-completion.ps1

Register-ArgumentCompleter -CommandName msgraphgolangtestingtool.exe,msgraphgolangtestingtool,'.\msgraphgolangtestingtool.exe','.\msgraphgolangtestingtool' -ScriptBlock {
    param($commandName, $parameterName, $wordToComplete, $commandAst, $fakeBoundParameters)

    # Define valid actions
    $actions = @('getevents', 'sendmail', 'sendinvite', 'getinbox')

    # Define log levels
    $logLevels = @('DEBUG', 'INFO', 'WARN', 'ERROR')

    # Define shell types for completion flag
    $shellTypes = @('bash', 'powershell')

    # All flags that accept values
    $flags = @(
        '-action', '-tenantid', '-clientid', '-secret', '-pfx', '-pfxpass',
        '-thumbprint', '-mailbox', '-to', '-cc', '-bcc', '-subject', '-body',
        '-bodyHTML', '-attachments', '-invite-subject', '-start', '-end',
        '-proxy', '-count', '-maxretries', '-retrydelay', '-loglevel',
        '-completion', '-verbose', '-version', '-help'
    )

    # Get the last word from command line
    $lastWord = ''
    if ($commandAst.CommandElements.Count -gt 1) {
        $lastWord = $commandAst.CommandElements[-2].ToString()
    }

    # Provide context-specific completions based on the previous flag
    switch ($lastWord) {
        '-action' {
            $actions | Where-Object { $_ -like "$wordToComplete*" } | ForEach-Object {
                [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterValue', "Action: $_")
            }
            return
        }
        '-loglevel' {
            $logLevels | Where-Object { $_ -like "$wordToComplete*" } | ForEach-Object {
                [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterValue', "Log Level: $_")
            }
            return
        }
        '-completion' {
            $shellTypes | Where-Object { $_ -like "$wordToComplete*" } | ForEach-Object {
                [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterValue', "Shell: $_")
            }
            return
        }
        '-pfx' {
            # File completion for PFX files
            Get-ChildItem -Path "$wordToComplete*" -File -ErrorAction SilentlyContinue |
                Where-Object { $_.Extension -in @('.pfx', '.p12') -or $wordToComplete -eq '' } |
                ForEach-Object {
                    [System.Management.Automation.CompletionResult]::new(
                        $_.FullName,
                        $_.Name,
                        'ParameterValue',
                        "Certificate: $($_.Name)"
                    )
                }
            return
        }
        '-attachments' {
            # File completion for any file type
            Get-ChildItem -Path "$wordToComplete*" -File -ErrorAction SilentlyContinue |
                ForEach-Object {
                    [System.Management.Automation.CompletionResult]::new(
                        $_.FullName,
                        $_.Name,
                        'ParameterValue',
                        "File: $($_.Name)"
                    )
                }
            return
        }
    }

    # Default: complete with flag names
    $flags | Where-Object { $_ -like "$wordToComplete*" } | ForEach-Object {
        $description = switch ($_) {
            '-action' { 'Operation to perform (getevents, sendmail, sendinvite, getinbox)' }
            '-tenantid' { 'Azure Tenant ID (GUID)' }
            '-clientid' { 'Application (Client) ID (GUID)' }
            '-secret' { 'Client Secret for authentication' }
            '-pfx' { 'Path to .pfx certificate file' }
            '-pfxpass' { 'Password for .pfx certificate' }
            '-thumbprint' { 'Certificate thumbprint (Windows Certificate Store)' }
            '-mailbox' { 'Target user email address' }
            '-to' { 'Comma-separated TO recipients' }
            '-cc' { 'Comma-separated CC recipients' }
            '-bcc' { 'Comma-separated BCC recipients' }
            '-subject' { 'Email subject line' }
            '-body' { 'Email body (text)' }
            '-bodyHTML' { 'Email body (HTML)' }
            '-attachments' { 'Comma-separated file paths to attach' }
            '-invite-subject' { 'Calendar invite subject' }
            '-start' { 'Start time for calendar invite (RFC3339)' }
            '-end' { 'End time for calendar invite (RFC3339)' }
            '-proxy' { 'HTTP/HTTPS proxy URL' }
            '-count' { 'Number of items to retrieve (default: 3)' }
            '-maxretries' { 'Maximum retry attempts (default: 3)' }
            '-retrydelay' { 'Retry delay in milliseconds (default: 2000)' }
            '-loglevel' { 'Logging level (DEBUG, INFO, WARN, ERROR)' }
            '-completion' { 'Generate completion script (bash or powershell)' }
            '-verbose' { 'Enable verbose output' }
            '-version' { 'Show version information' }
            '-help' { 'Show help message' }
            default { $_ }
        }
        [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterName', $description)
    }
}

Write-Host "PowerShell completion for msgraphgolangtestingtool loaded successfully!" -ForegroundColor Green
Write-Host "Try typing: msgraphgolangtestingtool.exe -<TAB>" -ForegroundColor Cyan
`
}
