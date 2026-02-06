package gitlab

import (
	"testing"
	"time"
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
