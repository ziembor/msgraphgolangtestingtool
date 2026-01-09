//go:build !integration
// +build !integration

package protocol

import (
	"bufio"
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestReadResponse(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantCode    int
		wantMessage string
		wantLines   int
		wantErr     bool
	}{
		{
			name:        "Single line success",
			input:       "250 OK\r\n",
			wantCode:    250,
			wantMessage: "OK",
			wantLines:   1,
			wantErr:     false,
		},
		{
			name:        "Single line with no message",
			input:       "220 \r\n",
			wantCode:    220,
			wantMessage: "",
			wantLines:   1,
			wantErr:     false,
		},
		{
			name: "Multiline response",
			input: "250-smtp.example.com\r\n" +
				"250-PIPELINING\r\n" +
				"250-SIZE 35882577\r\n" +
				"250 HELP\r\n",
			wantCode:    250,
			wantMessage: "smtp.example.com\nPIPELINING\nSIZE 35882577\nHELP",
			wantLines:   4,
			wantErr:     false,
		},
		{
			name:        "Error response",
			input:       "550 Mailbox unavailable\r\n",
			wantCode:    550,
			wantMessage: "Mailbox unavailable",
			wantLines:   1,
			wantErr:     false,
		},
		{
			name:    "Invalid response - too short",
			input:   "25\r\n",
			wantErr: true,
		},
		{
			name:    "Invalid response - non-numeric code",
			input:   "ABC OK\r\n",
			wantErr: true,
		},
		{
			name:    "Invalid separator",
			input:   "250XOK\r\n",
			wantErr: true,
		},
		{
			name: "Code mismatch in multiline",
			input: "250-First line\r\n" +
				"251 Second line\r\n",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := bufio.NewReader(strings.NewReader(tt.input))
			resp, err := ReadResponse(reader)

			if (err != nil) != tt.wantErr {
				t.Errorf("ReadResponse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			if resp.Code != tt.wantCode {
				t.Errorf("ReadResponse() Code = %d, want %d", resp.Code, tt.wantCode)
			}

			if resp.Message != tt.wantMessage {
				t.Errorf("ReadResponse() Message = %q, want %q", resp.Message, tt.wantMessage)
			}

			if len(resp.Lines) != tt.wantLines {
				t.Errorf("ReadResponse() Lines count = %d, want %d", len(resp.Lines), tt.wantLines)
			}
		})
	}
}

func TestReadResponseWithTimeout_Success(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		timeout     time.Duration
		wantCode    int
		wantMessage string
		wantErr     bool
	}{
		{
			name:        "Valid response within timeout",
			input:       "250 OK\r\n",
			timeout:     1 * time.Second,
			wantCode:    250,
			wantMessage: "OK",
			wantErr:     false,
		},
		{
			name: "Multiline response within timeout",
			input: "250-First\r\n" +
				"250 Last\r\n",
			timeout:     1 * time.Second,
			wantCode:    250,
			wantMessage: "First\nLast",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := bufio.NewReader(strings.NewReader(tt.input))
			resp, err := ReadResponseWithTimeout(reader, tt.timeout)

			if (err != nil) != tt.wantErr {
				t.Errorf("ReadResponseWithTimeout() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			if resp.Code != tt.wantCode {
				t.Errorf("ReadResponseWithTimeout() Code = %d, want %d", resp.Code, tt.wantCode)
			}

			if resp.Message != tt.wantMessage {
				t.Errorf("ReadResponseWithTimeout() Message = %q, want %q", resp.Message, tt.wantMessage)
			}
		})
	}
}

func TestReadResponseWithTimeout_Timeout(t *testing.T) {
	// Create a reader that never returns data (simulates hanging server)
	reader := bufio.NewReader(&neverEndingReader{})

	// Use a very short timeout for testing
	timeout := 50 * time.Millisecond

	start := time.Now()
	resp, err := ReadResponseWithTimeout(reader, timeout)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("ReadResponseWithTimeout() expected timeout error, got nil")
	}

	if resp != nil {
		t.Errorf("ReadResponseWithTimeout() expected nil response on timeout, got %v", resp)
	}

	if !strings.Contains(err.Error(), "timeout") {
		t.Errorf("ReadResponseWithTimeout() error should contain 'timeout', got: %v", err)
	}

	// Verify timeout occurred approximately at the expected time (with some tolerance)
	if elapsed < timeout || elapsed > timeout+100*time.Millisecond {
		t.Errorf("ReadResponseWithTimeout() elapsed time = %v, want approximately %v", elapsed, timeout)
	}
}

func TestReadResponseWithTimeout_ErrorPropagation(t *testing.T) {
	// Create a reader that returns an invalid response
	reader := bufio.NewReader(strings.NewReader("invalid\r\n"))

	resp, err := ReadResponseWithTimeout(reader, 1*time.Second)

	if err == nil {
		t.Fatal("ReadResponseWithTimeout() expected error for invalid response, got nil")
	}

	if resp != nil {
		t.Errorf("ReadResponseWithTimeout() expected nil response on error, got %v", resp)
	}

	// Should not be a timeout error
	if strings.Contains(err.Error(), "timeout") {
		t.Errorf("ReadResponseWithTimeout() should propagate read error, not timeout: %v", err)
	}
}

// TestSMTPResponseMethods tests the helper methods on SMTPResponse
func TestSMTPResponseMethods(t *testing.T) {
	tests := []struct {
		name                  string
		code                  int
		wantIsSuccess         bool
		wantIsTemporaryError  bool
		wantIsPermanentError  bool
		wantCodeClass         int
		wantIsAuthRequired    bool
		wantIsMailboxUnavail  bool
		wantIsRateLimited     bool
	}{
		{"Success 2xx", 250, true, false, false, 2, false, false, false},
		{"Success 220", 220, true, false, false, 2, false, false, false},
		{"Temporary 4xx", 421, false, true, false, 4, false, false, true},
		{"Temporary 450", 450, false, true, false, 4, false, false, true},
		{"Permanent 5xx", 550, false, false, true, 5, false, true, false},
		{"Auth required 530", 530, false, false, true, 5, true, false, false},
		{"Mailbox unavail 550", 550, false, false, true, 5, false, true, false},
		{"Mailbox unavail 551", 551, false, false, true, 5, false, true, false},
		{"Rate limited 451", 451, false, true, false, 4, false, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &SMTPResponse{Code: tt.code, Message: "Test"}

			if got := resp.IsSuccess(); got != tt.wantIsSuccess {
				t.Errorf("IsSuccess() = %v, want %v", got, tt.wantIsSuccess)
			}

			if got := resp.IsTemporaryError(); got != tt.wantIsTemporaryError {
				t.Errorf("IsTemporaryError() = %v, want %v", got, tt.wantIsTemporaryError)
			}

			if got := resp.IsPermanentError(); got != tt.wantIsPermanentError {
				t.Errorf("IsPermanentError() = %v, want %v", got, tt.wantIsPermanentError)
			}

			if got := resp.GetCodeClass(); got != tt.wantCodeClass {
				t.Errorf("GetCodeClass() = %v, want %v", got, tt.wantCodeClass)
			}

			if got := resp.IsAuthRequired(); got != tt.wantIsAuthRequired {
				t.Errorf("IsAuthRequired() = %v, want %v", got, tt.wantIsAuthRequired)
			}

			if got := resp.IsMailboxUnavailable(); got != tt.wantIsMailboxUnavail {
				t.Errorf("IsMailboxUnavailable() = %v, want %v", got, tt.wantIsMailboxUnavail)
			}

			if got := resp.IsRateLimited(); got != tt.wantIsRateLimited {
				t.Errorf("IsRateLimited() = %v, want %v", got, tt.wantIsRateLimited)
			}
		})
	}
}

func TestSMTPResponseString(t *testing.T) {
	tests := []struct {
		name     string
		resp     *SMTPResponse
		wantStr  string
	}{
		{
			name:     "Single line",
			resp:     &SMTPResponse{Code: 250, Message: "OK", Lines: []string{"OK"}},
			wantStr:  "250 OK",
		},
		{
			name:     "Multiline",
			resp:     &SMTPResponse{Code: 250, Message: "First\nSecond\nThird", Lines: []string{"First", "Second", "Third"}},
			wantStr:  "250 (multiline, 3 lines)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.resp.String(); got != tt.wantStr {
				t.Errorf("String() = %q, want %q", got, tt.wantStr)
			}
		})
	}
}

// neverEndingReader is a reader that blocks forever (simulates hanging server)
type neverEndingReader struct{}

func (r *neverEndingReader) Read(p []byte) (n int, err error) {
	// Block forever to simulate a hanging server
	select {}
}

// slowReader simulates a slow server that sends data byte by byte with delays
type slowReader struct {
	data  []byte
	pos   int
	delay time.Duration
}

func (r *slowReader) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.data) {
		return 0, bytes.ErrTooLarge
	}

	// Simulate slow server by adding delay
	time.Sleep(r.delay)

	// Return one byte at a time
	p[0] = r.data[r.pos]
	r.pos++
	return 1, nil
}

func TestReadResponseWithTimeout_SlowResponse(t *testing.T) {
	// Create a reader that sends data very slowly (1 byte every 10ms)
	input := "250 OK\r\n"
	reader := bufio.NewReader(&slowReader{
		data:  []byte(input),
		delay: 10 * time.Millisecond,
	})

	// Timeout should be long enough to read the full response
	timeout := 200 * time.Millisecond

	resp, err := ReadResponseWithTimeout(reader, timeout)

	if err != nil {
		t.Fatalf("ReadResponseWithTimeout() unexpected error: %v", err)
	}

	if resp.Code != 250 {
		t.Errorf("ReadResponseWithTimeout() Code = %d, want 250", resp.Code)
	}

	if resp.Message != "OK" {
		t.Errorf("ReadResponseWithTimeout() Message = %q, want %q", resp.Message, "OK")
	}
}
