package parsers

import (
	"testing"

	"github.com/gbjohnso/gitlab-python-scanner/internal/rules"
)

// ============================================================================
// .python-version Tests
// ============================================================================

func TestParsePythonVersionFile(t *testing.T) {
	tests := []struct {
		name       string
		content    string
		wantFound  bool
		wantVer    string
		wantConf   float64
	}{
		{
			name:      "simple version",
			content:   "3.11",
			wantFound: true,
			wantVer:   "3.11",
			wantConf:  1.0,
		},
		{
			name:      "full version",
			content:   "3.11.5",
			wantFound: true,
			wantVer:   "3.11.5",
			wantConf:  1.0,
		},
		{
			name:      "with python prefix",
			content:   "python-3.11.5",
			wantFound: true,
			wantVer:   "3.11.5",
			wantConf:  1.0,
		},
		{
			name:      "with whitespace",
			content:   "  3.11.5  \n",
			wantFound: true,
			wantVer:   "3.11.5",
			wantConf:  1.0,
		},
		{
			name:      "Python with capital P",
			content:   "Python-3.11",
			wantFound: true,
			wantVer:   "3.11",
			wantConf:  1.0,
		},
		{
			name:      "py prefix",
			content:   "py3.11",
			wantFound: true,
			wantVer:   "3.11",
			wantConf:  1.0,
		},
		{
			name:      "invalid content",
			content:   "not a version",
			wantFound: false,
		},
		{
			name:      "empty file",
			content:   "",
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParsePythonVersionFile([]byte(tt.content), ".python-version")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.Found != tt.wantFound {
				t.Errorf("Found = %v, want %v", result.Found, tt.wantFound)
			}

			if tt.wantFound {
				if result.Version != tt.wantVer {
					t.Errorf("Version = %v, want %v", result.Version, tt.wantVer)
				}
				if result.Confidence != tt.wantConf {
					t.Errorf("Confidence = %v, want %v", result.Confidence, tt.wantConf)
				}
				if result.Source != ".python-version" {
					t.Errorf("Source = %v, want .python-version", result.Source)
				}
			}
		})
	}
}

func TestGetPythonVersionFileRule(t *testing.T) {
	rule := GetPythonVersionFileRule()
	
	if rule.Name != "python-version-file" {
		t.Errorf("rule name = %v, want python-version-file", rule.Name)
	}
	
	if rule.Priority != 1 {
		t.Errorf("priority = %d, want 1", rule.Priority)
	}
	
	if !rule.Matches(".python-version", "/path/.python-version") {
		t.Error("rule should match .python-version")
	}
}

// ============================================================================
// runtime.txt Tests
// ============================================================================

func TestParseRuntimeTxt(t *testing.T) {
	tests := []struct {
		name       string
		content    string
		wantFound  bool
		wantVer    string
		wantConf   float64
	}{
		{
			name:      "heroku format",
			content:   "python-3.11.5",
			wantFound: true,
			wantVer:   "3.11.5",
			wantConf:  0.95,
		},
		{
			name:      "short version",
			content:   "python-3.11",
			wantFound: true,
			wantVer:   "3.11",
			wantConf:  0.95,
		},
		{
			name:      "with whitespace",
			content:   "  python-3.11  \n",
			wantFound: true,
			wantVer:   "3.11",
			wantConf:  0.95,
		},
		{
			name:      "invalid format",
			content:   "ruby-2.7",
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseRuntimeTxt([]byte(tt.content), "runtime.txt")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.Found != tt.wantFound {
				t.Errorf("Found = %v, want %v", result.Found, tt.wantFound)
			}

			if tt.wantFound {
				if result.Version != tt.wantVer {
					t.Errorf("Version = %v, want %v", result.Version, tt.wantVer)
				}
				if result.Confidence != tt.wantConf {
					t.Errorf("Confidence = %v, want %v", result.Confidence, tt.wantConf)
				}
			}
		})
	}
}

// ============================================================================
// setup.py Tests
// ============================================================================

func TestParseSetupPy(t *testing.T) {
	tests := []struct {
		name       string
		content    string
		wantFound  bool
		wantVer    string
		wantConf   float64
	}{
		{
			name: "simple python_requires",
			content: `setup(
    name="mypackage",
    python_requires='>=3.11'
)`,
			wantFound: true,
			wantVer:   "3.11",
			wantConf:  0.9,
		},
		{
			name: "double quotes",
			content: `setup(
    name="mypackage",
    python_requires=">=3.10,<4.0"
)`,
			wantFound: true,
			wantVer:   "3.10",
			wantConf:  0.9,
		},
		{
			name: "exact version",
			content: `setup(
    name="mypackage",
    python_requires="==3.11.5"
)`,
			wantFound: true,
			wantVer:   "3.11.5",
			wantConf:  0.9,
		},
		{
			name: "no python_requires",
			content: `setup(
    name="mypackage"
)`,
			wantFound: false,
		},
		{
			name:      "invalid format",
			content:   "import setuptools",
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseSetupPy([]byte(tt.content), "setup.py")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.Found != tt.wantFound {
				t.Errorf("Found = %v, want %v", result.Found, tt.wantFound)
			}

			if tt.wantFound && result.Version != tt.wantVer {
				t.Errorf("Version = %v, want %v", result.Version, tt.wantVer)
			}
		})
	}
}

// ============================================================================
// Pipfile Tests
// ============================================================================

func TestParsePipfile(t *testing.T) {
	tests := []struct {
		name       string
		content    string
		wantFound  bool
		wantVer    string
		wantConf   float64
	}{
		{
			name: "python_version",
			content: `[requires]
python_version = "3.11"`,
			wantFound: true,
			wantVer:   "3.11",
			wantConf:  0.9,
		},
		{
			name: "python_full_version",
			content: `[requires]
python_full_version = "3.11.5"`,
			wantFound: true,
			wantVer:   "3.11.5",
			wantConf:  0.9,
		},
		{
			name: "both versions (full takes precedence)",
			content: `[requires]
python_version = "3.10"
python_full_version = "3.11.5"`,
			wantFound: true,
			wantVer:   "3.11.5",
			wantConf:  0.9,
		},
		{
			name: "no requires section",
			content: `[packages]
requests = "*"`,
			wantFound: false,
		},
		{
			name:      "invalid toml",
			content:   "[broken",
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParsePipfile([]byte(tt.content), "Pipfile")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.Found != tt.wantFound {
				t.Errorf("Found = %v, want %v", result.Found, tt.wantFound)
			}

			if tt.wantFound && result.Version != tt.wantVer {
				t.Errorf("Version = %v, want %v", result.Version, tt.wantVer)
			}
		})
	}
}

// ============================================================================
// requirements.txt Tests
// ============================================================================

func TestParseRequirementsTxt(t *testing.T) {
	tests := []struct {
		name       string
		content    string
		wantFound  bool
		wantVer    string
		wantConf   float64
	}{
		{
			name: "python comment",
			content: `# Python 3.11
requests>=2.28.0`,
			wantFound: true,
			wantVer:   "3.11",
			wantConf:  0.6,
		},
		{
			name: "requires python comment",
			content: `# Requires Python >= 3.11
requests>=2.28.0
django>=4.2`,
			wantFound: true,
			wantVer:   "3.11",
			wantConf:  0.6,
		},
		{
			name: "py comment",
			content: `# py >= 3.10
requests>=2.28.0`,
			wantFound: true,
			wantVer:   "3.10",
			wantConf:  0.6,
		},
		{
			name: "full version in comment",
			content: `# Python 3.11.5
requests>=2.28.0`,
			wantFound: true,
			wantVer:   "3.11.5",
			wantConf:  0.6,
		},
		{
			name: "no python version",
			content: `requests>=2.28.0
django>=4.2`,
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseRequirementsTxt([]byte(tt.content), "requirements.txt")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.Found != tt.wantFound {
				t.Errorf("Found = %v, want %v", result.Found, tt.wantFound)
			}

			if tt.wantFound {
				if result.Version != tt.wantVer {
					t.Errorf("Version = %v, want %v", result.Version, tt.wantVer)
				}
				if result.Confidence != tt.wantConf {
					t.Errorf("Confidence = %v, want %v", result.Confidence, tt.wantConf)
				}
			}
		})
	}
}

// ============================================================================
// .gitlab-ci.yml Tests
// ============================================================================

func TestParseGitLabCI(t *testing.T) {
	tests := []struct {
		name       string
		content    string
		wantFound  bool
		wantVer    string
		wantConf   float64
	}{
		{
			name: "python image",
			content: `test:
  image: python:3.11
  script:
    - pytest`,
			wantFound: true,
			wantVer:   "3.11",
			wantConf:  0.75,
		},
		{
			name: "python slim image",
			content: `test:
  image: python:3.11-slim
  script:
    - pytest`,
			wantFound: true,
			wantVer:   "3.11",
			wantConf:  0.75,
		},
		{
			name: "full version",
			content: `test:
  image: python:3.11.5-alpine
  script:
    - pytest`,
			wantFound: true,
			wantVer:   "3.11.5",
			wantConf:  0.75,
		},
		{
			name: "no python image",
			content: `test:
  image: ubuntu:latest
  script:
    - pytest`,
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseGitLabCI([]byte(tt.content), ".gitlab-ci.yml")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.Found != tt.wantFound {
				t.Errorf("Found = %v, want %v", result.Found, tt.wantFound)
			}

			if tt.wantFound && result.Version != tt.wantVer {
				t.Errorf("Version = %v, want %v", result.Version, tt.wantVer)
			}
		})
	}
}

// ============================================================================
// Dockerfile Tests
// ============================================================================

func TestParseDockerfile(t *testing.T) {
	tests := []struct {
		name       string
		content    string
		wantFound  bool
		wantVer    string
		wantConf   float64
	}{
		{
			name: "python base image",
			content: `FROM python:3.11
WORKDIR /app
COPY . .`,
			wantFound: true,
			wantVer:   "3.11",
			wantConf:  0.8,
		},
		{
			name: "python slim image",
			content: `FROM python:3.11-slim
RUN pip install -r requirements.txt`,
			wantFound: true,
			wantVer:   "3.11",
			wantConf:  0.8,
		},
		{
			name: "full version",
			content: `FROM python:3.11.5-alpine
WORKDIR /app`,
			wantFound: true,
			wantVer:   "3.11.5",
			wantConf:  0.8,
		},
		{
			name: "non-python image",
			content: `FROM ubuntu:latest
RUN apt-get install python3`,
			wantFound: false,
		},
		{
			name: "python install but not FROM",
			content: `FROM ubuntu:latest
RUN apt-get install python:3.11`,
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseDockerfile([]byte(tt.content), "Dockerfile")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.Found != tt.wantFound {
				t.Errorf("Found = %v, want %v", result.Found, tt.wantFound)
			}

			if tt.wantFound && result.Version != tt.wantVer {
				t.Errorf("Version = %v, want %v", result.Version, tt.wantVer)
			}
		})
	}
}

// ============================================================================
// tox.ini Tests
// ============================================================================

func TestParseToxIni(t *testing.T) {
	tests := []struct {
		name       string
		content    string
		wantFound  bool
		wantVer    string
		wantConf   float64
	}{
		{
			name: "py311 envlist",
			content: `[tox]
envlist = py311,py312`,
			wantFound: true,
			wantVer:   "3.11",
			wantConf:  0.7,
		},
		{
			name: "py39 envlist",
			content: `[tox]
envlist = py39,py310`,
			wantFound: true,
			wantVer:   "3.9",
			wantConf:  0.7,
		},
		{
			name: "py310 envlist",
			content: `[tox]
envlist = py310`,
			wantFound: true,
			wantVer:   "3.10",
			wantConf:  0.7,
		},
		{
			name: "toml format",
			content: `[tox]
envlist = "py311,py312"`,
			wantFound: true,
			wantVer:   "3.11",
			wantConf:  0.7,
		},
		{
			name: "no envlist",
			content: `[tox]
skipsdist = True`,
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseToxIni([]byte(tt.content), "tox.ini")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.Found != tt.wantFound {
				t.Errorf("Found = %v, want %v", result.Found, tt.wantFound)
			}

			if tt.wantFound && result.Version != tt.wantVer {
				t.Errorf("Version = %v, want %v", result.Version, tt.wantVer)
			}
		})
	}
}

func TestExtractPythonVersionFromToxEnv(t *testing.T) {
	tests := []struct {
		envlist string
		want    string
	}{
		{"py311", "3.11"},
		{"py39", "3.9"},
		{"py310", "3.10"},
		{"py312", "3.12"},
		{"py311,py312", "3.11"}, // Returns first match
		{"flake8,py311", "3.11"},
		{"invalid", ""},
	}

	for _, tt := range tests {
		t.Run(tt.envlist, func(t *testing.T) {
			got := extractPythonVersionFromToxEnv(tt.envlist)
			if got != tt.want {
				t.Errorf("extractPythonVersionFromToxEnv(%q) = %q, want %q", tt.envlist, got, tt.want)
			}
		})
	}
}

// ============================================================================
// Helper Function Tests
// ============================================================================

func TestExtractPythonVersion(t *testing.T) {
	tests := []struct {
		input     string
		want      string
		wantError bool
	}{
		{"3.11", "3.11", false},
		{"3.11.5", "3.11.5", false},
		{"  3.11  ", "3.11", false},
		{"3.9", "3.9", false},
		{"invalid", "", true},
		{"", "", true},
		{"python", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := extractPythonVersion(tt.input)
			
			if tt.wantError {
				if err == nil {
					t.Errorf("expected error for input %q", tt.input)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if got != tt.want {
					t.Errorf("extractPythonVersion(%q) = %q, want %q", tt.input, got, tt.want)
				}
			}
		})
	}
}

// ============================================================================
// Rule Tests
// ============================================================================

func TestAllRulesAreValid(t *testing.T) {
	rules := []struct {
		name    string
		getRule func() *rules.SearchRule
	}{
		{"python-version-file", GetPythonVersionFileRule},
		{"runtime-txt", GetRuntimeTxtRule},
		{"setup-py", GetSetupPyRule},
		{"pipfile", GetPipfileRule},
		{"requirements-txt", GetRequirementsTxtRule},
		{"gitlab-ci", GetGitLabCIRule},
		{"dockerfile", GetDockerfileRule},
		{"tox-ini", GetToxIniRule},
	}

	for _, tt := range rules {
		t.Run(tt.name, func(t *testing.T) {
			rule := tt.getRule()
			
			if err := rule.Validate(); err != nil {
				t.Errorf("rule validation failed: %v", err)
			}
			
			if !rule.Enabled {
				t.Error("rule should be enabled")
			}
			
			if rule.Parser == nil {
				t.Error("rule should have a parser")
			}
		})
	}
}

func TestRulePriorities(t *testing.T) {
	// Verify rules are in the correct priority order
	priorities := map[string]int{
		"python-version-file": 1,
		"runtime-txt":         2,
		"setup-py":            8,
		"pipfile":             9,
		"pyproject-toml":      10,
		"dockerfile":          11,
		"gitlab-ci":           12,
		"tox-ini":             13,
		"requirements-txt":    15,
	}

	rules := map[string]func() *rules.SearchRule{
		"python-version-file": GetPythonVersionFileRule,
		"runtime-txt":         GetRuntimeTxtRule,
		"setup-py":            GetSetupPyRule,
		"pipfile":             GetPipfileRule,
		"pyproject-toml":      GetPyprojectTomlRule,
		"dockerfile":          GetDockerfileRule,
		"gitlab-ci":           GetGitLabCIRule,
		"tox-ini":             GetToxIniRule,
		"requirements-txt":    GetRequirementsTxtRule,
	}

	for name, getRule := range rules {
		t.Run(name, func(t *testing.T) {
			rule := getRule()
			expectedPriority := priorities[name]
			
			if rule.Priority != expectedPriority {
				t.Errorf("priority = %d, want %d", rule.Priority, expectedPriority)
			}
		})
	}
}
