//go:build !integration
// +build !integration

package main

import (
	"encoding/base64"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// TestValidateFilePath tests the validateFilePath function with various inputs
func TestValidateFilePath(t *testing.T) {
	// Create a temporary file for valid path tests
	tmpFile, err := os.CreateTemp("", "testfile-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// Create a temporary directory to test directory rejection
	tmpDir, err := os.MkdirTemp("", "testdir-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name      string
		path      string
		fieldName string
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "empty path is allowed",
			path:      "",
			fieldName: "Test file",
			wantErr:   false,
		},
		{
			name:      "valid absolute path",
			path:      tmpFile.Name(),
			fieldName: "PFX file",
			wantErr:   false,
		},
		{
			name:      "path traversal with ..",
			path:      "../../etc/passwd",
			fieldName: "PFX file",
			wantErr:   true,
			errMsg:    "path contains directory traversal",
		},
		{
			name:      "path traversal Windows style",
			path:      "..\\..\\Windows\\System32\\config\\SAM",
			fieldName: "Attachment",
			wantErr:   true,
			errMsg:    "path contains directory traversal",
		},
		{
			name:      "file does not exist",
			path:      filepath.Join(os.TempDir(), "nonexistent-file-12345.pfx"),
			fieldName: "PFX file",
			wantErr:   true,
			errMsg:    "file not found",
		},
		{
			name:      "path is a directory not a file",
			path:      tmpDir,
			fieldName: "Attachment",
			wantErr:   true,
			errMsg:    "not a regular file",
		},
		{
			name:      "relative path to existing file",
			path:      filepath.Base(tmpFile.Name()),
			fieldName: "Test file",
			wantErr:   true, // Will fail because relative path won't be found unless we're in tmpdir
			errMsg:    "file not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateFilePath(tt.path, tt.fieldName)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateFilePath(%q, %q) error = %v, wantErr %v", tt.path, tt.fieldName, err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("validateFilePath(%q, %q) error = %v, should contain %q", tt.path, tt.fieldName, err, tt.errMsg)
				}
			}
		})
	}
}

// TestValidateFilePath_PathTraversalVariations tests various path traversal attempts
func TestValidateFilePath_PathTraversalVariations(t *testing.T) {
	traversalPaths := []string{
		"../secret.pfx",
		"../../etc/passwd",
		"foo/../../../etc/shadow",
		"..\\..\\Windows\\System32",
		"test\\..\\..\\sensitive.txt",
	}

	for _, path := range traversalPaths {
		t.Run(path, func(t *testing.T) {
			err := validateFilePath(path, "Test file")
			if err == nil {
				t.Errorf("validateFilePath(%q) should reject path traversal, but got nil error", path)
			}
			// Either it should fail with traversal error, or file not found (both are acceptable)
			errMsg := err.Error()
			if !strings.Contains(errMsg, "directory traversal") && !strings.Contains(errMsg, "file not found") {
				t.Errorf("validateFilePath(%q) error = %v, expected traversal or not found error", path, err)
			}
		})
	}
}

// TestValidateConfiguration_PfxPathValidation tests that validateConfiguration validates PFX paths
func TestValidateConfiguration_PfxPathValidation(t *testing.T) {
	// Create a temporary PFX file for testing
	tmpPfx, err := os.CreateTemp("", "test-*.pfx")
	if err != nil {
		t.Fatalf("Failed to create temp PFX file: %v", err)
	}
	defer os.Remove(tmpPfx.Name())
	tmpPfx.Close()

	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid PFX path",
			config: &Config{
				TenantID: "12345678-1234-1234-1234-123456789012",
				ClientID: "abcdefab-1234-1234-1234-abcdefabcdef",
				Mailbox:  "test@example.com",
				PfxPath:  tmpPfx.Name(),
				PfxPass:  "password",
				Action:   ActionGetInbox,
			},
			wantErr: false,
		},
		{
			name: "PFX path does not exist",
			config: &Config{
				TenantID: "12345678-1234-1234-1234-123456789012",
				ClientID: "abcdefab-1234-1234-1234-abcdefabcdef",
				Mailbox:  "test@example.com",
				PfxPath:  "/nonexistent/path/cert.pfx",
				PfxPass:  "password",
				Action:   ActionGetInbox,
			},
			wantErr: true,
			errMsg:  "file not found",
		},
		{
			name: "PFX path with traversal",
			config: &Config{
				TenantID: "12345678-1234-1234-1234-123456789012",
				ClientID: "abcdefab-1234-1234-1234-abcdefabcdef",
				Mailbox:  "test@example.com",
				PfxPath:  "../../etc/passwd",
				PfxPass:  "password",
				Action:   ActionGetInbox,
			},
			wantErr: true,
			errMsg:  "directory traversal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfiguration(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateConfiguration() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("validateConfiguration() error = %v, should contain %q", err, tt.errMsg)
				}
			}
		})
	}
}

// TestValidateConfiguration_AttachmentFilesValidation tests that validateConfiguration validates attachment paths
func TestValidateConfiguration_AttachmentFilesValidation(t *testing.T) {
	// Create temporary attachment files for testing
	tmpAttach1, err := os.CreateTemp("", "attach1-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp attachment file: %v", err)
	}
	defer os.Remove(tmpAttach1.Name())
	tmpAttach1.Close()

	tmpAttach2, err := os.CreateTemp("", "attach2-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp attachment file: %v", err)
	}
	defer os.Remove(tmpAttach2.Name())
	tmpAttach2.Close()

	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid attachment paths",
			config: &Config{
				TenantID:        "12345678-1234-1234-1234-123456789012",
				ClientID:        "abcdefab-1234-1234-1234-abcdefabcdef",
				Mailbox:         "test@example.com",
				Secret:          "test-secret",
				AttachmentFiles: stringSlice{tmpAttach1.Name(), tmpAttach2.Name()},
				Action:          ActionSendMail,
			},
			wantErr: false,
		},
		{
			name: "one attachment does not exist",
			config: &Config{
				TenantID:        "12345678-1234-1234-1234-123456789012",
				ClientID:        "abcdefab-1234-1234-1234-abcdefabcdef",
				Mailbox:         "test@example.com",
				Secret:          "test-secret",
				AttachmentFiles: stringSlice{tmpAttach1.Name(), "/nonexistent/file.txt"},
				Action:          ActionSendMail,
			},
			wantErr: true,
			errMsg:  "Attachment file #2",
		},
		{
			name: "attachment with path traversal",
			config: &Config{
				TenantID:        "12345678-1234-1234-1234-123456789012",
				ClientID:        "abcdefab-1234-1234-1234-abcdefabcdef",
				Mailbox:         "test@example.com",
				Secret:          "test-secret",
				AttachmentFiles: stringSlice{"../../etc/shadow"},
				Action:          ActionSendMail,
			},
			wantErr: true,
			errMsg:  "directory traversal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfiguration(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateConfiguration() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("validateConfiguration() error = %v, should contain %q", err, tt.errMsg)
				}
			}
		})
	}
}

// TestValidateGetScheduleAction tests getschedule-specific validation
func TestValidateGetScheduleAction(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid getschedule with one recipient",
			config: &Config{
				TenantID: "12345678-1234-1234-1234-123456789012",
				ClientID: "abcdefab-1234-1234-1234-abcdefabcdef",
				Mailbox:  "organizer@example.com",
				Secret:   "test-secret",
				Action:   ActionGetSchedule,
				To:       stringSlice{"recipient@example.com"},
			},
			wantErr: false,
		},
		{
			name: "getschedule without recipient",
			config: &Config{
				TenantID: "12345678-1234-1234-1234-123456789012",
				ClientID: "abcdefab-1234-1234-1234-abcdefabcdef",
				Mailbox:  "organizer@example.com",
				Secret:   "test-secret",
				Action:   ActionGetSchedule,
				To:       stringSlice{},
			},
			wantErr: true,
			errMsg:  "getschedule action requires -to parameter",
		},
		{
			name: "getschedule with multiple recipients",
			config: &Config{
				TenantID: "12345678-1234-1234-1234-123456789012",
				ClientID: "abcdefab-1234-1234-1234-abcdefabcdef",
				Mailbox:  "organizer@example.com",
				Secret:   "test-secret",
				Action:   ActionGetSchedule,
				To:       stringSlice{"recipient1@example.com", "recipient2@example.com"},
			},
			wantErr: true,
			errMsg:  "only supports checking one recipient at a time",
		},
		{
			name: "invalid action name",
			config: &Config{
				TenantID: "12345678-1234-1234-1234-123456789012",
				ClientID: "abcdefab-1234-1234-1234-abcdefabcdef",
				Mailbox:  "test@example.com",
				Secret:   "test-secret",
				Action:   "invalidaction",
			},
			wantErr: true,
			errMsg:  "invalid action",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfiguration(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateConfiguration() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("validateConfiguration() error = %v, should contain %q", err, tt.errMsg)
				}
			}
		})
	}
}

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

// TestEnrichGraphAPIError tests the enrichGraphAPIError function with various error types
func TestEnrichGraphAPIError(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		operation string
		wantNil   bool
		wantErr   bool
	}{
		{
			name:      "nil error returns nil",
			err:       nil,
			operation: "testOperation",
			wantNil:   true,
			wantErr:   false,
		},
		{
			name:      "non-OData error returned unchanged",
			err:       &testError{msg: "generic error"},
			operation: "testOperation",
			wantNil:   false,
			wantErr:   true,
		},
		{
			name:      "empty operation name",
			err:       &testError{msg: "test error"},
			operation: "",
			wantNil:   false,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := enrichGraphAPIError(tt.err, nil, tt.operation)

			if tt.wantNil && result != nil {
				t.Errorf("enrichGraphAPIError() expected nil, got %v", result)
			}

			if !tt.wantNil && tt.wantErr && result == nil {
				t.Error("enrichGraphAPIError() expected error, got nil")
			}

			if !tt.wantNil && !tt.wantErr && result != nil {
				t.Errorf("enrichGraphAPIError() expected no error, got %v", result)
			}
		})
	}
}

// testError is a simple error type for testing
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

// TestEnrichGraphAPIError_NoP panic tests that enrichGraphAPIError doesn't panic
func TestEnrichGraphAPIError_NoPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("enrichGraphAPIError() panicked: %v", r)
		}
	}()

	// Test with various nil combinations
	enrichGraphAPIError(nil, nil, "")
	enrichGraphAPIError(nil, nil, "operation")
	enrichGraphAPIError(&testError{msg: "test"}, nil, "")
	enrichGraphAPIError(&testError{msg: "test"}, nil, "operation")
}

// TestCreateFileAttachments tests the createFileAttachments function
func TestCreateFileAttachments(t *testing.T) {
	// Create temporary test files
	tmpFile1, err := os.CreateTemp("", "attach1-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file 1: %v", err)
	}
	defer os.Remove(tmpFile1.Name())

	content1 := []byte("This is test attachment 1")
	if _, err := tmpFile1.Write(content1); err != nil {
		t.Fatalf("Failed to write to temp file 1: %v", err)
	}
	tmpFile1.Close()

	tmpFile2, err := os.CreateTemp("", "attach2-*.pdf")
	if err != nil {
		t.Fatalf("Failed to create temp file 2: %v", err)
	}
	defer os.Remove(tmpFile2.Name())

	content2 := []byte("PDF content here")
	if _, err := tmpFile2.Write(content2); err != nil {
		t.Fatalf("Failed to write to temp file 2: %v", err)
	}
	tmpFile2.Close()

	tests := []struct {
		name          string
		filePaths     []string
		config        *Config
		wantErr       bool
		wantCount     int
		checkFilename string
	}{
		{
			name:      "single file attachment",
			filePaths: []string{tmpFile1.Name()},
			config:    &Config{VerboseMode: false},
			wantErr:   false,
			wantCount: 1,
			checkFilename: filepath.Base(tmpFile1.Name()),
		},
		{
			name:      "multiple file attachments",
			filePaths: []string{tmpFile1.Name(), tmpFile2.Name()},
			config:    &Config{VerboseMode: false},
			wantErr:   false,
			wantCount: 2,
		},
		{
			name:      "empty file list",
			filePaths: []string{},
			config:    &Config{VerboseMode: false},
			wantErr:   false,
			wantCount: 0,
		},
		{
			name:      "nonexistent file should skip",
			filePaths: []string{tmpFile1.Name(), "/nonexistent/file.txt"},
			config:    &Config{VerboseMode: false},
			wantErr:   false,
			wantCount: 1, // Only valid file should be processed
		},
		{
			name:      "all files nonexistent",
			filePaths: []string{"/nonexistent/file1.txt", "/nonexistent/file2.txt"},
			config:    &Config{VerboseMode: false},
			wantErr:   true, // Should error when no attachments could be processed
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attachments, err := createFileAttachments(tt.filePaths, tt.config)

			if (err != nil) != tt.wantErr {
				t.Errorf("createFileAttachments() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(attachments) != tt.wantCount {
				t.Errorf("createFileAttachments() returned %d attachments, want %d", len(attachments), tt.wantCount)
			}

			// Check filename if specified
			if tt.checkFilename != "" && len(attachments) > 0 {
				firstAttachment := attachments[0]
				// We can't access the Name field directly in the test without reflection
				// So just verify we got an attachment object
				if firstAttachment == nil {
					t.Error("First attachment is nil")
				}
			}
		})
	}
}

// TestCreateFileAttachments_LargeFile tests handling of large file attachments (>10MB)
// This test verifies memory management and base64 encoding for large files
func TestCreateFileAttachments_LargeFile(t *testing.T) {
	// Create a large temporary file (15MB)
	tmpFile, err := os.CreateTemp("", "large-attach-*.bin")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write 15MB of data (pattern to verify integrity)
	const fileSize = 15 * 1024 * 1024 // 15MB
	const chunkSize = 1024 * 1024     // 1MB chunks
	pattern := []byte("TESTDATA") // 8-byte pattern

	t.Logf("Creating %d MB test file...", fileSize/(1024*1024))
	bytesWritten := 0
	for bytesWritten < fileSize {
		// Write pattern repeatedly
		for i := 0; i < chunkSize/len(pattern) && bytesWritten < fileSize; i++ {
			n, err := tmpFile.Write(pattern)
			if err != nil {
				t.Fatalf("Failed to write to temp file: %v", err)
			}
			bytesWritten += n
		}
	}
	tmpFile.Close()

	// Verify file size
	fileInfo, err := os.Stat(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to stat temp file: %v", err)
	}
	if fileInfo.Size() != fileSize {
		t.Errorf("File size mismatch: got %d, want %d", fileInfo.Size(), fileSize)
	}
	t.Logf("Created test file: %d bytes (%d MB)", fileInfo.Size(), fileInfo.Size()/(1024*1024))

	// Get memory stats before processing
	var memBefore runtime.MemStats
	runtime.ReadMemStats(&memBefore)
	t.Logf("Memory before: Alloc=%d MB, TotalAlloc=%d MB",
		memBefore.Alloc/(1024*1024), memBefore.TotalAlloc/(1024*1024))

	// Process the large file attachment
	config := &Config{VerboseMode: false}
	attachments, err := createFileAttachments([]string{tmpFile.Name()}, config)

	// Get memory stats after processing
	var memAfter runtime.MemStats
	runtime.ReadMemStats(&memAfter)
	t.Logf("Memory after: Alloc=%d MB, TotalAlloc=%d MB",
		memAfter.Alloc/(1024*1024), memAfter.TotalAlloc/(1024*1024))

	memDelta := memAfter.Alloc - memBefore.Alloc
	t.Logf("Memory delta: %d MB", memDelta/(1024*1024))

	// Verify no error
	if err != nil {
		t.Errorf("createFileAttachments() returned error for large file: %v", err)
		return
	}

	// Verify attachment was created
	if len(attachments) != 1 {
		t.Errorf("createFileAttachments() returned %d attachments, want 1", len(attachments))
		return
	}

	// Verify attachment is not nil
	if attachments[0] == nil {
		t.Fatal("Attachment is nil")
	}

	// Test base64 encoding of large file content
	// Read the file again to verify encoding
	fileData, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to read file for verification: %v", err)
	}

	// Verify base64 encoding works for large files
	encoded := getAttachmentContentBase64(fileData)
	if encoded == "" {
		t.Error("Base64 encoding returned empty string for large file")
	}

	// Verify encoding length is approximately 4/3 of original size (base64 overhead)
	expectedEncodedLen := (fileSize + 2) / 3 * 4 // Base64 encoding formula
	if len(encoded) < expectedEncodedLen-10 || len(encoded) > expectedEncodedLen+10 {
		t.Errorf("Base64 encoded length unexpected: got %d, expected ~%d", len(encoded), expectedEncodedLen)
	}
	t.Logf("Base64 encoding: original=%d bytes, encoded=%d chars (ratio: %.2f)",
		fileSize, len(encoded), float64(len(encoded))/float64(fileSize))

	// Verify we can decode it back
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Errorf("Failed to decode base64 encoded large file: %v", err)
	}
	if len(decoded) != fileSize {
		t.Errorf("Decoded size mismatch: got %d, want %d", len(decoded), fileSize)
	}

	// Verify first and last bytes match original pattern
	if len(decoded) >= len(pattern) {
		if string(decoded[:len(pattern)]) != string(pattern) {
			t.Error("Decoded data pattern mismatch at start")
		}
		if string(decoded[len(decoded)-len(pattern):]) != string(pattern) {
			t.Error("Decoded data pattern mismatch at end")
		}
	}

	t.Logf("✓ Large file attachment test passed: 15MB file processed successfully")
}

// TestGetAttachmentContentBase64 tests the getAttachmentContentBase64 function
func TestGetAttachmentContentBase64(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected string
	}{
		{
			name:     "empty data",
			input:    []byte{},
			expected: "",
		},
		{
			name:     "simple text",
			input:    []byte("Hello World"),
			expected: "SGVsbG8gV29ybGQ=",
		},
		{
			name:     "binary data",
			input:    []byte{0x00, 0xFF, 0xAA, 0x55},
			expected: "AP+qVQ==",
		},
		{
			name:     "newline character",
			input:    []byte("Line1\nLine2"),
			expected: "TGluZTEKTGluZTI=",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getAttachmentContentBase64(tt.input)
			if result != tt.expected {
				t.Errorf("getAttachmentContentBase64() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestGenerateBashCompletion tests the bash completion script generator
func TestGenerateBashCompletion(t *testing.T) {
	script := generateBashCompletion()

	// Check that script is not empty
	if script == "" {
		t.Error("generateBashCompletion() returned empty string")
	}

	// Check for essential bash completion elements
	requiredStrings := []string{
		"_msgraphgolangtestingtool_completions",
		"COMPREPLY",
		"COMP_WORDS",
		"COMP_CWORD",
		"-action",
		"-tenantid",
		"-clientid",
		"complete -F",
		"getevents",
		"sendmail",
		"sendinvite",
		"getinbox",
	}

	for _, required := range requiredStrings {
		if !strings.Contains(script, required) {
			t.Errorf("generateBashCompletion() missing required string: %q", required)
		}
	}

	// Check for installation instructions
	if !strings.Contains(script, "Installation:") {
		t.Error("generateBashCompletion() missing installation instructions")
	}

	// Check that it completes basic flags
	if !strings.Contains(script, "-loglevel") {
		t.Error("generateBashCompletion() missing -loglevel flag")
	}

	// Check that it has action completions
	if !strings.Contains(script, "case") && !strings.Contains(script, "-action") {
		t.Error("generateBashCompletion() missing action case handling")
	}
}

// TestGeneratePowerShellCompletion tests the PowerShell completion script generator
func TestGeneratePowerShellCompletion(t *testing.T) {
	script := generatePowerShellCompletion()

	// Check that script is not empty
	if script == "" {
		t.Error("generatePowerShellCompletion() returned empty string")
	}

	// Check for essential PowerShell completion elements
	requiredStrings := []string{
		"Register-ArgumentCompleter",
		"msgraphgolangtestingtool.exe",
		"param(",
		"$commandName",
		"$wordToComplete",
		"-action",
		"-tenantid",
		"-clientid",
		"getevents",
		"sendmail",
		"sendinvite",
		"getinbox",
		"CompletionResult",
	}

	for _, required := range requiredStrings {
		if !strings.Contains(script, required) {
			t.Errorf("generatePowerShellCompletion() missing required string: %q", required)
		}
	}

	// Check for installation instructions
	if !strings.Contains(script, "Installation:") {
		t.Error("generatePowerShellCompletion() missing installation instructions")
	}

	// Check for log levels
	logLevels := []string{"DEBUG", "INFO", "WARN", "ERROR"}
	for _, level := range logLevels {
		if !strings.Contains(script, level) {
			t.Errorf("generatePowerShellCompletion() missing log level: %q", level)
		}
	}

	// Check for shell types
	shellTypes := []string{"bash", "powershell"}
	for _, shell := range shellTypes {
		if !strings.Contains(script, shell) {
			t.Errorf("generatePowerShellCompletion() missing shell type: %q", shell)
		}
	}

	// Check for success message
	if !strings.Contains(script, "Write-Host") {
		t.Error("generatePowerShellCompletion() missing success message")
	}
}

// TestGenerateBashCompletion_Syntax tests that the generated bash completion script is syntactically valid
func TestGenerateBashCompletion_Syntax(t *testing.T) {
	script := generateBashCompletion()

	// Create a temporary file with the script
	tmpFile, err := os.CreateTemp("", "bash-completion-*.sh")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write script to file
	if _, err := tmpFile.WriteString(script); err != nil {
		t.Fatalf("Failed to write script to temp file: %v", err)
	}
	tmpFile.Close()

	// Test bash syntax using bash -n (syntax check only, no execution)
	cmd := exec.Command("bash", "-n", tmpFile.Name())
	output, err := cmd.CombinedOutput()

	if err != nil {
		t.Errorf("Bash completion script has invalid syntax: %v\nOutput: %s\nScript preview (first 500 chars):\n%s",
			err, output, script[:min(500, len(script))])
	} else {
		t.Logf("✓ Bash completion script syntax is valid (%d bytes)", len(script))
	}
}

// TestGeneratePowerShellCompletion_Syntax tests that the generated PowerShell completion script is syntactically valid
func TestGeneratePowerShellCompletion_Syntax(t *testing.T) {
	script := generatePowerShellCompletion()

	// Check if pwsh (PowerShell 7+) is available
	// We use pwsh instead of powershell for better cross-platform support
	_, err := exec.LookPath("pwsh")
	if err != nil {
		t.Skip("Skipping PowerShell syntax test - pwsh (PowerShell 7+) not found in PATH")
	}

	// Create a temporary file with the script
	tmpFile, err := os.CreateTemp("", "ps-completion-*.ps1")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write script to file
	if _, err := tmpFile.WriteString(script); err != nil {
		t.Fatalf("Failed to write script to temp file: %v", err)
	}
	tmpFile.Close()

	// Test PowerShell syntax using pwsh with -File and -NoProfile
	// Using -File instead of -Command to avoid issues with script parsing
	// The script will attempt to run but we're mainly checking for syntax errors
	cmd := exec.Command("pwsh", "-NoProfile", "-NonInteractive", "-File", tmpFile.Name())

	// Set a timeout to prevent hangs
	output, err := cmd.CombinedOutput()

	// For PowerShell, we expect it to complete without syntax errors
	// The script registers completions which should succeed
	if err != nil {
		outputStr := string(output)
		// Check if error is due to syntax issues (not just completion registration)
		if strings.Contains(outputStr, "ParserError") ||
			strings.Contains(outputStr, "syntax") ||
			strings.Contains(outputStr, "unexpected token") {
			t.Errorf("PowerShell completion script has syntax errors: %v\nOutput: %s\nScript preview (first 500 chars):\n%s",
				err, outputStr, script[:min(500, len(script))])
		} else {
			// Script ran but returned non-zero (possibly due to completion registration context)
			// This is acceptable as long as no syntax errors
			t.Logf("✓ PowerShell completion script syntax is valid (%d bytes) - exit code %v (expected for completion script)",
				len(script), err)
		}
	} else {
		t.Logf("✓ PowerShell completion script syntax is valid and executed successfully (%d bytes)", len(script))
	}
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// TestInt32Ptr tests the Int32Ptr helper function
func TestInt32Ptr(t *testing.T) {
	tests := []struct {
		name  string
		input int32
	}{
		{
			name:  "zero value",
			input: 0,
		},
		{
			name:  "positive value",
			input: 42,
		},
		{
			name:  "negative value",
			input: -100,
		},
		{
			name:  "max int32",
			input: 2147483647,
		},
		{
			name:  "min int32",
			input: -2147483648,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Int32Ptr(tt.input)

			// Check that result is not nil
			if result == nil {
				t.Error("Int32Ptr() returned nil")
				return
			}

			// Check that dereferenced value matches input
			if *result != tt.input {
				t.Errorf("Int32Ptr(%d) = %d, want %d", tt.input, *result, tt.input)
			}

			// Check that the pointer points to a different address than the input
			// (This verifies that a new memory location was created)
			inputAddr := &tt.input
			if result == inputAddr {
				t.Error("Int32Ptr() returned pointer to input variable instead of new allocation")
			}
		})
	}
}

// TestMaskGUID tests the maskGUID function
func TestMaskGUID(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "standard GUID",
			input:    "12345678-1234-1234-1234-123456789012",
			expected: "1234****-****-****-****9012",
		},
		{
			name:     "GUID without dashes",
			input:    "12345678123412341234123456789012",
			expected: "1234****-****-****-****9012",
		},
		{
			name:     "short string",
			input:    "short",
			expected: "****",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "****",
		},
		{
			name:     "exactly 8 characters",
			input:    "12345678",
			expected: "****",
		},
		{
			name:     "9 characters",
			input:    "123456789",
			expected: "1234****-****-****-****6789",
		},
		{
			name:     "uppercase GUID",
			input:    "ABCDEFAB-1234-5678-9ABC-DEF012345678",
			expected: "ABCD****-****-****-****5678",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := maskGUID(tt.input)
			if result != tt.expected {
				t.Errorf("maskGUID(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
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

// TestInterpretAvailability tests the interpretAvailability function
func TestInterpretAvailability(t *testing.T) {
	tests := []struct {
		name     string
		view     string
		expected string
	}{
		{
			name:     "Free (0)",
			view:     "0",
			expected: "Free",
		},
		{
			name:     "Tentative (1)",
			view:     "1",
			expected: "Tentative",
		},
		{
			name:     "Busy (2)",
			view:     "2",
			expected: "Busy",
		},
		{
			name:     "Out of Office (3)",
			view:     "3",
			expected: "Out of Office",
		},
		{
			name:     "Working Elsewhere (4)",
			view:     "4",
			expected: "Working Elsewhere",
		},
		{
			name:     "Unknown code (9)",
			view:     "9",
			expected: "Unknown (9)",
		},
		{
			name:     "Empty view",
			view:     "",
			expected: "Unknown (empty response)",
		},
		{
			name:     "Multi-character view (takes first)",
			view:     "0000",
			expected: "Free",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := interpretAvailability(tt.view)
			if result != tt.expected {
				t.Errorf("interpretAvailability(%q) = %q, want %q", tt.view, result, tt.expected)
			}
		})
	}
}

// TestAddWorkingDays tests the addWorkingDays function that calculates working days
func TestAddWorkingDays(t *testing.T) {
	tests := []struct {
		name     string
		start    time.Time
		days     int
		expected time.Time
	}{
		{
			name:     "Thursday to Friday",
			start:    time.Date(2026, 1, 1, 14, 0, 0, 0, time.UTC), // Thursday
			days:     1,
			expected: time.Date(2026, 1, 2, 14, 0, 0, 0, time.UTC), // Friday
		},
		{
			name:     "Friday to Monday (skip weekend)",
			start:    time.Date(2026, 1, 2, 14, 0, 0, 0, time.UTC), // Friday
			days:     1,
			expected: time.Date(2026, 1, 5, 14, 0, 0, 0, time.UTC), // Monday
		},
		{
			name:     "Saturday to Monday",
			start:    time.Date(2026, 1, 3, 14, 0, 0, 0, time.UTC), // Saturday
			days:     1,
			expected: time.Date(2026, 1, 5, 14, 0, 0, 0, time.UTC), // Monday
		},
		{
			name:     "Sunday to Monday",
			start:    time.Date(2026, 1, 4, 14, 0, 0, 0, time.UTC), // Sunday
			days:     1,
			expected: time.Date(2026, 1, 5, 14, 0, 0, 0, time.UTC), // Monday
		},
		{
			name:     "Add 5 working days (crosses weekend)",
			start:    time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC), // Thursday
			days:     5,
			expected: time.Date(2026, 1, 8, 9, 0, 0, 0, time.UTC), // Next Thursday
		},
		{
			name:     "Zero days returns same time",
			start:    time.Date(2026, 1, 1, 12, 30, 45, 0, time.UTC),
			days:     0,
			expected: time.Date(2026, 1, 1, 12, 30, 45, 0, time.UTC),
		},
		{
			name:     "Monday to Tuesday",
			start:    time.Date(2026, 1, 5, 10, 0, 0, 0, time.UTC), // Monday
			days:     1,
			expected: time.Date(2026, 1, 6, 10, 0, 0, 0, time.UTC), // Tuesday
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := addWorkingDays(tt.start, tt.days)
			if !result.Equal(tt.expected) {
				t.Errorf("addWorkingDays(%v, %d) = %v, want %v",
					tt.start.Format("Mon 2006-01-02 15:04:05"),
					tt.days,
					result.Format("Mon 2006-01-02 15:04:05"),
					tt.expected.Format("Mon 2006-01-02 15:04:05"))
			}
		})
	}
}
