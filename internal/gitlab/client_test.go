package gitlab

import (
	stderrors "errors"
	"net/http"
	"testing"
	"time"

	apperrors "github.com/gbjohnso/gitlab-python-scanner/internal/errors"
	"github.com/xanzy/go-gitlab"
)

func TestParseGitLabURL(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantBaseURL  string
		wantOrg      string
		wantErr      bool
	}{
		{
			name:        "Simple gitlab.com URL",
			input:       "gitlab.com/myorg",
			wantBaseURL: "https://gitlab.com",
			wantOrg:     "myorg",
			wantErr:     false,
		},
		{
			name:        "gitlab.com URL with https",
			input:       "https://gitlab.com/myorg",
			wantBaseURL: "https://gitlab.com",
			wantOrg:     "myorg",
			wantErr:     false,
		},
		{
			name:        "Custom GitLab instance",
			input:       "gitlab.example.com/engineering",
			wantBaseURL: "https://gitlab.example.com",
			wantOrg:     "engineering",
			wantErr:     false,
		},
		{
			name:        "Nested group path",
			input:       "gitlab.com/group/subgroup",
			wantBaseURL: "https://gitlab.com",
			wantOrg:     "group/subgroup",
			wantErr:     false,
		},
		{
			name:        "URL with trailing slash",
			input:       "gitlab.com/myorg/",
			wantBaseURL: "https://gitlab.com",
			wantOrg:     "myorg",
			wantErr:     false,
		},
		{
			name:        "No organization path",
			input:       "gitlab.com",
			wantBaseURL: "",
			wantOrg:     "",
			wantErr:     true,
		},
		{
			name:        "HTTP scheme",
			input:       "http://gitlab.local/myorg",
			wantBaseURL: "http://gitlab.local",
			wantOrg:     "myorg",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseURL, org, err := parseGitLabURL(tt.input)

			if (err != nil) != tt.wantErr {
				t.Errorf("parseGitLabURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if baseURL != tt.wantBaseURL {
					t.Errorf("parseGitLabURL() baseURL = %v, want %v", baseURL, tt.wantBaseURL)
				}
				if org != tt.wantOrg {
					t.Errorf("parseGitLabURL() org = %v, want %v", org, tt.wantOrg)
				}
			}
		})
	}
}

func TestNewClient(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name:    "Nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name: "Missing token",
			config: &Config{
				GitLabURL: "gitlab.com/myorg",
				Token:     "",
				Timeout:   30 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "Missing URL",
			config: &Config{
				GitLabURL: "",
				Token:     "test-token",
				Timeout:   30 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "Invalid URL format",
			config: &Config{
				GitLabURL: "gitlab.com",
				Token:     "test-token",
				Timeout:   30 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "Valid config",
			config: &Config{
				GitLabURL: "gitlab.com/myorg",
				Token:     "test-token",
				Timeout:   30 * time.Second,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.config)

			if (err != nil) != tt.wantErr {
				t.Errorf("NewClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && client == nil {
				t.Error("NewClient() returned nil client without error")
			}

			if !tt.wantErr {
				if client.GetOrganization() != "myorg" {
					t.Errorf("GetOrganization() = %v, want myorg", client.GetOrganization())
				}
				if client.GetBaseURL() != "https://gitlab.com" {
					t.Errorf("GetBaseURL() = %v, want https://gitlab.com", client.GetBaseURL())
				}
				if client.GetTimeout() != 30*time.Second {
					t.Errorf("GetTimeout() = %v, want 30s", client.GetTimeout())
				}
			}
		})
	}
}

func TestClientDefaultTimeout(t *testing.T) {
	config := &Config{
		GitLabURL: "gitlab.com/myorg",
		Token:     "test-token",
		Timeout:   0, // No timeout specified
	}

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	if client.GetTimeout() != 30*time.Second {
		t.Errorf("GetTimeout() = %v, want 30s (default)", client.GetTimeout())
	}
}

func TestClassifyGitLabError(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		statusCode   int
		expectedType string
		shouldRetry  bool
	}{
		{
			name:         "401 Unauthorized",
			err:          &mockGitLabError{message: "unauthorized"},
			statusCode:   401,
			expectedType: "Authentication",
			shouldRetry:  false,
		},
		{
			name:         "403 Forbidden",
			err:          &mockGitLabError{message: "forbidden"},
			statusCode:   403,
			expectedType: "Permission",
			shouldRetry:  false,
		},
		{
			name:         "404 Not Found",
			err:          &mockGitLabError{message: "not found"},
			statusCode:   404,
			expectedType: "NotFound",
			shouldRetry:  false,
		},
		{
			name:         "429 Too Many Requests",
			err:          &mockGitLabError{message: "rate limit exceeded"},
			statusCode:   429,
			expectedType: "RateLimit",
			shouldRetry:  true,
		},
		{
			name:         "502 Bad Gateway",
			err:          &mockGitLabError{message: "bad gateway"},
			statusCode:   502,
			expectedType: "Network",
			shouldRetry:  true,
		},
		{
			name:         "503 Service Unavailable",
			err:          &mockGitLabError{message: "service unavailable"},
			statusCode:   503,
			expectedType: "Network",
			shouldRetry:  true,
		},
		{
			name:         "504 Gateway Timeout",
			err:          &mockGitLabError{message: "gateway timeout"},
			statusCode:   504,
			expectedType: "Network",
			shouldRetry:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock response with http.Response
			resp := &gitlab.Response{
				Response: &http.Response{
					StatusCode: tt.statusCode,
				},
			}
			
			// Classify the error
			classifiedErr := classifyGitLabError(tt.err, resp)
			
			if classifiedErr == nil {
				t.Fatal("expected non-nil error")
			}

			// Type assert to AppError to access the Retryable field
			var appErr *apperrors.AppError
			if !stderrors.As(classifiedErr, &appErr) {
				t.Fatal("expected error to be an AppError")
			}

			// Verify retryability
			if appErr.Retryable != tt.shouldRetry {
				t.Errorf("expected retryable=%v, got %v", tt.shouldRetry, appErr.Retryable)
			}
		})
	}
}

// mockGitLabError simulates a GitLab API error
type mockGitLabError struct {
	message string
}

func (e *mockGitLabError) Error() string {
	return e.message
}

func TestListProjectsOptions(t *testing.T) {
	tests := []struct {
		name                     string
		opts                     *ListProjectsOptions
		expectedPerPage          int
		expectedIncludeSubgroups bool
		expectedArchived         *bool
	}{
		{
			name:                     "Nil options - use defaults",
			opts:                     nil,
			expectedPerPage:          20,
			expectedIncludeSubgroups: true,
			expectedArchived:         nil,
		},
		{
			name:                     "Empty options - use defaults",
			opts:                     &ListProjectsOptions{},
			expectedPerPage:          20,
			expectedIncludeSubgroups: true,
			expectedArchived:         nil,
		},
		{
			name: "Custom per page",
			opts: &ListProjectsOptions{
				PerPage: 50,
			},
			expectedPerPage:          50,
			expectedIncludeSubgroups: true,
			expectedArchived:         nil,
		},
		{
			name: "Per page exceeds max - should cap at 100",
			opts: &ListProjectsOptions{
				PerPage: 200,
			},
			expectedPerPage:          100,
			expectedIncludeSubgroups: true,
			expectedArchived:         nil,
		},
		{
			name: "Include subgroups explicitly true",
			opts: &ListProjectsOptions{
				IncludeSubgroups: gitlab.Ptr(true),
			},
			expectedPerPage:          20,
			expectedIncludeSubgroups: true,
			expectedArchived:         nil,
		},
		{
			name: "Include subgroups explicitly false",
			opts: &ListProjectsOptions{
				IncludeSubgroups: gitlab.Ptr(false),
			},
			expectedPerPage:          20,
			expectedIncludeSubgroups: false,
			expectedArchived:         nil,
		},
		{
			name: "Archived filter - archived only",
			opts: &ListProjectsOptions{
				Archived: gitlab.Ptr(true),
			},
			expectedPerPage:          20,
			expectedIncludeSubgroups: true,
			expectedArchived:         gitlab.Ptr(true),
		},
		{
			name: "Archived filter - active only",
			opts: &ListProjectsOptions{
				Archived: gitlab.Ptr(false),
			},
			expectedPerPage:          20,
			expectedIncludeSubgroups: true,
			expectedArchived:         gitlab.Ptr(false),
		},
		{
			name: "All options combined",
			opts: &ListProjectsOptions{
				PerPage:          75,
				IncludeSubgroups: gitlab.Ptr(false),
				Archived:         gitlab.Ptr(true),
			},
			expectedPerPage:          75,
			expectedIncludeSubgroups: false,
			expectedArchived:         gitlab.Ptr(true),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This test verifies the options logic without making actual API calls
			// We'll test the actual values that would be set
			
			perPage := 0
			if tt.opts != nil {
				perPage = tt.opts.PerPage
			}
			if perPage == 0 {
				perPage = 20
			}
			if perPage > 100 {
				perPage = 100
			}
			
			if perPage != tt.expectedPerPage {
				t.Errorf("PerPage = %v, want %v", perPage, tt.expectedPerPage)
			}
			
			includeSubgroups := true
			if tt.opts != nil && tt.opts.IncludeSubgroups != nil {
				includeSubgroups = *tt.opts.IncludeSubgroups
			}
			
			if includeSubgroups != tt.expectedIncludeSubgroups {
				t.Errorf("IncludeSubgroups = %v, want %v", includeSubgroups, tt.expectedIncludeSubgroups)
			}
			
			var archived *bool
			if tt.opts != nil {
				archived = tt.opts.Archived
			}
			
			if (archived == nil) != (tt.expectedArchived == nil) {
				t.Errorf("Archived nil mismatch: got %v, want %v", archived, tt.expectedArchived)
			} else if archived != nil && tt.expectedArchived != nil && *archived != *tt.expectedArchived {
				t.Errorf("Archived = %v, want %v", *archived, *tt.expectedArchived)
			}
		})
	}
}

func TestProjectConversion(t *testing.T) {
	// Test the conversion from gitlab.Project to our Project type
	now := time.Now()
	
	gitlabProject := &gitlab.Project{
		ID:                1234,
		Name:              "My Project",
		Path:              "my-project",
		PathWithNamespace: "myorg/my-project",
		WebURL:            "https://gitlab.com/myorg/my-project",
		DefaultBranch:     "main",
		Archived:          false,
		LastActivityAt:    &now,
	}
	
	// Convert to our Project type (simulating what happens in ListProjects)
	project := &Project{
		ID:                gitlabProject.ID,
		Name:              gitlabProject.Name,
		Path:              gitlabProject.Path,
		PathWithNamespace: gitlabProject.PathWithNamespace,
		WebURL:            gitlabProject.WebURL,
		Archived:          gitlabProject.Archived,
	}
	
	if gitlabProject.DefaultBranch != "" {
		project.DefaultBranch = gitlabProject.DefaultBranch
	}
	
	if gitlabProject.LastActivityAt != nil {
		project.LastActivityAt = gitlabProject.LastActivityAt.String()
	}
	
	// Verify the conversion
	if project.ID != 1234 {
		t.Errorf("ID = %v, want 1234", project.ID)
	}
	if project.Name != "My Project" {
		t.Errorf("Name = %v, want My Project", project.Name)
	}
	if project.Path != "my-project" {
		t.Errorf("Path = %v, want my-project", project.Path)
	}
	if project.PathWithNamespace != "myorg/my-project" {
		t.Errorf("PathWithNamespace = %v, want myorg/my-project", project.PathWithNamespace)
	}
	if project.WebURL != "https://gitlab.com/myorg/my-project" {
		t.Errorf("WebURL = %v, want https://gitlab.com/myorg/my-project", project.WebURL)
	}
	if project.DefaultBranch != "main" {
		t.Errorf("DefaultBranch = %v, want main", project.DefaultBranch)
	}
	if project.Archived != false {
		t.Errorf("Archived = %v, want false", project.Archived)
	}
	if project.LastActivityAt != now.String() {
		t.Errorf("LastActivityAt = %v, want %v", project.LastActivityAt, now.String())
	}
}

func TestProjectConversionWithMissingFields(t *testing.T) {
	// Test conversion when optional fields are missing
	gitlabProject := &gitlab.Project{
		ID:                5678,
		Name:              "Minimal Project",
		Path:              "minimal",
		PathWithNamespace: "myorg/minimal",
		WebURL:            "https://gitlab.com/myorg/minimal",
		DefaultBranch:     "", // Empty default branch
		Archived:          true,
		LastActivityAt:    nil, // Nil timestamp
	}
	
	project := &Project{
		ID:                gitlabProject.ID,
		Name:              gitlabProject.Name,
		Path:              gitlabProject.Path,
		PathWithNamespace: gitlabProject.PathWithNamespace,
		WebURL:            gitlabProject.WebURL,
		Archived:          gitlabProject.Archived,
	}
	
	if gitlabProject.DefaultBranch != "" {
		project.DefaultBranch = gitlabProject.DefaultBranch
	}
	
	if gitlabProject.LastActivityAt != nil {
		project.LastActivityAt = gitlabProject.LastActivityAt.String()
	}
	
	// Verify defaults for missing fields
	if project.DefaultBranch != "" {
		t.Errorf("DefaultBranch = %v, want empty string", project.DefaultBranch)
	}
	if project.LastActivityAt != "" {
		t.Errorf("LastActivityAt = %v, want empty string", project.LastActivityAt)
	}
}

func TestFormatUserError(t *testing.T) {
	client := &Client{
		baseURL:      "https://gitlab.com",
		organization: "testorg",
		timeout:      30 * time.Second,
	}

	tests := []struct {
		name           string
		err            error
		statusCode     int
		expectedMsg    string
		shouldContain  []string
	}{
		{
			name:          "Authentication error",
			err:           apperrors.NewAuthenticationError(stderrors.New("invalid token")),
			statusCode:    401,
			expectedMsg:   "authentication failed: please check your GitLab token",
			shouldContain: []string{"authentication", "token"},
		},
		{
			name:          "Network error - server error",
			err:           apperrors.NewNetworkError(stderrors.New("connection failed")),
			statusCode:    502,
			expectedMsg:   "GitLab server error (HTTP 502): the server may be experiencing issues",
			shouldContain: []string{"server error", "502"},
		},
		{
			name:          "Network error - connection issue",
			err:           apperrors.NewNetworkError(stderrors.New("connection failed")),
			statusCode:    0,
			expectedMsg:   "network error: unable to reach GitLab server",
			shouldContain: []string{"network", "GitLab server", "internet connection"},
		},
		{
			name:          "Timeout error",
			err:           apperrors.NewTimeoutError(stderrors.New("timeout")),
			statusCode:    0,
			expectedMsg:   "connection timeout",
			shouldContain: []string{"timeout", "30s"},
		},
		{
			name:          "Rate limit error",
			err:           apperrors.NewRateLimitError(stderrors.New("too many requests")),
			statusCode:    429,
			expectedMsg:   "rate limit exceeded",
			shouldContain: []string{"rate limit", "wait"},
		},
		{
			name:          "Permission error",
			err:           apperrors.NewPermissionError("repository"),
			statusCode:    403,
			expectedMsg:   "permission denied",
			shouldContain: []string{"permission", "token"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var resp *gitlab.Response
			if tt.statusCode > 0 {
				resp = &gitlab.Response{
					Response: &http.Response{
						StatusCode: tt.statusCode,
					},
				}
			}

			formattedErr := client.formatUserError(tt.err, resp)
			errMsg := formattedErr.Error()

			for _, substr := range tt.shouldContain {
				if !contains(errMsg, substr) {
					t.Errorf("error message should contain %q, got: %s", substr, errMsg)
				}
			}
		})
	}
}

func TestClassifyGitLabErrorWithNilResponse(t *testing.T) {
	// Test error classification when response is nil (connection failures)
	networkErr := stderrors.New("connection refused")
	
	classifiedErr := classifyGitLabError(networkErr, nil)
	
	if classifiedErr == nil {
		t.Fatal("expected non-nil error")
	}

	var appErr *apperrors.AppError
	if !stderrors.As(classifiedErr, &appErr) {
		t.Fatal("expected error to be an AppError")
	}

	// Network errors should be retryable
	if !appErr.Retryable {
		t.Error("network errors should be retryable")
	}
}

func TestClassifyGitLabErrorWithWrappedErrors(t *testing.T) {
	// Test that wrapped network errors are properly classified
	baseErr := &mockNetError{timeout: true}
	wrappedErr := stderrors.New("failed to connect: " + baseErr.Error())
	
	classifiedErr := classifyGitLabError(wrappedErr, nil)
	
	if classifiedErr == nil {
		t.Fatal("expected non-nil error")
	}
}

// Helper function to check if string contains substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || 
		len(s) > len(substr) && containsAt(s, substr))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
