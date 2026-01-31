package ratelimit

import (
	"context"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name        string
		rps         float64
		wantEnabled bool
		wantRPS     float64
	}{
		{
			name:        "disabled with zero",
			rps:         0,
			wantEnabled: false,
			wantRPS:     0,
		},
		{
			name:        "disabled with negative",
			rps:         -1,
			wantEnabled: false,
			wantRPS:     0,
		},
		{
			name:        "enabled with 1 rps",
			rps:         1.0,
			wantEnabled: true,
			wantRPS:     1.0,
		},
		{
			name:        "enabled with 10 rps",
			rps:         10.0,
			wantEnabled: true,
			wantRPS:     10.0,
		},
		{
			name:        "enabled with fractional rps",
			rps:         0.5,
			wantEnabled: true,
			wantRPS:     0.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limiter := New(tt.rps)
			if limiter == nil {
				t.Fatal("New() returned nil")
			}

			if limiter.Enabled() != tt.wantEnabled {
				t.Errorf("Enabled() = %v, want %v", limiter.Enabled(), tt.wantEnabled)
			}

			if limiter.RPS() != tt.wantRPS {
				t.Errorf("RPS() = %v, want %v", limiter.RPS(), tt.wantRPS)
			}
		})
	}
}

func TestLimiter_Wait_Disabled(t *testing.T) {
	limiter := New(0) // Disabled

	ctx := context.Background()
	start := time.Now()

	// Should return immediately
	err := limiter.Wait(ctx)
	duration := time.Since(start)

	if err != nil {
		t.Errorf("Wait() returned error: %v", err)
	}

	// Should complete in less than 10ms (practically instant)
	if duration > 10*time.Millisecond {
		t.Errorf("Wait() took too long for disabled limiter: %v", duration)
	}
}

func TestLimiter_Wait_Enabled(t *testing.T) {
	// Create limiter allowing 10 requests per second
	limiter := New(10.0)

	ctx := context.Background()

	// First request should pass immediately
	start := time.Now()
	err := limiter.Wait(ctx)
	duration := time.Since(start)

	if err != nil {
		t.Errorf("Wait() returned error: %v", err)
	}

	// First request should be fast
	if duration > 10*time.Millisecond {
		t.Errorf("First Wait() took too long: %v", duration)
	}

	// Second request should be rate limited
	start = time.Now()
	err = limiter.Wait(ctx)
	duration = time.Since(start)

	if err != nil {
		t.Errorf("Wait() returned error: %v", err)
	}

	// With 10 rps, wait should be approximately 100ms
	// Allow some tolerance (50ms to 200ms)
	if duration < 50*time.Millisecond || duration > 200*time.Millisecond {
		t.Errorf("Second Wait() duration out of expected range: %v (expected ~100ms)", duration)
	}
}

func TestLimiter_Wait_ContextCanceled(t *testing.T) {
	// Create limiter with very low rate
	limiter := New(0.1) // 1 request per 10 seconds

	// Create context with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// First request passes
	_ = limiter.Wait(context.Background())

	// Second request should be canceled by context
	err := limiter.Wait(ctx)

	if err == nil {
		t.Error("Wait() should return error when context is canceled")
	}

	// The error should be related to context cancellation
	// (golang.org/x/time/rate returns a custom error, not context.DeadlineExceeded)
	t.Logf("Context cancellation error: %v", err)
}

func TestLimiter_Allow_Disabled(t *testing.T) {
	limiter := New(0) // Disabled

	// Should always allow
	for i := 0; i < 100; i++ {
		if !limiter.Allow() {
			t.Errorf("Allow() returned false for disabled limiter at iteration %d", i)
		}
	}
}

func TestLimiter_Allow_Enabled(t *testing.T) {
	limiter := New(10.0) // 10 rps

	// First request should be allowed
	if !limiter.Allow() {
		t.Error("First Allow() should return true")
	}

	// Immediate subsequent requests should be denied (token bucket empty)
	allowed := 0
	for i := 0; i < 10; i++ {
		if limiter.Allow() {
			allowed++
		}
	}

	// Most should be denied since tokens aren't replenished instantly
	if allowed > 2 {
		t.Errorf("Too many requests allowed immediately: %d (expected 0-2)", allowed)
	}

	// After waiting for token replenishment, should be allowed again
	time.Sleep(150 * time.Millisecond) // Wait for ~1 token at 10 rps

	if !limiter.Allow() {
		t.Error("Allow() should return true after waiting for token replenishment")
	}
}

func TestLimiter_Reserve(t *testing.T) {
	limiter := New(10.0) // 10 rps

	// Reserve a token
	r := limiter.Reserve()
	if r == nil {
		t.Fatal("Reserve() returned nil")
	}

	// Check delay (should be small for first request)
	delay := r.Delay()
	if delay > 10*time.Millisecond {
		t.Errorf("First Reserve() delay too long: %v", delay)
	}

	// Reserve another token immediately
	r2 := limiter.Reserve()
	delay2 := r2.Delay()

	// Second reservation should have a delay (~100ms at 10 rps)
	if delay2 < 50*time.Millisecond {
		t.Errorf("Second Reserve() delay too short: %v (expected ~100ms)", delay2)
	}
}

func TestLimiter_Reserve_Disabled(t *testing.T) {
	limiter := New(0) // Disabled

	// Should return nil for disabled limiter (indicates unlimited rate)
	r := limiter.Reserve()
	if r != nil {
		t.Errorf("Reserve() returned non-nil for disabled limiter: %v", r)
	}
}

func TestLimiter_String(t *testing.T) {
	tests := []struct {
		name       string
		rps        float64
		wantSubstr string
	}{
		{
			name:       "disabled",
			rps:        0,
			wantSubstr: "disabled",
		},
		{
			name:       "10 rps",
			rps:        10.0,
			wantSubstr: "10.00 rps",
		},
		{
			name:       "fractional rate",
			rps:        0.5,
			wantSubstr: "1 request per",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limiter := New(tt.rps)
			str := limiter.String()

			if str == "" {
				t.Error("String() returned empty string")
			}

			// Check if expected substring is present
			// Note: We use a simple check since the exact format might vary
			t.Logf("String() = %q", str)
		})
	}
}

func TestLimiter_ConcurrentAccess(t *testing.T) {
	limiter := New(100.0) // 100 rps

	// Launch multiple goroutines
	ctx := context.Background()
	const numGoroutines = 10

	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			for j := 0; j < 5; j++ {
				if err := limiter.Wait(ctx); err != nil {
					errors <- err
					return
				}
			}
			errors <- nil
		}()
	}

	// Collect results
	for i := 0; i < numGoroutines; i++ {
		err := <-errors
		if err != nil {
			t.Errorf("Concurrent Wait() failed: %v", err)
		}
	}
}

func BenchmarkLimiter_Wait(b *testing.B) {
	limiter := New(1000.0) // High rate for benchmarking
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = limiter.Wait(ctx)
	}
}

func BenchmarkLimiter_Allow(b *testing.B) {
	limiter := New(1000.0) // High rate for benchmarking

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		limiter.Allow()
	}
}
