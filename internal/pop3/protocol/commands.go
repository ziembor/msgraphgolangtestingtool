// Package protocol provides POP3 protocol command builders and utilities.
package protocol

import (
	"fmt"
	"strings"
)

// POP3 command constants
const (
	CRLF = "\r\n"
)

// sanitizeCRLF removes CR and LF characters to prevent command injection.
func sanitizeCRLF(input string) string {
	result := strings.ReplaceAll(input, "\r", "")
	result = strings.ReplaceAll(result, "\n", "")
	return result
}

// USER builds a USER command for the first step of USER/PASS authentication.
// Format: USER <username>
func USER(username string) string {
	return fmt.Sprintf("USER %s%s", sanitizeCRLF(username), CRLF)
}

// PASS builds a PASS command for the second step of USER/PASS authentication.
// Format: PASS <password>
func PASS(password string) string {
	return fmt.Sprintf("PASS %s%s", sanitizeCRLF(password), CRLF)
}

// APOP builds an APOP command for challenge-response authentication.
// Format: APOP <username> <digest>
// The digest is MD5(timestamp + password) where timestamp is from the greeting.
func APOP(username, digest string) string {
	return fmt.Sprintf("APOP %s %s%s", sanitizeCRLF(username), sanitizeCRLF(digest), CRLF)
}

// STAT builds a STAT command to get mailbox statistics.
// Format: STAT
// Response: +OK <count> <size>
func STAT() string {
	return "STAT" + CRLF
}

// LIST builds a LIST command to get message sizes.
// Format: LIST [msg]
// If msg is 0, lists all messages.
func LIST(msg int) string {
	if msg > 0 {
		return fmt.Sprintf("LIST %d%s", msg, CRLF)
	}
	return "LIST" + CRLF
}

// UIDL builds a UIDL command to get unique message identifiers.
// Format: UIDL [msg]
// If msg is 0, lists all messages.
func UIDL(msg int) string {
	if msg > 0 {
		return fmt.Sprintf("UIDL %d%s", msg, CRLF)
	}
	return "UIDL" + CRLF
}

// RETR builds a RETR command to retrieve a message.
// Format: RETR <msg>
func RETR(msg int) string {
	return fmt.Sprintf("RETR %d%s", msg, CRLF)
}

// DELE builds a DELE command to mark a message for deletion.
// Format: DELE <msg>
func DELE(msg int) string {
	return fmt.Sprintf("DELE %d%s", msg, CRLF)
}

// TOP builds a TOP command to retrieve message headers and first n lines.
// Format: TOP <msg> <n>
func TOP(msg, lines int) string {
	return fmt.Sprintf("TOP %d %d%s", msg, lines, CRLF)
}

// NOOP builds a NOOP command (no operation).
// Format: NOOP
func NOOP() string {
	return "NOOP" + CRLF
}

// RSET builds a RSET command to reset deletion marks.
// Format: RSET
func RSET() string {
	return "RSET" + CRLF
}

// QUIT builds a QUIT command to end the session.
// Format: QUIT
func QUIT() string {
	return "QUIT" + CRLF
}

// CAPA builds a CAPA command to request server capabilities.
// Format: CAPA
func CAPA() string {
	return "CAPA" + CRLF
}

// STLS builds a STLS command to start TLS negotiation.
// Format: STLS
func STLS() string {
	return "STLS" + CRLF
}

// AUTH builds an AUTH command for SASL authentication.
// Format: AUTH <mechanism> [initial-response]
func AUTH(mechanism string, initialResponse string) string {
	mechanism = sanitizeCRLF(mechanism)
	if initialResponse != "" {
		return fmt.Sprintf("AUTH %s %s%s", mechanism, sanitizeCRLF(initialResponse), CRLF)
	}
	return fmt.Sprintf("AUTH %s%s", mechanism, CRLF)
}

// XOAUTH2Token builds an XOAUTH2 token for SASL authentication.
// Format: user=<email>\x01auth=Bearer <token>\x01\x01 (base64 encoded)
func XOAUTH2Token(username, accessToken string) string {
	// Build the XOAUTH2 string format
	return fmt.Sprintf("user=%s\x01auth=Bearer %s\x01\x01", username, accessToken)
}
