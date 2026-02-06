package errors

import (
	"context"
	"errors"
	"net"
	"syscall"
	"testing"
	"time"
)

func TestClassifyError(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		expectedType ErrorType
		retryable    bool
	}{
		{
			name:         "nil error",
			err:          nil,
			expectedType: ErrorTypeUnknown,
			retryable:    false,
		},
		{
			name:         "network timeout",
			err:          &mockNetError{timeout: true},
			expectedType: ErrorTypeTimeout,
			retryable:    true,
		},
		{
			name:         "network error",
			err:          &mockNetError{timeout: false},
			expectedType: ErrorTypeNetwork,
			retryable:    true,
		},
		{
			name:         "connection refused",
			err:          syscall.ECONNREFUSED,
			expectedType: ErrorTypeNetwork,
			retryable:    true,
		},
		{
			name:         "authentication error",
			err:          NewAuthenticationError(errors.New("invalid token")),
			expectedType: ErrorTypeAuthentication,
			retryable:    false,
		},
		{
			name:         "unknown error",
			err:          errors.New("some error"),
			expectedType: ErrorTypeUnknown,
			retryable:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err == nil {
				appErr := ClassifyError(tt.err)
				if appErr != nil {
					t.Errorf("expected nil for nil error, got %v", appErr)
				}
				return
			}

			appErr := ClassifyError(tt.err)
			if appErr == nil {
				t.Fatal("expected non-nil AppError")
			}

			if appErr.Type != tt.expectedType {
				t.Errorf("expected error type %v, got %v", tt.expectedType, appErr.Type)
			}

			if appErr.Retryable != tt.retryable {
				t.Errorf("expected retryable=%v, got %v", tt.retryable, appErr.Retryable)
			}
		})
	}
}

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "network error is retryable",
			err:      NewNetworkError(errors.New("connection failed")),
			expected: true,
		},
		{
			name:     "timeout error is retryable",
			err:      NewTimeoutError(errors.New("timeout")),
			expected: true,
		},
		{
			name:     "authentication error is not retryable",
			err:      NewAuthenticationError(errors.New("invalid token")),
			expected: false,
		},
		{
			name:     "standard error is not retryable",
			err:      errors.New("some error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsRetryable(tt.err)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestRetryWithBackoff(t *testing.T) {
	t.Run("success on first attempt", func(t *testing.T) {
		attempts := 0
		err := RetryWithBackoff(context.Background(), DefaultRetryConfig(), func() error {
			attempts++
			return nil
		})

		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}

		if attempts != 1 {
			t.Errorf("expected 1 attempt, got %d", attempts)
		}
	})

	t.Run("retry on retryable error", func(t *testing.T) {
		attempts := 0
		config := &RetryConfig{
			MaxAttempts:  3,
			InitialDelay: 10 * time.Millisecond,
			MaxDelay:     100 * time.Millisecond,
			Multiplier:   2.0,
			ShouldRetry: func(err error) bool {
				return IsRetryable(err)
			},
		}

		err := RetryWithBackoff(context.Background(), config, func() error {
			attempts++
			if attempts < 3 {
				return NewNetworkError(errors.New("connection failed"))
			}
			return nil
		})

		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}

		if attempts != 3 {
			t.Errorf("expected 3 attempts, got %d", attempts)
		}
	})

	t.Run("no retry on non-retryable error", func(t *testing.T) {
		attempts := 0
		config := DefaultRetryConfig()

		err := RetryWithBackoff(context.Background(), config, func() error {
			attempts++
			return NewAuthenticationError(errors.New("invalid token"))
		})

		if err == nil {
			t.Error("expected error, got nil")
		}

		if attempts != 1 {
			t.Errorf("expected 1 attempt, got %d", attempts)
		}
	})

	t.Run("max attempts exceeded", func(t *testing.T) {
		attempts := 0
		config := &RetryConfig{
			MaxAttempts:  3,
			InitialDelay: 10 * time.Millisecond,
			MaxDelay:     100 * time.Millisecond,
			Multiplier:   2.0,
			ShouldRetry: func(err error) bool {
				return IsRetryable(err)
			},
		}

		err := RetryWithBackoff(context.Background(), config, func() error {
			attempts++
			return NewNetworkError(errors.New("connection failed"))
		})

		if err == nil {
			t.Error("expected error, got nil")
		}

		if attempts != 3 {
			t.Errorf("expected 3 attempts, got %d", attempts)
		}
	})

	t.Run("context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		attempts := 0
		config := &RetryConfig{
			MaxAttempts:  5,
			InitialDelay: 100 * time.Millisecond,
			MaxDelay:     1 * time.Second,
			Multiplier:   2.0,
			ShouldRetry: func(err error) bool {
				return IsRetryable(err)
			},
		}

		// Cancel context after first attempt
		go func() {
			time.Sleep(50 * time.Millisecond)
			cancel()
		}()

		err := RetryWithBackoff(ctx, config, func() error {
			attempts++
			return NewNetworkError(errors.New("connection failed"))
		})

		if err == nil {
			t.Error("expected error, got nil")
		}

		// Should stop early due to context cancellation
		if attempts > 2 {
			t.Errorf("expected at most 2 attempts due to context cancellation, got %d", attempts)
		}
	})
}

func TestCalculateDelay(t *testing.T) {
	tests := []struct {
		name         string
		attempt      int
		initialDelay time.Duration
		maxDelay     time.Duration
		multiplier   float64
		expected     time.Duration
	}{
		{
			name:         "first attempt",
			attempt:      1,
			initialDelay: 1 * time.Second,
			maxDelay:     30 * time.Second,
			multiplier:   2.0,
			expected:     1 * time.Second,
		},
		{
			name:         "second attempt",
			attempt:      2,
			initialDelay: 1 * time.Second,
			maxDelay:     30 * time.Second,
			multiplier:   2.0,
			expected:     2 * time.Second,
		},
		{
			name:         "third attempt",
			attempt:      3,
			initialDelay: 1 * time.Second,
			maxDelay:     30 * time.Second,
			multiplier:   2.0,
			expected:     4 * time.Second,
		},
		{
			name:         "exceeds max delay",
			attempt:      10,
			initialDelay: 1 * time.Second,
			maxDelay:     10 * time.Second,
			multiplier:   2.0,
			expected:     10 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateDelay(tt.attempt, tt.initialDelay, tt.maxDelay, tt.multiplier)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// mockNetError implements net.Error for testing
type mockNetError struct {
	timeout bool
}

func (m *mockNetError) Error() string {
	if m.timeout {
		return "timeout"
	}
	return "network error"
}

func (m *mockNetError) Timeout() bool {
	return m.timeout
}

func (m *mockNetError) Temporary() bool {
	return true
}

var _ net.Error = (*mockNetError)(nil)
