package protocol

import (
	"testing"
)

func TestNewCapabilities(t *testing.T) {
	lines := []string{
		"TOP",
		"USER",
		"SASL PLAIN LOGIN XOAUTH2",
		"UIDL",
		"RESP-CODES",
		"PIPELINING",
		"STLS",
		"EXPIRE 30",
		"IMPLEMENTATION Dovecot",
	}

	caps := NewCapabilities(lines)

	if caps == nil {
		t.Fatal("NewCapabilities returned nil")
	}

	// Test that raw lines are preserved
	if len(caps.Raw()) != len(lines) {
		t.Errorf("Raw() returned %d lines, want %d", len(caps.Raw()), len(lines))
	}
}

func TestCapabilities_Has(t *testing.T) {
	lines := []string{
		"TOP",
		"USER",
		"UIDL",
		"STLS",
	}
	caps := NewCapabilities(lines)

	tests := []struct {
		name     string
		cap      string
		expected bool
	}{
		{"has TOP", "TOP", true},
		{"has top lowercase", "top", true},
		{"has USER", "USER", true},
		{"has UIDL", "UIDL", true},
		{"has STLS", "STLS", true},
		{"missing SASL", "SASL", false},
		{"missing PIPELINING", "PIPELINING", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := caps.Has(tt.cap)
			if result != tt.expected {
				t.Errorf("Has(%q) = %v, want %v", tt.cap, result, tt.expected)
			}
		})
	}
}

func TestCapabilities_Get(t *testing.T) {
	lines := []string{
		"SASL PLAIN LOGIN XOAUTH2",
		"EXPIRE 30",
		"IMPLEMENTATION Dovecot Mail Server",
		"TOP",
	}
	caps := NewCapabilities(lines)

	tests := []struct {
		name     string
		cap      string
		expected []string
	}{
		{"SASL args", "SASL", []string{"PLAIN", "LOGIN", "XOAUTH2"}},
		{"EXPIRE args", "EXPIRE", []string{"30"}},
		{"IMPLEMENTATION args", "IMPLEMENTATION", []string{"Dovecot", "Mail", "Server"}},
		{"TOP no args", "TOP", nil},
		{"missing cap", "MISSING", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := caps.Get(tt.cap)
			if len(result) != len(tt.expected) {
				t.Errorf("Get(%q) = %v, want %v", tt.cap, result, tt.expected)
				return
			}
			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("Get(%q)[%d] = %q, want %q", tt.cap, i, result[i], tt.expected[i])
				}
			}
		})
	}
}

func TestCapabilities_SupportsSTLS(t *testing.T) {
	tests := []struct {
		name     string
		lines    []string
		expected bool
	}{
		{"has STLS", []string{"STLS", "USER"}, true},
		{"no STLS", []string{"USER", "TOP"}, false},
		{"empty caps", []string{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			caps := NewCapabilities(tt.lines)
			if caps.SupportsSTLS() != tt.expected {
				t.Errorf("SupportsSTLS() = %v, want %v", caps.SupportsSTLS(), tt.expected)
			}
		})
	}
}

func TestCapabilities_SupportsAuth(t *testing.T) {
	tests := []struct {
		name     string
		lines    []string
		expected bool
	}{
		{"has SASL", []string{"SASL PLAIN"}, true},
		{"no SASL", []string{"USER", "TOP"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			caps := NewCapabilities(tt.lines)
			if caps.SupportsAuth() != tt.expected {
				t.Errorf("SupportsAuth() = %v, want %v", caps.SupportsAuth(), tt.expected)
			}
		})
	}
}

func TestCapabilities_GetAuthMechanisms(t *testing.T) {
	tests := []struct {
		name     string
		lines    []string
		expected []string
	}{
		{"multiple mechanisms", []string{"SASL PLAIN LOGIN XOAUTH2"}, []string{"PLAIN", "LOGIN", "XOAUTH2"}},
		{"single mechanism", []string{"SASL PLAIN"}, []string{"PLAIN"}},
		{"no SASL", []string{"USER"}, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			caps := NewCapabilities(tt.lines)
			result := caps.GetAuthMechanisms()
			if len(result) != len(tt.expected) {
				t.Errorf("GetAuthMechanisms() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestCapabilities_SupportsXOAUTH2(t *testing.T) {
	tests := []struct {
		name     string
		lines    []string
		expected bool
	}{
		{"has XOAUTH2", []string{"SASL PLAIN XOAUTH2"}, true},
		{"no XOAUTH2", []string{"SASL PLAIN LOGIN"}, false},
		{"no SASL", []string{"USER"}, false},
		{"xoauth2 lowercase", []string{"SASL xoauth2"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			caps := NewCapabilities(tt.lines)
			if caps.SupportsXOAUTH2() != tt.expected {
				t.Errorf("SupportsXOAUTH2() = %v, want %v", caps.SupportsXOAUTH2(), tt.expected)
			}
		})
	}
}

func TestCapabilities_SupportsPlain(t *testing.T) {
	tests := []struct {
		name     string
		lines    []string
		expected bool
	}{
		{"has PLAIN", []string{"SASL PLAIN LOGIN"}, true},
		{"no PLAIN", []string{"SASL LOGIN"}, false},
		{"no SASL", []string{"USER"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			caps := NewCapabilities(tt.lines)
			if caps.SupportsPlain() != tt.expected {
				t.Errorf("SupportsPlain() = %v, want %v", caps.SupportsPlain(), tt.expected)
			}
		})
	}
}

func TestCapabilities_SupportsUIDL(t *testing.T) {
	tests := []struct {
		name     string
		lines    []string
		expected bool
	}{
		{"has UIDL", []string{"UIDL", "USER"}, true},
		{"no UIDL", []string{"USER", "TOP"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			caps := NewCapabilities(tt.lines)
			if caps.SupportsUIDL() != tt.expected {
				t.Errorf("SupportsUIDL() = %v, want %v", caps.SupportsUIDL(), tt.expected)
			}
		})
	}
}

func TestCapabilities_SupportsTOP(t *testing.T) {
	tests := []struct {
		name     string
		lines    []string
		expected bool
	}{
		{"has TOP", []string{"TOP", "USER"}, true},
		{"no TOP", []string{"USER", "UIDL"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			caps := NewCapabilities(tt.lines)
			if caps.SupportsTOP() != tt.expected {
				t.Errorf("SupportsTOP() = %v, want %v", caps.SupportsTOP(), tt.expected)
			}
		})
	}
}

func TestCapabilities_SupportsUSER(t *testing.T) {
	tests := []struct {
		name     string
		lines    []string
		expected bool
	}{
		{"has USER", []string{"USER", "TOP"}, true},
		{"no USER", []string{"TOP", "UIDL"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			caps := NewCapabilities(tt.lines)
			if caps.SupportsUSER() != tt.expected {
				t.Errorf("SupportsUSER() = %v, want %v", caps.SupportsUSER(), tt.expected)
			}
		})
	}
}

func TestCapabilities_SupportsPipelining(t *testing.T) {
	tests := []struct {
		name     string
		lines    []string
		expected bool
	}{
		{"has PIPELINING", []string{"PIPELINING", "USER"}, true},
		{"no PIPELINING", []string{"USER", "TOP"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			caps := NewCapabilities(tt.lines)
			if caps.SupportsPipelining() != tt.expected {
				t.Errorf("SupportsPipelining() = %v, want %v", caps.SupportsPipelining(), tt.expected)
			}
		})
	}
}

func TestCapabilities_SupportsRESPCodes(t *testing.T) {
	tests := []struct {
		name     string
		lines    []string
		expected bool
	}{
		{"has RESP-CODES", []string{"RESP-CODES", "USER"}, true},
		{"no RESP-CODES", []string{"USER", "TOP"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			caps := NewCapabilities(tt.lines)
			if caps.SupportsRESPCodes() != tt.expected {
				t.Errorf("SupportsRESPCodes() = %v, want %v", caps.SupportsRESPCodes(), tt.expected)
			}
		})
	}
}

func TestCapabilities_GetExpirePolicy(t *testing.T) {
	tests := []struct {
		name     string
		lines    []string
		expected string
	}{
		{"expire 30 days", []string{"EXPIRE 30"}, "30"},
		{"expire never", []string{"EXPIRE NEVER"}, "NEVER"},
		{"no expire", []string{"USER"}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			caps := NewCapabilities(tt.lines)
			if caps.GetExpirePolicy() != tt.expected {
				t.Errorf("GetExpirePolicy() = %q, want %q", caps.GetExpirePolicy(), tt.expected)
			}
		})
	}
}

func TestCapabilities_GetImplementation(t *testing.T) {
	tests := []struct {
		name     string
		lines    []string
		expected string
	}{
		{"simple implementation", []string{"IMPLEMENTATION Dovecot"}, "Dovecot"},
		{"multi-word", []string{"IMPLEMENTATION Dovecot Mail Server"}, "Dovecot Mail Server"},
		{"no implementation", []string{"USER"}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			caps := NewCapabilities(tt.lines)
			if caps.GetImplementation() != tt.expected {
				t.Errorf("GetImplementation() = %q, want %q", caps.GetImplementation(), tt.expected)
			}
		})
	}
}

func TestCapabilities_All(t *testing.T) {
	lines := []string{"TOP", "USER", "UIDL"}
	caps := NewCapabilities(lines)

	all := caps.All()
	if len(all) != 3 {
		t.Errorf("All() returned %d items, want 3", len(all))
	}

	// Check that all expected caps are present (order may vary)
	expected := map[string]bool{"TOP": true, "USER": true, "UIDL": true}
	for _, cap := range all {
		if !expected[cap] {
			t.Errorf("All() returned unexpected capability: %q", cap)
		}
	}
}

func TestCapabilities_String(t *testing.T) {
	lines := []string{"TOP", "USER"}
	caps := NewCapabilities(lines)

	str := caps.String()
	if str == "" {
		t.Error("String() returned empty string")
	}
}
