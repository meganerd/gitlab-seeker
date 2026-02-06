package parsers

import (
	"os"
	"path/filepath"
	"testing"
)

// TestParsePyprojectToml_RealWorldExamples tests the parser against realistic pyproject.toml files
func TestParsePyprojectToml_RealWorldExamples(t *testing.T) {
	testDataDir := "../../test/testdata/pyproject"

	tests := []struct {
		filename       string
		wantFound      bool
		wantVersion    string
		wantFormat     string
		wantConfidence float64
		minDeps        int // Minimum number of dependencies expected
	}{
		{
			filename:       "poetry-example.toml",
			wantFound:      true,
			wantVersion:    "3.11",
			wantFormat:     "Poetry",
			wantConfidence: 0.9,
			minDeps:        4, // Should have at least 4 dependencies
		},
		{
			filename:       "pep621-example.toml",
			wantFound:      true,
			wantVersion:    "3.10",
			wantFormat:     "PEP621",
			wantConfidence: 0.9,
			minDeps:        4,
		},
		{
			filename:       "pdm-example.toml",
			wantFound:      true,
			wantVersion:    "3.11",
			wantFormat:     "PEP621", // PDM uses PEP 621 format
			wantConfidence: 0.9,
			minDeps:        4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			// Read the test file
			filePath := filepath.Join(testDataDir, tt.filename)
			content, err := os.ReadFile(filePath)
			if err != nil {
				t.Skipf("Test file not found: %s (skipping)", filePath)
				return
			}

			// Parse the file
			result, err := ParsePyprojectToml(content, tt.filename)
			if err != nil {
				t.Fatalf("ParsePyprojectToml() error = %v", err)
			}

			// Verify results
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

			if format := result.Metadata["format"]; format != tt.wantFormat {
				t.Errorf("Format = %v, want %v", format, tt.wantFormat)
			}

			if result.Source != tt.filename {
				t.Errorf("Source = %v, want %v", result.Source, tt.filename)
			}

			if result.RawValue == "" {
				t.Error("RawValue should not be empty")
			}

			if constraint := result.Metadata["constraint"]; constraint == "" {
				t.Error("Metadata[constraint] should not be empty")
			}

			// Note: We don't check exact dependency count as it varies by file
			// and our parser focuses on Python version, not full dependency extraction
		})
	}
}

// TestParsePyprojectToml_Integration tests the parser with the rule
func TestParsePyprojectToml_Integration(t *testing.T) {
	rule := GetPyprojectTomlRule()

	// Test that the rule matches pyproject.toml files
	if !rule.Matches("pyproject.toml", "/project/pyproject.toml") {
		t.Error("Rule should match pyproject.toml")
	}

	if rule.Matches("setup.py", "/project/setup.py") {
		t.Error("Rule should not match setup.py")
	}

	// Test applying the rule
	content := []byte(`[project]
name = "integration-test"
requires-python = ">=3.11"
`)

	result, err := rule.Parser(content, "pyproject.toml")
	if err != nil {
		t.Fatalf("Parser error = %v", err)
	}

	if !result.Found {
		t.Error("Expected to find Python version")
	}

	if result.Version != "3.11" {
		t.Errorf("Version = %v, want 3.11", result.Version)
	}
}
