package parsers

import (
	"testing"
)

func TestParseRequirementsTxtDependencies(t *testing.T) {
	tests := []struct {
		name           string
		content        string
		wantFound      bool
		wantVersion    string
		wantDepCount   string
		wantConfidence float64
		wantDep1       string
		wantDep2       string
	}{
		{
			name: "simple dependencies",
			content: `requests
django
flask`,
			wantFound:    true,
			wantDepCount: "3",
			wantDep1:     "requests",
			wantDep2:     "django",
		},
		{
			name: "dependencies with version specifiers",
			content: `requests>=2.28.0
django==4.2.0
flask~=2.3.0`,
			wantFound:    true,
			wantDepCount: "3",
			wantDep1:     "requests>=2.28.0",
			wantDep2:     "django==4.2.0",
		},
		{
			name: "complex version specifiers",
			content: `requests>=2.28.0,<3.0.0
django>=4.0,!=4.1.0,<5.0
flask~=2.3.0`,
			wantFound:    true,
			wantDepCount: "3",
			wantDep1:     "requests>=2.28.0,<3.0.0",
			wantDep2:     "django>=4.0,!=4.1.0,<5.0",
		},
		{
			name: "dependencies with extras",
			content: `requests[security]>=2.28.0
celery[redis,auth]>=5.0.0`,
			wantFound:    true,
			wantDepCount: "2",
			wantDep1:     "requests>=2.28.0",
			wantDep2:     "celery>=5.0.0",
		},
		{
			name: "dependencies with environment markers",
			content: `requests>=2.28.0; python_version>='3.8'
django>=4.0; sys_platform=='linux'`,
			wantFound:    true,
			wantDepCount: "2",
			wantDep1:     "requests>=2.28.0",
			wantDep2:     "django>=4.0",
		},
		{
			name: "dependencies with hashes",
			content: `requests==2.28.0 --hash=sha256:abc123
django==4.2.0 --hash=sha256:def456 --hash=sha256:ghi789`,
			wantFound:    true,
			wantDepCount: "2",
			wantDep1:     "requests==2.28.0",
			wantDep2:     "django==4.2.0",
		},
		{
			name: "dependencies with inline comments",
			content: `requests>=2.28.0  # HTTP library
django>=4.0  # Web framework
flask>=2.3.0  # Micro framework`,
			wantFound:    true,
			wantDepCount: "3",
			wantDep1:     "requests>=2.28.0",
			wantDep2:     "django>=4.0",
		},
		{
			name: "mixed format with comments",
			content: `# Core dependencies
requests>=2.28.0
django>=4.0

# Testing
pytest>=7.0.0
pytest-cov>=4.0.0`,
			wantFound:    true,
			wantDepCount: "4",
			wantDep1:     "requests>=2.28.0",
			wantDep2:     "django>=4.0",
		},
		{
			name: "with python version comment",
			content: `# Python 3.11
requests>=2.28.0
django>=4.2.0`,
			wantFound:      true,
			wantVersion:    "3.11",
			wantDepCount:   "2",
			wantConfidence: 0.6,
			wantDep1:       "requests>=2.28.0",
			wantDep2:       "django>=4.2.0",
		},
		{
			name: "with requires python comment",
			content: `# Requires Python >= 3.11
requests>=2.28.0
django>=4.2.0`,
			wantFound:      true,
			wantVersion:    "3.11",
			wantDepCount:   "2",
			wantConfidence: 0.6,
			wantDep1:       "requests>=2.28.0",
		},
		{
			name: "editable installs",
			content: `-e git+https://github.com/user/repo.git@main#egg=package
-e ./local-package
requests>=2.28.0`,
			wantFound:    true,
			wantDepCount: "3",
			wantDep1:     "git+https://github.com/user/repo.git@main#egg=package",
			wantDep2:     "./local-package",
		},
		{
			name: "with options",
			content: `--index-url https://pypi.org/simple
--extra-index-url https://private.pypi.org
requests>=2.28.0
django>=4.0`,
			wantFound:    true,
			wantDepCount: "2",
			wantDep1:     "requests>=2.28.0",
			wantDep2:     "django>=4.0",
		},
		{
			name: "recursive requirements ignored",
			content: `-r base.txt
-r testing.txt
requests>=2.28.0`,
			wantFound:    true,
			wantDepCount: "1",
			wantDep1:     "requests>=2.28.0",
		},
		{
			name: "empty file",
			content: `

`,
			wantFound: false,
		},
		{
			name: "only comments",
			content: `# This is a comment
# Another comment`,
			wantFound: false,
		},
		{
			name: "only options",
			content: `--index-url https://pypi.org/simple
--extra-index-url https://private.pypi.org`,
			wantFound: false,
		},
		{
			name: "real world example",
			content: `# Production Dependencies
# Requires Python >= 3.11

# Web Framework
django>=4.2.0,<5.0.0
djangorestframework>=3.14.0

# Database
psycopg2-binary>=2.9.0
redis>=4.5.0

# Task Queue
celery[redis]>=5.2.0

# HTTP Client
requests>=2.28.0
httpx>=0.24.0; python_version>='3.8'

# Utilities
python-dotenv>=1.0.0
pydantic>=2.0.0`,
			wantFound:      true,
			wantVersion:    "3.11",
			wantDepCount:   "9",
			wantConfidence: 0.6,
			wantDep1:       "django>=4.2.0,<5.0.0",
			wantDep2:       "djangorestframework>=3.14.0",
		},
		{
			name: "package names with hyphens and underscores",
			content: `python-dateutil>=2.8.0
my_package>=1.0.0
some-other-package>=2.0.0`,
			wantFound:    true,
			wantDepCount: "3",
			wantDep1:     "python-dateutil>=2.8.0",
			wantDep2:     "my_package>=1.0.0",
		},
		{
			name: "case variations",
			content: `Django>=4.2.0
REQUESTS>=2.28.0
flask>=2.3.0`,
			wantFound:    true,
			wantDepCount: "3",
			wantDep1:     "Django>=4.2.0",
			wantDep2:     "REQUESTS>=2.28.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseRequirementsTxtDependencies([]byte(tt.content), "requirements.txt")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.Found != tt.wantFound {
				t.Errorf("Found = %v, want %v", result.Found, tt.wantFound)
			}

			if !result.Found {
				return // No need to check other fields
			}

			if tt.wantVersion != "" && result.Version != tt.wantVersion {
				t.Errorf("Version = %v, want %v", result.Version, tt.wantVersion)
			}

			if tt.wantDepCount != "" {
				if got := result.Metadata["dependency_count"]; got != tt.wantDepCount {
					t.Errorf("dependency_count = %v, want %v", got, tt.wantDepCount)
				}
			}

			if tt.wantConfidence > 0 {
				if result.Confidence != tt.wantConfidence {
					t.Errorf("Confidence = %v, want %v", result.Confidence, tt.wantConfidence)
				}
			}

			if tt.wantDep1 != "" {
				if got := result.Metadata["dependency_1"]; got != tt.wantDep1 {
					t.Errorf("dependency_1 = %v, want %v", got, tt.wantDep1)
				}
			}

			if tt.wantDep2 != "" {
				if got := result.Metadata["dependency_2"]; got != tt.wantDep2 {
					t.Errorf("dependency_2 = %v, want %v", got, tt.wantDep2)
				}
			}
		})
	}
}

func TestParseRequirementLine(t *testing.T) {
	tests := []struct {
		name          string
		line          string
		wantName      string
		wantSpecifier string
		wantExtras    []string
		wantMarkers   string
		wantEditable  bool
		wantError     bool
	}{
		{
			name:      "simple package",
			line:      "requests",
			wantName:  "requests",
			wantError: false,
		},
		{
			name:          "package with version",
			line:          "requests>=2.28.0",
			wantName:      "requests",
			wantSpecifier: ">=2.28.0",
			wantError:     false,
		},
		{
			name:          "package with exact version",
			line:          "django==4.2.0",
			wantName:      "django",
			wantSpecifier: "==4.2.0",
			wantError:     false,
		},
		{
			name:          "package with compatible version",
			line:          "flask~=2.3.0",
			wantName:      "flask",
			wantSpecifier: "~=2.3.0",
			wantError:     false,
		},
		{
			name:          "package with range",
			line:          "requests>=2.0,<3.0",
			wantName:      "requests",
			wantSpecifier: ">=2.0,<3.0",
			wantError:     false,
		},
		{
			name:          "package with extras",
			line:          "requests[security]>=2.28.0",
			wantName:      "requests",
			wantSpecifier: ">=2.28.0",
			wantExtras:    []string{"security"},
			wantError:     false,
		},
		{
			name:          "package with multiple extras",
			line:          "celery[redis,auth]>=5.0.0",
			wantName:      "celery",
			wantSpecifier: ">=5.0.0",
			wantExtras:    []string{"redis", "auth"},
			wantError:     false,
		},
		{
			name:          "package with environment markers",
			line:          "requests>=2.28.0; python_version>='3.8'",
			wantName:      "requests",
			wantSpecifier: ">=2.28.0",
			wantMarkers:   "python_version>='3.8'",
			wantError:     false,
		},
		{
			name:          "package with extras and markers",
			line:          "requests[security]>=2.28.0; python_version>='3.8'",
			wantName:      "requests",
			wantSpecifier: ">=2.28.0",
			wantExtras:    []string{"security"},
			wantMarkers:   "python_version>='3.8'",
			wantError:     false,
		},
		{
			name:         "editable git install",
			line:         "-e git+https://github.com/user/repo.git@main#egg=package",
			wantName:     "git+https://github.com/user/repo.git@main#egg=package",
			wantEditable: true,
			wantError:    false,
		},
		{
			name:         "editable local install",
			line:         "-e ./local-package",
			wantName:     "./local-package",
			wantEditable: true,
			wantError:    false,
		},
		{
			name:      "package with hyphen",
			line:      "python-dateutil>=2.8.0",
			wantName:  "python-dateutil",
			wantSpecifier: ">=2.8.0",
			wantError: false,
		},
		{
			name:          "package with underscore",
			line:          "my_package>=1.0.0",
			wantName:      "my_package",
			wantSpecifier: ">=1.0.0",
			wantError:     false,
		},
		{
			name:      "option line should error",
			line:      "--index-url https://pypi.org/simple",
			wantError: true,
		},
		{
			name:      "empty line should error",
			line:      "",
			wantError: true,
		},
		{
			name:          "inline comment stripped",
			line:          "requests>=2.28.0  # HTTP library",
			wantName:      "requests",
			wantSpecifier: ">=2.28.0",
			wantError:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := parseRequirementLine(tt.line)
			
			if tt.wantError {
				if err == nil && req != nil && !req.IsRequirementFile {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if req == nil {
				t.Fatal("expected requirement but got nil")
			}

			if req.Name != tt.wantName {
				t.Errorf("Name = %v, want %v", req.Name, tt.wantName)
			}

			if req.Specifier != tt.wantSpecifier {
				t.Errorf("Specifier = %v, want %v", req.Specifier, tt.wantSpecifier)
			}

			if tt.wantExtras != nil {
				if len(req.Extras) != len(tt.wantExtras) {
					t.Errorf("Extras length = %v, want %v", len(req.Extras), len(tt.wantExtras))
				} else {
					for i, extra := range tt.wantExtras {
						if req.Extras[i] != extra {
							t.Errorf("Extras[%d] = %v, want %v", i, req.Extras[i], extra)
						}
					}
				}
			}

			if req.Markers != tt.wantMarkers {
				t.Errorf("Markers = %v, want %v", req.Markers, tt.wantMarkers)
			}

			if req.IsEditable != tt.wantEditable {
				t.Errorf("IsEditable = %v, want %v", req.IsEditable, tt.wantEditable)
			}
		})
	}
}

func TestGetRequirementsTxtDependencyRule(t *testing.T) {
	rule := GetRequirementsTxtDependencyRule()
	
	if rule == nil {
		t.Fatal("GetRequirementsTxtDependencyRule returned nil")
	}

	// Check basic rule properties
	if rule.Name != "requirements-txt-dependencies" {
		t.Errorf("Name = %v, want %v", rule.Name, "requirements-txt-dependencies")
	}

	if rule.Priority != 15 {
		t.Errorf("Priority = %v, want %v", rule.Priority, 15)
	}
}

func TestRequirementsTxtDependencyRuleIntegration(t *testing.T) {
	// Test the full rule with various content
	rule := GetRequirementsTxtDependencyRule()

	testCases := []struct {
		name         string
		content      string
		wantFound    bool
		wantDepCount string
	}{
		{
			name: "typical requirements file",
			content: `requests>=2.28.0
django>=4.2.0
celery[redis]>=5.0.0`,
			wantFound:    true,
			wantDepCount: "3",
		},
		{
			name: "empty file",
			content: ``,
			wantFound: false,
		},
		{
			name: "with python version",
			content: `# Python 3.11
requests>=2.28.0`,
			wantFound:    true,
			wantDepCount: "1",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := rule.Parser([]byte(tc.content), "requirements.txt")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.Found != tc.wantFound {
				t.Errorf("Found = %v, want %v", result.Found, tc.wantFound)
			}

			if tc.wantDepCount != "" {
				if got := result.Metadata["dependency_count"]; got != tc.wantDepCount {
					t.Errorf("dependency_count = %v, want %v", got, tc.wantDepCount)
				}
			}
		})
	}
}
