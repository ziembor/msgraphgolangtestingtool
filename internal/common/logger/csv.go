package logger

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// CSVLogger handles CSV logging operations with periodic buffering
type CSVLogger struct {
	writer     *csv.Writer
	file       *os.File
	toolName   string    // Tool name for filename (e.g., "msgraphgolangtestingtool", "smtptool")
	action     string    // Action being performed
	rowCount   int       // Number of rows written since last flush
	lastFlush  time.Time // Time of last flush
	flushEvery int       // Flush every N rows
}

// NewCSVLogger creates a new CSV logger for the specified tool and action.
// The toolName parameter differentiates between tools (e.g., "msgraphgolangtestingtool" or "smtptool").
// Filename pattern: %TEMP%\_{toolName}_{action}_{date}.csv
//
// Examples:
//   - _msgraphgolangtestingtool_sendmail_2026-01-09.csv
//   - _smtptool_teststarttls_2026-01-09.csv
func NewCSVLogger(toolName, action string) (*CSVLogger, error) {
	// Get temp directory
	tempDir := os.TempDir()

	// Create filename with tool name, action, and current date
	dateStr := time.Now().Format("2006-01-02")
	fileName := fmt.Sprintf("_%s_%s_%s.csv", toolName, action, dateStr)
	filePath := filepath.Join(tempDir, fileName)

	// Open or create file (append mode)
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("could not create CSV log file: %w", err)
	}

	logger := &CSVLogger{
		writer:     csv.NewWriter(file),
		file:       file,
		toolName:   toolName,
		action:     action,
		rowCount:   0,
		lastFlush:  time.Now(),
		flushEvery: 10, // Flush every 10 rows or on close
	}

	fmt.Printf("Logging to: %s\n\n", filePath)
	return logger, nil
}

// WriteHeader writes a CSV header with the provided column names.
// This should be called once after creating the logger if the file is new.
// The timestamp column is automatically prepended to the header.
func (l *CSVLogger) WriteHeader(columns []string) error {
	// Prepend "Timestamp" to the header
	header := append([]string{"Timestamp"}, columns...)
	if err := l.writer.Write(header); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}
	l.writer.Flush()
	return l.writer.Error()
}

// WriteRow writes a row to the CSV file with periodic buffering.
// The timestamp is automatically prepended to each row.
// Rows are flushed every N rows or every 5 seconds to balance performance and data safety.
func (l *CSVLogger) WriteRow(row []string) error {
	if l.writer == nil {
		return fmt.Errorf("CSV writer is not initialized")
	}

	// Prepend timestamp
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	fullRow := append([]string{timestamp}, row...)

	if err := l.writer.Write(fullRow); err != nil {
		return fmt.Errorf("failed to write CSV row: %w", err)
	}

	l.rowCount++

	// Flush every N rows or every 5 seconds
	if l.rowCount%l.flushEvery == 0 || time.Since(l.lastFlush) > 5*time.Second {
		l.writer.Flush()
		l.lastFlush = time.Now()
		if err := l.writer.Error(); err != nil {
			return fmt.Errorf("failed to flush CSV: %w", err)
		}
	}

	return nil
}

// Close closes the CSV file, ensuring all buffered data is flushed.
// Always call this method when done logging to prevent data loss.
func (l *CSVLogger) Close() error {
	if l.writer != nil {
		l.writer.Flush() // Always flush remaining rows on close
		if err := l.writer.Error(); err != nil {
			return fmt.Errorf("error flushing CSV on close: %w", err)
		}
	}
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

// ShouldWriteHeader checks if the CSV file is new (empty) and needs a header.
// Returns true if the file was just created or is empty.
func (l *CSVLogger) ShouldWriteHeader() (bool, error) {
	fileInfo, err := l.file.Stat()
	if err != nil {
		return false, fmt.Errorf("could not stat CSV file: %w", err)
	}
	return fileInfo.Size() == 0, nil
}
