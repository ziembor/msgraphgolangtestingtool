package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"msgraphgolangtestingtool/internal/common/validation"
)

// Config holds all smtptool configuration.
type Config struct {
	// Core configuration
	ShowVersion bool
	Action      string

	// SMTP server configuration
	Host    string
	Port    int
	Timeout time.Duration

	// Authentication
	Username   string
	Password   string
	AuthMethod string // PLAIN, LOGIN, CRAM-MD5, or "auto"

	// Email configuration (for sendmail)
	From    string
	To      []string
	Subject string
	Body    string

	// TLS configuration
	StartTLS   bool   // Force STARTTLS
	SkipVerify bool   // Skip TLS certificate verification
	TLSVersion string // Minimum TLS version (1.2, 1.3)

	// Network configuration
	ProxyURL   string
	MaxRetries int
	RetryDelay time.Duration

	// Runtime configuration
	VerboseMode  bool
	LogLevel     string
	OutputFormat string
}

// Action constants
const (
	ActionTestConnect  = "testconnect"
	ActionTestStartTLS = "teststarttls"
	ActionTestAuth     = "testauth"
	ActionSendMail     = "sendmail"
)

// NewConfig creates a new Config with default values.
func NewConfig() *Config {
	return &Config{
		Port:         25,
		Timeout:      30 * time.Second,
		AuthMethod:   "auto",
		Subject:      "SMTP Test",
		Body:         "This is a test message from smtptool",
		StartTLS:     false, // Auto-detect
		SkipVerify:   false,
		TLSVersion:   "1.2",
		MaxRetries:   3,
		RetryDelay:   2000 * time.Millisecond,
		VerboseMode:  false,
		LogLevel:     "INFO",
		OutputFormat: "text",
	}
}

// parseAndConfigureFlags parses command-line flags and environment variables.
func parseAndConfigureFlags() *Config {
	config := NewConfig()

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "SMTP Connectivity Testing Tool - Part of msgraphgolangtestingtool suite\n\n")
		fmt.Fprintf(flag.CommandLine.Output(), "Repository: https://github.com/ziembor/msgraphgolangtestingtool\n\n")
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(flag.CommandLine.Output(), "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(flag.CommandLine.Output(), "\nEnvironment Variables:\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  All flags can be set via environment variables with SMTP prefix\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  Example: SMTPHOST, SMTPPORT, SMTPUSERNAME\n\n")
		fmt.Fprintf(flag.CommandLine.Output(), "Actions:\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  testconnect   - Test TCP connection and capabilities\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  teststarttls  - Test TLS/SSL with comprehensive diagnostics\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  testauth      - Test SMTP authentication\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  sendmail      - Send test email\n\n")
		fmt.Fprintf(flag.CommandLine.Output(), "Examples:\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  %s -action testconnect -host smtp.example.com -port 25\n", os.Args[0])
		fmt.Fprintf(flag.CommandLine.Output(), "  %s -action teststarttls -host smtp.example.com -port 587\n", os.Args[0])
		fmt.Fprintf(flag.CommandLine.Output(), "  %s -action testauth -host smtp.example.com -port 587 -username user@example.com -password secret\n", os.Args[0])
		fmt.Fprintf(flag.CommandLine.Output(), "  %s -action sendmail -host smtp.example.com -port 587 -username user@example.com -password secret -from sender@example.com -to recipient@example.com\n\n", os.Args[0])
	}

	// Define flags
	showVersion := flag.Bool("version", false, "Show version information")
	action := flag.String("action", "", "Action to perform (testconnect, teststarttls, testauth, sendmail)")
	host := flag.String("host", "", "SMTP server hostname or IP address (env: SMTPHOST)")
	port := flag.Int("port", 25, "SMTP server port (env: SMTPPORT)")
	timeout := flag.Int("timeout", 30, "Connection timeout in seconds (env: SMTPTIMEOUT)")
	username := flag.String("username", "", "SMTP username for authentication (env: SMTPUSERNAME)")
	password := flag.String("password", "", "SMTP password for authentication (env: SMTPPASSWORD)")
	authMethod := flag.String("authmethod", "auto", "Authentication method: PLAIN, LOGIN, CRAM-MD5, auto (env: SMTPAUTHMETHOD)")
	from := flag.String("from", "", "Sender email address for sendmail (env: SMTPFROM)")
	to := flag.String("to", "", "Comma-separated recipient email addresses (env: SMTPTO)")
	subject := flag.String("subject", "SMTP Test", "Email subject (env: SMTPSUBJECT)")
	body := flag.String("body", "This is a test message from smtptool", "Email body text (env: SMTPBODY)")
	startTLS := flag.Bool("starttls", false, "Force STARTTLS usage (env: SMTPSTARTTLS)")
	skipVerify := flag.Bool("skipverify", false, "Skip TLS certificate verification (insecure) (env: SMTPSKIPVERIFY)")
	tlsVersion := flag.String("tlsversion", "1.2", "Minimum TLS version: 1.2, 1.3 (env: SMTPTLSVERSION)")
	proxyURL := flag.String("proxy", "", "HTTP/HTTPS proxy URL (env: SMTPPROXY)")
	maxRetries := flag.Int("maxretries", 3, "Maximum retry attempts (env: SMTPMAXRETRIES)")
	retryDelay := flag.Int("retrydelay", 2000, "Retry delay in milliseconds (env: SMTPRETRYDELAY)")
	verbose := flag.Bool("verbose", false, "Enable verbose output")
	logLevel := flag.String("loglevel", "INFO", "Logging level: DEBUG, INFO, WARN, ERROR")
	outputFormat := flag.String("output", "text", "Output format: text, json (env: SMTPOUTPUT)")

	flag.Parse()

	// Apply flags to config
	config.ShowVersion = *showVersion
	config.Action = *action
	config.Host = *host
	config.Port = *port
	config.Timeout = time.Duration(*timeout) * time.Second
	config.Username = *username
	config.Password = *password
	config.AuthMethod = *authMethod
	config.From = *from
	if *to != "" {
		config.To = strings.Split(*to, ",")
	}
	config.Subject = *subject
	config.Body = *body
	config.StartTLS = *startTLS
	config.SkipVerify = *skipVerify
	config.TLSVersion = *tlsVersion
	config.ProxyURL = *proxyURL
	config.MaxRetries = *maxRetries
	config.RetryDelay = time.Duration(*retryDelay) * time.Millisecond
	config.VerboseMode = *verbose
	config.LogLevel = *logLevel
	config.OutputFormat = *outputFormat

	// Apply environment variables (if flags not set)
	applyEnvironmentVariables(config)

	return config
}

// applyEnvironmentVariables applies environment variables to config.
func applyEnvironmentVariables(config *Config) {
	if config.Action == "" {
		config.Action = os.Getenv("SMTPACTION")
	}
	if config.Host == "" {
		config.Host = os.Getenv("SMTPHOST")
	}
	if portStr := os.Getenv("SMTPPORT"); portStr != "" && config.Port == 25 {
		if port, err := strconv.Atoi(portStr); err == nil {
			config.Port = port
		}
	}
	if config.Username == "" {
		config.Username = os.Getenv("SMTPUSERNAME")
	}
	if config.Password == "" {
		config.Password = os.Getenv("SMTPPASSWORD")
	}
	if config.From == "" {
		config.From = os.Getenv("SMTPFROM")
	}
	if toStr := os.Getenv("SMTPTO"); toStr != "" && len(config.To) == 0 {
		config.To = strings.Split(toStr, ",")
	}
}

// validateConfiguration validates the configuration.
func validateConfiguration(config *Config) error {
	// Validate action
	validActions := []string{ActionTestConnect, ActionTestStartTLS, ActionTestAuth, ActionSendMail}
	valid := false
	for _, a := range validActions {
		if config.Action == a {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("invalid action: %s (must be one of: %s)", config.Action, strings.Join(validActions, ", "))
	}

	// Validate host (required for all actions)
	if config.Host == "" {
		return fmt.Errorf("host is required (-host flag)")
	}
	if err := validation.ValidateHostname(config.Host); err != nil {
		return fmt.Errorf("invalid host: %w", err)
	}

	// Validate port
	if err := validation.ValidatePort(config.Port); err != nil {
		return fmt.Errorf("invalid port: %w", err)
	}

	// Action-specific validation
	switch config.Action {
	case ActionTestAuth:
		if config.Username == "" {
			return fmt.Errorf("testauth requires -username")
		}
		if config.Password == "" {
			return fmt.Errorf("testauth requires -password")
		}

	case ActionSendMail:
		if config.From == "" {
			return fmt.Errorf("sendmail requires -from")
		}
		if err := validation.ValidateEmail(config.From); err != nil {
			return fmt.Errorf("invalid sender email: %w", err)
		}
		if len(config.To) == 0 {
			return fmt.Errorf("sendmail requires -to")
		}
		for _, email := range config.To {
			if err := validation.ValidateEmail(strings.TrimSpace(email)); err != nil {
				return fmt.Errorf("invalid recipient email: %w", err)
			}
		}
		if config.Subject == "" {
			return fmt.Errorf("sendmail requires -subject")
		}
	}

	return nil
}
