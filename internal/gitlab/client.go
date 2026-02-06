package gitlab

import (
	"fmt"
	"net/url"
	"strings"
	"time"

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
	if c.client == nil {
		return fmt.Errorf("GitLab client is not initialized")
	}

	// Try to get the current user to verify authentication
	_, resp, err := c.client.Users.CurrentUser()
	if err != nil {
		// Check if it's an authentication error
		if resp != nil && resp.StatusCode == 401 {
			return fmt.Errorf("authentication failed: invalid token")
		}
		return fmt.Errorf("failed to connect to GitLab: %w", err)
	}

	return nil
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
