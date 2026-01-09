package protocol

import (
	"fmt"
	"strings"
)

// SMTP command builders following RFC 5321
// All commands include proper CRLF line endings as required by the SMTP protocol.
//
// Defense-in-Depth: All command builders sanitize input parameters to remove
// CRLF sequences that could be used for command injection attacks. While this
// tool accepts input from trusted sources (CLI flags), this sanitization provides
// an additional layer of protection.

// sanitizeCRLF removes carriage return and line feed characters from input strings
// to prevent SMTP command injection attacks. This is a defense-in-depth measure.
func sanitizeCRLF(input string) string {
	input = strings.ReplaceAll(input, "\r", "")
	input = strings.ReplaceAll(input, "\n", "")
	return input
}

// EHLO sends an Extended SMTP greeting with the specified hostname.
// This command initiates an SMTP session and requests the server's capabilities.
// Example: EHLO smtptool.local
func EHLO(hostname string) string {
	return fmt.Sprintf("EHLO %s\r\n", sanitizeCRLF(hostname))
}

// HELO sends a standard SMTP greeting (legacy, use EHLO if possible).
// Example: HELO smtptool.local
func HELO(hostname string) string {
	return fmt.Sprintf("HELO %s\r\n", sanitizeCRLF(hostname))
}

// STARTTLS sends the STARTTLS command to upgrade the connection to TLS.
// After receiving a 220 response, the client should initiate TLS handshake.
func STARTTLS() string {
	return "STARTTLS\r\n"
}

// AUTH sends an authentication command with the specified mechanism.
// If initialResponse is provided, it's included in the command (e.g., for PLAIN).
// Example: AUTH PLAIN AGpvaG5AZXhhbXBsZS5jb20AcGFzc3dvcmQ=
func AUTH(mechanism string, initialResponse string) string {
	if initialResponse != "" {
		return fmt.Sprintf("AUTH %s %s\r\n", sanitizeCRLF(mechanism), sanitizeCRLF(initialResponse))
	}
	return fmt.Sprintf("AUTH %s\r\n", sanitizeCRLF(mechanism))
}

// MAILFROM sends the MAIL FROM command specifying the sender address.
// The address should NOT include angle brackets - they're added automatically.
// Example: MAIL FROM:<sender@example.com>
func MAILFROM(address string) string {
	return fmt.Sprintf("MAIL FROM:<%s>\r\n", sanitizeCRLF(address))
}

// RCPTTO sends the RCPT TO command specifying a recipient address.
// The address should NOT include angle brackets - they're added automatically.
// Example: RCPT TO:<recipient@example.com>
func RCPTTO(address string) string {
	return fmt.Sprintf("RCPT TO:<%s>\r\n", sanitizeCRLF(address))
}

// DATA sends the DATA command to begin message transmission.
// After receiving a 354 response, send the message body followed by <CRLF>.<CRLF>
func DATA() string {
	return "DATA\r\n"
}

// RSET sends the RESET command to abort the current mail transaction.
// This resets the SMTP session state without closing the connection.
func RSET() string {
	return "RSET\r\n"
}

// NOOP sends a no-operation command (used for connection keep-alive).
// The server should respond with 250 OK.
func NOOP() string {
	return "NOOP\r\n"
}

// QUIT sends the QUIT command to terminate the SMTP session.
// The server should respond with 221 and close the connection.
func QUIT() string {
	return "QUIT\r\n"
}

// VRFY sends the VERIFY command to check if a mailbox exists.
// Many servers disable this for security/privacy reasons.
func VRFY(address string) string {
	return fmt.Sprintf("VRFY %s\r\n", sanitizeCRLF(address))
}

// EXPN sends the EXPAND command to expand a mailing list.
// Many servers disable this for security/privacy reasons.
func EXPN(mailingList string) string {
	return fmt.Sprintf("EXPN %s\r\n", sanitizeCRLF(mailingList))
}

// HELP sends the HELP command to request server help information.
func HELP() string {
	return "HELP\r\n"
}
