package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

// Config holds all configuration for jmaptool.
type Config struct {
	// Connection settings
	Host        string
	Port        int
	Username    string
	Password    string
	AccessToken string

	// Action
	Action string

	// Authentication
	AuthMethod string // auto, basic, bearer

	// TLS settings
	SkipVerify bool

	// Logging
	VerboseMode bool
	LogLevel    string
	LogFormat   string

	// Other
	ShowVersion bool
}

// parseAndConfigureFlags parses command-line flags and environment variables.
func parseAndConfigureFlags() *Config {
	config := &Config{}

	// Define flags
	flag.StringVar(&config.Action, "action", "", "Action to perform: testconnect, testauth, getmailboxes")
	flag.StringVar(&config.Host, "host", "", "JMAP server hostname")
	flag.IntVar(&config.Port, "port", 443, "JMAP server port (default: 443)")
	flag.StringVar(&config.Username, "username", "", "Username for authentication")
	flag.StringVar(&config.Password, "password", "", "Password for authentication")
	flag.StringVar(&config.AccessToken, "accesstoken", "", "Access token for Bearer authentication")
	flag.StringVar(&config.AuthMethod, "authmethod", "auto", "Authentication method: auto, basic, bearer")
	flag.BoolVar(&config.SkipVerify, "skipverify", false, "Skip TLS certificate verification")
	flag.BoolVar(&config.VerboseMode, "verbose", false, "Enable verbose output")
	flag.StringVar(&config.LogLevel, "loglevel", "info", "Log level: debug, info, warn, error")
	flag.StringVar(&config.LogFormat, "logformat", "csv", "Log format: csv, json")
	flag.BoolVar(&config.ShowVersion, "version", false, "Show version information")

	// Custom usage
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: jmaptool [options]\n\n")
		fmt.Fprintf(os.Stderr, "JMAP testing tool for testing JMAP server connectivity and operations.\n\n")
		fmt.Fprintf(os.Stderr, "Actions:\n")
		fmt.Fprintf(os.Stderr, "  testconnect   Test JMAP server connectivity and discover session\n")
		fmt.Fprintf(os.Stderr, "  testauth      Test authentication\n")
		fmt.Fprintf(os.Stderr, "  getmailboxes  Get list of mailboxes\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nEnvironment variables:\n")
		fmt.Fprintf(os.Stderr, "  JMAP_HOST        Server hostname\n")
		fmt.Fprintf(os.Stderr, "  JMAP_PORT        Server port\n")
		fmt.Fprintf(os.Stderr, "  JMAP_USERNAME    Username\n")
		fmt.Fprintf(os.Stderr, "  JMAP_PASSWORD    Password\n")
		fmt.Fprintf(os.Stderr, "  JMAP_ACCESSTOKEN Access token\n")
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  jmaptool -action testconnect -host jmap.fastmail.com\n")
		fmt.Fprintf(os.Stderr, "  jmaptool -action testauth -host jmap.fastmail.com -username user@example.com -accesstoken \"token\"\n")
		fmt.Fprintf(os.Stderr, "  jmaptool -action getmailboxes -host jmap.fastmail.com -username user@example.com -accesstoken \"token\"\n")
	}

	flag.Parse()

	// Read from environment variables if not set via flags
	if config.Host == "" {
		config.Host = os.Getenv("JMAP_HOST")
	}
	if config.Port == 443 {
		if envPort := os.Getenv("JMAP_PORT"); envPort != "" {
			fmt.Sscanf(envPort, "%d", &config.Port)
		}
	}
	if config.Username == "" {
		config.Username = os.Getenv("JMAP_USERNAME")
	}
	if config.Password == "" {
		config.Password = os.Getenv("JMAP_PASSWORD")
	}
	if config.AccessToken == "" {
		config.AccessToken = os.Getenv("JMAP_ACCESSTOKEN")
	}

	return config
}

// validateConfiguration validates the configuration.
func validateConfiguration(config *Config) error {
	// Validate action
	validActions := map[string]bool{
		"testconnect":  true,
		"testauth":     true,
		"getmailboxes": true,
	}

	action := strings.ToLower(config.Action)
	if !validActions[action] {
		return fmt.Errorf("invalid action: %s (valid: testconnect, testauth, getmailboxes)", config.Action)
	}
	config.Action = action

	// Validate host
	if config.Host == "" {
		return fmt.Errorf("host is required")
	}

	// Validate auth method
	config.AuthMethod = strings.ToLower(config.AuthMethod)
	validAuthMethods := map[string]bool{
		"auto":   true,
		"basic":  true,
		"bearer": true,
	}
	if !validAuthMethods[config.AuthMethod] {
		return fmt.Errorf("invalid auth method: %s (valid: auto, basic, bearer)", config.AuthMethod)
	}

	// Validate credentials for auth actions
	if config.Action == "testauth" || config.Action == "getmailboxes" {
		if config.AccessToken == "" && config.Password == "" {
			return fmt.Errorf("either password or accesstoken is required for %s", config.Action)
		}
	}

	return nil
}
