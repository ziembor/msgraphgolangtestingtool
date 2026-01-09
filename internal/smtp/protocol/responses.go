package protocol

import (
	"bufio"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// DefaultResponseTimeout is the default timeout for reading SMTP responses.
// This prevents indefinite hangs when communicating with misbehaving servers.
const DefaultResponseTimeout = 30 * time.Second

// SMTPResponse represents a parsed SMTP server response.
// SMTP responses consist of a 3-digit code and optional message text.
// Multiline responses are supported (indicated by a hyphen after the code).
type SMTPResponse struct {
	Code    int      // 3-digit response code (e.g., 220, 250, 550)
	Message string   // Full response message (multiline responses joined with \n)
	Lines   []string // Individual lines of the response message
}

// ReadResponse reads and parses an SMTP response from the provided reader.
// Handles both single-line and multiline responses according to RFC 5321.
//
// Single-line format: "250 OK"
// Multiline format:
//
//	250-First line
//	250-Second line
//	250 Last line
//
// The hyphen after the code indicates more lines follow.
// A space after the code indicates the final line.
func ReadResponse(reader *bufio.Reader) (*SMTPResponse, error) {
	var lines []string
	var code int

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("failed to read response: %w", err)
		}

		// Remove CRLF or LF
		line = strings.TrimRight(line, "\r\n")

		// Parse response code (first 3 characters)
		if len(line) < 3 {
			return nil, fmt.Errorf("invalid SMTP response (too short): %s", line)
		}

		lineCode, err := strconv.Atoi(line[0:3])
		if err != nil {
			return nil, fmt.Errorf("invalid response code: %s", line)
		}

		// First line sets the code
		if code == 0 {
			code = lineCode
		} else if code != lineCode {
			return nil, fmt.Errorf("response code mismatch: %d vs %d", code, lineCode)
		}

		// Get message part (skip code and separator)
		message := ""
		if len(line) > 4 {
			message = line[4:]
		}
		lines = append(lines, message)

		// Check if this is the last line (space after code indicates end)
		if len(line) > 3 && line[3] == ' ' {
			break
		}
		// Hyphen after code means more lines follow
		if len(line) > 3 && line[3] != '-' {
			return nil, fmt.Errorf("invalid response format (expected - or space after code): %s", line)
		}
	}

	return &SMTPResponse{
		Code:    code,
		Message: strings.Join(lines, "\n"),
		Lines:   lines,
	}, nil
}

// ReadResponseWithTimeout reads and parses an SMTP response with a timeout.
// This prevents indefinite hangs when communicating with misbehaving SMTP servers.
//
// The timeout parameter specifies the maximum time to wait for a complete response.
// If the timeout is exceeded, an error is returned.
//
// Example usage:
//
//	resp, err := protocol.ReadResponseWithTimeout(reader, 30*time.Second)
//	if err != nil {
//	    // Handle timeout or read error
//	}
func ReadResponseWithTimeout(reader *bufio.Reader, timeout time.Duration) (*SMTPResponse, error) {
	type result struct {
		resp *SMTPResponse
		err  error
	}

	resultCh := make(chan result, 1)

	// Read response in goroutine
	go func() {
		resp, err := ReadResponse(reader)
		resultCh <- result{resp, err}
	}()

	// Wait for response or timeout
	select {
	case r := <-resultCh:
		return r.resp, r.err
	case <-time.After(timeout):
		return nil, fmt.Errorf("timeout waiting for SMTP response after %v", timeout)
	}
}

// IsSuccess checks if the response code indicates success (2xx).
func (r *SMTPResponse) IsSuccess() bool {
	return r.Code >= 200 && r.Code < 300
}

// IsTemporaryError checks if the response code indicates a temporary failure (4xx).
// These errors can be retried.
func (r *SMTPResponse) IsTemporaryError() bool {
	return r.Code >= 400 && r.Code < 500
}

// IsPermanentError checks if the response code indicates a permanent failure (5xx).
// These errors should not be retried.
func (r *SMTPResponse) IsPermanentError() bool {
	return r.Code >= 500 && r.Code < 600
}

// String returns a human-readable representation of the response.
func (r *SMTPResponse) String() string {
	if len(r.Lines) == 1 {
		return fmt.Sprintf("%d %s", r.Code, r.Message)
	}
	return fmt.Sprintf("%d (multiline, %d lines)", r.Code, len(r.Lines))
}

// GetCodeClass returns the response code class (2, 4, or 5).
// Useful for categorizing responses without checking specific codes.
func (r *SMTPResponse) GetCodeClass() int {
	return r.Code / 100
}

// IsAuthRequired checks if the response indicates authentication is required.
// Common codes: 530 (Authentication required)
func (r *SMTPResponse) IsAuthRequired() bool {
	return r.Code == 530
}

// IsMailboxUnavailable checks if the response indicates the mailbox doesn't exist.
// Common codes: 550 (Mailbox unavailable), 551 (User not local)
func (r *SMTPResponse) IsMailboxUnavailable() bool {
	return r.Code == 550 || r.Code == 551
}

// IsRateLimited checks if the response indicates rate limiting.
// Common codes: 421 (Service not available), 450 (Mailbox busy)
func (r *SMTPResponse) IsRateLimited() bool {
	return r.Code == 421 || r.Code == 450 || r.Code == 451
}
