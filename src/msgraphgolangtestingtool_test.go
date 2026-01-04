package main

import (
	"reflect"
	"testing"
)

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
		{"extra spaces", "a@example.com  ,  , b@example.com", []string{"a@example.com", "b@example.com"}},
		{"leading comma", ",a@example.com", []string{"a@example.com"}},
		{"only spaces", "   ,   ,   ", nil},
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
		{"three items", stringSlice{"a@example.com", "b@example.com", "c@example.com"}, "a@example.com,b@example.com,c@example.com"},
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

// Test createRecipients function
func TestCreateRecipients(t *testing.T) {
	tests := []struct {
		name     string
		emails   []string
		wantLen  int
		wantAddr string // First recipient address to verify
	}{
		{"empty list", []string{}, 0, ""},
		{"single recipient", []string{"user1@example.com"}, 1, "user1@example.com"},
		{"multiple recipients", []string{"user1@example.com", "user2@example.com"}, 2, "user1@example.com"},
		{"three recipients", []string{"a@example.com", "b@example.com", "c@example.com"}, 3, "a@example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recipients := createRecipients(tt.emails)

			if len(recipients) != tt.wantLen {
				t.Errorf("Expected %d recipients, got %d", tt.wantLen, len(recipients))
			}

			// Verify first recipient address if we have any
			if tt.wantLen > 0 {
				addr := recipients[0].GetEmailAddress()
				if addr == nil || addr.GetAddress() == nil || *addr.GetAddress() != tt.wantAddr {
					t.Errorf("First recipient address = %v, want %q", addr, tt.wantAddr)
				}
			}
		})
	}
}

// Test maskSecret function
func TestMaskSecret(t *testing.T) {
	tests := []struct {
		name     string
		secret   string
		expected string
	}{
		{"empty", "", "********"},
		{"single char", "x", "********"},
		{"two chars", "ab", "********"},
		{"short", "abc", "********"},
		{"exactly 8 chars", "12345678", "********"},
		{"9 chars - shows first/last 4", "123456789", "1234********6789"},
		{"long secret", "very-long-secret-string", "very********ring"},
		{"12 chars", "abcdefghijkl", "abcd********ijkl"},
		{"medium", "my-secret-key", "my-s********-key"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := maskSecret(tt.secret)
			if result != tt.expected {
				t.Errorf("maskSecret(%q) = %q, want %q", tt.secret, result, tt.expected)
			}
		})
	}
}

// Test validateConfiguration function (basic tests without format validation)
func TestValidateConfiguration(t *testing.T) {
	tests := []struct {
		name    string
		flags   *Flags
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid with secret",
			flags: &Flags{
				TenantID: "12345678-1234-1234-1234-123456789012",
				ClientID: "abcdefgh-5678-9012-abcd-ef1234567890",
				Mailbox:  "user@example.com",
				Secret:   "my-secret",
				Action:   "getevents",
			},
			wantErr: false,
		},
		{
			name: "valid with pfx",
			flags: &Flags{
				TenantID: "12345678-1234-1234-1234-123456789012",
				ClientID: "abcdefgh-5678-9012-abcd-ef1234567890",
				Mailbox:  "user@example.com",
				PfxPath:  "/path/to/cert.pfx",
				Action:   "getevents",
			},
			wantErr: false,
		},
		{
			name: "valid with thumbprint",
			flags: &Flags{
				TenantID:   "12345678-1234-1234-1234-123456789012",
				ClientID:   "abcdefgh-5678-9012-abcd-ef1234567890",
				Mailbox:    "user@example.com",
				Thumbprint: "ABC123DEF456",
				Action:     "getevents",
			},
			wantErr: false,
		},
		{
			name: "missing tenant ID",
			flags: &Flags{
				ClientID: "abcdefgh-5678-9012-abcd-ef1234567890",
				Mailbox:  "user@example.com",
				Secret:   "my-secret",
			},
			wantErr: true,
			errMsg:  "Tenant ID cannot be empty",
		},
		{
			name: "missing client ID",
			flags: &Flags{
				TenantID: "12345678-1234-1234-1234-123456789012",
				Mailbox:  "user@example.com",
				Secret:   "my-secret",
			},
			wantErr: true,
			errMsg:  "Client ID cannot be empty",
		},
		{
			name: "missing mailbox",
			flags: &Flags{
				TenantID: "12345678-1234-1234-1234-123456789012",
				ClientID: "abcdefgh-5678-9012-abcd-ef1234567890",
				Secret:   "my-secret",
			},
			wantErr: true,
			errMsg:  "invalid mailbox: email cannot be empty",
		},
		{
			name: "no authentication method",
			flags: &Flags{
				TenantID: "12345678-1234-1234-1234-123456789012",
				ClientID: "abcdefgh-5678-9012-abcd-ef1234567890",
				Mailbox:  "user@example.com",
			},
			wantErr: true,
			errMsg:  "missing authentication: must provide one of -secret, -pfx, or -thumbprint",
		},
		{
			name: "multiple authentication methods - secret and pfx",
			flags: &Flags{
				TenantID: "12345678-1234-1234-1234-123456789012",
				ClientID: "abcdefgh-5678-9012-abcd-ef1234567890",
				Mailbox:  "user@example.com",
				Secret:   "my-secret",
				PfxPath:  "/path/to/cert.pfx",
			},
			wantErr: true,
			errMsg:  "multiple authentication methods provided: use only one of -secret, -pfx, or -thumbprint",
		},
		{
			name: "multiple authentication methods - all three",
			flags: &Flags{
				TenantID:   "12345678-1234-1234-1234-123456789012",
				ClientID:   "abcdefgh-5678-9012-abcd-ef1234567890",
				Mailbox:    "user@example.com",
				Secret:     "my-secret",
				PfxPath:    "/path/to/cert.pfx",
				Thumbprint: "ABC123",
			},
			wantErr: true,
			errMsg:  "multiple authentication methods provided: use only one of -secret, -pfx, or -thumbprint",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfiguration(tt.flags)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateConfiguration() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err.Error() != tt.errMsg {
				t.Errorf("validateConfiguration() error message = %q, want %q", err.Error(), tt.errMsg)
			}
		})
	}
}

// Test Flags struct initialization
func TestFlagsStruct(t *testing.T) {
	flags := &Flags{
		ShowVersion: false,
		TenantID:    "test-tenant",
		ClientID:    "test-client",
		Mailbox:     "test@example.com",
		Action:      "sendmail",
		Secret:      "test-secret",
		Count:       5,
	}

	if flags.TenantID != "test-tenant" {
		t.Errorf("TenantID = %q, want %q", flags.TenantID, "test-tenant")
	}
	if flags.Count != 5 {
		t.Errorf("Count = %d, want %d", flags.Count, 5)
	}
	if flags.Action != "sendmail" {
		t.Errorf("Action = %q, want %q", flags.Action, "sendmail")
	}
}

// Test Config struct
func TestConfigStruct(t *testing.T) {
	config := &Config{
		VerboseMode: true,
	}

	if !config.VerboseMode {
		t.Errorf("VerboseMode = %v, want %v", config.VerboseMode, true)
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
		{"valid with dots", "first.last@example.com", false},
		{"no @", "userexample.com", true},
		{"empty", "", true},
		{"no domain", "user@", true},
		{"no local part", "@example.com", true},
		{"multiple @", "user@@example.com", true},
		{"only @", "@", true},
		{"with spaces gets trimmed", "  user@example.com  ", false},
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
		{"all invalid", []string{"invalid1", "invalid2"}, "recipients", true},
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
		{"valid GUID lowercase", "abcdefgh-1234-5678-90ab-cdef12345678", "Test ID", false},
		{"too short", "12345678-1234-1234-1234", "Test ID", true},
		{"too long", "12345678-1234-1234-1234-1234567890123", "Test ID", true},
		{"no dashes", "12345678123412341234123456789012", "Test ID", true},
		{"wrong dash positions", "1234567-81234-1234-1234-123456789012", "Test ID", true},
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
		{"valid RFC3339 with offset", "2026-01-15T14:00:00+01:00", "Start time", false},
		{"empty allowed", "", "Start time", false},
		{"invalid format", "2026-01-15 14:00:00", "Start time", true},
		{"missing T", "2026-01-15T14:00:00", "Start time", true},
		{"invalid date", "2026-13-01T14:00:00Z", "Start time", true},
		{"missing timezone", "2026-01-15T14:00:00", "Start time", true},
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

// Test enhanced validateConfiguration with format checking
func TestValidateConfigurationEnhanced(t *testing.T) {
	tests := []struct {
		name    string
		flags   *Flags
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid configuration",
			flags: &Flags{
				TenantID: "12345678-1234-1234-1234-123456789012",
				ClientID: "abcdefgh-1234-5678-90ab-cdef12345678",
				Mailbox:  "user@example.com",
				Secret:   "my-secret",
				Action:   "getevents",
			},
			wantErr: false,
		},
		{
			name: "invalid tenant GUID",
			flags: &Flags{
				TenantID: "invalid-guid",
				ClientID: "abcdefgh-1234-5678-90ab-cdef12345678",
				Mailbox:  "user@example.com",
				Secret:   "my-secret",
			},
			wantErr: true,
		},
		{
			name: "invalid mailbox email",
			flags: &Flags{
				TenantID: "12345678-1234-1234-1234-123456789012",
				ClientID: "abcdefgh-1234-5678-90ab-cdef12345678",
				Mailbox:  "invalid-email",
				Secret:   "my-secret",
			},
			wantErr: true,
		},
		{
			name: "invalid To recipient",
			flags: &Flags{
				TenantID: "12345678-1234-1234-1234-123456789012",
				ClientID: "abcdefgh-1234-5678-90ab-cdef12345678",
				Mailbox:  "user@example.com",
				Secret:   "my-secret",
				To:       []string{"invalid-email"},
				Action:   "sendmail",
			},
			wantErr: true,
		},
		{
			name: "invalid start time",
			flags: &Flags{
				TenantID:  "12345678-1234-1234-1234-123456789012",
				ClientID:  "abcdefgh-1234-5678-90ab-cdef12345678",
				Mailbox:   "user@example.com",
				Secret:    "my-secret",
				StartTime: "invalid-time",
				Action:    "sendinvite",
			},
			wantErr: true,
		},
		{
			name: "invalid action",
			flags: &Flags{
				TenantID: "12345678-1234-1234-1234-123456789012",
				ClientID: "abcdefgh-1234-5678-90ab-cdef12345678",
				Mailbox:  "user@example.com",
				Secret:   "my-secret",
				Action:   "invalidaction",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfiguration(tt.flags)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateConfiguration() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
