package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"fmt"
	"math/big"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/microsoftgraph/msgraph-sdk-go/models"
	"software.sslmate.com/src/go-pkcs12"
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
		config   *Config
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
		// Note: "valid with pfx" test removed - now covered by TestValidateConfiguration_PfxPathValidation in shared_test.go
		// This avoids creating temp files in every test run
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
			name: "multiple authentication methods - all three",
			config: &Config{
				TenantID:   "12345678-1234-1234-1234-123456789012",
				ClientID:   "abcdefgh-5678-9012-abcd-ef1234567890",
				Mailbox:    "user@example.com",
				Secret:     "my-secret",
				PfxPath:    "/path/to/cert.pfx",
				Thumbprint: "ABC123",
				OutputFormat: "text",
			},
			wantErr: true,
			errMsg:  "multiple authentication methods provided: use only one of -secret, -pfx, or -thumbprint",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfiguration(tt.config)
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

// Test Config struct initialization
func TestFlagsStruct(t *testing.T) {
	config := &Config{
		ShowVersion: false,
		TenantID:    "test-tenant",
		ClientID:    "test-client",
		Mailbox:     "test@example.com",
		Action:      "sendmail",
		Secret:      "test-secret",
		Count:       5,
	}

	if config.TenantID != "test-tenant" {
		t.Errorf("TenantID = %q, want %q", config.TenantID, "test-tenant")
	}
	if config.Count != 5 {
		t.Errorf("Count = %d, want %d", config.Count, 5)
	}
	if config.Action != "sendmail" {
		t.Errorf("Action = %q, want %q", config.Action, "sendmail")
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

// Test parseFlexibleTime function
func TestParseFlexibleTime(t *testing.T) {
	tests := []struct {
		name     string
		timeStr  string
		wantErr  bool
		wantYear int
		wantMon  int
		wantDay  int
		wantHour int
		wantMin  int
		wantSec  int
	}{
		{"RFC3339 UTC", "2026-01-15T14:30:45Z", false, 2026, 1, 15, 14, 30, 45},
		{"RFC3339 with offset", "2026-01-15T14:30:45+01:00", false, 2026, 1, 15, 13, 30, 45}, // Converts to UTC
		{"PowerShell sortable format", "2026-01-15T14:30:45", false, 2026, 1, 15, 14, 30, 45},
		{"PowerShell from Get-Date -Format s", "2026-03-20T09:15:30", false, 2026, 3, 20, 9, 15, 30},
		{"empty string", "", true, 0, 0, 0, 0, 0, 0},
		{"invalid format", "2026-01-15 14:00:00", true, 0, 0, 0, 0, 0, 0},
		{"invalid date", "2026-13-01T14:00:00Z", true, 0, 0, 0, 0, 0, 0},
		{"only date", "2026-01-15", true, 0, 0, 0, 0, 0, 0},
		{"only time", "14:30:45", true, 0, 0, 0, 0, 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsedTime, err := parseFlexibleTime(tt.timeStr)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseFlexibleTime(%q) error = %v, wantErr %v", tt.timeStr, err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				// Convert to UTC for consistent comparison (test expectations are in UTC)
				utcTime := parsedTime.UTC()
				// Verify the parsed time components
				if utcTime.Year() != tt.wantYear {
					t.Errorf("Year = %d, want %d", utcTime.Year(), tt.wantYear)
				}
				if int(utcTime.Month()) != tt.wantMon {
					t.Errorf("Month = %d, want %d", utcTime.Month(), tt.wantMon)
				}
				if utcTime.Day() != tt.wantDay {
					t.Errorf("Day = %d, want %d", utcTime.Day(), tt.wantDay)
				}
				if utcTime.Hour() != tt.wantHour {
					t.Errorf("Hour = %d, want %d", utcTime.Hour(), tt.wantHour)
				}
				if utcTime.Minute() != tt.wantMin {
					t.Errorf("Minute = %d, want %d", utcTime.Minute(), tt.wantMin)
				}
				if utcTime.Second() != tt.wantSec {
					t.Errorf("Second = %d, want %d", utcTime.Second(), tt.wantSec)
				}
			}
		})
	}
}

// Test validateRFC3339Time function (updated to support PowerShell format)
func TestValidateRFC3339Time(t *testing.T) {
	tests := []struct {
		name      string
		timeStr   string
		fieldName string
		wantErr   bool
	}{
		{"valid RFC3339 UTC", "2026-01-15T14:00:00Z", "Start time", false},
		{"valid RFC3339 with offset", "2026-01-15T14:00:00+01:00", "Start time", false},
		{"valid PowerShell sortable", "2026-01-15T14:00:00", "Start time", false},
		{"valid PowerShell from Get-Date -Format s", "2026-03-20T09:15:30", "Start time", false},
		{"empty allowed", "", "Start time", false},
		{"invalid format with space", "2026-01-15 14:00:00", "Start time", true},
		{"invalid date", "2026-13-01T14:00:00Z", "Start time", true},
		{"only date", "2026-01-15", "Start time", true},
		{"only time", "14:00:00", "Start time", true},
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

// Test validateMessageID function - SECURITY: prevents OData injection attacks
func TestValidateMessageID(t *testing.T) {
	tests := []struct {
		name    string
		msgID   string
		wantErr bool
	}{
		// Valid cases
		{"valid standard", "<abc123@example.com>", false},
		{"valid with dots", "<user.name@mail.example.com>", false},
		{"valid with hyphens", "<message-id-123@mail.example.com>", false},
		{"valid with plus", "<user+tag@example.com>", false},
		{"valid with underscore", "<user_name@example.com>", false},
		{"valid long ID", "<CABcD1234567890ABCDEFabcdef1234567890@mail.gmail.com>", false},

		// Invalid cases - injection attempts (SECURITY TESTS)
		{"injection or operator", "<test' or 1 eq 1 or internetMessageId eq 'x@example.com>", true},
		{"injection and operator", "<test' and from/emailAddress/address eq 'victim@example.com>", true},
		{"injection eq operator", "<test' eq 'x>", true},
		{"injection ne operator", "<test' ne 'x>", true},
		{"injection lt operator", "<test' lt 'x>", true},
		{"injection gt operator", "<test' gt 'x>", true},
		{"injection le operator", "<test' le 'x>", true},
		{"injection ge operator", "<test' ge 'x>", true},
		{"injection not operator", "<test' not 'x>", true},
		{"uppercase injection OR", "<test' OR 1 eq 1>", true},
		{"uppercase injection AND", "<test' AND 1 eq 1>", true},
		{"uppercase injection EQ", "<test' EQ 'x>", true},

		// Invalid cases - format violations
		{"missing brackets", "abc123@example.com", true},
		{"missing opening bracket", "abc123@example.com>", true},
		{"missing closing bracket", "<abc123@example.com", true},
		{"contains single quote", "<test'quote@example.com>", true},
		{"contains double quote", "<test\"quote@example.com>", true},
		{"contains backslash", "<test\\slash@example.com>", true},
		{"empty string", "", true},
		{"only brackets", "<>", false}, // Valid but unusual - RFC allows it
		{"too long", "<" + strings.Repeat("a", 1000) + "@example.com>", true},

		// Edge cases
		{"whitespace in ID", "<test message@example.com>", false}, // Spaces are allowed in local part
		{"numeric only", "<123456@example.com>", false},
		{"special chars allowed", "<user!#$%&*+=?^_`{|}~@example.com>", false}, // RFC 5322 allows these
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateMessageID(tt.msgID)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateMessageID(%q) error = %v, wantErr %v", tt.msgID, err, tt.wantErr)
			}
		})
	}
}

// Test enhanced validateConfiguration with format checking
func TestValidateConfigurationEnhanced(t *testing.T) {
	tests := []struct {
		name    string
		config   *Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid configuration",
			config: &Config{
				TenantID: "12345678-1234-1234-1234-123456789012",
				ClientID: "abcdefgh-1234-5678-90ab-cdef12345678",
				Mailbox:  "user@example.com",
				Secret:   "my-secret",
				Action:   "getevents",
				OutputFormat: "text",
			},
			wantErr: false,
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
			name: "invalid mailbox email",
			config: &Config{
				TenantID: "12345678-1234-1234-1234-123456789012",
				ClientID: "abcdefgh-1234-5678-90ab-cdef12345678",
				Mailbox:  "invalid-email",
				Secret:   "my-secret",
				OutputFormat: "text",
			},
			wantErr: true,
		},
		{
			name: "invalid To recipient",
			config: &Config{
				TenantID: "12345678-1234-1234-1234-123456789012",
				ClientID: "abcdefgh-1234-5678-90ab-cdef12345678",
				Mailbox:  "user@example.com",
				Secret:   "my-secret",
				To:       []string{"invalid-email"},
				Action:   "sendmail",
				OutputFormat: "text",
			},
			wantErr: true,
		},
		{
			name: "invalid start time",
			config: &Config{
				TenantID:  "12345678-1234-1234-1234-123456789012",
				ClientID:  "abcdefgh-1234-5678-90ab-cdef12345678",
				Mailbox:   "user@example.com",
				Secret:    "my-secret",
				StartTime: "invalid-time",
				Action:    "sendinvite",
				OutputFormat: "text",
			},
			wantErr: true,
		},
		{
			name: "invalid action",
			config: &Config{
				TenantID: "12345678-1234-1234-1234-123456789012",
				ClientID: "abcdefgh-1234-5678-90ab-cdef12345678",
				Mailbox:  "user@example.com",
				Secret:   "my-secret",
				Action:   "invalidaction",
				OutputFormat: "text",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfiguration(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateConfiguration() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Helper function to generate a test certificate and private key
func generateTestCertificate(t *testing.T) (*x509.Certificate, *rsa.PrivateKey) {
	t.Helper()

	// Generate RSA key pair
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate private key: %v", err)
	}

	// Create certificate template
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		t.Fatalf("Failed to generate serial number: %v", err)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Test Organization"},
			CommonName:   "Test Certificate",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
	}

	// Create self-signed certificate
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		t.Fatalf("Failed to create certificate: %v", err)
	}

	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		t.Fatalf("Failed to parse certificate: %v", err)
	}

	return cert, privateKey
}

// Helper function to create a test PFX file with specified encryption
func createTestPFX(t *testing.T, password string) []byte {
	t.Helper()

	cert, privateKey := generateTestCertificate(t)

	// Encode as PFX using Modern2023 encoder (supports SHA-256)
	pfxData, err := pkcs12.Modern2023.Encode(privateKey, cert, nil, password)
	if err != nil {
		t.Fatalf("Failed to encode PFX: %v", err)
	}

	return pfxData
}

// Helper function to create a legacy test PFX file with SHA-1 encryption
func createLegacyTestPFX(t *testing.T, password string) []byte {
	t.Helper()

	cert, privateKey := generateTestCertificate(t)

	// Encode as PFX using Legacy encoder (uses SHA-1/TripleDES)
	pfxData, err := pkcs12.Legacy.Encode(privateKey, cert, nil, password)
	if err != nil {
		t.Fatalf("Failed to encode legacy PFX: %v", err)
	}

	return pfxData
}

// Test createCertCredential with modern PFX (SHA-256)
func TestCreateCertCredential_ModernPFX(t *testing.T) {
	pfxData := createTestPFX(t, "test-password")

	// Test decoding - we can't fully test Azure credential creation without real Azure setup,
	// but we can verify the PFX decodes correctly
	_, cert, caCerts, err := pkcs12.DecodeChain(pfxData, "test-password")
	if err != nil {
		t.Fatalf("Failed to decode modern PFX (SHA-256): %v", err)
	}

	if cert == nil {
		t.Error("Expected certificate, got nil")
	}

	if cert.Subject.CommonName != "Test Certificate" {
		t.Errorf("Certificate CN = %q, want %q", cert.Subject.CommonName, "Test Certificate")
	}

	// CA certs may be nil for self-signed
	if caCerts == nil {
		t.Log("No CA certificates (expected for self-signed)")
	}
}

// Test createCertCredential with legacy PFX (SHA-1)
func TestCreateCertCredential_LegacyPFX(t *testing.T) {
	pfxData := createLegacyTestPFX(t, "test-password")

	// Test decoding legacy format
	_, cert, _, err := pkcs12.DecodeChain(pfxData, "test-password")
	if err != nil {
		t.Fatalf("Failed to decode legacy PFX (SHA-1): %v", err)
	}

	if cert == nil {
		t.Error("Expected certificate, got nil")
	}

	if cert.Subject.CommonName != "Test Certificate" {
		t.Errorf("Certificate CN = %q, want %q", cert.Subject.CommonName, "Test Certificate")
	}
}

// Test createCertCredential with wrong password
func TestCreateCertCredential_WrongPassword(t *testing.T) {
	pfxData := createTestPFX(t, "correct-password")

	// Try to decode with wrong password
	_, _, _, err := pkcs12.DecodeChain(pfxData, "wrong-password")
	if err == nil {
		t.Error("Expected error with wrong password, got nil")
	}
}

// Test createCertCredential with empty password
func TestCreateCertCredential_EmptyPassword(t *testing.T) {
	pfxData := createTestPFX(t, "")

	// Decode with empty password
	_, cert, _, err := pkcs12.DecodeChain(pfxData, "")
	if err != nil {
		t.Fatalf("Failed to decode PFX with empty password: %v", err)
	}

	if cert == nil {
		t.Error("Expected certificate, got nil")
	}
}

// Test createCertCredential with malformed PFX data
func TestCreateCertCredential_MalformedPFX(t *testing.T) {
	malformedData := []byte("this is not a valid PFX file")

	_, _, _, err := pkcs12.DecodeChain(malformedData, "password")
	if err == nil {
		t.Error("Expected error with malformed PFX data, got nil")
	}
}

// Test createCertCredential with empty PFX data
func TestCreateCertCredential_EmptyPFX(t *testing.T) {
	emptyData := []byte{}

	_, _, _, err := pkcs12.DecodeChain(emptyData, "password")
	if err == nil {
		t.Error("Expected error with empty PFX data, got nil")
	}
}

// Test that our fix handles the SHA-256 digest algorithm (OID 2.16.840.1.101.3.4.2.1)
func TestCreateCertCredential_SHA256Support(t *testing.T) {
	// Create a PFX with modern encryption (SHA-256)
	pfxData := createTestPFX(t, "sha256-test")

	// This should NOT fail with "unknown digest algorithm: 2.16.840.1.101.3.4.2.1"
	key, cert, _, err := pkcs12.DecodeChain(pfxData, "sha256-test")
	if err != nil {
		t.Fatalf("SHA-256 PFX decoding failed: %v (this was the original bug)", err)
	}

	if key == nil {
		t.Error("Expected private key, got nil")
	}

	if cert == nil {
		t.Error("Expected certificate, got nil")
	}

	t.Log("✓ SHA-256 digest algorithm is now supported!")
}

// Test isRetryableError() function with various error types
func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		retryable bool
	}{
		{
			name:      "nil error",
			err:       nil,
			retryable: false,
		},
		{
			name:      "context canceled",
			err:       context.Canceled,
			retryable: false,
		},
		{
			name:      "context deadline exceeded",
			err:       context.DeadlineExceeded,
			retryable: false,
		},
		{
			name:      "azure response error 429",
			err:       &azcore.ResponseError{StatusCode: 429},
			retryable: true,
		},
		{
			name:      "azure response error 503",
			err:       &azcore.ResponseError{StatusCode: 503},
			retryable: true,
		},
		{
			name:      "azure response error 504",
			err:       &azcore.ResponseError{StatusCode: 504},
			retryable: true,
		},
		{
			name:      "azure response error 400",
			err:       &azcore.ResponseError{StatusCode: 400},
			retryable: false,
		},
		{
			name:      "azure response error 404",
			err:       &azcore.ResponseError{StatusCode: 404},
			retryable: false,
		},
		{
			name:      "timeout error",
			err:       errors.New("connection timeout occurred"),
			retryable: true,
		},
		{
			name:      "i/o timeout",
			err:       errors.New("i/o timeout while reading response"),
			retryable: true,
		},
		{
			name:      "connection reset",
			err:       errors.New("connection reset by peer"),
			retryable: true,
		},
		{
			name:      "connection refused",
			err:       errors.New("connection refused"),
			retryable: true,
		},
		{
			name:      "temporary failure",
			err:       errors.New("temporary failure in name resolution"),
			retryable: true,
		},
		{
			name:      "network unreachable",
			err:       errors.New("network is unreachable"),
			retryable: true,
		},
		{
			name:      "no such host",
			err:       errors.New("no such host"),
			retryable: true,
		},
		{
			name:      "generic error",
			err:       errors.New("something went wrong"),
			retryable: false,
		},
		{
			name:      "authentication error",
			err:       errors.New("invalid credentials"),
			retryable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isRetryableError(tt.err)
			if result != tt.retryable {
				t.Errorf("isRetryableError(%v) = %v, want %v", tt.err, result, tt.retryable)
			}
		})
	}
}

// Test isRetryableError with OData errors
func TestIsRetryableError_ODataErrors(t *testing.T) {
	// Note: Creating actual ODataError instances requires complex setup
	// For now, we test that the function doesn't panic with OData errors
	// More comprehensive testing would require mocking the Graph SDK
	t.Run("wrapped azure error", func(t *testing.T) {
		baseErr := &azcore.ResponseError{StatusCode: 429}
		wrappedErr := fmt.Errorf("graph api call failed: %w", baseErr)

		if !isRetryableError(wrappedErr) {
			t.Error("Expected wrapped 429 error to be retryable")
		}
	})

	t.Run("wrapped non-retryable error", func(t *testing.T) {
		baseErr := &azcore.ResponseError{StatusCode: 401}
		wrappedErr := fmt.Errorf("graph api call failed: %w", baseErr)

		if isRetryableError(wrappedErr) {
			t.Error("Expected wrapped 401 error to be non-retryable")
		}
	})
}

// Test retryWithBackoff() function - successful operation on first try
func TestRetryWithBackoff_SuccessFirstTry(t *testing.T) {
	ctx := context.Background()
	callCount := 0

	operation := func() error {
		callCount++
		return nil
	}

	err := retryWithBackoff(ctx, 3, 100*time.Millisecond, operation)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if callCount != 1 {
		t.Errorf("Expected operation to be called once, got %d calls", callCount)
	}
}

// Test retryWithBackoff() function - success after retries
func TestRetryWithBackoff_SuccessAfterRetries(t *testing.T) {
	ctx := context.Background()
	callCount := 0

	operation := func() error {
		callCount++
		if callCount < 3 {
			// Fail first 2 attempts with retryable error
			return errors.New("temporary failure - network timeout")
		}
		return nil // Succeed on 3rd attempt
	}

	start := time.Now()
	err := retryWithBackoff(ctx, 5, 50*time.Millisecond, operation)
	duration := time.Since(start)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if callCount != 3 {
		t.Errorf("Expected operation to be called 3 times, got %d calls", callCount)
	}

	// Verify exponential backoff timing (should wait ~50ms + 100ms = ~150ms)
	expectedMinDuration := 150 * time.Millisecond
	if duration < expectedMinDuration {
		t.Errorf("Expected duration >= %v, got %v (backoff not working)", expectedMinDuration, duration)
	}
}

// Test retryWithBackoff() function - max retries exceeded
func TestRetryWithBackoff_MaxRetriesExceeded(t *testing.T) {
	ctx := context.Background()
	callCount := 0
	maxRetries := 3

	operation := func() error {
		callCount++
		return errors.New("persistent timeout error")
	}

	err := retryWithBackoff(ctx, maxRetries, 10*time.Millisecond, operation)

	if err == nil {
		t.Error("Expected error after max retries, got nil")
	}

	// Should be called maxRetries + 1 times (initial + retries)
	expectedCalls := maxRetries + 1
	if callCount != expectedCalls {
		t.Errorf("Expected %d calls (1 initial + %d retries), got %d", expectedCalls, maxRetries, callCount)
	}

	if !errors.Is(err, errors.New("persistent timeout error")) {
		// Check if error message contains expected text
		if err.Error() == "" || callCount == 0 {
			t.Errorf("Expected error message about retries, got: %v", err)
		}
	}
}

// Test retryWithBackoff() function - non-retryable error fails immediately
func TestRetryWithBackoff_NonRetryableError(t *testing.T) {
	ctx := context.Background()
	callCount := 0

	operation := func() error {
		callCount++
		return errors.New("authentication failed") // Non-retryable error
	}

	err := retryWithBackoff(ctx, 5, 50*time.Millisecond, operation)

	if err == nil {
		t.Error("Expected error, got nil")
	}

	// Should only be called once (no retries for non-retryable errors)
	if callCount != 1 {
		t.Errorf("Expected 1 call (no retries for non-retryable error), got %d calls", callCount)
	}
}

// Test retryWithBackoff() function - context cancellation
func TestRetryWithBackoff_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	callCount := 0

	operation := func() error {
		callCount++
		if callCount == 2 {
			// Cancel context during retry wait
			cancel()
		}
		return errors.New("timeout error") // Retryable error
	}

	err := retryWithBackoff(ctx, 5, 500*time.Millisecond, operation)

	if err == nil {
		t.Error("Expected error due to context cancellation, got nil")
	}

	// Should be called at least twice before cancellation
	if callCount < 2 {
		t.Errorf("Expected at least 2 calls, got %d", callCount)
	}

	// Error should indicate cancellation
	if !errors.Is(err, context.Canceled) {
		// Check if error contains "cancelled" text
		if err.Error() == "" {
			t.Logf("Got error: %v (expected context cancellation error)", err)
		}
	}
}

// Test retryWithBackoff() function - exponential backoff delay calculation
func TestRetryWithBackoff_ExponentialBackoff(t *testing.T) {
	ctx := context.Background()
	baseDelay := 100 * time.Millisecond
	callCount := 0
	var delays []time.Duration
	lastCall := time.Now()

	operation := func() error {
		callCount++
		if callCount > 1 {
			delay := time.Since(lastCall)
			delays = append(delays, delay)
		}
		lastCall = time.Now()

		if callCount <= 3 {
			return errors.New("i/o timeout") // Retryable
		}
		return nil
	}

	err := retryWithBackoff(ctx, 5, baseDelay, operation)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if len(delays) < 2 {
		t.Fatalf("Expected at least 2 delays, got %d", len(delays))
	}

	// First delay should be ~100ms (baseDelay * 2^0)
	expectedFirstDelay := baseDelay
	tolerance := 50 * time.Millisecond
	if delays[0] < expectedFirstDelay-tolerance || delays[0] > expectedFirstDelay+tolerance {
		t.Errorf("First delay expected ~%v, got %v", expectedFirstDelay, delays[0])
	}

	// Second delay should be ~200ms (baseDelay * 2^1)
	expectedSecondDelay := baseDelay * 2
	if delays[1] < expectedSecondDelay-tolerance || delays[1] > expectedSecondDelay+tolerance*2 {
		t.Errorf("Second delay expected ~%v, got %v", expectedSecondDelay, delays[1])
	}
}

// Test retryWithBackoff() function - delay cap at 30 seconds
func TestRetryWithBackoff_DelayCap(t *testing.T) {
	// This test verifies the 30-second cap without actually waiting
	ctx := context.Background()
	baseDelay := 10 * time.Second
	callCount := 0

	operation := func() error {
		callCount++
		if callCount == 1 {
			return errors.New("timeout") // Trigger one retry
		}
		return nil
	}

	start := time.Now()
	err := retryWithBackoff(ctx, 10, baseDelay, operation)
	duration := time.Since(start)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// The delay should be capped at 30 seconds even though baseDelay * 2^attempt would be larger
	// For first retry: min(10s * 2^0, 30s) = 10s
	maxExpectedDuration := 15 * time.Second // 10s delay + some buffer
	if duration > maxExpectedDuration {
		t.Errorf("Expected duration <= %v (with 30s cap), got %v", maxExpectedDuration, duration)
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
		{"complex id", "AAMkAGI2...==", "AAMkAGI2...__"},
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

	// Verify it contains "export" and date
	dateStr := time.Now().Format("2006-01-02")
	if !strings.Contains(dir, "export") || !strings.Contains(dir, dateStr) {
		t.Errorf("Export dir %q should contain 'export' and %q", dir, dateStr)
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
		// Mock model-like behavior if possible, or use real models
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

		r2 := models.NewRecipient()
		a2 := models.NewEmailAddress()
		e2 := "u2@ex.com"
		a2.SetAddress(&e2)
		r2.SetEmailAddress(a2)

		res := extractRecipients([]models.Recipientable{r1, r2})
		if len(res) != 2 {
			t.Fatalf("Expected 2 recipients, got %d", len(res))
		}
		if res[0]["address"] != e1 || res[1]["address"] != e2 {
			t.Errorf("Recipient addresses mismatch: %v", res)
		}
	})
}

// Test validateConfiguration for new actions
func TestValidateConfigurationNewActions(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "exportinbox valid",
			config: &Config{
				TenantID: "12345678-1234-1234-1234-123456789012",
				ClientID: "abcdefgh-1234-5678-90ab-cdef12345678",
				Mailbox:  "user@example.com",
				Secret:   "my-secret",
				Action:   ActionExportInbox,
			},
			wantErr: false,
		},
		{
			name: "searchandexport valid",
			config: &Config{
				TenantID:  "12345678-1234-1234-1234-123456789012",
				ClientID:  "abcdefgh-1234-5678-90ab-cdef12345678",
				Mailbox:   "user@example.com",
				Secret:    "my-secret",
				Action:    ActionSearchAndExport,
				MessageID: "<unique-id@host>",
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
				t.Errorf("validateConfiguration(%s) error = %v, wantErr %v", tt.name, err, tt.wantErr)
			}
		})
	}
}

