package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gbjohnso/gitlab-python-scanner/internal/rules"
)

func TestLoadConfig_YAML(t *testing.T) {
	yamlContent := `
version: "1.0"
rules:
  - name: python-version-file
    description: Parse .python-version file
    priority: 10
    enabled: true
    tags:
      - explicit
      - version-file
    match:
      file_pattern: ".python-version"
      max_file_size: 1024
    parser:
      type: simple_version
      config:
        confidence: 1.0

  - name: pyproject-toml
    description: Parse pyproject.toml for Python version
    priority: 20
    enabled: true
    tags:
      - config-file
    match:
      file_pattern: "pyproject.toml"
      required_content: "\\[tool\\.poetry\\]|\\[project\\]"
    parser:
      type: pyproject_toml

settings:
  default_enabled: true
  default_priority: 50
`

	// Write temp file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to write temp config: %v", err)
	}

	// Load config
	config, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// Validate
	if config.Version != "1.0" {
		t.Errorf("Expected version 1.0, got %s", config.Version)
	}

	if len(config.Rules) != 2 {
		t.Fatalf("Expected 2 rules, got %d", len(config.Rules))
	}

	// Check first rule
	rule1 := config.Rules[0]
	if rule1.Name != "python-version-file" {
		t.Errorf("Expected name 'python-version-file', got %s", rule1.Name)
	}
	if rule1.Priority != 10 {
		t.Errorf("Expected priority 10, got %d", rule1.Priority)
	}
	if !*rule1.Enabled {
		t.Error("Expected rule to be enabled")
	}
	if len(rule1.Tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(rule1.Tags))
	}
	if rule1.Match.FilePattern != ".python-version" {
		t.Errorf("Expected file pattern '.python-version', got %s", rule1.Match.FilePattern)
	}
	if rule1.Parser.Type != "simple_version" {
		t.Errorf("Expected parser type 'simple_version', got %s", rule1.Parser.Type)
	}

	// Check settings
	if !config.Settings.DefaultEnabled {
		t.Error("Expected default_enabled to be true")
	}
	if config.Settings.DefaultPriority != 50 {
		t.Errorf("Expected default priority 50, got %d", config.Settings.DefaultPriority)
	}
}

func TestLoadConfig_JSON(t *testing.T) {
	jsonContent := `{
  "version": "1.0",
  "rules": [
    {
      "name": "python-version-file",
      "description": "Parse .python-version file",
      "priority": 10,
      "enabled": true,
      "tags": ["explicit"],
      "match": {
        "file_pattern": ".python-version"
      },
      "parser": {
        "type": "simple_version"
      }
    }
  ],
  "settings": {
    "default_enabled": true,
    "default_priority": 50
  }
}`

	// Write temp file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	if err := os.WriteFile(configPath, []byte(jsonContent), 0644); err != nil {
		t.Fatalf("Failed to write temp config: %v", err)
	}

	// Load config
	config, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// Validate
	if config.Version != "1.0" {
		t.Errorf("Expected version 1.0, got %s", config.Version)
	}

	if len(config.Rules) != 1 {
		t.Fatalf("Expected 1 rule, got %d", len(config.Rules))
	}
}

func TestSaveConfig_YAML(t *testing.T) {
	config := &Config{
		Version: "1.0",
		Rules: []RuleConfig{
			{
				Name:        "test-rule",
				Description: "Test rule",
				Priority:    10,
				Enabled:     boolPtr(true),
				Tags:        []string{"test"},
				Match: MatchConfig{
					FilePattern: "*.txt",
				},
				Parser: ParserConfig{
					Type: "simple_version",
				},
			},
		},
		Settings: SettingsConfig{
			DefaultEnabled:  true,
			DefaultPriority: 50,
		},
	}

	// Save to temp file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	
	if err := SaveConfig(config, configPath); err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	// Load it back
	loaded, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// Verify
	if loaded.Version != config.Version {
		t.Errorf("Version mismatch: expected %s, got %s", config.Version, loaded.Version)
	}

	if len(loaded.Rules) != len(config.Rules) {
		t.Errorf("Rules count mismatch: expected %d, got %d", len(config.Rules), len(loaded.Rules))
	}
}

func TestSaveConfig_JSON(t *testing.T) {
	config := &Config{
		Version: "1.0",
		Rules: []RuleConfig{
			{
				Name:     "test-rule",
				Priority: 10,
				Match: MatchConfig{
					FilePattern: "*.txt",
				},
				Parser: ParserConfig{
					Type: "simple_version",
				},
			},
		},
	}

	// Save to temp file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	
	if err := SaveConfig(config, configPath); err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	// Load it back
	loaded, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// Verify
	if loaded.Version != config.Version {
		t.Errorf("Version mismatch: expected %s, got %s", config.Version, loaded.Version)
	}
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &Config{
				Version: "1.0",
				Rules: []RuleConfig{
					{
						Name: "test-rule",
						Match: MatchConfig{
							FilePattern: "*.txt",
						},
						Parser: ParserConfig{
							Type: "simple_version",
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "missing version",
			config: &Config{
				Rules: []RuleConfig{
					{
						Name: "test-rule",
						Match: MatchConfig{
							FilePattern: "*.txt",
						},
						Parser: ParserConfig{
							Type: "simple_version",
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "no rules",
			config: &Config{
				Version: "1.0",
				Rules:   []RuleConfig{},
			},
			wantErr: true,
		},
		{
			name: "duplicate rule names",
			config: &Config{
				Version: "1.0",
				Rules: []RuleConfig{
					{
						Name: "test-rule",
						Match: MatchConfig{
							FilePattern: "*.txt",
						},
						Parser: ParserConfig{
							Type: "simple_version",
						},
					},
					{
						Name: "test-rule",
						Match: MatchConfig{
							FilePattern: "*.md",
						},
						Parser: ParserConfig{
							Type: "simple_version",
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "missing match condition",
			config: &Config{
				Version: "1.0",
				Rules: []RuleConfig{
					{
						Name: "test-rule",
						Match: MatchConfig{},
						Parser: ParserConfig{
							Type: "simple_version",
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid regex pattern",
			config: &Config{
				Version: "1.0",
				Rules: []RuleConfig{
					{
						Name: "test-rule",
						Match: MatchConfig{
							PathPattern: "[invalid(",
						},
						Parser: ParserConfig{
							Type: "simple_version",
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "missing parser type",
			config: &Config{
				Version: "1.0",
				Rules: []RuleConfig{
					{
						Name: "test-rule",
						Match: MatchConfig{
							FilePattern: "*.txt",
						},
						Parser: ParserConfig{},
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfigToRegistry(t *testing.T) {
	config := &Config{
		Version: "1.0",
		Rules: []RuleConfig{
			{
				Name:        "test-rule",
				Description: "Test rule",
				Priority:    10,
				Enabled:     boolPtr(true),
				Tags:        []string{"test"},
				Match: MatchConfig{
					FilePattern: ".python-version",
					MaxFileSize: 1024,
				},
				Parser: ParserConfig{
					Type: "simple_version",
					Config: map[string]interface{}{
						"confidence": 1.0,
					},
				},
			},
		},
		Settings: SettingsConfig{
			DefaultEnabled:  true,
			DefaultPriority: 50,
		},
	}

	parserRegistry := NewDefaultParserRegistry()
	registry, err := config.ToRegistry(parserRegistry)
	if err != nil {
		t.Fatalf("ToRegistry failed: %v", err)
	}

	// Verify registry
	if registry.Count() != 1 {
		t.Errorf("Expected 1 rule in registry, got %d", registry.Count())
	}

	rule := registry.Get("test-rule")
	if rule == nil {
		t.Fatal("Rule not found in registry")
	}

	if rule.Name != "test-rule" {
		t.Errorf("Expected rule name 'test-rule', got %s", rule.Name)
	}

	if rule.Priority != 10 {
		t.Errorf("Expected priority 10, got %d", rule.Priority)
	}

	if !rule.Enabled {
		t.Error("Expected rule to be enabled")
	}

	// Test the rule matches expected files
	if !rule.Matches(".python-version", "/some/path/.python-version") {
		t.Error("Rule should match .python-version file")
	}

	// Test the parser works
	content := []byte("3.11.5")
	result, err := rule.Parser(content, ".python-version")
	if err != nil {
		t.Errorf("Parser failed: %v", err)
	}

	if !result.Found {
		t.Error("Expected parser to find version")
	}

	if result.Version != "3.11.5" {
		t.Errorf("Expected version '3.11.5', got %s", result.Version)
	}
}

func TestFromRegistry(t *testing.T) {
	// Create a registry with some rules
	registry := rules.NewRegistry()
	
	enabled := true
	rule := &rules.SearchRule{
		Name:        "test-rule",
		Description: "Test rule",
		Priority:    10,
		Enabled:     enabled,
		Tags:        []string{"test"},
		Condition: rules.MatchCondition{
			FilePattern: "*.txt",
			MaxFileSize: 1024,
		},
		Parser: func(content []byte, filename string) (*rules.SearchResult, error) {
			return &rules.SearchResult{Found: true, Version: "1.0"}, nil
		},
	}

	if err := registry.Register(rule); err != nil {
		t.Fatalf("Failed to register rule: %v", err)
	}

	// Convert to config
	config := FromRegistry(registry)

	// Verify
	if len(config.Rules) != 1 {
		t.Fatalf("Expected 1 rule, got %d", len(config.Rules))
	}

	ruleConfig := config.Rules[0]
	if ruleConfig.Name != "test-rule" {
		t.Errorf("Expected name 'test-rule', got %s", ruleConfig.Name)
	}

	if ruleConfig.Priority != 10 {
		t.Errorf("Expected priority 10, got %d", ruleConfig.Priority)
	}

	if !*ruleConfig.Enabled {
		t.Error("Expected rule to be enabled")
	}
}

func TestRuleConfigToSearchRule(t *testing.T) {
	ruleConfig := RuleConfig{
		Name:        "test-rule",
		Description: "Test rule",
		Priority:    10,
		Enabled:     boolPtr(true),
		Tags:        []string{"test"},
		Match: MatchConfig{
			FilePattern:     "*.txt",
			PathPattern:     "^/test/.*",
			RequiredContent: "version",
			MaxFileSize:     1024,
		},
		Parser: ParserConfig{
			Type: "simple_version",
		},
	}

	parserRegistry := NewDefaultParserRegistry()
	rule, err := ruleConfig.ToSearchRule(parserRegistry, true, 50)
	if err != nil {
		t.Fatalf("ToSearchRule failed: %v", err)
	}

	// Verify rule properties
	if rule.Name != "test-rule" {
		t.Errorf("Expected name 'test-rule', got %s", rule.Name)
	}

	if rule.Description != "Test rule" {
		t.Errorf("Expected description 'Test rule', got %s", rule.Description)
	}

	if rule.Priority != 10 {
		t.Errorf("Expected priority 10, got %d", rule.Priority)
	}

	if !rule.Enabled {
		t.Error("Expected rule to be enabled")
	}

	if len(rule.Tags) != 1 || rule.Tags[0] != "test" {
		t.Errorf("Expected tags ['test'], got %v", rule.Tags)
	}

	if rule.Condition.FilePattern != "*.txt" {
		t.Errorf("Expected file pattern '*.txt', got %s", rule.Condition.FilePattern)
	}

	if rule.Condition.MaxFileSize != 1024 {
		t.Errorf("Expected max file size 1024, got %d", rule.Condition.MaxFileSize)
	}

	// Verify regex patterns were compiled
	if rule.Condition.PathPattern == nil {
		t.Error("Expected path pattern to be compiled")
	}

	if rule.Condition.RequiredContent == nil {
		t.Error("Expected required content pattern to be compiled")
	}

	// Verify parser was assigned
	if rule.Parser == nil {
		t.Error("Expected parser to be assigned")
	}
}

// Helper function
func boolPtr(b bool) *bool {
	return &b
}
