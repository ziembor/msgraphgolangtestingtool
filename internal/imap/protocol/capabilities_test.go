package protocol

import (
	"testing"
)

func TestNewCapabilities(t *testing.T) {
	caps := []string{"IMAP4rev1", "STARTTLS", "AUTH=PLAIN", "AUTH=LOGIN", "AUTH=XOAUTH2", "IDLE", "NAMESPACE"}

	c := NewCapabilities(caps)

	if c == nil {
		t.Fatal("NewCapabilities returned nil")
	}

	// Check that capabilities are preserved
	all := c.All()
	if len(all) != len(caps) {
		t.Errorf("All() returned %d items, want %d", len(all), len(caps))
	}
}

func TestCapabilities_Has(t *testing.T) {
	caps := NewCapabilities([]string{"IMAP4rev1", "STARTTLS", "IDLE"})

	tests := []struct {
		cap      string
		expected bool
	}{
		{"IMAP4rev1", true},
		{"imap4rev1", true}, // case insensitive
		{"STARTTLS", true},
		{"IDLE", true},
		{"AUTH=PLAIN", false},
		{"NAMESPACE", false},
	}

	for _, tt := range tests {
		result := caps.Has(tt.cap)
		if result != tt.expected {
			t.Errorf("Has(%q) = %v, want %v", tt.cap, result, tt.expected)
		}
	}
}

func TestCapabilities_All(t *testing.T) {
	caps := NewCapabilities([]string{"IMAP4rev1", "STARTTLS", "IDLE"})

	all := caps.All()
	if len(all) != 3 {
		t.Errorf("All() returned %d items, want 3", len(all))
	}
}

func TestCapabilities_String(t *testing.T) {
	caps := NewCapabilities([]string{"IMAP4rev1", "STARTTLS"})

	str := caps.String()
	if str == "" {
		t.Error("String() returned empty string")
	}
}

func TestCapabilities_SupportsSTARTTLS(t *testing.T) {
	tests := []struct {
		name     string
		caps     []string
		expected bool
	}{
		{"has STARTTLS", []string{"IMAP4rev1", "STARTTLS"}, true},
		{"no STARTTLS", []string{"IMAP4rev1", "IDLE"}, false},
		{"empty caps", []string{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			caps := NewCapabilities(tt.caps)
			if caps.SupportsSTARTTLS() != tt.expected {
				t.Errorf("SupportsSTARTTLS() = %v, want %v", caps.SupportsSTARTTLS(), tt.expected)
			}
		})
	}
}

func TestCapabilities_SupportsPlain(t *testing.T) {
	tests := []struct {
		name     string
		caps     []string
		expected bool
	}{
		{"has AUTH=PLAIN", []string{"IMAP4rev1", "AUTH=PLAIN"}, true},
		{"no AUTH=PLAIN", []string{"IMAP4rev1", "AUTH=LOGIN"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			caps := NewCapabilities(tt.caps)
			if caps.SupportsPlain() != tt.expected {
				t.Errorf("SupportsPlain() = %v, want %v", caps.SupportsPlain(), tt.expected)
			}
		})
	}
}

func TestCapabilities_SupportsLogin(t *testing.T) {
	tests := []struct {
		name     string
		caps     []string
		expected bool
	}{
		{"LOGIN enabled by default", []string{"IMAP4rev1"}, true},
		{"LOGIN disabled", []string{"IMAP4rev1", "LOGINDISABLED"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			caps := NewCapabilities(tt.caps)
			if caps.SupportsLogin() != tt.expected {
				t.Errorf("SupportsLogin() = %v, want %v", caps.SupportsLogin(), tt.expected)
			}
		})
	}
}

func TestCapabilities_SupportsXOAUTH2(t *testing.T) {
	tests := []struct {
		name     string
		caps     []string
		expected bool
	}{
		{"has AUTH=XOAUTH2", []string{"IMAP4rev1", "AUTH=XOAUTH2"}, true},
		{"has auth=xoauth2 lowercase", []string{"IMAP4rev1", "auth=xoauth2"}, true},
		{"no XOAUTH2", []string{"IMAP4rev1", "AUTH=PLAIN"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			caps := NewCapabilities(tt.caps)
			if caps.SupportsXOAUTH2() != tt.expected {
				t.Errorf("SupportsXOAUTH2() = %v, want %v", caps.SupportsXOAUTH2(), tt.expected)
			}
		})
	}
}

func TestCapabilities_SupportsIDLE(t *testing.T) {
	tests := []struct {
		name     string
		caps     []string
		expected bool
	}{
		{"has IDLE", []string{"IMAP4rev1", "IDLE"}, true},
		{"no IDLE", []string{"IMAP4rev1"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			caps := NewCapabilities(tt.caps)
			if caps.SupportsIDLE() != tt.expected {
				t.Errorf("SupportsIDLE() = %v, want %v", caps.SupportsIDLE(), tt.expected)
			}
		})
	}
}

func TestCapabilities_SupportsNAMESPACE(t *testing.T) {
	tests := []struct {
		name     string
		caps     []string
		expected bool
	}{
		{"has NAMESPACE", []string{"IMAP4rev1", "NAMESPACE"}, true},
		{"no NAMESPACE", []string{"IMAP4rev1"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			caps := NewCapabilities(tt.caps)
			if caps.SupportsNAMESPACE() != tt.expected {
				t.Errorf("SupportsNAMESPACE() = %v, want %v", caps.SupportsNAMESPACE(), tt.expected)
			}
		})
	}
}

func TestCapabilities_SupportsCONDSTORE(t *testing.T) {
	tests := []struct {
		name     string
		caps     []string
		expected bool
	}{
		{"has CONDSTORE", []string{"IMAP4rev1", "CONDSTORE"}, true},
		{"no CONDSTORE", []string{"IMAP4rev1"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			caps := NewCapabilities(tt.caps)
			if caps.SupportsCONDSTORE() != tt.expected {
				t.Errorf("SupportsCONDSTORE() = %v, want %v", caps.SupportsCONDSTORE(), tt.expected)
			}
		})
	}
}

func TestCapabilities_GetAuthMechanisms(t *testing.T) {
	tests := []struct {
		name     string
		caps     []string
		expected []string
	}{
		{
			name:     "multiple auth mechanisms",
			caps:     []string{"IMAP4rev1", "AUTH=PLAIN", "AUTH=LOGIN", "AUTH=XOAUTH2"},
			expected: []string{"PLAIN", "LOGIN", "XOAUTH2"},
		},
		{
			name:     "single auth mechanism",
			caps:     []string{"IMAP4rev1", "AUTH=PLAIN"},
			expected: []string{"PLAIN"},
		},
		{
			name:     "no auth mechanisms",
			caps:     []string{"IMAP4rev1", "IDLE"},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			caps := NewCapabilities(tt.caps)
			mechanisms := caps.GetAuthMechanisms()

			if len(mechanisms) != len(tt.expected) {
				t.Errorf("GetAuthMechanisms() returned %d mechanisms, want %d", len(mechanisms), len(tt.expected))
				return
			}

			// Convert to map for order-independent comparison
			expectedMap := make(map[string]bool)
			for _, m := range tt.expected {
				expectedMap[m] = true
			}

			for _, m := range mechanisms {
				if !expectedMap[m] {
					t.Errorf("GetAuthMechanisms() returned unexpected mechanism: %q", m)
				}
			}
		})
	}
}

func TestCapabilities_Empty(t *testing.T) {
	caps := NewCapabilities([]string{})

	if caps.Has("IMAP4rev1") {
		t.Error("Empty capabilities should not have any capability")
	}

	if len(caps.All()) != 0 {
		t.Errorf("All() should return empty slice for empty capabilities")
	}

	if len(caps.GetAuthMechanisms()) != 0 {
		t.Error("GetAuthMechanisms() should return empty slice for empty capabilities")
	}
}

func TestCapabilities_CaseInsensitive(t *testing.T) {
	caps := NewCapabilities([]string{"imap4rev1", "starttls", "auth=plain"})

	// All lookups should be case-insensitive
	if !caps.Has("IMAP4REV1") {
		t.Error("Has() should be case-insensitive")
	}
	if !caps.Has("Imap4rev1") {
		t.Error("Has() should be case-insensitive")
	}
	if !caps.SupportsSTARTTLS() {
		t.Error("SupportsSTARTTLS() should find lowercase starttls")
	}
	if !caps.SupportsPlain() {
		t.Error("SupportsPlain() should find lowercase auth=plain")
	}
}

func TestCapabilities_SelectBestAuthMechanism(t *testing.T) {
	tests := []struct {
		name           string
		caps           []string
		hasAccessToken bool
		expected       string
	}{
		{
			name:           "prefer XOAUTH2 with token",
			caps:           []string{"AUTH=PLAIN", "AUTH=XOAUTH2"},
			hasAccessToken: true,
			expected:       "XOAUTH2",
		},
		{
			name:           "PLAIN without token",
			caps:           []string{"AUTH=PLAIN", "AUTH=XOAUTH2"},
			hasAccessToken: false,
			expected:       "PLAIN",
		},
		{
			name:           "LOGIN fallback",
			caps:           []string{"AUTH=LOGIN"},
			hasAccessToken: false,
			expected:       "LOGIN",
		},
		{
			name:           "LOGIN when not disabled",
			caps:           []string{"IMAP4rev1"},
			hasAccessToken: false,
			expected:       "LOGIN",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			caps := NewCapabilities(tt.caps)
			result := caps.SelectBestAuthMechanism(tt.hasAccessToken)
			if result != tt.expected {
				t.Errorf("SelectBestAuthMechanism(%v) = %q, want %q", tt.hasAccessToken, result, tt.expected)
			}
		})
	}
}
