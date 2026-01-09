package logger

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
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

	// Open or create file (append mode) with restrictive permissions (0600)
	// This ensures only the owner can read/write the file, protecting sensitive data
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return nil, fmt.Errorf("could not create CSV log file: %w", err)
	}

	// Apply platform-specific restrictive permissions for additional security
	if err := setRestrictivePermissions(file, filePath); err != nil {
		// Log warning but continue - file creation succeeded
		log.Printf("Warning: Failed to set restrictive permissions on log file %s: %v", filePath, err)
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

// setRestrictivePermissions sets platform-specific restrictive permissions on the CSV log file.
// On Unix/Linux/macOS: Sets permissions to 0600 (owner read/write only)
// On Windows: Attempts to set ACLs to restrict access to current user only
//
// This function provides defense-in-depth security to protect sensitive data that may
// appear in logs (e.g., error messages containing credentials). It's called after file
// creation to ensure restrictive permissions regardless of the system's umask.
//
// Returns an error if permission setting fails, but file creation has already succeeded.
func setRestrictivePermissions(file *os.File, filePath string) error {
	if runtime.GOOS == "windows" {
		// On Windows, file permissions are handled through ACLs (Access Control Lists).
		// Setting proper ACLs requires using Windows-specific APIs, which is complex
		// and requires the golang.org/x/sys/windows package.
		//
		// For now, we rely on the file being created in %TEMP% which typically has
		// appropriate user-specific permissions on Windows systems.
		//
		// Note: OpenFile with 0600 on Windows still creates the file, but Windows
		// uses inherited ACLs from the parent directory rather than Unix-style permissions.
		//
		// Future enhancement: Implement proper Windows ACL setting using:
		// - golang.org/x/sys/windows package
		// - SetNamedSecurityInfo or similar Windows APIs
		// - Create a DACL that grants access only to current user

		// No error - Windows relies on directory ACLs
		return nil
	}

	// Unix/Linux/macOS: Set file permissions to 0600 (owner read/write only)
	// This ensures that even if the file was created with a permissive umask,
	// we explicitly restrict access to the owner only.
	if err := file.Chmod(0600); err != nil {
		return fmt.Errorf("failed to chmod file to 0600: %w", err)
	}

	return nil
}
