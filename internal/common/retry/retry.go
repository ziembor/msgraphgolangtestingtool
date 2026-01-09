package retry

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"
)

// IsRetryableError determines if an error is transient and worth retrying.
// Returns true for network timeouts, connection errors, and temporary failures.
// Returns false for context cancellation, permanent errors, and authentication failures.
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Check for context cancellation - never retry these
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	// Check error message for common transient patterns
	errMsg := strings.ToLower(err.Error())
	transientPatterns := []string{
		"timeout",
		"connection reset",
		"connection refused",
		"temporary failure",
		"try again",
		"i/o timeout",
		"no such host",
		"network is unreachable",
		"broken pipe",
		"connection timed out",
	}

	for _, pattern := range transientPatterns {
		if strings.Contains(errMsg, pattern) {
			return true
		}
	}

	return false
}

// IsSMTPRetryableError determines if an SMTP error code is retryable.
// Returns true for 4xx temporary SMTP errors.
// Returns false for 5xx permanent SMTP errors and 2xx/3xx success codes.
func IsSMTPRetryableError(smtpCode int) bool {
	// 4xx codes are temporary failures - retry
	if smtpCode >= 400 && smtpCode < 500 {
		return true
	}
	// 5xx codes are permanent failures - don't retry
	// 2xx/3xx are success codes - don't retry
	return false
}

// RetryWithBackoff wraps an operation with exponential backoff retry logic.
// The operation is retried up to maxRetries times with exponentially increasing delays.
// Base delay doubles on each attempt (capped at 30 seconds).
// Context cancellation is respected and will stop retries immediately.
//
// Example usage:
//
//	err := retry.RetryWithBackoff(ctx, 3, 2*time.Second, func() error {
//	    return doSomethingThatMightFail()
//	})
func RetryWithBackoff(ctx context.Context, maxRetries int, baseDelay time.Duration, operation func() error) error {
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		// Execute the operation
		lastErr = operation()

		// Success - return immediately
		if lastErr == nil {
			if attempt > 0 {
				log.Printf("Operation succeeded after %d retries", attempt)
			}
			return nil
		}

		// Check if error is retryable
		if !IsRetryableError(lastErr) {
			// Non-retryable error - fail immediately
			return lastErr
		}

		// Last attempt failed - return error
		if attempt == maxRetries {
			return fmt.Errorf("operation failed after %d retries: %w", maxRetries, lastErr)
		}

		// Calculate exponential backoff delay (cap at 30 seconds)
		delay := baseDelay * time.Duration(1<<uint(attempt))
		if delay > 30*time.Second {
			delay = 30 * time.Second
		}

		log.Printf("Retryable error encountered (attempt %d/%d): %v. Retrying in %v...",
			attempt+1, maxRetries, lastErr, delay)

		// Wait with context cancellation support
		select {
		case <-ctx.Done():
			return fmt.Errorf("retry cancelled: %w", ctx.Err())
		case <-time.After(delay):
			// Continue to next retry attempt
		}
	}

	return lastErr
}
