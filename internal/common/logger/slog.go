package logger

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
)

// SetupLogger configures a structured logger based on the provided configuration.
// Valid levels are: DEBUG, INFO, WARN, ERROR
// If verboseMode is true, it overrides logLevel to DEBUG.
// Returns a configured *slog.Logger that can be used throughout the application.
func SetupLogger(verboseMode bool, logLevel string) *slog.Logger {
	// Determine log level
	level := ParseLogLevel(logLevel)

	// Verbose mode overrides log level to DEBUG
	if verboseMode {
		level = slog.LevelDebug
	}

	// Create handler options with the determined level
	opts := &slog.HandlerOptions{
		Level: level,
	}

	// Create a text handler that writes to stderr
	handler := slog.NewTextHandler(os.Stderr, opts)

	// Create and return the logger
	return slog.New(handler)
}

// ParseLogLevel converts a string log level to slog.Level.
// Defaults to INFO if an invalid level is provided.
func ParseLogLevel(levelStr string) slog.Level {
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

// LogDebug logs a debug message if debug level is enabled
func LogDebug(logger *slog.Logger, msg string, args ...any) {
	if logger != nil {
		logger.Debug(msg, args...)
	}
}

// LogInfo logs an informational message
func LogInfo(logger *slog.Logger, msg string, args ...any) {
	if logger != nil {
		logger.Info(msg, args...)
	}
}

// LogWarn logs a warning message
func LogWarn(logger *slog.Logger, msg string, args ...any) {
	if logger != nil {
		logger.Warn(msg, args...)
	}
}

// LogError logs an error message
func LogError(logger *slog.Logger, msg string, args ...any) {
	if logger != nil {
		logger.Error(msg, args...)
	}
}

// LogVerbose is a helper for verbose logging that writes directly to stderr.
// This is useful for diagnostic output that bypasses the structured logger.
func LogVerbose(verbose bool, format string, args ...interface{}) {
	if verbose {
		prefix := "[VERBOSE] "
		fmt.Fprintf(os.Stderr, prefix+format+"\n", args...)
	}
}
