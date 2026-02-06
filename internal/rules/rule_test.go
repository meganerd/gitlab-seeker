package rules

import (
	"context"
	"fmt"
	"regexp"
	"testing"
)

// Mock parser that always succeeds
func mockParserSuccess(content []byte, filename string) (*SearchResult, error) {
	return &SearchResult{
		Found:      true,
		Version:    "3.11.0",
		Source:     filename,
		Confidence: 1.0,
		RawValue:   string(content),
	}, nil
}

// Mock parser that never finds a match
func mockParserNoMatch(content []byte, filename string) (*SearchResult, error) {
	return &SearchResult{Found: false}, nil
}

// Mock parser that returns an error
func mockParserError(content []byte, filename string) (*SearchResult, error) {
	return nil, fmt.Errorf("test error")
}

func TestSearchRuleMatches(t *testing.T) {
	tests := []struct {
		name     string
		rule     *SearchRule
		filename string
		filepath string
		expected bool
	}{
		{
			name: "Exact filename match",
			rule: &SearchRule{
				Name:    "test-rule",
				Enabled: true,
				Condition: MatchCondition{
					FilePattern: ".python-version",
				},
			},
			filename: ".python-version",
			filepath: "/project/.python-version",
			expected: true,
		},
		{
			name: "Wildcard filename match",
			rule: &SearchRule{
				Name:    "test-rule",
				Enabled: true,
				Condition: MatchCondition{
					FilePattern: "*.toml",
				},
			},
			filename: "pyproject.toml",
			filepath: "/project/pyproject.toml",
			expected: true,
		},
		{
			name: "Wildcard no match",
			rule: &SearchRule{
				Name:    "test-rule",
				Enabled: true,
				Condition: MatchCondition{
					FilePattern: "*.toml",
				},
			},
			filename: "setup.py",
			filepath: "/project/setup.py",
			expected: false,
		},
		{
			name: "Path pattern match",
			rule: &SearchRule{
				Name:    "test-rule",
				Enabled: true,
				Condition: MatchCondition{
					FilePattern: "Dockerfile",
					PathPattern: regexp.MustCompile("^.*/Dockerfile$"),
				},
			},
			filename: "Dockerfile",
			filepath: "/project/docker/Dockerfile",
			expected: true,
		},
		{
			name: "Path pattern no match",
			rule: &SearchRule{
				Name:    "test-rule",
				Enabled: true,
				Condition: MatchCondition{
					FilePattern: "Dockerfile",
					PathPattern: regexp.MustCompile("^.*/test/.*$"),
				},
			},
			filename: "Dockerfile",
			filepath: "/project/docker/Dockerfile",
			expected: false,
		},
		{
			name: "Disabled rule never matches",
			rule: &SearchRule{
				Name:    "test-rule",
				Enabled: false,
				Condition: MatchCondition{
					FilePattern: ".python-version",
				},
			},
			filename: ".python-version",
			filepath: "/project/.python-version",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.rule.Matches(tt.filename, tt.filepath)
			if result != tt.expected {
				t.Errorf("Matches() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestSearchRuleApply(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		rule        *SearchRule
		content     []byte
		filename    string
		expectFound bool
		expectError bool
	}{
		{
			name: "Successful parsing",
			rule: &SearchRule{
				Name:    "test-rule",
				Enabled: true,
				Parser:  mockParserSuccess,
				Condition: MatchCondition{
					FilePattern: ".python-version",
				},
			},
			content:     []byte("3.11.0"),
			filename:    ".python-version",
			expectFound: true,
			expectError: false,
		},
		{
			name: "No match found",
			rule: &SearchRule{
				Name:    "test-rule",
				Enabled: true,
				Parser:  mockParserNoMatch,
				Condition: MatchCondition{
					FilePattern: "*.txt",
				},
			},
			content:     []byte("some content"),
			filename:    "readme.txt",
			expectFound: false,
			expectError: false,
		},
		{
			name: "Disabled rule returns error",
			rule: &SearchRule{
				Name:    "test-rule",
				Enabled: false,
				Parser:  mockParserSuccess,
				Condition: MatchCondition{
					FilePattern: ".python-version",
				},
			},
			content:     []byte("3.11.0"),
			filename:    ".python-version",
			expectFound: false,
			expectError: true,
		},
		{
			name: "File size exceeds limit",
			rule: &SearchRule{
				Name:    "test-rule",
				Enabled: true,
				Parser:  mockParserSuccess,
				Condition: MatchCondition{
					FilePattern: "*.txt",
					MaxFileSize: 10,
				},
			},
			content:     []byte("This content is too long for the limit"),
			filename:    "file.txt",
			expectFound: false,
			expectError: true,
		},
		{
			name: "Required content not present",
			rule: &SearchRule{
				Name:    "test-rule",
				Enabled: true,
				Parser:  mockParserSuccess,
				Condition: MatchCondition{
					FilePattern:     "*.toml",
					RequiredContent: regexp.MustCompile("python"),
				},
			},
			content:     []byte("[tool.poetry]\nname = 'test'"),
			filename:    "pyproject.toml",
			expectFound: false,
			expectError: false,
		},
		{
			name: "Required content present",
			rule: &SearchRule{
				Name:    "test-rule",
				Enabled: true,
				Parser:  mockParserSuccess,
				Condition: MatchCondition{
					FilePattern:     "*.toml",
					RequiredContent: regexp.MustCompile("python"),
				},
			},
			content:     []byte("[tool.poetry.dependencies]\npython = '^3.11'"),
			filename:    "pyproject.toml",
			expectFound: true,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.rule.Apply(ctx, tt.content, tt.filename)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result == nil {
				t.Fatal("Expected result, got nil")
			}

			if result.Found != tt.expectFound {
				t.Errorf("Found = %v, want %v", result.Found, tt.expectFound)
			}

			// Verify source is populated for successful matches
			if result.Found && result.Source == "" {
				t.Error("Expected Source to be populated for successful match")
			}
		})
	}
}

func TestSearchRuleValidate(t *testing.T) {
	tests := []struct {
		name      string
		rule      *SearchRule
		expectErr bool
	}{
		{
			name: "Valid rule with file pattern",
			rule: &SearchRule{
				Name:    "test-rule",
				Parser:  mockParserSuccess,
				Enabled: true,
				Condition: MatchCondition{
					FilePattern: ".python-version",
				},
			},
			expectErr: false,
		},
		{
			name: "Valid rule with path pattern",
			rule: &SearchRule{
				Name:    "test-rule",
				Parser:  mockParserSuccess,
				Enabled: true,
				Condition: MatchCondition{
					PathPattern: regexp.MustCompile(".*Dockerfile$"),
				},
			},
			expectErr: false,
		},
		{
			name: "Missing name",
			rule: &SearchRule{
				Name:    "",
				Parser:  mockParserSuccess,
				Enabled: true,
				Condition: MatchCondition{
					FilePattern: ".python-version",
				},
			},
			expectErr: true,
		},
		{
			name: "Missing parser",
			rule: &SearchRule{
				Name:    "test-rule",
				Parser:  nil,
				Enabled: true,
				Condition: MatchCondition{
					FilePattern: ".python-version",
				},
			},
			expectErr: true,
		},
		{
			name: "Missing match conditions",
			rule: &SearchRule{
				Name:      "test-rule",
				Parser:    mockParserSuccess,
				Enabled:   true,
				Condition: MatchCondition{},
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.rule.Validate()
			if tt.expectErr && err == nil {
				t.Error("Expected validation error, got nil")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Unexpected validation error: %v", err)
			}
		})
	}
}

func TestSearchRuleClone(t *testing.T) {
	original := &SearchRule{
		Name:        "original",
		Description: "Original rule",
		Priority:    10,
		Enabled:     true,
		Parser:      mockParserSuccess,
		Tags:        []string{"tag1", "tag2"},
		Condition: MatchCondition{
			FilePattern:     "*.py",
			PathPattern:     regexp.MustCompile(".*test.*"),
			RequiredContent: regexp.MustCompile("python"),
			MaxFileSize:     1024,
		},
	}

	clone := original.Clone()

	// Verify all fields are copied
	if clone.Name != original.Name {
		t.Errorf("Name = %v, want %v", clone.Name, original.Name)
	}
	if clone.Description != original.Description {
		t.Errorf("Description = %v, want %v", clone.Description, original.Description)
	}
	if clone.Priority != original.Priority {
		t.Errorf("Priority = %v, want %v", clone.Priority, original.Priority)
	}
	if clone.Enabled != original.Enabled {
		t.Errorf("Enabled = %v, want %v", clone.Enabled, original.Enabled)
	}

	// Verify tags are deep copied
	if len(clone.Tags) != len(original.Tags) {
		t.Errorf("Tags length = %v, want %v", len(clone.Tags), len(original.Tags))
	}

	// Modify clone's tags and verify original is unchanged
	if len(clone.Tags) > 0 {
		clone.Tags[0] = "modified"
		if original.Tags[0] == "modified" {
			t.Error("Modifying clone's tags affected original")
		}
	}

	// Verify condition fields are copied
	if clone.Condition.FilePattern != original.Condition.FilePattern {
		t.Error("FilePattern not copied correctly")
	}
	if clone.Condition.MaxFileSize != original.Condition.MaxFileSize {
		t.Error("MaxFileSize not copied correctly")
	}
}

func TestGlobToRegex(t *testing.T) {
	tests := []struct {
		glob     string
		text     string
		expected bool
	}{
		{"*.txt", "file.txt", true},
		{"*.txt", "file.py", false},
		{"test*", "test123", true},
		{"test*", "mytest", false},
		{"?.txt", "a.txt", true},
		{"?.txt", "ab.txt", false},
		{"file.txt", "file.txt", true},
		{"file.txt", "other.txt", false},
		{"Dockerfile*", "Dockerfile", true},
		{"Dockerfile*", "Dockerfile.dev", true},
		{"*file*", "myfile.txt", true},
	}

	for _, tt := range tests {
		t.Run(tt.glob+" matches "+tt.text, func(t *testing.T) {
			regex := globToRegex(tt.glob)
			matched, err := regexp.MatchString(regex, tt.text)
			if err != nil {
				t.Fatalf("Regex compilation error: %v", err)
			}
			if matched != tt.expected {
				t.Errorf("globToRegex(%q).Match(%q) = %v, want %v (regex: %q)",
					tt.glob, tt.text, matched, tt.expected, regex)
			}
		})
	}
}

func TestRuleBuilder(t *testing.T) {
	t.Run("Build valid rule", func(t *testing.T) {
		rule, err := NewRuleBuilder("test-rule").
			Description("Test rule description").
			Priority(10).
			FilePattern("*.py").
			PathPattern(".*test.*").
			RequiredContent("import.*").
			MaxFileSize(1024).
			Parser(mockParserSuccess).
			Tags("test", "python").
			Build()

		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if rule.Name != "test-rule" {
			t.Errorf("Name = %v, want test-rule", rule.Name)
		}
		if rule.Description != "Test rule description" {
			t.Errorf("Description = %v, want 'Test rule description'", rule.Description)
		}
		if rule.Priority != 10 {
			t.Errorf("Priority = %v, want 10", rule.Priority)
		}
		if !rule.Enabled {
			t.Error("Expected rule to be enabled by default")
		}
		if len(rule.Tags) != 2 {
			t.Errorf("Tags length = %v, want 2", len(rule.Tags))
		}
	})

	t.Run("Build with invalid path pattern", func(t *testing.T) {
		_, err := NewRuleBuilder("test-rule").
			FilePattern("*.py").
			PathPattern("[invalid(regex").
			Parser(mockParserSuccess).
			Build()

		if err == nil {
			t.Error("Expected error for invalid regex, got nil")
		}
	})

	t.Run("Build without parser fails validation", func(t *testing.T) {
		_, err := NewRuleBuilder("test-rule").
			FilePattern("*.py").
			Build()

		if err == nil {
			t.Error("Expected validation error for missing parser, got nil")
		}
	})

	t.Run("MustBuild panics on error", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic, got none")
			}
		}()

		NewRuleBuilder("test-rule").
			FilePattern("*.py").
			MustBuild() // Missing parser, should panic
	})

	t.Run("MustBuild succeeds", func(t *testing.T) {
		rule := NewRuleBuilder("test-rule").
			FilePattern("*.py").
			Parser(mockParserSuccess).
			MustBuild()

		if rule == nil {
			t.Error("Expected rule, got nil")
		}
	})

	t.Run("Builder default values", func(t *testing.T) {
		rule, err := NewRuleBuilder("test-rule").
			FilePattern("*.py").
			Parser(mockParserSuccess).
			Build()

		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if rule.Priority != 50 {
			t.Errorf("Default priority = %v, want 50", rule.Priority)
		}
		if !rule.Enabled {
			t.Error("Default enabled should be true")
		}
	})

	t.Run("Builder with Enabled(false)", func(t *testing.T) {
		rule, err := NewRuleBuilder("test-rule").
			FilePattern("*.py").
			Parser(mockParserSuccess).
			Enabled(false).
			Build()

		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if rule.Enabled {
			t.Error("Expected rule to be disabled")
		}
	})
}

func TestMatchPattern(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		filename string
		expected bool
	}{
		{"Exact match", "pyproject.toml", "pyproject.toml", true},
		{"Exact no match", "pyproject.toml", "setup.py", false},
		{"Wildcard extension", "*.py", "script.py", true},
		{"Wildcard extension no match", "*.py", "script.txt", false},
		{"Wildcard prefix", "test_*", "test_feature.py", true},
		{"Wildcard prefix no match", "test_*", "feature_test.py", false},
		{"Wildcard both ends", "*file*", "myfile.txt", true},
		{"Question mark single char", "?.py", "a.py", true},
		{"Question mark no match", "?.py", "ab.py", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matched, err := matchPattern(tt.pattern, tt.filename)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if matched != tt.expected {
				t.Errorf("matchPattern(%q, %q) = %v, want %v",
					tt.pattern, tt.filename, matched, tt.expected)
			}
		})
	}
}

func TestSearchResultMetadata(t *testing.T) {
	result := &SearchResult{
		Found:      true,
		Version:    "3.11.0",
		Source:     ".python-version",
		Confidence: 1.0,
		RawValue:   "3.11.0",
		Metadata: map[string]string{
			"file_size": "8",
			"encoding":  "utf-8",
		},
	}

	if result.Metadata["file_size"] != "8" {
		t.Error("Metadata not properly stored")
	}
}
