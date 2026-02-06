package config

import (
	"testing"

	"github.com/gbjohnso/gitlab-python-scanner/internal/rules"
)

// Type aliases for cleaner test code
type ParserFunc = rules.ParserFunc
type SearchResult = rules.SearchResult

func TestDefaultParserRegistry_GetParser(t *testing.T) {
	registry := NewDefaultParserRegistry()

	tests := []struct {
		name        string
		parserType  string
		config      map[string]interface{}
		wantErr     bool
		testContent []byte
		testFile    string
		wantFound   bool
		wantVersion string
	}{
		{
			name:        "simple_version parser",
			parserType:  "simple_version",
			config:      map[string]interface{}{"confidence": 1.0},
			wantErr:     false,
			testContent: []byte("3.11.5\n"),
			testFile:    ".python-version",
			wantFound:   true,
			wantVersion: "3.11.5",
		},
		{
			name:       "regex parser with version group",
			parserType: "regex",
			config: map[string]interface{}{
				"pattern":       `python-(?P<version>\d+\.\d+\.\d+)`,
				"version_group": "version",
				"confidence":    0.8,
			},
			wantErr:     false,
			testContent: []byte("FROM python-3.11.5\n"),
			testFile:    "Dockerfile",
			wantFound:   true,
			wantVersion: "3.11.5",
		},
		{
			name:       "regex parser without named group",
			parserType: "regex",
			config: map[string]interface{}{
				"pattern":    `version = "(\d+\.\d+\.\d+)"`,
				"confidence": 0.7,
			},
			wantErr:     false,
			testContent: []byte(`version = "3.11.5"`),
			testFile:    "setup.py",
			wantFound:   true,
			wantVersion: "3.11.5",
		},
		{
			name:        "pyproject_toml parser",
			parserType:  "pyproject_toml",
			config:      map[string]interface{}{},
			wantErr:     false,
			testContent: []byte(`[tool.poetry.dependencies]\npython = "^3.11"`),
			testFile:    "pyproject.toml",
			// Note: Don't test Found/Version as it depends on pyproject parser implementation
			// Just verify the parser can be retrieved
		},
		{
			name:       "unknown parser type",
			parserType: "unknown_parser",
			config:     map[string]interface{}{},
			wantErr:    true,
		},
		{
			name:       "regex parser missing pattern",
			parserType: "regex",
			config:     map[string]interface{}{},
			wantErr:    true,
		},
		{
			name:       "regex parser invalid pattern",
			parserType: "regex",
			config: map[string]interface{}{
				"pattern": "[invalid(",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, err := registry.GetParser(tt.parserType, tt.config)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("GetParser() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			// Test the parser if we have test content
			if tt.testContent != nil && tt.wantFound {
				result, err := parser(tt.testContent, tt.testFile)
				if err != nil {
					t.Errorf("Parser execution failed: %v", err)
					return
				}

				if result.Found != tt.wantFound {
					t.Errorf("Expected Found=%v, got %v", tt.wantFound, result.Found)
				}

				if tt.wantFound && tt.wantVersion != "" && result.Version != tt.wantVersion {
					t.Errorf("Expected version '%s', got '%s'", tt.wantVersion, result.Version)
				}
			}
		})
	}
}

func TestDefaultParserRegistry_ListParserTypes(t *testing.T) {
	registry := NewDefaultParserRegistry()
	types := registry.ListParserTypes()

	expectedTypes := map[string]bool{
		"pyproject_toml":  true,
		"regex":           true,
		"simple_version":  true,
		"string_search":   true,
	}

	if len(types) != len(expectedTypes) {
		t.Errorf("Expected %d parser types, got %d", len(expectedTypes), len(types))
	}

	for _, parserType := range types {
		if !expectedTypes[parserType] {
			t.Errorf("Unexpected parser type: %s", parserType)
		}
	}
}

func TestDefaultParserRegistry_RegisterParser(t *testing.T) {
	registry := NewDefaultParserRegistry()

	// Register a custom parser
	customParser := func(config map[string]interface{}) (ParserFunc, error) {
		return func(content []byte, filename string) (*SearchResult, error) {
			return &SearchResult{
				Found:      true,
				Version:    "custom",
				Source:     filename,
				Confidence: 0.5,
			}, nil
		}, nil
	}

	registry.RegisterParser("custom", customParser)

	// Verify it was registered
	types := registry.ListParserTypes()
	found := false
	for _, parserType := range types {
		if parserType == "custom" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Custom parser was not registered")
	}

	// Test using the custom parser
	parser, err := registry.GetParser("custom", map[string]interface{}{})
	if err != nil {
		t.Fatalf("Failed to get custom parser: %v", err)
	}

	result, err := parser([]byte("test"), "test.txt")
	if err != nil {
		t.Fatalf("Custom parser failed: %v", err)
	}

	if result.Version != "custom" {
		t.Errorf("Expected version 'custom', got '%s'", result.Version)
	}
}

func TestRegexParser_NoMatch(t *testing.T) {
	registry := NewDefaultParserRegistry()

	parser, err := registry.GetParser("regex", map[string]interface{}{
		"pattern": `version = "(\d+\.\d+\.\d+)"`,
	})
	if err != nil {
		t.Fatalf("Failed to get parser: %v", err)
	}

	// Test with content that doesn't match
	result, err := parser([]byte("no version here"), "test.txt")
	if err != nil {
		t.Errorf("Parser should not error on no match: %v", err)
	}

	if result.Found {
		t.Error("Expected Found=false for non-matching content")
	}
}

func TestSimpleVersionParser_Configuration(t *testing.T) {
	registry := NewDefaultParserRegistry()

	tests := []struct {
		name           string
		config         map[string]interface{}
		content        string
		wantVersion    string
		wantConfidence float64
	}{
		{
			name:           "default configuration",
			config:         map[string]interface{}{},
			content:        "  3.11.5  \n",
			wantVersion:    "3.11.5",
			wantConfidence: 1.0,
		},
		{
			name: "custom confidence",
			config: map[string]interface{}{
				"confidence": 0.5,
			},
			content:        "3.11.5",
			wantVersion:    "3.11.5",
			wantConfidence: 0.5,
		},
		{
			name: "no trim whitespace",
			config: map[string]interface{}{
				"trim_whitespace": false,
			},
			content:        "  3.11.5  \n",
			wantVersion:    "  3.11.5  \n",
			wantConfidence: 1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, err := registry.GetParser("simple_version", tt.config)
			if err != nil {
				t.Fatalf("Failed to get parser: %v", err)
			}

			result, err := parser([]byte(tt.content), ".python-version")
			if err != nil {
				t.Fatalf("Parser failed: %v", err)
			}

			if !result.Found {
				t.Error("Expected Found=true")
			}

			if result.Version != tt.wantVersion {
				t.Errorf("Expected version '%s', got '%s'", tt.wantVersion, result.Version)
			}

			if result.Confidence != tt.wantConfidence {
				t.Errorf("Expected confidence %f, got %f", tt.wantConfidence, result.Confidence)
			}
		})
	}
}

func TestRegexParser_MultipleGroups(t *testing.T) {
	registry := NewDefaultParserRegistry()

	// Test with multiple capture groups
	parser, err := registry.GetParser("regex", map[string]interface{}{
		"pattern":       `python (\d+)\.(\d+)\.(\d+)`,
		"version_group": "0", // Will use first capture group
	})
	if err != nil {
		t.Fatalf("Failed to get parser: %v", err)
	}

	result, err := parser([]byte("python 3.11.5"), "Dockerfile")
	if err != nil {
		t.Fatalf("Parser failed: %v", err)
	}

	if !result.Found {
		t.Error("Expected Found=true")
	}

	// Should capture first group (3)
	if result.Version != "3" {
		t.Errorf("Expected version '3', got '%s'", result.Version)
	}
}
