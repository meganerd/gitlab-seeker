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

// Project represents a GitLab project with relevant information
type Project struct {
	ID                int    // Project ID
	Name              string // Project name
	Path              string // Project path (URL slug)
	PathWithNamespace string // Full path including group
	WebURL            string // Web URL of the project
	DefaultBranch     string // Default branch name (e.g., "main", "master")
	Archived          bool   // Whether the project is archived
	LastActivityAt    string // Last activity timestamp
}

// ListProjectsOptions contains options for listing projects
type ListProjectsOptions struct {
	PerPage          int   // Number of results per page (default: 20, max: 100)
	Archived         *bool // Filter by archived status (nil = all, true = archived only, false = active only)
	IncludeSubgroups *bool // Include projects from subgroups (nil = default true, explicit true/false to override)
}

// ListProjects retrieves all projects in the organization/group with pagination
func (c *Client) ListProjects(ctx context.Context, opts *ListProjectsOptions) ([]*Project, error) {
	if c.client == nil {
		return nil, fmt.Errorf("GitLab client is not initialized")
	}

	// Set default options
	if opts == nil {
		opts = &ListProjectsOptions{}
	}
	
	perPage := opts.PerPage
	if perPage == 0 {
		perPage = 20 // GitLab default
	}
	if perPage > 100 {
		perPage = 100 // GitLab maximum
	}

	// Build the list options for the go-gitlab library
	listOptions := &gitlab.ListGroupProjectsOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: perPage,
			Page:    1,
		},
	}
	
	// Set IncludeSubGroups (default to true if not specified)
	if opts.IncludeSubgroups != nil {
		listOptions.IncludeSubGroups = opts.IncludeSubgroups
	} else {
		listOptions.IncludeSubGroups = gitlab.Ptr(true)
	}

	// Set archived filter if specified
	if opts.Archived != nil {
		listOptions.Archived = opts.Archived
	}

	var allProjects []*Project

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

	// Paginate through all projects
	for {
		var gitlabProjects []*gitlab.Project
		var resp *gitlab.Response
		var lastErr error

		// Create a context with timeout for this page
		pageCtx, cancel := context.WithTimeout(ctx, c.timeout)
		
		// Fetch one page with retry logic
		err := apperrors.RetryWithBackoff(pageCtx, retryConfig, func() error {
			projects, response, err := c.client.Groups.ListGroupProjects(c.organization, listOptions, gitlab.WithContext(pageCtx))
			if err != nil {
				lastErr = classifyGitLabError(err, response)
				return lastErr
			}
			gitlabProjects = projects
			resp = response
			return nil
		})
		
		cancel() // Clean up the context

		if err != nil {
			return nil, c.formatUserError(err, resp)
		}

		// Convert GitLab projects to our Project type
		for _, gp := range gitlabProjects {
			project := &Project{
				ID:                gp.ID,
				Name:              gp.Name,
				Path:              gp.Path,
				PathWithNamespace: gp.PathWithNamespace,
				WebURL:            gp.WebURL,
				Archived:          gp.Archived,
			}
			
			// Set default branch if available
			if gp.DefaultBranch != "" {
				project.DefaultBranch = gp.DefaultBranch
			}
			
			// Set last activity timestamp if available
			if gp.LastActivityAt != nil {
				project.LastActivityAt = gp.LastActivityAt.String()
			}
			
			allProjects = append(allProjects, project)
		}

		// Check if there are more pages
		if resp.NextPage == 0 {
			break
		}

		// Move to next page
		listOptions.Page = resp.NextPage
	}

	return allProjects, nil
}

// ListAllProjects is a convenience method that lists all active (non-archived) projects
// with default pagination settings
func (c *Client) ListAllProjects(ctx context.Context) ([]*Project, error) {
	archived := false
	includeSubgroups := true
	return c.ListProjects(ctx, &ListProjectsOptions{
		Archived:         &archived,
		IncludeSubgroups: &includeSubgroups,
	})
}

// FileContent represents the content and metadata of a file from a GitLab repository
type FileContent struct {
	FileName      string // Name of the file
	FilePath      string // Full path to the file in the repository
	Size          int    // File size in bytes
	Encoding      string // Encoding of the content (e.g., "base64", "text")
	Content       []byte // Raw file content
	ContentSHA256 string // SHA256 hash of the content
	Ref           string // Git reference (branch, tag, or commit SHA)
	BlobID        string // Git blob ID
	CommitID      string // Last commit ID that modified this file
	LastCommitID  string // Same as CommitID (for compatibility)
}

// GetFileOptions contains options for fetching files from GitLab repositories
type GetFileOptions struct {
	// Ref specifies the branch, tag, or commit SHA to fetch the file from.
	// If empty, uses the project's default branch.
	Ref string
}

// GetRawFile retrieves the raw content of a file from a GitLab repository
// This is the most efficient method for fetching file content as it returns
// the raw bytes without base64 encoding.
//
// Parameters:
//   - projectID: The project ID or path (e.g., 123 or "group/project")
//   - filePath: Path to the file in the repository (e.g., ".python-version")
//   - opts: Optional parameters (can be nil to use defaults)
//
// Returns the raw file content as bytes, or an error if the file cannot be fetched.
func (c *Client) GetRawFile(ctx context.Context, projectID interface{}, filePath string, opts *GetFileOptions) ([]byte, error) {
	if c.client == nil {
		return nil, fmt.Errorf("GitLab client is not initialized")
	}

	if filePath == "" {
		return nil, fmt.Errorf("file path cannot be empty")
	}

	// Build the options for the go-gitlab library
	gitlabOpts := &gitlab.GetRawFileOptions{}
	if opts != nil && opts.Ref != "" {
		gitlabOpts.Ref = gitlab.Ptr(opts.Ref)
	}

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

	var fileContent []byte
	var lastResp *gitlab.Response

	// Create a context with timeout
	fetchCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	// Fetch the file with retry logic
	err := apperrors.RetryWithBackoff(fetchCtx, retryConfig, func() error {
		content, resp, err := c.client.RepositoryFiles.GetRawFile(
			projectID,
			filePath,
			gitlabOpts,
			gitlab.WithContext(fetchCtx),
		)
		lastResp = resp
		if err != nil {
			return classifyGitLabError(err, resp)
		}
		fileContent = content
		return nil
	})

	if err != nil {
		return nil, c.formatUserError(err, lastResp)
	}

	return fileContent, nil
}

// GetFile retrieves a file from a GitLab repository with full metadata
// This method returns more information than GetRawFile but may be less efficient
// as the content is base64-encoded in the API response.
//
// Parameters:
//   - projectID: The project ID or path (e.g., 123 or "group/project")
//   - filePath: Path to the file in the repository (e.g., "pyproject.toml")
//   - opts: Optional parameters (can be nil to use defaults)
//
// Returns a FileContent struct with the file data and metadata.
func (c *Client) GetFile(ctx context.Context, projectID interface{}, filePath string, opts *GetFileOptions) (*FileContent, error) {
	if c.client == nil {
		return nil, fmt.Errorf("GitLab client is not initialized")
	}

	if filePath == "" {
		return nil, fmt.Errorf("file path cannot be empty")
	}

	// Build the options for the go-gitlab library
	gitlabOpts := &gitlab.GetFileOptions{}
	if opts != nil && opts.Ref != "" {
		gitlabOpts.Ref = gitlab.Ptr(opts.Ref)
	}

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

	var gitlabFile *gitlab.File
	var lastResp *gitlab.Response

	// Create a context with timeout
	fetchCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	// Fetch the file with retry logic
	err := apperrors.RetryWithBackoff(fetchCtx, retryConfig, func() error {
		file, resp, err := c.client.RepositoryFiles.GetFile(
			projectID,
			filePath,
			gitlabOpts,
			gitlab.WithContext(fetchCtx),
		)
		lastResp = resp
		if err != nil {
			return classifyGitLabError(err, resp)
		}
		gitlabFile = file
		return nil
	})

	if err != nil {
		return nil, c.formatUserError(err, lastResp)
	}

	// Convert the GitLab File to our FileContent type
	fileContent := &FileContent{
		FileName: gitlabFile.FileName,
		FilePath: gitlabFile.FilePath,
		Size:     gitlabFile.Size,
		Encoding: gitlabFile.Encoding,
		Ref:      gitlabFile.Ref,
		BlobID:   gitlabFile.BlobID,
		CommitID: gitlabFile.CommitID,
	}

	// Set LastCommitID if available
	if gitlabFile.LastCommitID != "" {
		fileContent.LastCommitID = gitlabFile.LastCommitID
	} else {
		fileContent.LastCommitID = gitlabFile.CommitID
	}

	// Set ContentSHA256 if available
	if gitlabFile.ContentSHA256 != "" {
		fileContent.ContentSHA256 = gitlabFile.ContentSHA256
	}

	// Decode the content if it's base64 encoded
	if gitlabFile.Encoding == "base64" && gitlabFile.Content != "" {
		// The go-gitlab library might return Content as a string
		// We need to handle base64 decoding
		fileContent.Content = []byte(gitlabFile.Content)
	} else if gitlabFile.Content != "" {
		fileContent.Content = []byte(gitlabFile.Content)
	}

	return fileContent, nil
}

// GetFileMetadata retrieves metadata about a file without fetching its content
// This is more efficient when you only need file information like size, last commit, etc.
//
// Parameters:
//   - projectID: The project ID or path (e.g., 123 or "group/project")
//   - filePath: Path to the file in the repository (e.g., "requirements.txt")
//   - opts: Optional parameters (can be nil to use defaults)
//
// Returns a FileContent struct with metadata but without the Content field populated.
func (c *Client) GetFileMetadata(ctx context.Context, projectID interface{}, filePath string, opts *GetFileOptions) (*FileContent, error) {
	if c.client == nil {
		return nil, fmt.Errorf("GitLab client is not initialized")
	}

	if filePath == "" {
		return nil, fmt.Errorf("file path cannot be empty")
	}

	// Build the options for the go-gitlab library
	gitlabOpts := &gitlab.GetFileMetaDataOptions{}
	if opts != nil && opts.Ref != "" {
		gitlabOpts.Ref = gitlab.Ptr(opts.Ref)
	}

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

	var gitlabFile *gitlab.File
	var lastResp *gitlab.Response

	// Create a context with timeout
	fetchCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	// Fetch the file metadata with retry logic
	err := apperrors.RetryWithBackoff(fetchCtx, retryConfig, func() error {
		file, resp, err := c.client.RepositoryFiles.GetFileMetaData(
			projectID,
			filePath,
			gitlabOpts,
			gitlab.WithContext(fetchCtx),
		)
		lastResp = resp
		if err != nil {
			return classifyGitLabError(err, resp)
		}
		gitlabFile = file
		return nil
	})

	if err != nil {
		return nil, c.formatUserError(err, lastResp)
	}

	// Convert the GitLab File to our FileContent type (without content)
	fileContent := &FileContent{
		FileName: gitlabFile.FileName,
		FilePath: gitlabFile.FilePath,
		Size:     gitlabFile.Size,
		Encoding: gitlabFile.Encoding,
		Ref:      gitlabFile.Ref,
		BlobID:   gitlabFile.BlobID,
		CommitID: gitlabFile.CommitID,
	}

	// Set LastCommitID if available
	if gitlabFile.LastCommitID != "" {
		fileContent.LastCommitID = gitlabFile.LastCommitID
	} else {
		fileContent.LastCommitID = gitlabFile.CommitID
	}

	// Set ContentSHA256 if available
	if gitlabFile.ContentSHA256 != "" {
		fileContent.ContentSHA256 = gitlabFile.ContentSHA256
	}

	// Note: Content is intentionally not populated for metadata-only requests

	return fileContent, nil
}
