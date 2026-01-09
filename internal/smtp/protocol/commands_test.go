//go:build !integration
// +build !integration

package protocol

import (
	"strings"
	"testing"
)

// TestSanitizeCRLF tests CRLF injection prevention (CRITICAL SECURITY)
func TestSanitizeCRLF(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// Normal cases
		{"Normal string", "example.com", "example.com"},
		{"Empty string", "", ""},
		{"String with spaces", "smtp server", "smtp server"},
		{"String with special chars", "user@example.com", "user@example.com"},

		// Security: CRLF injection attempts (CRITICAL)
		{"Security: CRLF injection", "host.com\r\nVRFY admin", "host.comVRFY admin"},
		{"Security: Newline only", "hostname\nDATA", "hostnameDATA"},
		{"Security: Carriage return only", "host\rQUIT", "hostQUIT"},
		{"Security: Multiple CRLF", "host\r\n\r\nQUIT", "hostQUIT"},
		{"Security: CRLF at start", "\r\nQUIT", "QUIT"},
		{"Security: CRLF at end", "EHLO host\r\n", "EHLO host"},
		{"Security: Mixed newlines", "host\r\nVRFY\nadmin\rQUIT", "hostVRFYadminQUIT"},
		{"Security: Email injection", "user@example.com\r\nRCPT TO:<attacker@evil.com>", "user@example.comRCPT TO:<attacker@evil.com>"},
		{"Security: Command injection", "host.com\r\nMAIL FROM:<evil@attacker.com>", "host.comMAIL FROM:<evil@attacker.com>"},
		{"Security: Multiple commands", "host\r\nDATA\r\nQUIT\r\n", "hostDATAQUIT"},
		{"Security: Embedded CRLF in middle", "smtp\r\nserver\r\nattack", "smtpserverattack"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeCRLF(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeCRLF(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestEHLO tests the EHLO command builder
func TestEHLO(t *testing.T) {
	tests := []struct {
		name     string
		hostname string
		want     string
	}{
		{"Normal hostname", "smtptool.local", "EHLO smtptool.local\r\n"},
		{"Hostname with domain", "mail.example.com", "EHLO mail.example.com\r\n"},
		{"Security: CRLF injection", "host.com\r\nVRFY admin", "EHLO host.comVRFY admin\r\n"},
		{"Security: Command injection", "host\r\nQUIT", "EHLO hostQUIT\r\n"},
		{"Empty hostname", "", "EHLO \r\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EHLO(tt.hostname)
			if got != tt.want {
				t.Errorf("EHLO(%q) = %q, want %q", tt.hostname, got, tt.want)
			}
			// Verify CRLF ending
			if !strings.HasSuffix(got, "\r\n") {
				t.Errorf("EHLO() does not end with CRLF")
			}
		})
	}
}

// TestHELO tests the HELO command builder
func TestHELO(t *testing.T) {
	tests := []struct {
		name     string
		hostname string
		want     string
	}{
		{"Normal hostname", "smtptool.local", "HELO smtptool.local\r\n"},
		{"Security: CRLF injection", "host.com\r\nDATA", "HELO host.comDATA\r\n"},
		{"Security: Newline injection", "host\nQUIT", "HELO hostQUIT\r\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HELO(tt.hostname)
			if got != tt.want {
				t.Errorf("HELO(%q) = %q, want %q", tt.hostname, got, tt.want)
			}
			if !strings.HasSuffix(got, "\r\n") {
				t.Errorf("HELO() does not end with CRLF")
			}
		})
	}
}

// TestSTARTTLS tests the STARTTLS command (static)
func TestSTARTTLS(t *testing.T) {
	want := "STARTTLS\r\n"
	got := STARTTLS()
	if got != want {
		t.Errorf("STARTTLS() = %q, want %q", got, want)
	}
}

// TestAUTH tests the AUTH command builder
func TestAUTH(t *testing.T) {
	tests := []struct {
		name            string
		mechanism       string
		initialResponse string
		want            string
	}{
		{"AUTH without response", "PLAIN", "", "AUTH PLAIN\r\n"},
		{"AUTH with response", "PLAIN", "AGpvaG5AZXhhbXBsZS5jb20AcGFzc3dvcmQ=", "AUTH PLAIN AGpvaG5AZXhhbXBsZS5jb20AcGFzc3dvcmQ=\r\n"},
		{"AUTH LOGIN", "LOGIN", "", "AUTH LOGIN\r\n"},
		{"Security: CRLF in mechanism", "PLAIN\r\nQUIT", "", "AUTH PLAINQUIT\r\n"},
		{"Security: CRLF in response", "PLAIN", "response\r\nDATA", "AUTH PLAIN responseDATA\r\n"},
		{"Security: CRLF in both", "PLAIN\r\nQUIT", "test\r\nDATA", "AUTH PLAINQUIT testDATA\r\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AUTH(tt.mechanism, tt.initialResponse)
			if got != tt.want {
				t.Errorf("AUTH(%q, %q) = %q, want %q", tt.mechanism, tt.initialResponse, got, tt.want)
			}
			if !strings.HasSuffix(got, "\r\n") {
				t.Errorf("AUTH() does not end with CRLF")
			}
		})
	}
}

// TestMAILFROM tests the MAIL FROM command builder
func TestMAILFROM(t *testing.T) {
	tests := []struct {
		name    string
		address string
		want    string
	}{
		{"Normal address", "sender@example.com", "MAIL FROM:<sender@example.com>\r\n"},
		{"Empty address", "", "MAIL FROM:<>\r\n"},
		{"Security: CRLF injection", "user@evil.com\r\nRCPT TO:<victim@bank.com>", "MAIL FROM:<user@evil.comRCPT TO:<victim@bank.com>>\r\n"},
		{"Security: Command injection", "sender@test.com\r\nDATA", "MAIL FROM:<sender@test.comDATA>\r\n"},
		{"Security: Multiple recipients injection", "user@example.com\r\nRCPT TO:<attacker@evil.com>", "MAIL FROM:<user@example.comRCPT TO:<attacker@evil.com>>\r\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MAILFROM(tt.address)
			if got != tt.want {
				t.Errorf("MAILFROM(%q) = %q, want %q", tt.address, got, tt.want)
			}
			// Verify format
			if !strings.HasPrefix(got, "MAIL FROM:<") {
				t.Errorf("MAILFROM() does not start with 'MAIL FROM:<'")
			}
			if !strings.HasSuffix(got, ">\r\n") {
				t.Errorf("MAILFROM() does not end with '>\\r\\n'")
			}
		})
	}
}

// TestRCPTTO tests the RCPT TO command builder
func TestRCPTTO(t *testing.T) {
	tests := []struct {
		name    string
		address string
		want    string
	}{
		{"Normal address", "recipient@example.com", "RCPT TO:<recipient@example.com>\r\n"},
		{"Empty address", "", "RCPT TO:<>\r\n"},
		{"Security: CRLF injection", "user@example.com\r\nDATA", "RCPT TO:<user@example.comDATA>\r\n"},
		{"Security: Cc injection", "user@example.com\r\nRCPT TO:<leak@evil.com>", "RCPT TO:<user@example.comRCPT TO:<leak@evil.com>>\r\n"},
		{"Security: Command after recipient", "victim@bank.com\r\nQUIT", "RCPT TO:<victim@bank.comQUIT>\r\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RCPTTO(tt.address)
			if got != tt.want {
				t.Errorf("RCPTTO(%q) = %q, want %q", tt.address, got, tt.want)
			}
			if !strings.HasPrefix(got, "RCPT TO:<") {
				t.Errorf("RCPTTO() does not start with 'RCPT TO:<'")
			}
			if !strings.HasSuffix(got, ">\r\n") {
				t.Errorf("RCPTTO() does not end with '>\\r\\n'")
			}
		})
	}
}

// TestDATA tests the DATA command (static)
func TestDATA(t *testing.T) {
	want := "DATA\r\n"
	got := DATA()
	if got != want {
		t.Errorf("DATA() = %q, want %q", got, want)
	}
}

// TestRSET tests the RSET command (static)
func TestRSET(t *testing.T) {
	want := "RSET\r\n"
	got := RSET()
	if got != want {
		t.Errorf("RSET() = %q, want %q", got, want)
	}
}

// TestNOOP tests the NOOP command (static)
func TestNOOP(t *testing.T) {
	want := "NOOP\r\n"
	got := NOOP()
	if got != want {
		t.Errorf("NOOP() = %q, want %q", got, want)
	}
}

// TestQUIT tests the QUIT command (static)
func TestQUIT(t *testing.T) {
	want := "QUIT\r\n"
	got := QUIT()
	if got != want {
		t.Errorf("QUIT() = %q, want %q", got, want)
	}
}

// TestVRFY tests the VRFY command builder
func TestVRFY(t *testing.T) {
	tests := []struct {
		name    string
		address string
		want    string
	}{
		{"Normal address", "admin@example.com", "VRFY admin@example.com\r\n"},
		{"Security: CRLF injection", "user@example.com\r\nQUIT", "VRFY user@example.comQUIT\r\n"},
		{"Security: Command injection", "admin\r\nMAIL FROM:<evil@attacker.com>", "VRFY adminMAIL FROM:<evil@attacker.com>\r\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := VRFY(tt.address)
			if got != tt.want {
				t.Errorf("VRFY(%q) = %q, want %q", tt.address, got, tt.want)
			}
			if !strings.HasSuffix(got, "\r\n") {
				t.Errorf("VRFY() does not end with CRLF")
			}
		})
	}
}

// TestEXPN tests the EXPN command builder
func TestEXPN(t *testing.T) {
	tests := []struct {
		name        string
		mailingList string
		want        string
	}{
		{"Normal mailing list", "admins", "EXPN admins\r\n"},
		{"Security: CRLF injection", "list\r\nQUIT", "EXPN listQUIT\r\n"},
		{"Security: Command injection", "staff\r\nDATA", "EXPN staffDATA\r\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EXPN(tt.mailingList)
			if got != tt.want {
				t.Errorf("EXPN(%q) = %q, want %q", tt.mailingList, got, tt.want)
			}
			if !strings.HasSuffix(got, "\r\n") {
				t.Errorf("EXPN() does not end with CRLF")
			}
		})
	}
}

// TestHELP tests the HELP command (static)
func TestHELP(t *testing.T) {
	want := "HELP\r\n"
	got := HELP()
	if got != want {
		t.Errorf("HELP() = %q, want %q", got, want)
	}
}

// TestCommandCRLFEndings verifies all commands end with CRLF
func TestCommandCRLFEndings(t *testing.T) {
	commands := []struct {
		name string
		cmd  string
	}{
		{"EHLO", EHLO("test.com")},
		{"HELO", HELO("test.com")},
		{"STARTTLS", STARTTLS()},
		{"AUTH", AUTH("PLAIN", "")},
		{"MAILFROM", MAILFROM("test@example.com")},
		{"RCPTTO", RCPTTO("test@example.com")},
		{"DATA", DATA()},
		{"RSET", RSET()},
		{"NOOP", NOOP()},
		{"QUIT", QUIT()},
		{"VRFY", VRFY("admin")},
		{"EXPN", EXPN("list")},
		{"HELP", HELP()},
	}

	for _, tc := range commands {
		t.Run(tc.name, func(t *testing.T) {
			if !strings.HasSuffix(tc.cmd, "\r\n") {
				t.Errorf("%s command does not end with CRLF: %q", tc.name, tc.cmd)
			}
		})
	}
}
