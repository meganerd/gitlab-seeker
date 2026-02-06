package gitlab

import (
	"context"
	stderrors "errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	apperrors "github.com/gbjohnso/gitlab-python-scanner/internal/errors"
	"github.com/xanzy/go-gitlab"
)

// Client wraps the GitLab API client with additional metadata
type Client struct {
	client       *gitlab.Client
	baseURL      string
	organization string
	timeout      time.Duration
}

// Config holds the configuration for creating a GitLab client
type Config struct {
	GitLabURL string        // Full URL including org/group (e.g., "gitlab.com/myorg")
	Token     string        // GitLab API token
	Timeout   time.Duration // API timeout duration
}

// NewClient creates a new GitLab API client with authentication
func NewClient(config *Config) (*Client, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	if config.Token == "" {
		return nil, fmt.Errorf("GitLab token is required")
	}

	if config.GitLabURL == "" {
		return nil, fmt.Errorf("GitLab URL is required")
	}

	// Parse the GitLab URL to extract base URL and organization
	baseURL, organization, err := parseGitLabURL(config.GitLabURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse GitLab URL: %w", err)
	}

	// Create the go-gitlab client
	gitlabClient, err := gitlab.NewClient(config.Token, gitlab.WithBaseURL(baseURL))
	if err != nil {
		return nil, fmt.Errorf("failed to create GitLab client: %w", err)
	}

	// Set timeout if provided
	timeout := config.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second // default timeout
	}

	client := &Client{
		client:       gitlabClient,
		baseURL:      baseURL,
		organization: organization,
		timeout:      timeout,
	}

	return client, nil
}

// parseGitLabURL extracts the base URL and organization/group from a GitLab URL
// Examples:
//   - "gitlab.com/myorg" -> "https://gitlab.com", "myorg"
//   - "https://gitlab.com/myorg" -> "https://gitlab.com", "myorg"
//   - "gitlab.example.com/group/subgroup" -> "https://gitlab.example.com", "group/subgroup"
func parseGitLabURL(gitlabURL string) (baseURL, organization string, err error) {
	// Ensure the URL has a scheme
	if !strings.HasPrefix(gitlabURL, "http://") && !strings.HasPrefix(gitlabURL, "https://") {
		gitlabURL = "https://" + gitlabURL
	}

	// Parse the URL
	parsedURL, err := url.Parse(gitlabURL)
	if err != nil {
		return "", "", fmt.Errorf("invalid URL format: %w", err)
	}

	// Extract the base URL (scheme + host)
	baseURL = fmt.Sprintf("%s://%s", parsedURL.Scheme, parsedURL.Host)

	// Extract the organization/group from the path
	// Remove leading/trailing slashes
	path := strings.Trim(parsedURL.Path, "/")
	if path == "" {
		return "", "", fmt.Errorf("organization/group path is missing from URL")
	}

	organization = path

	return baseURL, organization, nil
}

// TestConnection verifies that the client can connect to GitLab and authenticate
func (c *Client) TestConnection() error {
	return c.TestConnectionWithContext(context.Background())
}

// TestConnectionWithContext verifies that the client can connect to GitLab and authenticate
// with context support for timeout and cancellation
func (c *Client) TestConnectionWithContext(ctx context.Context) error {
	if c.client == nil {
		return fmt.Errorf("GitLab client is not initialized")
	}

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	// Configure retry for network failures
	retryConfig := &apperrors.RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 1 * time.Second,
		MaxDelay:     10 * time.Second,
		Multiplier:   2.0,
		ShouldRetry: func(err error) bool {
			return apperrors.IsRetryable(err)
		},
	}

	var lastResp *gitlab.Response
	err := apperrors.RetryWithBackoff(ctx, retryConfig, func() error {
		// Try to get the current user to verify authentication
		_, resp, err := c.client.Users.CurrentUser()
		lastResp = resp
		if err != nil {
			return classifyGitLabError(err, resp)
		}
		return nil
	})

	if err != nil {
		// Provide user-friendly error messages
		return c.formatUserError(err, lastResp)
	}

	return nil
}

// classifyGitLabError analyzes a GitLab API error and returns an appropriate AppError
func classifyGitLabError(err error, resp *gitlab.Response) error {
	if err == nil {
		return nil
	}

	// Check HTTP response status codes
	if resp != nil {
		switch resp.StatusCode {
		case http.StatusUnauthorized:
			return apperrors.NewAuthenticationError(err)
		case http.StatusForbidden:
			return apperrors.NewPermissionError("GitLab API access")
		case http.StatusNotFound:
			return apperrors.NewNotFoundError("GitLab resource")
		case http.StatusTooManyRequests:
			return apperrors.NewRateLimitError(err)
		case http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
			return apperrors.NewNetworkError(err)
		}
	}

	// Classify the underlying error
	return apperrors.ClassifyError(err)
}

// formatUserError formats an error for user-friendly display
func (c *Client) formatUserError(err error, resp *gitlab.Response) error {
	var appErr *apperrors.AppError
	if !stderrors.As(err, &appErr) {
		return err
	}

	switch appErr.Type {
	case apperrors.ErrorTypeAuthentication:
		return fmt.Errorf("authentication failed: please check your GitLab token")
	case apperrors.ErrorTypeNetwork:
		if resp != nil && resp.StatusCode >= 500 {
			return fmt.Errorf("GitLab server error (HTTP %d): the server may be experiencing issues. Please try again later", resp.StatusCode)
		}
		return fmt.Errorf("network error: unable to reach GitLab server. Please check your internet connection and the GitLab URL")
	case apperrors.ErrorTypeTimeout:
		return fmt.Errorf("connection timeout: GitLab server did not respond within %v. Please check your network or try increasing the timeout", c.timeout)
	case apperrors.ErrorTypeRateLimit:
		return fmt.Errorf("rate limit exceeded: too many requests to GitLab API. Please wait a moment before trying again")
	case apperrors.ErrorTypePermission:
		return fmt.Errorf("permission denied: your GitLab token does not have sufficient permissions")
	default:
		return err
	}
}

// GetOrganization returns the organization/group path
func (c *Client) GetOrganization() string {
	return c.organization
}

// GetBaseURL returns the base GitLab URL
func (c *Client) GetBaseURL() string {
	return c.baseURL
}

// GetClient returns the underlying go-gitlab client for advanced usage
func (c *Client) GetClient() *gitlab.Client {
	return c.client
}

// GetTimeout returns the configured timeout duration
func (c *Client) GetTimeout() time.Duration {
	return c.timeout
}
