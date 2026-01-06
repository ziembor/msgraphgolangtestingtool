package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"
)

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

// Verbose logging helper
func logVerbose(verbose bool, format string, args ...interface{}) {
	if verbose {
		prefix := "[VERBOSE] "
		fmt.Printf(prefix+format+"\n", args...)
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
