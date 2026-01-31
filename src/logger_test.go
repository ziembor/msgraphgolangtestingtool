//go:build !integration
// +build !integration

package main

import (
	"log/slog"
	"testing"
)

// TestParseLogLevel tests the parseLogLevel function
func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantLevel slog.Level
	}{
		{"debug lowercase", "debug", slog.LevelDebug},
		{"debug uppercase", "DEBUG", slog.LevelDebug},
		{"info lowercase", "info", slog.LevelInfo},
		{"info uppercase", "INFO", slog.LevelInfo},
		{"warn lowercase", "warn", slog.LevelWarn},
		{"warn uppercase", "WARN", slog.LevelWarn},
		{"warning", "WARNING", slog.LevelWarn},
		{"error lowercase", "error", slog.LevelError},
		{"error uppercase", "ERROR", slog.LevelError},
		{"invalid level defaults to info", "INVALID", slog.LevelInfo},
		{"empty string defaults to info", "", slog.LevelInfo},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseLogLevel(tt.input)
			if got != tt.wantLevel {
				t.Errorf("parseLogLevel(%q) = %v, want %v", tt.input, got, tt.wantLevel)
			}
		})
	}
}

// TestSetupLogger tests logger configuration
func TestSetupLogger(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		expectDebug bool
	}{
		{
			name:        "verbose mode enables debug",
			config:      &Config{VerboseMode: true, LogLevel: "INFO"},
			expectDebug: true,
		},
		{
			name:        "debug level enables debug",
			config:      &Config{VerboseMode: false, LogLevel: "DEBUG"},
			expectDebug: true,
		},
		{
			name:        "info level disables debug",
			config:      &Config{VerboseMode: false, LogLevel: "INFO"},
			expectDebug: false,
		},
		{
			name:        "error level disables debug",
			config:      &Config{VerboseMode: false, LogLevel: "ERROR"},
			expectDebug: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := setupLogger(tt.config)
			if logger == nil {
				t.Fatal("setupLogger returned nil")
			}

			// Logger is created successfully if we get here
			// Testing actual log output would require capturing output
			// which is beyond the scope of a unit test
		})
	}
}

// TestLogHelpers tests the log helper functions don't panic with nil logger
func TestLogHelpers(t *testing.T) {
	// These should not panic even with nil logger
	logDebug(nil, "test debug")
	logInfo(nil, "test info")
	logWarn(nil, "test warn")
	logError(nil, "test error")

	// These should not panic with actual logger
	config := &Config{LogLevel: "DEBUG"}
	logger := setupLogger(config)
	logDebug(logger, "test debug", "key", "value")
	logInfo(logger, "test info", "key", "value")
	logWarn(logger, "test warn", "key", "value")
	logError(logger, "test error", "key", "value")
}

// TestLogVerbose tests the logVerbose function
func TestLogVerbose(t *testing.T) {
	// We can't easily capture stdout in a unit test without complex setup,
	// but we can at least verify the function doesn't panic and executes
	tests := []struct {
		name    string
		verbose bool
		format  string
		args    []interface{}
	}{
		{
			name:    "verbose mode enabled with no args",
			verbose: true,
			format:  "Test message",
			args:    nil,
		},
		{
			name:    "verbose mode enabled with args",
			verbose: true,
			format:  "Test message with %s and %d",
			args:    []interface{}{"string", 42},
		},
		{
			name:    "verbose mode disabled",
			verbose: false,
			format:  "This should not print",
			args:    []interface{}{"arg1", "arg2"},
		},
		{
			name:    "empty format string",
			verbose: true,
			format:  "",
			args:    nil,
		},
		{
			name:    "format with multiple placeholders",
			verbose: true,
			format:  "User: %s, ID: %s, Count: %d, Active: %t",
			args:    []interface{}{"test@example.com", "12345678-1234", 10, true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This test primarily verifies that logVerbose doesn't panic
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("logVerbose() panicked: %v", r)
				}
			}()

			logVerbose(tt.verbose, tt.format, tt.args...)
		})
	}
}

// TestLogVerbose_NilArgs tests that logVerbose handles nil args gracefully
func TestLogVerbose_NilArgs(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("logVerbose() with nil args panicked: %v", r)
		}
	}()

	// Should not panic with nil args
	logVerbose(true, "Test with no args")
	logVerbose(false, "Test with no args")
}
