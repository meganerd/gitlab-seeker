package parsers

import (
	"testing"
)

func TestParsePyprojectToml_Poetry(t *testing.T) {
	tests := []struct {
		name           string
		content        string
		wantFound      bool
		wantVersion    string
		wantConfidence float64
		wantFormat     string
	}{
		{
			name: "Poetry with caret constraint",
			content: `[tool.poetry]
name = "my-project"

[tool.poetry.dependencies]
python = "^3.11"
requests = "^2.28.0"
`,
			wantFound:      true,
			wantVersion:    "3.11",
			wantConfidence: 0.9,
			wantFormat:     "Poetry",
		},
		{
			name: "Poetry with greater-than constraint",
			content: `[tool.poetry.dependencies]
python = ">=3.10"
django = "^4.2"
`,
			wantFound:      true,
			wantVersion:    "3.10",
			wantConfidence: 0.9,
			wantFormat:     "Poetry",
		},
		{
			name: "Poetry with exact version",
			content: `[tool.poetry.dependencies]
python = "==3.11.5"
flask = "^2.3.0"
`,
			wantFound:      true,
			wantVersion:    "3.11.5",
			wantConfidence: 0.9,
			wantFormat:     "Poetry",
		},
		{
			name: "Poetry with compatible release",
			content: `[tool.poetry.dependencies]
python = "~=3.11.0"
numpy = "^1.24"
`,
			wantFound:      true,
			wantVersion:    "3.11.0",
			wantConfidence: 0.9,
			wantFormat:     "Poetry",
		},
		{
			name: "Poetry with range constraint",
			content: `[tool.poetry.dependencies]
python = ">=3.10,<3.12"
pandas = "^2.0"
`,
			wantFound:      true,
			wantVersion:    "3.10",
			wantConfidence: 0.9,
			wantFormat:     "Poetry",
		},
		{
			name: "Poetry with wildcard",
			content: `[tool.poetry.dependencies]
python = "3.11.*"
pytest = "^7.4"
`,
			wantFound:      true,
			wantVersion:    "3.11",
			wantConfidence: 0.9,
			wantFormat:     "Poetry",
		},
		{
			name: "Poetry with dependency groups",
			content: `[tool.poetry.dependencies]
python = "^3.12"
fastapi = "^0.104.0"

[tool.poetry.group.dev.dependencies]
pytest = "^7.4.0"
black = "^23.0.0"
`,
			wantFound:      true,
			wantVersion:    "3.12",
			wantConfidence: 0.9,
			wantFormat:     "Poetry",
		},
		{
			name: "Poetry without python dependency",
			content: `[tool.poetry.dependencies]
requests = "^2.28.0"
django = "^4.2"
`,
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParsePyprojectToml([]byte(tt.content), "pyproject.toml")
			if err != nil {
				t.Fatalf("ParsePyprojectToml() error = %v", err)
			}

			if result.Found != tt.wantFound {
				t.Errorf("Found = %v, want %v", result.Found, tt.wantFound)
			}

			if !tt.wantFound {
				return // No further checks needed
			}

			if result.Version != tt.wantVersion {
				t.Errorf("Version = %v, want %v", result.Version, tt.wantVersion)
			}

			if result.Confidence != tt.wantConfidence {
				t.Errorf("Confidence = %v, want %v", result.Confidence, tt.wantConfidence)
			}

			if result.Source != "pyproject.toml" {
				t.Errorf("Source = %v, want pyproject.toml", result.Source)
			}

			if format, ok := result.Metadata["format"]; !ok || format != tt.wantFormat {
				t.Errorf("Metadata[format] = %v, want %v", format, tt.wantFormat)
			}

			if result.RawValue == "" {
				t.Error("RawValue should not be empty")
			}
		})
	}
}

func TestParsePyprojectToml_PEP621(t *testing.T) {
	tests := []struct {
		name           string
		content        string
		wantFound      bool
		wantVersion    string
		wantConfidence float64
		wantFormat     string
	}{
		{
			name: "PEP 621 with requires-python",
			content: `[project]
name = "my-project"
requires-python = ">=3.11"
dependencies = [
    "requests>=2.28.0",
    "django>=4.2",
]
`,
			wantFound:      true,
			wantVersion:    "3.11",
			wantConfidence: 0.9,
			wantFormat:     "PEP621",
		},
		{
			name: "PEP 621 with exact version",
			content: `[project]
name = "example"
requires-python = "==3.10.8"
`,
			wantFound:      true,
			wantVersion:    "3.10.8",
			wantConfidence: 0.9,
			wantFormat:     "PEP621",
		},
		{
			name: "PEP 621 with compatible release",
			content: `[project]
name = "test-project"
requires-python = "~=3.11.0"
dependencies = ["flask"]
`,
			wantFound:      true,
			wantVersion:    "3.11.0",
			wantConfidence: 0.9,
			wantFormat:     "PEP621",
		},
		{
			name: "PEP 621 with range",
			content: `[project]
requires-python = ">=3.9,<3.13"
`,
			wantFound:      true,
			wantVersion:    "3.9",
			wantConfidence: 0.9,
			wantFormat:     "PEP621",
		},
		{
			name: "PEP 621 without requires-python",
			content: `[project]
name = "my-project"
dependencies = ["requests"]
`,
			wantFound: false,
		},
		{
			name: "PEP 621 with optional dependencies",
			content: `[project]
name = "my-lib"
requires-python = ">=3.10"
dependencies = ["click"]

[project.optional-dependencies]
dev = ["pytest", "black"]
docs = ["sphinx"]
`,
			wantFound:      true,
			wantVersion:    "3.10",
			wantConfidence: 0.9,
			wantFormat:     "PEP621",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParsePyprojectToml([]byte(tt.content), "pyproject.toml")
			if err != nil {
				t.Fatalf("ParsePyprojectToml() error = %v", err)
			}

			if result.Found != tt.wantFound {
				t.Errorf("Found = %v, want %v", result.Found, tt.wantFound)
			}

			if !tt.wantFound {
				return
			}

			if result.Version != tt.wantVersion {
				t.Errorf("Version = %v, want %v", result.Version, tt.wantVersion)
			}

			if result.Confidence != tt.wantConfidence {
				t.Errorf("Confidence = %v, want %v", result.Confidence, tt.wantConfidence)
			}

			if format, ok := result.Metadata["format"]; !ok || format != tt.wantFormat {
				t.Errorf("Metadata[format] = %v, want %v", format, tt.wantFormat)
			}
		})
	}
}

func TestParsePyprojectToml_PDM(t *testing.T) {
	// PDM uses PEP 621 format, so it's essentially the same as PEP621 tests
	content := `[project]
name = "pdm-project"
requires-python = ">=3.11"
dependencies = [
    "requests>=2.28.0",
]

[tool.pdm.dev-dependencies]
test = ["pytest>=7.0"]
lint = ["black", "ruff"]
`

	result, err := ParsePyprojectToml([]byte(content), "pyproject.toml")
	if err != nil {
		t.Fatalf("ParsePyprojectToml() error = %v", err)
	}

	if !result.Found {
		t.Error("Expected to find Python version")
	}

	if result.Version != "3.11" {
		t.Errorf("Version = %v, want 3.11", result.Version)
	}

	// PDM uses PEP 621 format
	if format := result.Metadata["format"]; format != "PEP621" {
		t.Errorf("Format = %v, want PEP621", format)
	}
}

func TestParsePyprojectToml_Mixed(t *testing.T) {
	// File with both Poetry and PEP 621 sections (PEP 621 should take priority)
	content := `[project]
name = "mixed-project"
requires-python = ">=3.12"

[tool.poetry]
name = "mixed-project"

[tool.poetry.dependencies]
python = "^3.11"
`

	result, err := ParsePyprojectToml([]byte(content), "pyproject.toml")
	if err != nil {
		t.Fatalf("ParsePyprojectToml() error = %v", err)
	}

	if !result.Found {
		t.Error("Expected to find Python version")
	}

	// PEP 621 should take priority
	if result.Version != "3.12" {
		t.Errorf("Version = %v, want 3.12 (PEP621 should take priority)", result.Version)
	}

	if format := result.Metadata["format"]; format != "PEP621" {
		t.Errorf("Format = %v, want PEP621", format)
	}
}

func TestParsePyprojectToml_EdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		wantFound bool
	}{
		{
			name:      "Empty file",
			content:   "",
			wantFound: false,
		},
		{
			name:      "Invalid TOML",
			content:   `[broken toml syntax`,
			wantFound: false,
		},
		{
			name: "No python section",
			content: `[tool.mypy]
python_version = "3.11"
`,
			wantFound: false,
		},
		{
			name: "Empty python constraint",
			content: `[tool.poetry.dependencies]
python = ""
`,
			wantFound: false,
		},
		{
			name: "Build system only",
			content: `[build-system]
requires = ["poetry-core>=1.0.0"]
build-backend = "poetry.core.masonry.api"
`,
			wantFound: false,
		},
		{
			name: "Comments and whitespace",
			content: `# This is a comment
[project]
# More comments
requires-python = ">=3.11"  # Python version
`,
			wantFound: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParsePyprojectToml([]byte(tt.content), "pyproject.toml")
			if err != nil {
				t.Fatalf("ParsePyprojectToml() error = %v", err)
			}

			if result.Found != tt.wantFound {
				t.Errorf("Found = %v, want %v", result.Found, tt.wantFound)
			}
		})
	}
}

func TestExtractVersionFromConstraint(t *testing.T) {
	tests := []struct {
		name       string
		constraint string
		want       string
		wantErr    bool
	}{
		{
			name:       "Caret constraint",
			constraint: "^3.11",
			want:       "3.11",
			wantErr:    false,
		},
		{
			name:       "Greater than or equal",
			constraint: ">=3.10",
			want:       "3.10",
			wantErr:    false,
		},
		{
			name:       "Exact version",
			constraint: "==3.11.5",
			want:       "3.11.5",
			wantErr:    false,
		},
		{
			name:       "Compatible release",
			constraint: "~=3.11.0",
			want:       "3.11.0",
			wantErr:    false,
		},
		{
			name:       "Range constraint",
			constraint: ">=3.10,<3.12",
			want:       "3.10",
			wantErr:    false,
		},
		{
			name:       "Wildcard",
			constraint: "3.11.*",
			want:       "3.11",
			wantErr:    false,
		},
		{
			name:       "Simple version",
			constraint: "3.9",
			want:       "3.9",
			wantErr:    false,
		},
		{
			name:       "With whitespace",
			constraint: "  >=3.11  ",
			want:       "3.11",
			wantErr:    false,
		},
		{
			name:       "Empty constraint",
			constraint: "",
			wantErr:    true,
		},
		{
			name:       "No version number",
			constraint: "python",
			wantErr:    true,
		},
		{
			name:       "Three-part version",
			constraint: ">=3.11.2",
			want:       "3.11.2",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := extractVersionFromConstraint(tt.constraint)
			if (err != nil) != tt.wantErr {
				t.Errorf("extractVersionFromConstraint() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("extractVersionFromConstraint() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetPyprojectTomlRule(t *testing.T) {
	rule := GetPyprojectTomlRule()

	// Validate the rule
	if err := rule.Validate(); err != nil {
		t.Fatalf("GetPyprojectTomlRule() returned invalid rule: %v", err)
	}

	// Check properties
	if rule.Name != "pyproject-toml" {
		t.Errorf("Name = %v, want pyproject-toml", rule.Name)
	}

	if rule.Priority != 10 {
		t.Errorf("Priority = %v, want 10", rule.Priority)
	}

	if !rule.Enabled {
		t.Error("Rule should be enabled")
	}

	if rule.Condition.FilePattern != "pyproject.toml" {
		t.Errorf("FilePattern = %v, want pyproject.toml", rule.Condition.FilePattern)
	}

	if rule.Condition.MaxFileSize != 1024*1024 {
		t.Errorf("MaxFileSize = %v, want 1048576", rule.Condition.MaxFileSize)
	}

	if rule.Parser == nil {
		t.Error("Parser should not be nil")
	}

	// Check tags
	expectedTags := []string{"config", "toml", "dependencies", "poetry", "pdm", "pep621"}
	if len(rule.Tags) != len(expectedTags) {
		t.Errorf("Tags length = %v, want %v", len(rule.Tags), len(expectedTags))
	}
}

func TestParsePyprojectToml_Metadata(t *testing.T) {
	content := `[project]
name = "test-project"
requires-python = ">=3.11"
dependencies = [
    "requests",
    "django",
    "flask",
]
`

	result, err := ParsePyprojectToml([]byte(content), "pyproject.toml")
	if err != nil {
		t.Fatalf("ParsePyprojectToml() error = %v", err)
	}

	if !result.Found {
		t.Fatal("Expected to find Python version")
	}

	// Check metadata
	if constraint, ok := result.Metadata["constraint"]; !ok {
		t.Error("Metadata should contain 'constraint'")
	} else if constraint != ">=3.11" {
		t.Errorf("constraint = %v, want >=3.11", constraint)
	}

	if depCount, ok := result.Metadata["dependency_count"]; !ok {
		t.Error("Metadata should contain 'dependency_count'")
	} else if depCount != "3" {
		t.Errorf("dependency_count = %v, want 3", depCount)
	}
}

func TestParsePyprojectToml_RawValue(t *testing.T) {
	tests := []struct {
		name         string
		content      string
		wantRawValue string
	}{
		{
			name: "Poetry format",
			content: `[tool.poetry.dependencies]
python = "^3.11"
`,
			wantRawValue: "^3.11",
		},
		{
			name: "PEP 621 format",
			content: `[project]
requires-python = ">=3.10,<4.0"
`,
			wantRawValue: ">=3.10,<4.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParsePyprojectToml([]byte(tt.content), "pyproject.toml")
			if err != nil {
				t.Fatalf("ParsePyprojectToml() error = %v", err)
			}

			if !result.Found {
				t.Fatal("Expected to find Python version")
			}

			if result.RawValue != tt.wantRawValue {
				t.Errorf("RawValue = %v, want %v", result.RawValue, tt.wantRawValue)
			}
		})
	}
}
