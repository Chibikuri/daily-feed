package retry

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"
)

func TestWithBackoff_Success(t *testing.T) {
	config := Config{MaxRetries: 3, BaseDelay: 1 * time.Millisecond}
	attempts := 0
	
	operation := func(ctx context.Context) error {
		attempts++
		if attempts < 3 {
			return errors.New("temporary error")
		}
		return nil
	}
	
	err := WithBackoff(context.Background(), config, operation)
	if err != nil {
		t.Fatalf("Expected success, got error: %v", err)
	}
	
	if attempts != 3 {
		t.Fatalf("Expected 3 attempts, got %d", attempts)
	}
}

func TestWithBackoff_FailureAfterMaxRetries(t *testing.T) {
	config := Config{MaxRetries: 2, BaseDelay: 1 * time.Millisecond}
	attempts := 0
	
	operation := func(ctx context.Context) error {
		attempts++
		return errors.New("persistent error")
	}
	
	err := WithBackoff(context.Background(), config, operation)
	if err == nil {
		t.Fatal("Expected failure, got success")
	}
	
	if attempts != 3 { // MaxRetries + 1
		t.Fatalf("Expected 3 attempts, got %d", attempts)
	}
	
	if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
		// Check that it's our retry error
		expected := "operation failed after 3 attempts"
		if err.Error()[:len(expected)] != expected {
			t.Fatalf("Expected retry failure error, got: %v", err)
		}
	}
}

func TestWithBackoff_NonRetryableError(t *testing.T) {
	config := Config{MaxRetries: 3, BaseDelay: 1 * time.Millisecond}
	attempts := 0
	
	operation := func(ctx context.Context) error {
		attempts++
		return fmt.Errorf("unexpected status %d", http.StatusBadRequest)
	}
	
	err := WithBackoff(context.Background(), config, operation)
	if err == nil {
		t.Fatal("Expected failure, got success")
	}
	
	if attempts != 1 {
		t.Fatalf("Expected 1 attempt for non-retryable error, got %d", attempts)
	}
	
	expected := "non-retryable error"
	if err.Error()[:len(expected)] != expected {
		t.Fatalf("Expected non-retryable error, got: %v", err)
	}
}

func TestWithBackoff_ContextCancellation(t *testing.T) {
	config := Config{MaxRetries: 5, BaseDelay: 100 * time.Millisecond}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	
	attempts := 0
	operation := func(ctx context.Context) error {
		attempts++
		return errors.New("retryable error")
	}
	
	start := time.Now()
	err := WithBackoff(ctx, config, operation)
	duration := time.Since(start)
	
	if err == nil {
		t.Fatal("Expected context cancellation error")
	}
	
	if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
		t.Fatalf("Expected context error, got: %v", err)
	}
	
	// Should have aborted quickly due to context timeout
	if duration > 200*time.Millisecond {
		t.Fatalf("Expected quick abort, took %v", duration)
	}
}

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"nil error", nil, false},
		{"timeout error", errors.New("request timeout"), true},
		{"connection refused", errors.New("connection refused"), true},
		{"5xx server error", errors.New("unexpected status 500"), true},
		{"502 bad gateway", errors.New("unexpected status 502"), true},
		{"429 rate limit", errors.New("unexpected status 429"), true},
		{"400 bad request", errors.New("unexpected status 400"), false},
		{"401 unauthorized", errors.New("unexpected status 401"), false},
		{"403 forbidden", errors.New("unexpected status 403"), false},
		{"404 not found", errors.New("unexpected status 404"), false},
		{"unknown error", errors.New("some unknown error"), true},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isRetryableError(tt.err)
			if result != tt.expected {
				t.Errorf("isRetryableError(%v) = %v, expected %v", tt.err, result, tt.expected)
			}
		})
	}
}

func TestHTTPStatusRetryable(t *testing.T) {
	tests := []struct {
		status   int
		expected bool
	}{
		{200, false},
		{201, false},
		{400, false},
		{401, false},
		{403, false},
		{404, false},
		{429, true},  // Rate limiting
		{500, true},  // Server error
		{502, true},  // Bad gateway
		{503, true},  // Service unavailable
		{504, true},  // Gateway timeout
	}
	
	for _, tt := range tests {
		t.Run(fmt.Sprintf("status_%d", tt.status), func(t *testing.T) {
			result := HTTPStatusRetryable(tt.status)
			if result != tt.expected {
				t.Errorf("HTTPStatusRetryable(%d) = %v, expected %v", tt.status, result, tt.expected)
			}
		})
	}
}