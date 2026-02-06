package main

import (
	"flag"
	"os"
	"testing"
)

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "Valid config with all required fields",
			config: &Config{
				GitLabURL:   "gitlab.com/myorg",
				Token:       "test-token",
				LogFile:     "",
				Concurrency: 5,
				Timeout:     30,
			},
			wantErr: false,
		},
		{
			name: "Valid config with log file",
			config: &Config{
				GitLabURL:   "gitlab.com/myorg",
				Token:       "test-token",
				LogFile:     "results.log",
				Concurrency: 10,
				Timeout:     60,
			},
			wantErr: false,
		},
		{
			name: "Missing GitLab URL",
			config: &Config{
				GitLabURL:   "",
				Token:       "test-token",
				LogFile:     "",
				Concurrency: 5,
				Timeout:     30,
			},
			wantErr: true,
			errMsg:  "--url is required",
		},
		{
			name: "Missing token",
			config: &Config{
				GitLabURL:   "gitlab.com/myorg",
				Token:       "",
				LogFile:     "",
				Concurrency: 5,
				Timeout:     30,
			},
			wantErr: true,
			errMsg:  "--token is required (or set GITLAB_TOKEN environment variable)",
		},
		{
			name: "Missing both URL and token",
			config: &Config{
				GitLabURL:   "",
				Token:       "",
				LogFile:     "",
				Concurrency: 5,
				Timeout:     30,
			},
			wantErr: true,
			errMsg:  "--url is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfig(tt.config)

			if (err != nil) != tt.wantErr {
				t.Errorf("validateConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && err != nil && err.Error() != tt.errMsg {
				t.Errorf("validateConfig() error message = %v, want %v", err.Error(), tt.errMsg)
			}
		})
	}
}

func TestParseFlags(t *testing.T) {
	// Save original args and flags for cleanup
	oldArgs := os.Args
	oldCommandLine := flag.CommandLine
	defer func() {
		os.Args = oldArgs
		flag.CommandLine = oldCommandLine
	}()

	tests := []struct {
		name        string
		args        []string
		envToken    string
		wantURL     string
		wantToken   string
		wantLog     string
		wantConc    int
		wantTimeout int
	}{
		{
			name:        "All flags provided",
			args:        []string{"cmd", "--url", "gitlab.com/myorg", "--token", "abc123", "--log", "results.log", "--concurrency", "10", "--timeout", "60"},
			envToken:    "",
			wantURL:     "gitlab.com/myorg",
			wantToken:   "abc123",
			wantLog:     "results.log",
			wantConc:    10,
			wantTimeout: 60,
		},
		{
			name:        "Token from environment",
			args:        []string{"cmd", "--url", "gitlab.com/myorg"},
			envToken:    "env-token-123",
			wantURL:     "gitlab.com/myorg",
			wantToken:   "env-token-123",
			wantLog:     "",
			wantConc:    5,
			wantTimeout: 30,
		},
		{
			name:        "Default values",
			args:        []string{"cmd", "--url", "gitlab.com/test", "--token", "test-token"},
			envToken:    "",
			wantURL:     "gitlab.com/test",
			wantToken:   "test-token",
			wantLog:     "",
			wantConc:    5,
			wantTimeout: 30,
		},
		{
			name:        "Custom concurrency and timeout",
			args:        []string{"cmd", "--url", "gitlab.example.com/eng", "--token", "token123", "--concurrency", "20", "--timeout", "120"},
			envToken:    "",
			wantURL:     "gitlab.example.com/eng",
			wantToken:   "token123",
			wantLog:     "",
			wantConc:    20,
			wantTimeout: 120,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset flag.CommandLine for each test
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

			// Set environment variable if specified
			if tt.envToken != "" {
				os.Setenv("GITLAB_TOKEN", tt.envToken)
				defer os.Unsetenv("GITLAB_TOKEN")
			} else {
				os.Unsetenv("GITLAB_TOKEN")
			}

			// Set command line args
			os.Args = tt.args

			// Parse flags (skip program name in args)
			config := parseScanFlags(tt.args[1:])

			// Verify results
			if config.GitLabURL != tt.wantURL {
				t.Errorf("GitLabURL = %v, want %v", config.GitLabURL, tt.wantURL)
			}
			if config.Token != tt.wantToken {
				t.Errorf("Token = %v, want %v", config.Token, tt.wantToken)
			}
			if config.LogFile != tt.wantLog {
				t.Errorf("LogFile = %v, want %v", config.LogFile, tt.wantLog)
			}
			if config.Concurrency != tt.wantConc {
				t.Errorf("Concurrency = %v, want %v", config.Concurrency, tt.wantConc)
			}
			if config.Timeout != tt.wantTimeout {
				t.Errorf("Timeout = %v, want %v", config.Timeout, tt.wantTimeout)
			}
		})
	}
}

func TestConfigStruct(t *testing.T) {
	// Test that Config struct can be created and fields are accessible
	config := &Config{
		GitLabURL:   "gitlab.com/test",
		Token:       "test-token",
		LogFile:     "output.log",
		Concurrency: 8,
		Timeout:     45,
	}

	if config.GitLabURL != "gitlab.com/test" {
		t.Errorf("GitLabURL = %v, want gitlab.com/test", config.GitLabURL)
	}
	if config.Token != "test-token" {
		t.Errorf("Token = %v, want test-token", config.Token)
	}
	if config.LogFile != "output.log" {
		t.Errorf("LogFile = %v, want output.log", config.LogFile)
	}
	if config.Concurrency != 8 {
		t.Errorf("Concurrency = %v, want 8", config.Concurrency)
	}
	if config.Timeout != 45 {
		t.Errorf("Timeout = %v, want 45", config.Timeout)
	}
}

func TestParseFlagsTokenPriority(t *testing.T) {
	// Test that command line token takes priority over environment variable
	oldArgs := os.Args
	oldCommandLine := flag.CommandLine
	defer func() {
		os.Args = oldArgs
		flag.CommandLine = oldCommandLine
	}()

	// Reset flag.CommandLine
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

	// Set environment variable
	os.Setenv("GITLAB_TOKEN", "env-token")
	defer os.Unsetenv("GITLAB_TOKEN")

	// Set command line args with explicit token
	os.Args = []string{"cmd", "--url", "gitlab.com/test", "--token", "cli-token"}

	config := parseScanFlags(os.Args[1:])

	// CLI token should take priority
	if config.Token != "cli-token" {
		t.Errorf("Token = %v, want cli-token (CLI should override env)", config.Token)
	}
}

func TestParseSearchFlags(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		wantSearch    string
		wantRegex     bool
		wantFiles     int
		wantCaseSens  bool
		wantContext   int
	}{
		{
			name:       "basic search",
			args:       []string{"--url", "gitlab.com/org", "--token", "abc", "--search", "API_KEY"},
			wantSearch: "API_KEY",
		},
		{
			name:       "regex search with file patterns",
			args:       []string{"--url", "gitlab.com/org", "--token", "abc", "--search", "password\\s*=", "--regex", "--file", "*.py", "--file", "*.yml"},
			wantSearch: "password\\s*=",
			wantRegex:  true,
			wantFiles:  2,
		},
		{
			name:         "case sensitive with context",
			args:         []string{"--url", "gitlab.com/org", "--token", "abc", "--search", "TODO", "--case-sensitive", "--context", "3"},
			wantSearch:   "TODO",
			wantCaseSens: true,
			wantContext:  3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := parseSearchFlags(tt.args)

			if config.SearchTerm != tt.wantSearch {
				t.Errorf("SearchTerm = %q, want %q", config.SearchTerm, tt.wantSearch)
			}
			if config.IsRegex != tt.wantRegex {
				t.Errorf("IsRegex = %v, want %v", config.IsRegex, tt.wantRegex)
			}
			if len(config.FilePatterns) != tt.wantFiles {
				t.Errorf("FilePatterns count = %d, want %d", len(config.FilePatterns), tt.wantFiles)
			}
			if config.CaseSensitive != tt.wantCaseSens {
				t.Errorf("CaseSensitive = %v, want %v", config.CaseSensitive, tt.wantCaseSens)
			}
			if config.ContextLines != tt.wantContext {
				t.Errorf("ContextLines = %d, want %d", config.ContextLines, tt.wantContext)
			}
		})
	}
}

func TestValidateSearchConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *SearchConfig
		wantErr bool
	}{
		{
			name:    "valid with search term",
			config:  &SearchConfig{GitLabURL: "gitlab.com/org", Token: "tok", SearchTerm: "test"},
			wantErr: false,
		},
		{
			name:    "valid with config file",
			config:  &SearchConfig{GitLabURL: "gitlab.com/org", Token: "tok", ConfigFile: "config.yaml"},
			wantErr: false,
		},
		{
			name:    "missing url",
			config:  &SearchConfig{Token: "tok", SearchTerm: "test"},
			wantErr: true,
		},
		{
			name:    "missing token",
			config:  &SearchConfig{GitLabURL: "gitlab.com/org", SearchTerm: "test"},
			wantErr: true,
		},
		{
			name:    "missing search and config",
			config:  &SearchConfig{GitLabURL: "gitlab.com/org", Token: "tok"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSearchConfig(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateSearchConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
