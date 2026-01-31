//go:build !integration
// +build !integration

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
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

// TestValidateConfiguration tests validateConfiguration with various scenarios
func TestValidateConfiguration(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid with secret",
			config: &Config{
				TenantID: "12345678-1234-1234-1234-123456789012",
				ClientID: "abcdefgh-5678-9012-abcd-ef1234567890",
				Mailbox:  "user@example.com",
				Secret:   "my-secret",
				Action:   "getevents",
				OutputFormat: "text",
			},
			wantErr: false,
		},
		{
			name: "valid with thumbprint",
			config: &Config{
				TenantID:   "12345678-1234-1234-1234-123456789012",
				ClientID:   "abcdefgh-5678-9012-abcd-ef1234567890",
				Mailbox:    "user@example.com",
				Thumbprint: "ABC123DEF456",
				Action:     "getevents",
				OutputFormat: "text",
			},
			wantErr: false,
		},
		{
			name: "missing tenant ID",
			config: &Config{
				ClientID: "abcdefgh-5678-9012-abcd-ef1234567890",
				Mailbox:  "user@example.com",
				Secret:   "my-secret",
				OutputFormat: "text",
			},
			wantErr: true,
			errMsg:  "Tenant ID cannot be empty",
		},
		{
			name: "missing client ID",
			config: &Config{
				TenantID: "12345678-1234-1234-1234-123456789012",
				Mailbox:  "user@example.com",
				Secret:   "my-secret",
				OutputFormat: "text",
			},
			wantErr: true,
			errMsg:  "Client ID cannot be empty",
		},
		{
			name: "missing mailbox",
			config: &Config{
				TenantID: "12345678-1234-1234-1234-123456789012",
				ClientID: "abcdefgh-5678-9012-abcd-ef1234567890",
				Secret:   "my-secret",
				OutputFormat: "text",
			},
			wantErr: true,
			errMsg:  "invalid mailbox: email cannot be empty",
		},
		{
			name: "no authentication method",
			config: &Config{
				TenantID: "12345678-1234-1234-1234-123456789012",
				ClientID: "abcdefgh-5678-9012-abcd-ef1234567890",
				Mailbox:  "user@example.com",
				OutputFormat: "text",
			},
			wantErr: true,
			errMsg:  "missing authentication: must provide one of -secret, -pfx, or -thumbprint",
		},
		{
			name: "multiple authentication methods - secret and pfx",
			config: &Config{
				TenantID: "12345678-1234-1234-1234-123456789012",
				ClientID: "abcdefgh-5678-9012-abcd-ef1234567890",
				Mailbox:  "user@example.com",
				Secret:   "my-secret",
				PfxPath:  "/path/to/cert.pfx",
				OutputFormat: "text",
			},
			wantErr: true,
			errMsg:  "multiple authentication methods provided: use only one of -secret, -pfx, or -thumbprint",
		},
		{
			name: "invalid tenant GUID",
			config: &Config{
				TenantID: "invalid-guid",
				ClientID: "abcdefgh-1234-5678-90ab-cdef12345678",
				Mailbox:  "user@example.com",
				Secret:   "my-secret",
				OutputFormat: "text",
			},
			wantErr: true,
		},
		{
			name: "exportinbox valid",
			config: &Config{
				TenantID:     "12345678-1234-1234-1234-123456789012",
				ClientID:     "abcdefgh-1234-5678-90ab-cdef12345678",
				Mailbox:      "user@example.com",
				Secret:       "my-secret",
				Action:       ActionExportInbox,
				OutputFormat: "text",
			},
			wantErr: false,
		},
		{
			name: "searchandexport valid",
			config: &Config{
				TenantID:     "12345678-1234-1234-1234-123456789012",
				ClientID:     "abcdefgh-1234-5678-90ab-cdef12345678",
				Mailbox:      "user@example.com",
				Secret:       "my-secret",
				Action:       ActionSearchAndExport,
				MessageID:    "<unique-id@host>",
				OutputFormat: "text",
			},
			wantErr: false,
		},
		{
			name: "searchandexport missing messageid",
			config: &Config{
				TenantID: "12345678-1234-1234-1234-123456789012",
				ClientID: "abcdefgh-1234-5678-90ab-cdef12345678",
				Mailbox:  "user@example.com",
				Secret:   "my-secret",
				Action:   ActionSearchAndExport,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfiguration(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateConfiguration() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errMsg != "" {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("validateConfiguration() error = %v, should contain %q", err, tt.errMsg)
				}
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
				OutputFormat: "text",
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
				OutputFormat: "text",
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
				OutputFormat: "text",
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
				OutputFormat:    "text",
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
				OutputFormat:    "text",
			},
			wantErr: true,
			errMsg:  "Attachment file #2",
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
				OutputFormat: "text",
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
				OutputFormat: "text",
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
				OutputFormat: "text",
			},
			wantErr: true,
			errMsg:  "only supports checking one recipient at a time",
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

// Test stringSlice.Set() method
func TestStringSliceSet(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{"empty", "", nil},
		{"single", "a@example.com", []string{"a@example.com"}},
		{"multiple", "a@example.com,b@example.com", []string{"a@example.com", "b@example.com"}},
		{"with spaces", " a@example.com , b@example.com ", []string{"a@example.com", "b@example.com"}},
		{"trailing comma", "a@example.com,", []string{"a@example.com"}},
		{"three items", "a@example.com,b@example.com,c@example.com", []string{"a@example.com", "b@example.com", "c@example.com"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var s stringSlice
			err := s.Set(tt.input)
			if err != nil {
				t.Fatalf("Set() returned error: %v", err)
			}
			if !reflect.DeepEqual([]string(s), tt.expected) {
				t.Errorf("Set(%q) = %v, want %v", tt.input, s, tt.expected)
			}
		})
	}
}

// Test stringSlice.String() method
func TestStringSliceString(t *testing.T) {
	tests := []struct {
		name     string
		slice    stringSlice
		expected string
	}{
		{"nil", nil, ""},
		{"empty", stringSlice{}, ""},
		{"single", stringSlice{"a@example.com"}, "a@example.com"},
		{"multiple", stringSlice{"a@example.com", "b@example.com"}, "a@example.com,b@example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.slice.String()
			if result != tt.expected {
				t.Errorf("String() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// Test Config struct initialization and methods
func TestConfigStruct(t *testing.T) {
	config := NewConfig()

	if config.ShowVersion {
		t.Error("NewConfig defaults ShowVersion to true, but it should be false")
	}
	if config.MaxRetries != 3 {
		t.Errorf("NewConfig MaxRetries = %d, want 3", config.MaxRetries)
	}

	// Test manually populated struct
	manualConfig := &Config{
		VerboseMode: true,
		Count: 5,
		Action: "sendmail",
	}

	if !manualConfig.VerboseMode {
		t.Error("VerboseMode should be true")
	}
	if manualConfig.Count != 5 {
		t.Errorf("Count = %d, want 5", manualConfig.Count)
	}
}

// Test validateEmail function
func TestValidateEmail(t *testing.T) {
	tests := []struct {
		name    string
		email   string
		wantErr bool
	}{
		{"valid email", "user@example.com", false},
		{"valid with subdomain", "user@mail.example.com", false},
		{"no @", "userexample.com", true},
		{"empty", "", true},
		{"no domain", "user@", true},
		{"no local part", "@example.com", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateEmail(tt.email)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateEmail(%q) error = %v, wantErr %v", tt.email, err, tt.wantErr)
			}
		})
	}
}

// Test validateEmails function
func TestValidateEmails(t *testing.T) {
	tests := []struct {
		name      string
		emails    []string
		fieldName string
		wantErr   bool
	}{
		{"valid emails", []string{"user1@example.com", "user2@example.com"}, "recipients", false},
		{"empty list", []string{}, "recipients", false},
		{"one invalid", []string{"user1@example.com", "invalid"}, "recipients", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateEmails(tt.emails, tt.fieldName)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateEmails() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Test validateGUID function
func TestValidateGUID(t *testing.T) {
	tests := []struct {
		name      string
		guid      string
		fieldName string
		wantErr   bool
	}{
		{"valid GUID", "12345678-1234-1234-1234-123456789012", "Test ID", false},
		{"too short", "12345678-1234-1234-1234", "Test ID", true},
		{"empty", "", "Test ID", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateGUID(tt.guid, tt.fieldName)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateGUID(%q) error = %v, wantErr %v", tt.guid, err, tt.wantErr)
			}
		})
	}
}

// Test validateRFC3339Time function
func TestValidateRFC3339Time(t *testing.T) {
	tests := []struct {
		name      string
		timeStr   string
		fieldName string
		wantErr   bool
	}{
		{"valid RFC3339 UTC", "2026-01-15T14:00:00Z", "Start time", false},
		{"valid PowerShell sortable", "2026-01-15T14:00:00", "Start time", false},
		{"empty allowed", "", "Start time", false},
		{"invalid format", "2026-01-15 14:00:00", "Start time", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRFC3339Time(tt.timeStr, tt.fieldName)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateRFC3339Time(%q) error = %v, wantErr %v", tt.timeStr, err, tt.wantErr)
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
		"_msgraphtool_completions",
		"COMPREPLY",
		"COMP_WORDS",
		"-action",
		"-tenantid",
	}

	for _, required := range requiredStrings {
		if !strings.Contains(script, required) {
			t.Errorf("generateBashCompletion() missing required string: %q", required)
		}
	}
}

// TestGeneratePowerShellCompletion tests the PowerShell completion script generator
func TestGeneratePowerShellCompletion(t *testing.T) {
	script := generatePowerShellCompletion()

	if script == "" {
		t.Error("generatePowerShellCompletion() returned empty string")
	}

	requiredStrings := []string{
		"Register-ArgumentCompleter",
		"msgraphtool.exe",
		"param(",
		"-action",
	}

	for _, required := range requiredStrings {
		if !strings.Contains(script, required) {
			t.Errorf("generatePowerShellCompletion() missing required string: %q", required)
		}
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
		outputStr := string(output)
		// Check for common bash syntax errors, adjust as needed
		if strings.Contains(outputStr, "syntax error") || strings.Contains(outputStr, "unexpected") {
			t.Errorf("Bash completion script has invalid syntax: %v\nOutput: %s\nScript preview (first 500 chars):\n%s",
				err, outputStr, script[:minInt(500, len(script))])
		}
	}
}

// TestGeneratePowerShellCompletion_Syntax tests that the generated PowerShell completion script is syntactically valid
func TestGeneratePowerShellCompletion_Syntax(t *testing.T) {
	script := generatePowerShellCompletion()

	// Check if pwsh is available
	_, err := exec.LookPath("pwsh")
	if err != nil {
		t.Skip("Skipping PowerShell syntax test - pwsh not found")
	}

	// Create a temporary file with the script
	tmpFile, err := os.CreateTemp("", "ps-completion-*.ps1")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(script); err != nil {
		t.Fatalf("Failed to write script to temp file: %v", err)
	}
	tmpFile.Close()

	cmd := exec.Command("pwsh", "-NoProfile", "-NonInteractive", "-File", tmpFile.Name())
	output, err := cmd.CombinedOutput()

	if err != nil {
		outputStr := string(output)
		if strings.Contains(outputStr, "ParserError") || strings.Contains(outputStr, "syntax") {
			t.Errorf("PowerShell completion script has syntax errors: %v\nOutput: %s", err, outputStr)
		}
	}
}

// minInt returns the minimum of two integers (helper for tests)
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
