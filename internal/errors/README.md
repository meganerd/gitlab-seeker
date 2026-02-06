# Error Handling Package

This package provides comprehensive error handling capabilities for network operations, including:

- Custom error types with detailed classification
- Automatic retry logic with exponential backoff
- User-friendly error messages
- Context-aware timeout handling

## Features

### Custom Error Types

The package defines several error types to classify errors:

- `ErrorTypeNetwork` - Network-related errors (retryable)
- `ErrorTypeTimeout` - Timeout errors (retryable)
- `ErrorTypeAuthentication` - Authentication failures (not retryable)
- `ErrorTypeRateLimit` - Rate limiting errors (retryable)
- `ErrorTypeNotFound` - Resource not found errors (not retryable)
- `ErrorTypePermission` - Permission denied errors (not retryable)

### Error Classification

The `ClassifyError` function automatically analyzes errors and returns an appropriate `AppError`:

```go
err := someNetworkOperation()
appErr := errors.ClassifyError(err)

if appErr.Type == errors.ErrorTypeNetwork {
    // Handle network error
}

if errors.IsRetryable(err) {
    // Retry the operation
}
```

### Retry Logic with Exponential Backoff

The package provides automatic retry functionality:

```go
config := &errors.RetryConfig{
    MaxAttempts:  3,
    InitialDelay: 1 * time.Second,
    MaxDelay:     30 * time.Second,
    Multiplier:   2.0,
}

err := errors.RetryWithBackoff(ctx, config, func() error {
    return someNetworkOperation()
})
```

Default configuration:
- 3 max attempts
- 1 second initial delay
- 30 second max delay
- 2.0 exponential multiplier
- Retries only on retryable errors

### Context Support

All retry operations support context cancellation and timeout:

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
defer cancel()

err := errors.RetryWithBackoff(ctx, config, func() error {
    return performLongOperation()
})
```

## Usage Examples

### Basic Error Handling

```go
import apperrors "github.com/gbjohnso/gitlab-python-scanner/internal/errors"

func connectToGitLab() error {
    err := gitlab.Connect()
    if err != nil {
        // Classify the error
        appErr := apperrors.ClassifyError(err)
        
        // Provide user-friendly message based on error type
        switch appErr.Type {
        case apperrors.ErrorTypeNetwork:
            return fmt.Errorf("network error: please check your connection")
        case apperrors.ErrorTypeAuthentication:
            return fmt.Errorf("authentication failed: check your token")
        default:
            return err
        }
    }
    return nil
}
```

### Automatic Retry

```go
func fetchDataWithRetry(ctx context.Context) error {
    config := apperrors.DefaultRetryConfig()
    
    return apperrors.RetryWithBackoff(ctx, config, func() error {
        data, err := api.FetchData()
        if err != nil {
            // Classify the error to determine if it's retryable
            return apperrors.ClassifyError(err)
        }
        // Process data
        return nil
    })
}
```

### Custom Retry Logic

```go
func customRetry() error {
    config := &apperrors.RetryConfig{
        MaxAttempts:  5,
        InitialDelay: 500 * time.Millisecond,
        MaxDelay:     10 * time.Second,
        Multiplier:   1.5,
        ShouldRetry: func(err error) bool {
            // Custom retry logic
            return apperrors.IsNetworkError(err) || apperrors.IsTimeoutError(err)
        },
    }
    
    return apperrors.RetryWithBackoff(context.Background(), config, func() error {
        return performOperation()
    })
}
```

## Error Type Helpers

```go
// Check specific error types
if errors.IsNetworkError(err) {
    // Handle network error
}

if errors.IsTimeoutError(err) {
    // Handle timeout
}

if errors.IsRetryable(err) {
    // Can retry this operation
}
```

## Creating Custom Errors

```go
// Create specific error types
err := errors.NewNetworkError(netErr)
err := errors.NewTimeoutError(timeoutErr)
err := errors.NewAuthenticationError(authErr)
err := errors.NewRateLimitError(rateLimitErr)
err := errors.NewNotFoundError("resource-id")
err := errors.NewPermissionError("resource-name")
```

## Integration with GitLab Client

The error handling is integrated into the GitLab client for seamless error management:

```go
client, err := gitlab.NewClient(config)
if err != nil {
    log.Fatal(err)
}

// TestConnection automatically retries on network failures
err = client.TestConnection()
if err != nil {
    // Error messages are user-friendly and actionable
    log.Fatal(err)
}
```

## Best Practices

1. **Always use context**: Pass a context to retry operations for proper timeout and cancellation
2. **Classify errors**: Use `ClassifyError` to properly categorize errors before handling
3. **Don't retry non-retryable errors**: Check `IsRetryable` before implementing custom retry logic
4. **Set appropriate timeouts**: Configure both operation timeout and retry delays based on your use case
5. **Log retry attempts**: Consider logging retry attempts for debugging and monitoring
