//go:build !integration
// +build !integration

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
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"syscall"
	"time"

	msgraphsdk "github.com/microsoftgraph/msgraph-sdk-go"
)

// All shared types, constants, and business logic are in shared.go

func main() {
	// Handle -completion flag FIRST, before anything else runs
	// This ensures only completion script is output, all other flags are ignored
	for i, arg := range os.Args {
		if arg == "-completion" && i+1 < len(os.Args) {
			shellType := os.Args[i+1]
			if shellType == "bash" {
				fmt.Print(generateBashCompletion())
				os.Exit(0)
			} else if shellType == "powershell" {
				fmt.Print(generatePowerShellCompletion())
				os.Exit(0)
			} else {
				fmt.Fprintf(os.Stderr, "Error: Invalid completion shell type '%s'\n", shellType)
				fmt.Fprintf(os.Stderr, "Valid options: bash, powershell\n\n")
				fmt.Fprintf(os.Stderr, "Usage:\n")
				fmt.Fprintf(os.Stderr, "  %s -completion bash > msgraphgolangtestingtool-completion.bash\n", os.Args[0])
				fmt.Fprintf(os.Stderr, "  %s -completion powershell > msgraphgolangtestingtool-completion.ps1\n", os.Args[0])
				os.Exit(1)
			}
		}
	}

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

// parseAndConfigureFlags defines all command-line flags, parses them,
// applies environment variables, and returns a populated Config struct with
// all configuration values merged from defaults, environment variables, and
// command-line arguments (in that order of precedence).
func parseAndConfigureFlags() *Config {
	// Customize help output
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Microsoft Graph EXO Mails/Calendar Golang Testing Tool - Version %s\n\n", version)
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
	tenantID := flag.String("tenantid", "", "The Azure Tenant ID (env: MSGRAPHTENANTID)")
	clientID := flag.String("clientid", "", "The Application (Client) ID (env: MSGRAPHCLIENTID)")
	secret := flag.String("secret", "", "The Client Secret (env: MSGRAPHSECRET)")
	pfxPath := flag.String("pfx", "", "Path to the .pfx certificate file (env: MSGRAPHPFX)")
	pfxPass := flag.String("pfxpass", "", "Password for the .pfx file (env: MSGRAPHPFXPASS)")
	thumbprint := flag.String("thumbprint", "", "Thumbprint of the certificate in the CurrentUser\\My store (env: MSGRAPHTHUMBPRINT)")
	mailbox := flag.String("mailbox", "", "The target EXO mailbox email address (env: MSGRAPHMAILBOX)")

	// Recipient flags (using custom stringSlice type)
	var to, cc, bcc, attachmentFiles stringSlice
	flag.Var(&to, "to", "Comma-separated list of TO recipients (optional, defaults to mailbox if empty) (env: MSGRAPHTO)")
	flag.Var(&cc, "cc", "Comma-separated list of CC recipients (env: MSGRAPHCC)")
	flag.Var(&bcc, "bcc", "Comma-separated list of BCC recipients (env: MSGRAPHBCC)")

	// Email content flags
	subject := flag.String("subject", "Automated Tool Notification", "Subject of the email or calendar invite (env: MSGRAPHSUBJECT)")
	body := flag.String("body", "It's a test message, please ignore", "Body content of the email (text) (env: MSGRAPHBODY)")
	bodyHTML := flag.String("bodyHTML", "", "HTML body content of the email (optional, creates multipart message if both -body and -bodyHTML are provided) (env: MSGRAPHBODYHTML)")
	flag.Var(&attachmentFiles, "attachments", "Comma-separated list of file paths to attach (env: MSGRAPHATTACHMENTS)")

	// Calendar invite flags
	inviteSubject := flag.String("invite-subject", "", "")  // Deprecated: use -subject instead
	startTime := flag.String("start", "", "Start time for calendar invite (RFC3339 or PowerShell 'Get-Date -Format s' format). Examples: '2026-01-15T14:00:00Z', '2026-01-15T14:00:00'. Defaults to now if empty (env: MSGRAPHSTART)")
	endTime := flag.String("end", "", "End time for calendar invite (RFC3339 or PowerShell 'Get-Date -Format s' format). Examples: '2026-01-15T15:00:00Z', '2026-01-15T15:00:00'. Defaults to 1 hour after start if empty (env: MSGRAPHEND)")

	// Proxy configuration
	proxyURL := flag.String("proxy", "", "HTTP/HTTPS proxy URL (e.g., http://proxy.example.com:8080) (env: MSGRAPHPROXY)")

	// Retry configuration
	maxRetries := flag.Int("maxretries", 3, "Maximum retry attempts for transient failures (default: 3) (env: MSGRAPHMAXRETRIES)")
	retryDelay := flag.Int("retrydelay", 2000, "Base delay between retries in milliseconds (default: 2000ms) (env: MSGRAPHRETRYDELAY)")

	// Verbose mode
	verbose := flag.Bool("verbose", false, "Enable verbose output (shows configuration, tokens, API details)")

	// Log level
	logLevel := flag.String("loglevel", "INFO", "Logging level: DEBUG, INFO, WARN, ERROR (default: INFO)")

	// Count for getevents and getinbox
	count := flag.Int("count", 3, "Number of items to retrieve for getevents and getinbox actions (default: 3) (env: MSGRAPHCOUNT)")

	action := flag.String("action", "getinbox", "Action to perform: getevents, sendmail, sendinvite, getinbox, getschedule (env: MSGRAPHACTION)")
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

	// Apply MSGRAPHMAXRETRIES environment variable if flag wasn't provided
	maxRetriesFlagProvided := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "maxretries" {
			maxRetriesFlagProvided = true
		}
	})
	if !maxRetriesFlagProvided {
		if envMaxRetries := os.Getenv("MSGRAPHMAXRETRIES"); envMaxRetries != "" {
			if parsedMaxRetries, err := strconv.Atoi(envMaxRetries); err == nil && parsedMaxRetries >= 0 {
				*maxRetries = parsedMaxRetries
			}
		}
	}

	// Apply MSGRAPHRETRYDELAY environment variable if flag wasn't provided
	retryDelayFlagProvided := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "retrydelay" {
			retryDelayFlagProvided = true
		}
	})
	if !retryDelayFlagProvided {
		if envRetryDelay := os.Getenv("MSGRAPHRETRYDELAY"); envRetryDelay != "" {
			if parsedRetryDelay, err := strconv.Atoi(envRetryDelay); err == nil && parsedRetryDelay > 0 {
				*retryDelay = parsedRetryDelay
			}
		}
	}

	// Apply MSGRAPHLOGLEVEL environment variable if flag wasn't provided
	logLevelFlagProvided := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "loglevel" {
			logLevelFlagProvided = true
		}
	})
	if !logLevelFlagProvided {
		if envLogLevel := os.Getenv("MSGRAPHLOGLEVEL"); envLogLevel != "" {
			*logLevel = envLogLevel
		}
	}

	// Create and populate Config struct with all parsed values
	config := &Config{
		ShowVersion: *showVersion,
		TenantID:    *tenantID,
		ClientID:    *clientID,
		Mailbox:     *mailbox,
		Action:      *action,
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
		MaxRetries:      *maxRetries,
		RetryDelay:      time.Duration(*retryDelay) * time.Millisecond,
		VerboseMode:     *verbose,
		LogLevel:        *logLevel,
		Count:           *count,
	}

	// Print verbose configuration if enabled
	if config.VerboseMode {
		printVerboseConfig(*tenantID, *clientID, *secret, *pfxPath, *thumbprint, *mailbox, *action, *proxyURL, to.String(), cc.String(), bcc.String(), *subject, *body, *bodyHTML, attachmentFiles.String(), *inviteSubject, *startTime, *endTime)
	}

	return config
}

// initializeServices sets up CSV logging and proxy configuration based on
// the provided configuration. Creates a CSV logger for the specified action
// and configures HTTP/HTTPS proxy environment variables if a proxy URL is specified.
//
// Returns the CSV logger (or nil if initialization failed) and any error encountered.
// If CSV logger initialization fails, a warning is logged but execution continues.
func initializeServices(config *Config) (*CSVLogger, error) {
	// Initialize CSV logging
	logger, err := NewCSVLogger(config.Action)
	if err != nil {
		log.Printf("Warning: Could not initialize CSV logging: %v", err)
		logger = nil // Continue without logging
	}

	// Configure proxy if specified
	// Go's http package automatically uses HTTP_PROXY/HTTPS_PROXY environment variables
	if config.ProxyURL != "" {
		os.Setenv("HTTP_PROXY", config.ProxyURL)
		os.Setenv("HTTPS_PROXY", config.ProxyURL)
		fmt.Printf("Using proxy: %s\n", config.ProxyURL)
	}

	return logger, nil
}

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

// run is the main application entry point that orchestrates the tool's execution flow.
// It performs the following steps:
//  1. Sets up graceful shutdown handling for interrupt signals
//  2. Parses and validates configuration from flags and environment variables
//  3. Initializes services (CSV logging, proxy configuration)
//  4. Creates Microsoft Graph SDK client with appropriate authentication
//  5. Executes the requested action (getevents, sendmail, sendinvite, getinbox)
//
// Returns an error if any step fails, nil on successful completion.
func run() error {
	// 1. Setup signal handling for graceful shutdown
	ctx, cancel := setupSignalHandling()
	defer cancel()

	// 2. Parse command-line flags and apply environment variables
	config := parseAndConfigureFlags()

	// 3. Handle version flag early exit
	if config.ShowVersion {
		fmt.Printf("Microsoft Graph EXO Mails/Calendar Golang Testing Tool - Version %s\n", version)
		return nil
	}

	// 4. Validate configuration
	if err := validateConfiguration(config); err != nil {
		fmt.Printf("Error: %v\n", err)
		flag.Usage()
		os.Exit(1)
	}

	// 5. Setup structured logger
	slogger := setupLogger(config)
	slogger.Info("Application starting", "version", version, "action", config.Action)

	// 6. Initialize services (CSV logging and proxy)
	csvLogger, err := initializeServices(config)
	if err != nil {
		// Error already logged in initializeServices, continue without logger
	}
	if csvLogger != nil {
		defer csvLogger.Close()
	}

	// 7. Setup Microsoft Graph client
	client, err := setupGraphClient(ctx, config, slogger)
	if err != nil {
		return err
	}

	// 8. Execute the requested action
	return executeAction(ctx, client, config, csvLogger)
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
		"MSGRAPHMAXRETRIES",
		"MSGRAPHRETRYDELAY",
		"MSGRAPHLOGLEVEL",
	}

	for _, envVar := range msgraphEnvVars {
		if value := os.Getenv(envVar); value != "" {
			envVars[envVar] = value
		}
	}

	return envVars
}
