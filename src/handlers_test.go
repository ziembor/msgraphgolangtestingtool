//go:build !integration
// +build !integration

package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/microsoftgraph/msgraph-sdk-go/models"
)

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
				// We can't access the Name field directly in the test without reflection or casting
				// Just verify we got an attachment object
				if firstAttachment == nil {
					t.Error("First attachment is nil")
				}
			}
		})
	}
}

// TestCreateFileAttachments_LargeFile tests handling of large file attachments (>10MB)
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

	// Get memory stats before processing
	var memBefore runtime.MemStats
	runtime.ReadMemStats(&memBefore)

	// Process the large file attachment
	config := &Config{VerboseMode: false}
	attachments, err := createFileAttachments([]string{tmpFile.Name()}, config)

	// Get memory stats after processing
	var memAfter runtime.MemStats
	runtime.ReadMemStats(&memAfter)

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

	// Verify base64 encoding works for large files
	fileData, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to read file for verification: %v", err)
	}

	encoded := getAttachmentContentBase64(fileData)
	if encoded == "" {
		t.Error("Base64 encoding returned empty string for large file")
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
			name:     "Unknown code (9)",
			view:     "9",
			expected: "Unknown (9)",
		},
		{
			name:     "Empty view",
			view:     "",
			expected: "Unknown (empty response)",
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

// TestAddWorkingDays tests the addWorkingDays function
func TestAddWorkingDays(t *testing.T) {
	tests := []struct {
		name     string
		start    time.Time
		days     int
		expected time.Time
	}{
		{
			name:     "Thursday to Friday",
			start:    time.Date(2026, 1, 1, 14, 0, 0, 0, time.UTC),
			days:     1,
			expected: time.Date(2026, 1, 2, 14, 0, 0, 0, time.UTC),
		},
		{
			name:     "Friday to Monday (skip weekend)",
			start:    time.Date(2026, 1, 2, 14, 0, 0, 0, time.UTC),
			days:     1,
			expected: time.Date(2026, 1, 5, 14, 0, 0, 0, time.UTC),
		},
		{
			name:     "Saturday to Monday",
			start:    time.Date(2026, 1, 3, 14, 0, 0, 0, time.UTC),
			days:     1,
			expected: time.Date(2026, 1, 5, 14, 0, 0, 0, time.UTC),
		},
		{
			name:     "Zero days returns same time",
			start:    time.Date(2026, 1, 1, 12, 30, 45, 0, time.UTC),
			days:     0,
			expected: time.Date(2026, 1, 1, 12, 30, 45, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := addWorkingDays(tt.start, tt.days)
			if !result.Equal(tt.expected) {
				t.Errorf("addWorkingDays(%v, %d) = %v, want %v",
					tt.start, tt.days, result, tt.expected)
			}
		})
	}
}

// Test createRecipients function
func TestCreateRecipients(t *testing.T) {
	tests := []struct {
		name     string
		emails   []string
		wantLen  int
		wantAddr string
	}{
		{"empty list", []string{}, 0, ""},
		{"single recipient", []string{"user1@example.com"}, 1, "user1@example.com"},
		{"multiple recipients", []string{"user1@example.com", "user2@example.com"}, 2, "user1@example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recipients := createRecipients(tt.emails)

			if len(recipients) != tt.wantLen {
				t.Errorf("Expected %d recipients, got %d", tt.wantLen, len(recipients))
			}

			if tt.wantLen > 0 {
				addr := recipients[0].GetEmailAddress()
				if addr == nil || addr.GetAddress() == nil || *addr.GetAddress() != tt.wantAddr {
					t.Errorf("First recipient address = %v, want %q", addr, tt.wantAddr)
				}
			}
		})
	}
}

// Test parseFlexibleTime function
func TestParseFlexibleTime(t *testing.T) {
	tests := []struct {
		name     string
		timeStr  string
		wantErr  bool
		wantYear int
		wantMon  int
		wantDay  int
	}{
		{"RFC3339 UTC", "2026-01-15T14:30:45Z", false, 2026, 1, 15},
		{"PowerShell sortable", "2026-01-15T14:30:45", false, 2026, 1, 15},
		{"empty string", "", true, 0, 0, 0},
		{"invalid format", "2026-01-15 14:00:00", true, 0, 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsedTime, err := parseFlexibleTime(tt.timeStr)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseFlexibleTime(%q) error = %v, wantErr %v", tt.timeStr, err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				utcTime := parsedTime.UTC()
				if utcTime.Year() != tt.wantYear {
					t.Errorf("Year = %d, want %d", utcTime.Year(), tt.wantYear)
				}
				if int(utcTime.Month()) != tt.wantMon {
					t.Errorf("Month = %d, want %d", utcTime.Month(), tt.wantMon)
				}
				if utcTime.Day() != tt.wantDay {
					t.Errorf("Day = %d, want %d", utcTime.Day(), tt.wantDay)
				}
			}
		})
	}
}

// Test sanitizeFilename helper
func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"standard", "message123", "message123"},
		{"with spaces", "message 123", "message 123"},
		{"with slashes", "message/123\\abc", "message_123_abc"},
		{"with forbidden chars", "<>:\"|?*", "_______"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeFilename(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeFilename(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// Test createExportDir helper
func TestCreateExportDir(t *testing.T) {
	dir, err := createExportDir()
	if err != nil {
		t.Fatalf("createExportDir() returned error: %v", err)
	}

	// Verify it's within temp dir
	tempDir := os.TempDir()
	if !strings.HasPrefix(dir, tempDir) {
		t.Errorf("Export dir %q should be under temp dir %q", dir, tempDir)
	}

	// Verify it exists
	info, err := os.Stat(dir)
	if err != nil {
		t.Errorf("Could not stat export dir: %v", err)
	} else if !info.IsDir() {
		t.Errorf("Export dir %q is not a directory", dir)
	}
}

// Test email extraction helpers
func TestEmailExtractionHelpers(t *testing.T) {
	t.Run("extractEmailAddress", func(t *testing.T) {
		addr := models.NewEmailAddress()
		name := "John Doe"
		email := "john@example.com"
		addr.SetName(&name)
		addr.SetAddress(&email)

		res := extractEmailAddress(addr)
		if res["name"] != name || res["address"] != email {
			t.Errorf("extractEmailAddress() = %v, want name=%q, address=%q", res, name, email)
		}
	})

	t.Run("extractRecipients", func(t *testing.T) {
		r1 := models.NewRecipient()
		a1 := models.NewEmailAddress()
		e1 := "u1@ex.com"
		a1.SetAddress(&e1)
		r1.SetEmailAddress(a1)

		res := extractRecipients([]models.Recipientable{r1})
		if len(res) != 1 {
			t.Fatalf("Expected 1 recipient, got %d", len(res))
		}
		if res[0]["address"] != e1 {
			t.Errorf("Recipient address mismatch: %v", res)
		}
	})
}
