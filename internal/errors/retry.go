package errors

import (
	"context"
	"fmt"
	"math"
	"time"
)

// RetryConfig holds configuration for retry logic
type RetryConfig struct {
	// MaxAttempts is the maximum number of retry attempts
	MaxAttempts int
	// InitialDelay is the initial delay before the first retry
	InitialDelay time.Duration
	// MaxDelay is the maximum delay between retries
	MaxDelay time.Duration
	// Multiplier is the factor by which the delay increases after each retry
	Multiplier float64
	// ShouldRetry is an optional function to determine if an error should be retried
	ShouldRetry func(error) bool
}

// DefaultRetryConfig returns a default retry configuration
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 1 * time.Second,
		MaxDelay:     30 * time.Second,
		Multiplier:   2.0,
		ShouldRetry: func(err error) bool {
			return IsRetryable(err)
		},
	}
}

// RetryWithBackoff executes a function with exponential backoff retry logic
func RetryWithBackoff(ctx context.Context, config *RetryConfig, fn func() error) error {
	if config == nil {
		config = DefaultRetryConfig()
	}

	var lastErr error
	delay := config.InitialDelay

	for attempt := 1; attempt <= config.MaxAttempts; attempt++ {
		// Execute the function
		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if we should retry
		shouldRetry := config.ShouldRetry != nil && config.ShouldRetry(err)
		if !shouldRetry {
			return err
		}

		// Don't sleep after the last attempt
		if attempt == config.MaxAttempts {
			break
		}

		// Check context cancellation before sleeping
		select {
		case <-ctx.Done():
			return fmt.Errorf("retry cancelled: %w", ctx.Err())
		default:
		}

		// Sleep with exponential backoff
		select {
		case <-ctx.Done():
			return fmt.Errorf("retry cancelled: %w", ctx.Err())
		case <-time.After(delay):
		}

		// Calculate next delay with exponential backoff
		delay = time.Duration(float64(delay) * config.Multiplier)
		if delay > config.MaxDelay {
			delay = config.MaxDelay
		}
	}

	return fmt.Errorf("max retry attempts (%d) exceeded: %w", config.MaxAttempts, lastErr)
}

// RetryableOperation represents an operation that can be retried
type RetryableOperation struct {
	config *RetryConfig
}

// NewRetryableOperation creates a new retryable operation with the given config
func NewRetryableOperation(config *RetryConfig) *RetryableOperation {
	if config == nil {
		config = DefaultRetryConfig()
	}
	return &RetryableOperation{config: config}
}

// Execute runs the operation with retry logic
func (r *RetryableOperation) Execute(ctx context.Context, fn func() error) error {
	return RetryWithBackoff(ctx, r.config, fn)
}

// CalculateDelay calculates the delay for a specific attempt using exponential backoff
func CalculateDelay(attempt int, initialDelay, maxDelay time.Duration, multiplier float64) time.Duration {
	if attempt <= 0 {
		return initialDelay
	}

	delay := time.Duration(float64(initialDelay) * math.Pow(multiplier, float64(attempt-1)))
	if delay > maxDelay {
		return maxDelay
	}
	return delay
}
