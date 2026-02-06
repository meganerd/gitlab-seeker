package errors

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"syscall"
)

// ErrorType represents different categories of errors
type ErrorType int

const (
	// ErrorTypeUnknown represents an unknown error type
	ErrorTypeUnknown ErrorType = iota
	// ErrorTypeNetwork represents network-related errors
	ErrorTypeNetwork
	// ErrorTypeTimeout represents timeout errors
	ErrorTypeTimeout
	// ErrorTypeAuthentication represents authentication failures
	ErrorTypeAuthentication
	// ErrorTypeRateLimit represents rate limiting errors
	ErrorTypeRateLimit
	// ErrorTypeNotFound represents resource not found errors
	ErrorTypeNotFound
	// ErrorTypePermission represents permission denied errors
	ErrorTypePermission
)

// AppError represents a custom application error with additional context
type AppError struct {
	Type    ErrorType
	Message string
	Err     error
	// Retryable indicates if the operation can be retried
	Retryable bool
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

// Unwrap implements error unwrapping for error chains
func (e *AppError) Unwrap() error {
	return e.Err
}

// IsRetryable returns whether this error can be retried
func (e *AppError) IsRetryable() bool {
	return e.Retryable
}

// NewNetworkError creates a new network-related error
func NewNetworkError(err error) *AppError {
	return &AppError{
		Type:      ErrorTypeNetwork,
		Message:   "network operation failed",
		Err:       err,
		Retryable: true,
	}
}

// NewTimeoutError creates a new timeout error
func NewTimeoutError(err error) *AppError {
	return &AppError{
		Type:      ErrorTypeTimeout,
		Message:   "operation timed out",
		Err:       err,
		Retryable: true,
	}
}

// NewAuthenticationError creates a new authentication error
func NewAuthenticationError(err error) *AppError {
	return &AppError{
		Type:      ErrorTypeAuthentication,
		Message:   "authentication failed",
		Err:       err,
		Retryable: false,
	}
}

// NewRateLimitError creates a new rate limit error
func NewRateLimitError(err error) *AppError {
	return &AppError{
		Type:      ErrorTypeRateLimit,
		Message:   "rate limit exceeded",
		Err:       err,
		Retryable: true,
	}
}

// NewNotFoundError creates a new not found error
func NewNotFoundError(resource string) *AppError {
	return &AppError{
		Type:      ErrorTypeNotFound,
		Message:   fmt.Sprintf("resource not found: %s", resource),
		Retryable: false,
	}
}

// NewPermissionError creates a new permission denied error
func NewPermissionError(resource string) *AppError {
	return &AppError{
		Type:      ErrorTypePermission,
		Message:   fmt.Sprintf("permission denied: %s", resource),
		Retryable: false,
	}
}

// ClassifyError analyzes an error and returns an appropriate AppError
func ClassifyError(err error) *AppError {
	if err == nil {
		return nil
	}

	// Check if it's already an AppError
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr
	}

	// Check for network errors
	var netErr net.Error
	if errors.As(err, &netErr) {
		if netErr.Timeout() {
			return NewTimeoutError(err)
		}
		return NewNetworkError(err)
	}

	// Check for URL errors
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		if urlErr.Timeout() {
			return NewTimeoutError(err)
		}
		if urlErr.Err != nil {
			// Check the underlying error
			return ClassifyError(urlErr.Err)
		}
		return NewNetworkError(err)
	}

	// Check for syscall errors (connection refused, etc.)
	var syscallErr syscall.Errno
	if errors.As(err, &syscallErr) {
		switch syscallErr {
		case syscall.ECONNREFUSED, syscall.ECONNRESET, syscall.ENETUNREACH, syscall.EHOSTUNREACH:
			return NewNetworkError(err)
		case syscall.ETIMEDOUT:
			return NewTimeoutError(err)
		}
	}

	// Default to unknown error type
	return &AppError{
		Type:      ErrorTypeUnknown,
		Message:   "unknown error occurred",
		Err:       err,
		Retryable: false,
	}
}

// IsNetworkError checks if the error is a network-related error
func IsNetworkError(err error) bool {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Type == ErrorTypeNetwork
	}
	return false
}

// IsTimeoutError checks if the error is a timeout error
func IsTimeoutError(err error) bool {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Type == ErrorTypeTimeout
	}
	return false
}

// IsRetryable checks if the error can be retried
func IsRetryable(err error) bool {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Retryable
	}
	return false
}
