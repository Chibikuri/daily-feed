package retry

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"time"
)

func init() {
	// Initialize the random number generator for jitter
	rand.Seed(time.Now().UnixNano())
}

// Config holds retry configuration
type Config struct {
	MaxRetries int
	BaseDelay  time.Duration
}

// DefaultConfig returns a default retry configuration
func DefaultConfig() Config {
	return Config{
		MaxRetries: 3,
		BaseDelay:  1 * time.Second,
	}
}

// WithBackoff executes a function with exponential backoff retry logic
func WithBackoff(ctx context.Context, config Config, operation func(context.Context) error) error {
	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		err := operation(ctx)
		if err == nil {
			return nil
		}
		
		// Check if error is retryable
		if !isRetryableError(err) {
			return fmt.Errorf("non-retryable error: %w", err)
		}
		
		// Don't retry on the last attempt
		if attempt == config.MaxRetries {
			return fmt.Errorf("operation failed after %d attempts: %w", config.MaxRetries+1, err)
		}
		
		// Calculate exponential backoff delay with jitter
		baseDelay := config.BaseDelay * time.Duration(1<<attempt)
		jitter := time.Duration(rand.Int63n(int64(config.BaseDelay)))
		delay := baseDelay + jitter
		
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
			// Continue to next attempt
		}
	}
	
	return nil // Should never reach here
}

// isRetryableError determines if an error is worth retrying
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}
	
	errStr := strings.ToLower(err.Error())
	
	// Network-level errors are generally retryable
	if strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "connection reset") ||
		strings.Contains(errStr, "temporary failure") ||
		strings.Contains(errStr, "network") {
		return true
	}
	
	// Look for HTTP status codes in error messages
	// Only 5xx server errors and 429 rate limiting should be retried
	if strings.Contains(errStr, "status 5") || // 5xx errors
		strings.Contains(errStr, "status 429") { // Rate limiting
		return true
	}
	
	// Don't retry 4xx client errors (except 429)
	if strings.Contains(errStr, "status 4") {
		return false
	}
	
	// For unknown errors, err on the side of caution and retry
	// This handles cases where the specific error format isn't recognized
	return true
}

// HTTPStatusRetryable checks if an HTTP status code is retryable
func HTTPStatusRetryable(statusCode int) bool {
	// Retry on server errors (5xx) and rate limiting (429)
	return statusCode >= 500 || statusCode == http.StatusTooManyRequests
}