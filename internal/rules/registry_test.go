package rules

import (
	"context"
	"fmt"
	"testing"
)

// Helper: Create a simple test parser
func testParser(version string, found bool) ParserFunc {
	return func(content []byte, filename string) (*SearchResult, error) {
		if !found {
			return &SearchResult{Found: false}, nil
		}
		return &SearchResult{
			Found:      true,
			Version:    version,
			Source:     filename,
			Confidence: 0.9,
			RawValue:   string(content),
			Metadata:   map[string]string{"test": "data"},
		}, nil
	}
}

// Helper: Create a test rule
func testRule(name string, priority int, pattern string, parser ParserFunc) *SearchRule {
	return NewRuleBuilder(name).
		Priority(priority).
		FilePattern(pattern).
		Parser(parser).
		Tags("test").
		MustBuild()
}

func TestNewRegistry(t *testing.T) {
	reg := NewRegistry()
	if reg == nil {
		t.Fatal("NewRegistry returned nil")
	}
	if reg.Count() != 0 {
		t.Errorf("Expected empty registry, got %d rules", reg.Count())
	}
}

func TestRegistryRegister(t *testing.T) {
	tests := []struct {
		name      string
		rule      *SearchRule
		expectErr bool
	}{
		{
			name:      "valid rule",
			rule:      testRule("test1", 10, "*.py", testParser("3.11", true)),
			expectErr: false,
		},
		{
			name:      "nil rule",
			rule:      nil,
			expectErr: true,
		},
		{
			name: "invalid rule - no parser",
			rule: &SearchRule{
				Name: "invalid",
				Condition: MatchCondition{
					FilePattern: "*.py",
				},
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reg := NewRegistry()
			err := reg.Register(tt.rule)
			if (err != nil) != tt.expectErr {
				t.Errorf("Register() error = %v, expectErr %v", err, tt.expectErr)
			}
			if !tt.expectErr && reg.Count() != 1 {
				t.Errorf("Expected 1 rule after registration, got %d", reg.Count())
			}
		})
	}
}

func TestRegistryMustRegister(t *testing.T) {
	t.Run("valid rule doesn't panic", func(t *testing.T) {
		reg := NewRegistry()
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("MustRegister panicked: %v", r)
			}
		}()
		reg.MustRegister(testRule("test", 10, "*.py", testParser("3.11", true)))
	})

	t.Run("invalid rule panics", func(t *testing.T) {
		reg := NewRegistry()
		defer func() {
			if r := recover(); r == nil {
				t.Error("MustRegister should have panicked on invalid rule")
			}
		}()
		reg.MustRegister(nil)
	})
}

func TestRegistryUnregister(t *testing.T) {
	reg := NewRegistry()
	rule := testRule("test", 10, "*.py", testParser("3.11", true))
	reg.MustRegister(rule)

	// Unregister existing rule
	if !reg.Unregister("test") {
		t.Error("Unregister should return true for existing rule")
	}
	if reg.Count() != 0 {
		t.Errorf("Expected 0 rules after unregister, got %d", reg.Count())
	}

	// Unregister non-existent rule
	if reg.Unregister("nonexistent") {
		t.Error("Unregister should return false for non-existent rule")
	}
}

func TestRegistryGet(t *testing.T) {
	reg := NewRegistry()
	rule := testRule("test", 10, "*.py", testParser("3.11", true))
	reg.MustRegister(rule)

	// Get existing rule
	retrieved := reg.Get("test")
	if retrieved == nil {
		t.Fatal("Get returned nil for existing rule")
	}
	if retrieved.Name != "test" {
		t.Errorf("Expected rule name 'test', got '%s'", retrieved.Name)
	}

	// Get non-existent rule
	if reg.Get("nonexistent") != nil {
		t.Error("Get should return nil for non-existent rule")
	}
}

func TestRegistryList(t *testing.T) {
	reg := NewRegistry()
	
	// Add rules with different priorities
	reg.MustRegister(testRule("high", 5, "*.py", testParser("3.11", true)))
	reg.MustRegister(testRule("medium", 10, "*.toml", testParser("3.10", true)))
	reg.MustRegister(testRule("low", 20, "*.txt", testParser("3.9", true)))

	rules := reg.List()
	if len(rules) != 3 {
		t.Errorf("Expected 3 rules, got %d", len(rules))
	}

	// Verify sorting by priority (ascending)
	if rules[0].Name != "high" || rules[0].Priority != 5 {
		t.Error("First rule should be 'high' with priority 5")
	}
	if rules[1].Name != "medium" || rules[1].Priority != 10 {
		t.Error("Second rule should be 'medium' with priority 10")
	}
	if rules[2].Name != "low" || rules[2].Priority != 20 {
		t.Error("Third rule should be 'low' with priority 20")
	}
}

func TestRegistryListEnabled(t *testing.T) {
	reg := NewRegistry()
	
	rule1 := testRule("enabled1", 10, "*.py", testParser("3.11", true))
	rule2 := testRule("enabled2", 20, "*.toml", testParser("3.10", true))
	rule3 := testRule("disabled", 5, "*.txt", testParser("3.9", true))
	rule3.Enabled = false

	reg.MustRegister(rule1)
	reg.MustRegister(rule2)
	reg.MustRegister(rule3)

	enabled := reg.ListEnabled()
	if len(enabled) != 2 {
		t.Errorf("Expected 2 enabled rules, got %d", len(enabled))
	}

	// Verify only enabled rules are returned
	for _, rule := range enabled {
		if !rule.Enabled {
			t.Errorf("ListEnabled returned disabled rule: %s", rule.Name)
		}
	}
}

func TestRegistryEnableDisable(t *testing.T) {
	reg := NewRegistry()
	rule := testRule("test", 10, "*.py", testParser("3.11", true))
	reg.MustRegister(rule)

	// Disable
	if !reg.Disable("test") {
		t.Error("Disable should return true for existing rule")
	}
	if reg.Get("test").Enabled {
		t.Error("Rule should be disabled")
	}

	// Enable
	if !reg.Enable("test") {
		t.Error("Enable should return true for existing rule")
	}
	if !reg.Get("test").Enabled {
		t.Error("Rule should be enabled")
	}

	// Non-existent rule
	if reg.Enable("nonexistent") {
		t.Error("Enable should return false for non-existent rule")
	}
	if reg.Disable("nonexistent") {
		t.Error("Disable should return false for non-existent rule")
	}
}

func TestRegistryCount(t *testing.T) {
	reg := NewRegistry()
	if reg.Count() != 0 {
		t.Errorf("Expected count 0, got %d", reg.Count())
	}

	reg.MustRegister(testRule("test1", 10, "*.py", testParser("3.11", true)))
	if reg.Count() != 1 {
		t.Errorf("Expected count 1, got %d", reg.Count())
	}

	reg.MustRegister(testRule("test2", 20, "*.toml", testParser("3.10", true)))
	if reg.Count() != 2 {
		t.Errorf("Expected count 2, got %d", reg.Count())
	}

	reg.Unregister("test1")
	if reg.Count() != 1 {
		t.Errorf("Expected count 1 after unregister, got %d", reg.Count())
	}
}

func TestRegistryClear(t *testing.T) {
	reg := NewRegistry()
	reg.MustRegister(testRule("test1", 10, "*.py", testParser("3.11", true)))
	reg.MustRegister(testRule("test2", 20, "*.toml", testParser("3.10", true)))

	reg.Clear()
	if reg.Count() != 0 {
		t.Errorf("Expected count 0 after clear, got %d", reg.Count())
	}
}

func TestRegistryFindMatchingRules(t *testing.T) {
	reg := NewRegistry()
	
	// Add various rules
	reg.MustRegister(testRule("py-files", 10, "*.py", testParser("3.11", true)))
	reg.MustRegister(testRule("toml-files", 20, "*.toml", testParser("3.10", true)))
	reg.MustRegister(testRule("exact-match", 5, "setup.py", testParser("3.9", true)))
	
	disabled := testRule("disabled", 1, "*.py", testParser("3.12", true))
	disabled.Enabled = false
	reg.MustRegister(disabled)

	tests := []struct {
		name          string
		filename      string
		filepath      string
		expectedCount int
		expectedFirst string
	}{
		{
			name:          "matches py-files",
			filename:      "test.py",
			filepath:      "/path/to/test.py",
			expectedCount: 1,
			expectedFirst: "py-files",
		},
		{
			name:          "matches both py-files and exact-match",
			filename:      "setup.py",
			filepath:      "/path/to/setup.py",
			expectedCount: 2,
			expectedFirst: "exact-match", // Higher priority (lower number)
		},
		{
			name:          "matches toml-files",
			filename:      "pyproject.toml",
			filepath:      "/path/to/pyproject.toml",
			expectedCount: 1,
			expectedFirst: "toml-files",
		},
		{
			name:          "no matches",
			filename:      "README.md",
			filepath:      "/path/to/README.md",
			expectedCount: 0,
			expectedFirst: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := reg.FindMatchingRules(tt.filename, tt.filepath)
			if len(matches) != tt.expectedCount {
				t.Errorf("Expected %d matches, got %d", tt.expectedCount, len(matches))
			}
			if tt.expectedCount > 0 && matches[0].Name != tt.expectedFirst {
				t.Errorf("Expected first match to be '%s', got '%s'", tt.expectedFirst, matches[0].Name)
			}
		})
	}
}

func TestRegistryExecute(t *testing.T) {
	ctx := context.Background()
	content := []byte("3.11.5")

	t.Run("successful execution", func(t *testing.T) {
		reg := NewRegistry()
		reg.MustRegister(testRule("test", 10, "*.py", testParser("3.11", true)))

		result := reg.Execute(ctx, content, "test.py", "/path/test.py", DefaultExecutionOptions())
		
		if result.File != "test.py" {
			t.Errorf("Expected file 'test.py', got '%s'", result.File)
		}
		if result.RulesApplied != 1 {
			t.Errorf("Expected 1 rule applied, got %d", result.RulesApplied)
		}
		if len(result.Results) != 1 {
			t.Errorf("Expected 1 result, got %d", len(result.Results))
		}
		if result.BestResult == nil {
			t.Fatal("Expected BestResult to be set")
		}
		if result.BestResult.Version != "3.11" {
			t.Errorf("Expected version '3.11', got '%s'", result.BestResult.Version)
		}
	})

	t.Run("no matches", func(t *testing.T) {
		reg := NewRegistry()
		reg.MustRegister(testRule("test", 10, "*.toml", testParser("3.11", true)))

		result := reg.Execute(ctx, content, "test.py", "/path/test.py", DefaultExecutionOptions())
		
		if result.RulesApplied != 0 {
			t.Errorf("Expected 0 rules applied, got %d", result.RulesApplied)
		}
		if len(result.Results) != 0 {
			t.Errorf("Expected 0 results, got %d", len(result.Results))
		}
		if result.BestResult != nil {
			t.Error("Expected BestResult to be nil")
		}
	})

	t.Run("stop on first match", func(t *testing.T) {
		reg := NewRegistry()
		reg.MustRegister(testRule("first", 10, "*.py", testParser("3.11", true)))
		reg.MustRegister(testRule("second", 20, "*.py", testParser("3.10", true)))

		opts := ExecutionOptions{StopOnFirstMatch: true}
		result := reg.Execute(ctx, content, "test.py", "/path/test.py", opts)
		
		if len(result.Results) != 1 {
			t.Errorf("Expected 1 result with StopOnFirstMatch, got %d", len(result.Results))
		}
		if result.Results[0].Version != "3.11" {
			t.Errorf("Expected first match version '3.11', got '%s'", result.Results[0].Version)
		}
	})

	t.Run("max results limit", func(t *testing.T) {
		reg := NewRegistry()
		reg.MustRegister(testRule("first", 10, "*.py", testParser("3.11", true)))
		reg.MustRegister(testRule("second", 20, "*.py", testParser("3.10", true)))
		reg.MustRegister(testRule("third", 30, "*.py", testParser("3.9", true)))

		opts := ExecutionOptions{MaxResults: 2}
		result := reg.Execute(ctx, content, "test.py", "/path/test.py", opts)
		
		if len(result.Results) != 2 {
			t.Errorf("Expected 2 results with MaxResults=2, got %d", len(result.Results))
		}
	})

	t.Run("min confidence filter", func(t *testing.T) {
		reg := NewRegistry()
		
		highConfParser := func(content []byte, filename string) (*SearchResult, error) {
			return &SearchResult{
				Found:      true,
				Version:    "3.11",
				Confidence: 0.9,
			}, nil
		}
		lowConfParser := func(content []byte, filename string) (*SearchResult, error) {
			return &SearchResult{
				Found:      true,
				Version:    "3.10",
				Confidence: 0.5,
			}, nil
		}

		reg.MustRegister(testRule("high", 10, "*.py", highConfParser))
		reg.MustRegister(testRule("low", 20, "*.py", lowConfParser))

		opts := ExecutionOptions{MinConfidence: 0.8}
		result := reg.Execute(ctx, content, "test.py", "/path/test.py", opts)
		
		if len(result.Results) != 1 {
			t.Errorf("Expected 1 result with MinConfidence=0.8, got %d", len(result.Results))
		}
		if result.Results[0].Version != "3.11" {
			t.Errorf("Expected high confidence result, got version '%s'", result.Results[0].Version)
		}
	})

	t.Run("tag filtering", func(t *testing.T) {
		reg := NewRegistry()
		
		configRule := NewRuleBuilder("config").
			FilePattern("*.py").
			Parser(testParser("3.11", true)).
			Tags("config", "explicit").
			MustBuild()
		
		dockerRule := NewRuleBuilder("docker").
			FilePattern("*.py").
			Parser(testParser("3.10", true)).
			Tags("docker", "inferred").
			MustBuild()

		reg.MustRegister(configRule)
		reg.MustRegister(dockerRule)

		opts := ExecutionOptions{Tags: []string{"config"}}
		result := reg.Execute(ctx, content, "test.py", "/path/test.py", opts)
		
		if len(result.Results) != 1 {
			t.Errorf("Expected 1 result with tag filtering, got %d", len(result.Results))
		}
		if result.Results[0].Version != "3.11" {
			t.Errorf("Expected config rule result, got version '%s'", result.Results[0].Version)
		}
	})

	t.Run("context cancellation", func(t *testing.T) {
		reg := NewRegistry()
		reg.MustRegister(testRule("test", 10, "*.py", testParser("3.11", true)))

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		result := reg.Execute(ctx, content, "test.py", "/path/test.py", DefaultExecutionOptions())
		
		if len(result.Errors) == 0 {
			t.Error("Expected error from cancelled context")
		}
	})

	t.Run("parser errors", func(t *testing.T) {
		reg := NewRegistry()
		
		errorParser := func(content []byte, filename string) (*SearchResult, error) {
			return nil, fmt.Errorf("parser error")
		}
		
		reg.MustRegister(testRule("error", 10, "*.py", errorParser))

		result := reg.Execute(ctx, content, "test.py", "/path/test.py", DefaultExecutionOptions())
		
		if len(result.Errors) == 0 {
			t.Error("Expected error from failing parser")
		}
	})

	t.Run("best result selection", func(t *testing.T) {
		reg := NewRegistry()
		
		medConfParser := func(content []byte, filename string) (*SearchResult, error) {
			return &SearchResult{
				Found:      true,
				Version:    "3.11",
				Confidence: 0.7,
			}, nil
		}
		highConfParser := func(content []byte, filename string) (*SearchResult, error) {
			return &SearchResult{
				Found:      true,
				Version:    "3.10",
				Confidence: 0.9,
			}, nil
		}

		reg.MustRegister(testRule("med", 10, "*.py", medConfParser))
		reg.MustRegister(testRule("high", 20, "*.py", highConfParser))

		result := reg.Execute(ctx, content, "test.py", "/path/test.py", DefaultExecutionOptions())
		
		if result.BestResult.Confidence != 0.9 {
			t.Errorf("Expected best result confidence 0.9, got %.1f", result.BestResult.Confidence)
		}
		if result.BestResult.Version != "3.10" {
			t.Errorf("Expected best result version '3.10', got '%s'", result.BestResult.Version)
		}
	})
}

func TestRegistryExecuteFirstMatch(t *testing.T) {
	ctx := context.Background()
	content := []byte("3.11.5")

	reg := NewRegistry()
	reg.MustRegister(testRule("first", 10, "*.py", testParser("3.11", true)))
	reg.MustRegister(testRule("second", 20, "*.py", testParser("3.10", true)))

	result, err := reg.ExecuteFirstMatch(ctx, content, "test.py", "/path/test.py")
	if err != nil {
		t.Fatalf("ExecuteFirstMatch error: %v", err)
	}
	if result == nil {
		t.Fatal("Expected result, got nil")
	}
	if result.Version != "3.11" {
		t.Errorf("Expected version '3.11', got '%s'", result.Version)
	}
}

func TestRegistryExecuteBestMatch(t *testing.T) {
	ctx := context.Background()
	content := []byte("3.11.5")

	reg := NewRegistry()
	
	lowConfParser := func(content []byte, filename string) (*SearchResult, error) {
		return &SearchResult{
			Found:      true,
			Version:    "3.11",
			Confidence: 0.5,
		}, nil
	}
	highConfParser := func(content []byte, filename string) (*SearchResult, error) {
		return &SearchResult{
			Found:      true,
			Version:    "3.10",
			Confidence: 0.9,
		}, nil
	}

	reg.MustRegister(testRule("low", 10, "*.py", lowConfParser))
	reg.MustRegister(testRule("high", 20, "*.py", highConfParser))

	result, err := reg.ExecuteBestMatch(ctx, content, "test.py", "/path/test.py")
	if err != nil {
		t.Fatalf("ExecuteBestMatch error: %v", err)
	}
	if result == nil {
		t.Fatal("Expected result, got nil")
	}
	if result.Version != "3.10" {
		t.Errorf("Expected best match version '3.10', got '%s'", result.Version)
	}
	if result.Confidence != 0.9 {
		t.Errorf("Expected confidence 0.9, got %.1f", result.Confidence)
	}
}

func TestRegistryClone(t *testing.T) {
	original := NewRegistry()
	rule1 := testRule("test1", 10, "*.py", testParser("3.11", true))
	rule2 := testRule("test2", 20, "*.toml", testParser("3.10", true))
	original.MustRegister(rule1)
	original.MustRegister(rule2)

	clone := original.Clone()

	// Verify clone has same rules
	if clone.Count() != original.Count() {
		t.Errorf("Clone has different count: original=%d, clone=%d", original.Count(), clone.Count())
	}

	// Modify clone
	clone.Unregister("test1")
	clone.MustRegister(testRule("test3", 30, "*.txt", testParser("3.9", true)))

	// Original should be unchanged
	if original.Count() != 2 {
		t.Errorf("Original was modified, count=%d", original.Count())
	}
	if original.Get("test1") == nil {
		t.Error("Original rule 'test1' was removed")
	}
	if original.Get("test3") != nil {
		t.Error("Original has new rule 'test3'")
	}
}

func TestRegistryGetStatistics(t *testing.T) {
	reg := NewRegistry()
	
	rule1 := testRule("test1", 10, "*.py", testParser("3.11", true))
	rule1.Tags = []string{"config", "explicit"}
	
	rule2 := testRule("test2", 10, "*.toml", testParser("3.10", true))
	rule2.Tags = []string{"config", "toml"}
	
	rule3 := testRule("test3", 20, "*.txt", testParser("3.9", true))
	rule3.Enabled = false
	rule3.Tags = []string{"text"}

	reg.MustRegister(rule1)
	reg.MustRegister(rule2)
	reg.MustRegister(rule3)

	stats := reg.GetStatistics()

	if stats.TotalRules != 3 {
		t.Errorf("Expected total 3, got %d", stats.TotalRules)
	}
	if stats.EnabledRules != 2 {
		t.Errorf("Expected 2 enabled, got %d", stats.EnabledRules)
	}
	if stats.DisabledRules != 1 {
		t.Errorf("Expected 1 disabled, got %d", stats.DisabledRules)
	}
	if stats.RulesByPriority[10] != 2 {
		t.Errorf("Expected 2 rules with priority 10, got %d", stats.RulesByPriority[10])
	}
	if stats.RulesByPriority[20] != 1 {
		t.Errorf("Expected 1 rule with priority 20, got %d", stats.RulesByPriority[20])
	}
	if stats.RulesByTag["config"] != 2 {
		t.Errorf("Expected 2 rules with tag 'config', got %d", stats.RulesByTag["config"])
	}
}

func TestFilterByTags(t *testing.T) {
	rule1 := testRule("test1", 10, "*.py", testParser("3.11", true))
	rule1.Tags = []string{"config", "explicit"}
	
	rule2 := testRule("test2", 20, "*.toml", testParser("3.10", true))
	rule2.Tags = []string{"config", "toml"}
	
	rule3 := testRule("test3", 30, "*.txt", testParser("3.9", true))
	rule3.Tags = []string{"text"}

	rules := []*SearchRule{rule1, rule2, rule3}

	tests := []struct {
		name          string
		tags          []string
		expectedCount int
	}{
		{
			name:          "filter by config",
			tags:          []string{"config"},
			expectedCount: 2,
		},
		{
			name:          "filter by text",
			tags:          []string{"text"},
			expectedCount: 1,
		},
		{
			name:          "filter by multiple tags",
			tags:          []string{"explicit", "text"},
			expectedCount: 2,
		},
		{
			name:          "no matching tags",
			tags:          []string{"nonexistent"},
			expectedCount: 0,
		},
		{
			name:          "empty tag filter",
			tags:          []string{},
			expectedCount: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filtered := filterByTags(rules, tt.tags)
			if len(filtered) != tt.expectedCount {
				t.Errorf("Expected %d rules, got %d", tt.expectedCount, len(filtered))
			}
		})
	}
}

func TestDefaultExecutionOptions(t *testing.T) {
	opts := DefaultExecutionOptions()
	
	if opts.StopOnFirstMatch {
		t.Error("Expected StopOnFirstMatch to be false by default")
	}
	if opts.MaxResults != 0 {
		t.Errorf("Expected MaxResults to be 0, got %d", opts.MaxResults)
	}
	if opts.MinConfidence != 0.0 {
		t.Errorf("Expected MinConfidence to be 0.0, got %.1f", opts.MinConfidence)
	}
	if opts.Tags != nil {
		t.Error("Expected Tags to be nil")
	}
}

// Benchmark tests
func BenchmarkRegistryExecute(b *testing.B) {
	reg := NewRegistry()
	for i := 0; i < 10; i++ {
		name := fmt.Sprintf("rule%d", i)
		reg.MustRegister(testRule(name, i*10, "*.py", testParser("3.11", true)))
	}

	ctx := context.Background()
	content := []byte("3.11.5")
	opts := DefaultExecutionOptions()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reg.Execute(ctx, content, "test.py", "/path/test.py", opts)
	}
}

func BenchmarkRegistryFindMatchingRules(b *testing.B) {
	reg := NewRegistry()
	for i := 0; i < 100; i++ {
		name := fmt.Sprintf("rule%d", i)
		pattern := fmt.Sprintf("*.%d", i)
		reg.MustRegister(testRule(name, i, pattern, testParser("3.11", true)))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reg.FindMatchingRules("test.py", "/path/test.py")
	}
}
