package protocol

import (
	"encoding/base64"
	"testing"
)

func TestUSER(t *testing.T) {
	tests := []struct {
		username string
		expected string
	}{
		{"user@example.com", "USER user@example.com\r\n"},
		{"testuser", "USER testuser\r\n"},
		{"", "USER \r\n"},
	}

	for _, tt := range tests {
		result := USER(tt.username)
		if result != tt.expected {
			t.Errorf("USER(%q) = %q, want %q", tt.username, result, tt.expected)
		}
	}
}

func TestPASS(t *testing.T) {
	tests := []struct {
		password string
		expected string
	}{
		{"secret123", "PASS secret123\r\n"},
		{"", "PASS \r\n"},
	}

	for _, tt := range tests {
		result := PASS(tt.password)
		if result != tt.expected {
			t.Errorf("PASS(%q) = %q, want %q", tt.password, result, tt.expected)
		}
	}
}

func TestSTAT(t *testing.T) {
	expected := "STAT\r\n"
	result := STAT()
	if result != expected {
		t.Errorf("STAT() = %q, want %q", result, expected)
	}
}

func TestLIST(t *testing.T) {
	tests := []struct {
		msgNum   int
		expected string
	}{
		{0, "LIST\r\n"},
		{1, "LIST 1\r\n"},
		{100, "LIST 100\r\n"},
	}

	for _, tt := range tests {
		result := LIST(tt.msgNum)
		if result != tt.expected {
			t.Errorf("LIST(%d) = %q, want %q", tt.msgNum, result, tt.expected)
		}
	}
}

func TestUIDL(t *testing.T) {
	tests := []struct {
		msgNum   int
		expected string
	}{
		{0, "UIDL\r\n"},
		{1, "UIDL 1\r\n"},
		{50, "UIDL 50\r\n"},
	}

	for _, tt := range tests {
		result := UIDL(tt.msgNum)
		if result != tt.expected {
			t.Errorf("UIDL(%d) = %q, want %q", tt.msgNum, result, tt.expected)
		}
	}
}

func TestRETR(t *testing.T) {
	tests := []struct {
		msgNum   int
		expected string
	}{
		{1, "RETR 1\r\n"},
		{100, "RETR 100\r\n"},
	}

	for _, tt := range tests {
		result := RETR(tt.msgNum)
		if result != tt.expected {
			t.Errorf("RETR(%d) = %q, want %q", tt.msgNum, result, tt.expected)
		}
	}
}

func TestDELE(t *testing.T) {
	tests := []struct {
		msgNum   int
		expected string
	}{
		{1, "DELE 1\r\n"},
		{50, "DELE 50\r\n"},
	}

	for _, tt := range tests {
		result := DELE(tt.msgNum)
		if result != tt.expected {
			t.Errorf("DELE(%d) = %q, want %q", tt.msgNum, result, tt.expected)
		}
	}
}

func TestNOOP(t *testing.T) {
	expected := "NOOP\r\n"
	result := NOOP()
	if result != expected {
		t.Errorf("NOOP() = %q, want %q", result, expected)
	}
}

func TestRSET(t *testing.T) {
	expected := "RSET\r\n"
	result := RSET()
	if result != expected {
		t.Errorf("RSET() = %q, want %q", result, expected)
	}
}

func TestQUIT(t *testing.T) {
	expected := "QUIT\r\n"
	result := QUIT()
	if result != expected {
		t.Errorf("QUIT() = %q, want %q", result, expected)
	}
}

func TestCAPA(t *testing.T) {
	expected := "CAPA\r\n"
	result := CAPA()
	if result != expected {
		t.Errorf("CAPA() = %q, want %q", result, expected)
	}
}

func TestSTLS(t *testing.T) {
	expected := "STLS\r\n"
	result := STLS()
	if result != expected {
		t.Errorf("STLS() = %q, want %q", result, expected)
	}
}

func TestTOP(t *testing.T) {
	tests := []struct {
		msgNum   int
		lines    int
		expected string
	}{
		{1, 0, "TOP 1 0\r\n"},
		{1, 10, "TOP 1 10\r\n"},
		{5, 100, "TOP 5 100\r\n"},
	}

	for _, tt := range tests {
		result := TOP(tt.msgNum, tt.lines)
		if result != tt.expected {
			t.Errorf("TOP(%d, %d) = %q, want %q", tt.msgNum, tt.lines, result, tt.expected)
		}
	}
}

func TestAPOP(t *testing.T) {
	result := APOP("user", "digest123")
	expected := "APOP user digest123\r\n"
	if result != expected {
		t.Errorf("APOP() = %q, want %q", result, expected)
	}
}

func TestAUTH(t *testing.T) {
	tests := []struct {
		mechanism       string
		initialResponse string
		expected        string
	}{
		{"PLAIN", "", "AUTH PLAIN\r\n"},
		{"PLAIN", "dGVzdA==", "AUTH PLAIN dGVzdA==\r\n"},
		{"XOAUTH2", "token", "AUTH XOAUTH2 token\r\n"},
	}

	for _, tt := range tests {
		result := AUTH(tt.mechanism, tt.initialResponse)
		if result != tt.expected {
			t.Errorf("AUTH(%q, %q) = %q, want %q", tt.mechanism, tt.initialResponse, result, tt.expected)
		}
	}
}

func TestXOAUTH2Token(t *testing.T) {
	token := XOAUTH2Token("user@example.com", "ya29.token")

	// Should be in format: user=<email>\x01auth=Bearer <token>\x01\x01
	expected := "user=user@example.com\x01auth=Bearer ya29.token\x01\x01"
	if token != expected {
		t.Errorf("XOAUTH2Token() = %q, want %q", token, expected)
	}

	// Verify it can be base64 encoded (as would be done for AUTH command)
	encoded := base64.StdEncoding.EncodeToString([]byte(token))
	if len(encoded) == 0 {
		t.Error("XOAUTH2Token result should be base64 encodable")
	}
}

func TestSanitizeCRLF(t *testing.T) {
	// Test that CRLF injection is prevented
	result := USER("user\r\nQUIT\r\n")
	if result == "USER user\r\nQUIT\r\n\r\n" {
		t.Error("USER should sanitize CRLF to prevent injection")
	}

	// Should strip CR and LF
	expected := "USER userQUIT\r\n"
	if result != expected {
		t.Errorf("USER() with CRLF = %q, want %q", result, expected)
	}
}
