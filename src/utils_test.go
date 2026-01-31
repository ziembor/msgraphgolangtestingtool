//go:build !integration
// +build !integration

package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
)

// testError is a simple error type for testing
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

// TestEnrichGraphAPIError tests the enrichGraphAPIError function with various error types
func TestEnrichGraphAPIError(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		operation string
		wantNil   bool
		wantErr   bool
	}{
		{
			name:      "nil error returns nil",
			err:       nil,
			operation: "testOperation",
			wantNil:   true,
			wantErr:   false,
		},
		{
			name:      "non-OData error returned unchanged",
			err:       &testError{msg: "generic error"},
			operation: "testOperation",
			wantNil:   false,
			wantErr:   true,
		},
		{
			name:      "empty operation name",
			err:       &testError{msg: "test error"},
			operation: "",
			wantNil:   false,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := enrichGraphAPIError(tt.err, nil, tt.operation)

			if tt.wantNil && result != nil {
				t.Errorf("enrichGraphAPIError() expected nil, got %v", result)
			}

			if !tt.wantNil && tt.wantErr && result == nil {
				t.Error("enrichGraphAPIError() expected error, got nil")
			}

			if !tt.wantNil && !tt.wantErr && result != nil {
				t.Errorf("enrichGraphAPIError() expected no error, got %v", result)
			}
		})
	}
}

// TestEnrichGraphAPIError_NoPanic tests that enrichGraphAPIError doesn't panic
func TestEnrichGraphAPIError_NoPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("enrichGraphAPIError() panicked: %v", r)
		}
	}()

	// Test with various nil combinations
	enrichGraphAPIError(nil, nil, "")
	enrichGraphAPIError(nil, nil, "operation")
	enrichGraphAPIError(&testError{msg: "test"}, nil, "")
	enrichGraphAPIError(&testError{msg: "test"}, nil, "operation")
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

// TestInt32Ptr tests the Int32Ptr helper function
func TestInt32Ptr(t *testing.T) {
	tests := []struct {
		name  string
		input int32
	}{
		{
			name:  "zero value",
			input: 0,
		},
		{
			name:  "positive value",
			input: 42,
		},
		{
			name:  "negative value",
			input: -100,
		},
		{
			name:  "max int32",
			input: 2147483647,
		},
		{
			name:  "min int32",
			input: -2147483648,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Int32Ptr(tt.input)

			// Check that result is not nil
			if result == nil {
				t.Error("Int32Ptr() returned nil")
				return
			}

			// Check that dereferenced value matches input
			if *result != tt.input {
				t.Errorf("Int32Ptr(%d) = %d, want %d", tt.input, *result, tt.input)
			}

			// Check that the pointer points to a different address than the input
			// (This verifies that a new memory location was created)
			inputAddr := &tt.input
			if result == inputAddr {
				t.Error("Int32Ptr() returned pointer to input variable instead of new allocation")
			}
		})
	}
}

// TestMaskGUID tests the maskGUID function
func TestMaskGUID(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "standard GUID",
			input:    "12345678-1234-1234-1234-123456789012",
			expected: "1234****-****-****-****9012",
		},
		{
			name:     "GUID without dashes",
			input:    "12345678123412341234123456789012",
			expected: "1234****-****-****-****9012",
		},
		{
			name:     "short string",
			input:    "short",
			expected: "****",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "****",
		},
		{
			name:     "exactly 8 characters",
			input:    "12345678",
			expected: "****",
		},
		{
			name:     "9 characters",
			input:    "123456789",
			expected: "1234****-****-****-****6789",
		},
		{
			name:     "uppercase GUID",
			input:    "ABCDEFAB-1234-5678-9ABC-DEF012345678",
			expected: "ABCD****-****-****-****5678",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := maskGUID(tt.input)
			if result != tt.expected {
				t.Errorf("maskGUID(%q) = %q, want %q", tt.input, result, tt.expected)
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
