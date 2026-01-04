// Package main provides a portable CLI tool for interacting with Microsoft Graph API
// to manage Exchange Online (EXO) mailboxes. The tool supports sending emails,
// creating calendar events, and retrieving inbox messages and calendar events.
//
// Authentication methods supported:
//   - Client Secret: Standard App Registration secret
//   - PFX Certificate: Certificate file with private key
//   - Windows Certificate Store: Thumbprint-based certificate retrieval (Windows only)
//
// All operations are automatically logged to action-specific CSV files in the
// system temp directory for audit and troubleshooting purposes.
//
// Example usage:
//
//	msgraphgolangtestingtool.exe -tenantid "..." -clientid "..." -secret "..." -mailbox "user@example.com" -action sendmail
//
// Version information is embedded from the VERSION file at compile time using go:embed.
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

// Flags holds all command-line flag values and parsed configuration
type Flags struct {
	// Core flags
	ShowVersion bool
	TenantID    string
	ClientID    string
	Mailbox     string
	Action      string

	// Authentication flags
	Secret      string
	PfxPath     string
	PfxPass     string
	Thumbprint  string

	// Email recipients (using stringSlice type)
	To              stringSlice
	Cc              stringSlice
	Bcc             stringSlice
	AttachmentFiles stringSlice

	// Email content flags
	Subject  string
	Body     string
	BodyHTML string

	// Calendar invite flags
	InviteSubject string
	StartTime     string
	EndTime       string

	// Network and other flags
	ProxyURL string
	Verbose  bool
	Count    int
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

// setupSignalHandling configures graceful shutdown on interrupt signals
// Returns a cancellable context for use throughout the application
func setupSignalHandling() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())

	// Handle interrupt signals (Ctrl+C, SIGTERM)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\n\nReceived interrupt signal. Shutting down gracefully...")
		cancel()
	}()

	return ctx, cancel
}

// parseAndConfigureFlags defines all command-line flags, parses them,
// applies environment variables, and returns populated Flags and Config structs
func parseAndConfigureFlags() (*Flags, *Config) {
	// Customize help output
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Microsoft Graph GoLang Testing Tool - Version %s\n\n", version)
		fmt.Fprintf(flag.CommandLine.Output(), "Repository: https://github.com/ziembor/msgraphgolangtestingtool\n\n")
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(flag.CommandLine.Output(), "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(flag.CommandLine.Output(), "\nEnvironment Variables:\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  All flags can be set via environment variables with MSGRAPH prefix\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  Example: MSGRAPHTENANTID, MSGRAPHCLIENTID, MSGRAPHSECRET\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  Command-line flags take precedence over environment variables\n\n")
		fmt.Fprintf(flag.CommandLine.Output(), "Examples:\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  %s -tenantid \"...\" -clientid \"...\" -secret \"...\" -mailbox \"user@example.com\" -action getevents\n", os.Args[0])
		fmt.Fprintf(flag.CommandLine.Output(), "  %s -tenantid \"...\" -clientid \"...\" -thumbprint \"ABC123\" -mailbox \"user@example.com\" -action sendmail\n\n", os.Args[0])
	}

	// Define Command Line Parameters
	showVersion := flag.Bool("version", false, "Show version information")
	tenantID := flag.String("tenantid", "", "The Azure Tenant ID")
	clientID := flag.String("clientid", "", "The Application (Client) ID")
	secret := flag.String("secret", "", "The Client Secret")
	pfxPath := flag.String("pfx", "", "Path to the .pfx certificate file")
	pfxPass := flag.String("pfxpass", "", "Password for the .pfx file")
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

	// Create and populate Flags struct
	flags := &Flags{
		ShowVersion:     *showVersion,
		TenantID:        *tenantID,
		ClientID:        *clientID,
		Mailbox:         *mailbox,
		Action:          *action,
		Secret:          *secret,
		PfxPath:         *pfxPath,
		PfxPass:         *pfxPass,
		Thumbprint:      *thumbprint,
		To:              to,
		Cc:              cc,
		Bcc:             bcc,
		AttachmentFiles: attachmentFiles,
		Subject:         *subject,
		Body:            *body,
		BodyHTML:        *bodyHTML,
		InviteSubject:   *inviteSubject,
		StartTime:       *startTime,
		EndTime:         *endTime,
		ProxyURL:        *proxyURL,
		Verbose:         *verbose,
		Count:           *count,
	}

	// Create Config struct
	config := &Config{
		VerboseMode: *verbose,
	}

	// Print verbose configuration if enabled
	if config.VerboseMode {
		printVerboseConfig(*tenantID, *clientID, *secret, *pfxPath, *thumbprint, *mailbox, *action, *proxyURL, to.String(), cc.String(), bcc.String(), *subject, *body, *bodyHTML, attachmentFiles.String(), *inviteSubject, *startTime, *endTime)
	}

	return flags, config
}

// validateEmail performs basic email format validation
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

// validateGUID validates that a string is a valid GUID format
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

// validateRFC3339Time validates RFC3339 time format
func validateRFC3339Time(timeStr, fieldName string) error {
	if timeStr == "" {
		return nil // Empty is allowed (defaults are used)
	}
	_, err := time.Parse(time.RFC3339, timeStr)
	if err != nil {
		return fmt.Errorf("%s is not in valid RFC3339 format (e.g., 2026-01-15T14:00:00Z): %w", fieldName, err)
	}
	return nil
}

// validateConfiguration checks that required configuration parameters are valid
func validateConfiguration(flags *Flags) error {
	// Validate required fields with format checking
	if err := validateGUID(flags.TenantID, "Tenant ID"); err != nil {
		return err
	}
	if err := validateGUID(flags.ClientID, "Client ID"); err != nil {
		return err
	}
	if err := validateEmail(flags.Mailbox); err != nil {
		return fmt.Errorf("invalid mailbox: %w", err)
	}

	// Check that at least one authentication method is provided
	authMethodCount := 0
	if flags.Secret != "" {
		authMethodCount++
	}
	if flags.PfxPath != "" {
		authMethodCount++
	}
	if flags.Thumbprint != "" {
		authMethodCount++
	}

	if authMethodCount == 0 {
		return fmt.Errorf("missing authentication: must provide one of -secret, -pfx, or -thumbprint")
	}
	if authMethodCount > 1 {
		return fmt.Errorf("multiple authentication methods provided: use only one of -secret, -pfx, or -thumbprint")
	}

	// Validate email lists if provided
	if len(flags.To) > 0 {
		if err := validateEmails(flags.To, "To recipients"); err != nil {
			return err
		}
	}
	if len(flags.Cc) > 0 {
		if err := validateEmails(flags.Cc, "CC recipients"); err != nil {
			return err
		}
	}
	if len(flags.Bcc) > 0 {
		if err := validateEmails(flags.Bcc, "BCC recipients"); err != nil {
			return err
		}
	}

	// Validate RFC3339 times if provided
	if err := validateRFC3339Time(flags.StartTime, "Start time"); err != nil {
		return err
	}
	if err := validateRFC3339Time(flags.EndTime, "End time"); err != nil {
		return err
	}

	// Validate action
	validActions := map[string]bool{
		ActionGetEvents:  true,
		ActionSendMail:   true,
		ActionSendInvite: true,
		ActionGetInbox:   true,
	}
	if !validActions[flags.Action] {
		return fmt.Errorf("invalid action: %s (use: getevents, sendmail, sendinvite, getinbox)", flags.Action)
	}

	return nil
}

// initializeServices sets up CSV logging and proxy configuration
// Returns the CSV logger (or nil if initialization failed)
func initializeServices(flags *Flags) (*CSVLogger, error) {
	// Initialize CSV logging
	logger, err := NewCSVLogger(flags.Action)
	if err != nil {
		log.Printf("Warning: Could not initialize CSV logging: %v", err)
		logger = nil // Continue without logging
	}

	// Configure proxy if specified
	// Go's http package automatically uses HTTP_PROXY/HTTPS_PROXY environment variables
	if flags.ProxyURL != "" {
		os.Setenv("HTTP_PROXY", flags.ProxyURL)
		os.Setenv("HTTPS_PROXY", flags.ProxyURL)
		fmt.Printf("Using proxy: %s\n", flags.ProxyURL)
	}

	return logger, nil
}

// setupGraphClient creates credentials and initializes the Microsoft Graph SDK client
func setupGraphClient(ctx context.Context, flags *Flags, config *Config) (*msgraphsdk.GraphServiceClient, error) {
	// Setup Authentication
	cred, err := getCredential(flags.TenantID, flags.ClientID, flags.Secret, flags.PfxPath, flags.PfxPass, flags.Thumbprint, config)
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

// executeAction dispatches to the appropriate action handler based on flags.Action
func executeAction(ctx context.Context, client *msgraphsdk.GraphServiceClient, flags *Flags, config *Config, logger *CSVLogger) error {
	switch flags.Action {
	case ActionGetEvents:
		if err := listEvents(ctx, client, flags.Mailbox, flags.Count, config, logger); err != nil {
			return fmt.Errorf("failed to list events: %w", err)
		}
	case ActionSendMail:
		// If no recipients specified at all, default 'to' to the sender mailbox
		if len(flags.To) == 0 && len(flags.Cc) == 0 && len(flags.Bcc) == 0 {
			flags.To = []string{flags.Mailbox}
		}

		sendEmail(ctx, client, flags.Mailbox, flags.To, flags.Cc, flags.Bcc, flags.Subject, flags.Body, flags.BodyHTML, flags.AttachmentFiles, config, logger)
	case ActionSendInvite:
		createInvite(ctx, client, flags.Mailbox, flags.InviteSubject, flags.StartTime, flags.EndTime, config, logger)
	case ActionGetInbox:
		if err := listInbox(ctx, client, flags.Mailbox, flags.Count, config, logger); err != nil {
			return fmt.Errorf("failed to list inbox: %w", err)
		}
	default:
		return fmt.Errorf("unknown action: %s", flags.Action)
	}

	return nil
}

func run() error {
	// 1. Setup signal handling for graceful shutdown
	ctx, cancel := setupSignalHandling()
	defer cancel()

	// 2. Parse command-line flags and apply environment variables
	flags, config := parseAndConfigureFlags()

	// 3. Handle version flag early exit
	if flags.ShowVersion {
		fmt.Printf("Microsoft Graph Golang Testing Tool - Version %s\n", version)
		return nil
	}

	// 4. Validate configuration
	if err := validateConfiguration(flags); err != nil {
		fmt.Printf("Error: %v\n", err)
		flag.Usage()
		os.Exit(1)
	}

	// 5. Initialize services (CSV logging and proxy)
	logger, err := initializeServices(flags)
	if err != nil {
		// Error already logged in initializeServices, continue without logger
	}
	if logger != nil {
		defer logger.Close()
	}

	// 6. Setup Microsoft Graph client
	client, err := setupGraphClient(ctx, flags, config)
	if err != nil {
		return err
	}

	// 7. Execute the requested action
	return executeAction(ctx, client, flags, config, logger)
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
