package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/gbjohnso/gitlab-python-scanner/internal/rules"
	"gopkg.in/yaml.v3"
)

// RuleConfig represents a search rule configuration in YAML/JSON format
type RuleConfig struct {
	// Name is the unique identifier for the rule
	Name string `yaml:"name" json:"name"`

	// Description provides human-readable information
	Description string `yaml:"description,omitempty" json:"description,omitempty"`

	// Priority determines rule evaluation order (lower = higher priority)
	Priority int `yaml:"priority,omitempty" json:"priority,omitempty"`

	// Enabled indicates if the rule is active
	Enabled *bool `yaml:"enabled,omitempty" json:"enabled,omitempty"`

	// Tags for categorization
	Tags []string `yaml:"tags,omitempty" json:"tags,omitempty"`

	// Match conditions
	Match MatchConfig `yaml:"match" json:"match"`

	// Parser configuration
	Parser ParserConfig `yaml:"parser" json:"parser"`
}

// MatchConfig defines when a rule should be applied
type MatchConfig struct {
	// FilePattern is a glob pattern to match filenames
	FilePattern string `yaml:"file_pattern,omitempty" json:"file_pattern,omitempty"`

	// PathPattern is a regex to match file paths
	PathPattern string `yaml:"path_pattern,omitempty" json:"path_pattern,omitempty"`

	// RequiredContent is a regex that must exist in the file
	RequiredContent string `yaml:"required_content,omitempty" json:"required_content,omitempty"`

	// MaxFileSize is the maximum file size to process in bytes
	MaxFileSize int64 `yaml:"max_file_size,omitempty" json:"max_file_size,omitempty"`
}

// ParserConfig defines how to parse and extract information
type ParserConfig struct {
	// Type specifies the parser implementation to use
	// Built-in types: "pyproject_toml", "python_version_file", "regex", etc.
	Type string `yaml:"type" json:"type"`

	// Config contains parser-specific configuration
	Config map[string]interface{} `yaml:"config,omitempty" json:"config,omitempty"`
}

// Config represents the complete configuration file structure
type Config struct {
	// Version of the config file format
	Version string `yaml:"version,omitempty" json:"version,omitempty"`

	// Rules defines the search rules
	Rules []RuleConfig `yaml:"rules" json:"rules"`

	// Settings contains global configuration
	Settings SettingsConfig `yaml:"settings,omitempty" json:"settings,omitempty"`
}

// SettingsConfig contains global configuration settings
type SettingsConfig struct {
	// DefaultEnabled sets the default enabled state for rules
	DefaultEnabled bool `yaml:"default_enabled,omitempty" json:"default_enabled,omitempty"`

	// DefaultPriority sets the default priority for rules
	DefaultPriority int `yaml:"default_priority,omitempty" json:"default_priority,omitempty"`
}

// LoadConfig loads a configuration file (YAML or JSON) from the given path
func LoadConfig(path string) (*Config, error) {
	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Determine format based on file extension
	ext := filepath.Ext(path)
	
	var config Config
	
	switch ext {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &config); err != nil {
			return nil, fmt.Errorf("failed to parse YAML config: %w", err)
		}
	case ".json":
		if err := json.Unmarshal(data, &config); err != nil {
			return nil, fmt.Errorf("failed to parse JSON config: %w", err)
		}
	default:
		// Try YAML first, then JSON
		if err := yaml.Unmarshal(data, &config); err != nil {
			if jsonErr := json.Unmarshal(data, &config); jsonErr != nil {
				return nil, fmt.Errorf("failed to parse config as YAML or JSON: YAML error: %v, JSON error: %v", err, jsonErr)
			}
		}
	}

	// Set defaults
	if config.Version == "" {
		config.Version = "1.0"
	}

	return &config, nil
}

// SaveConfig saves a configuration to a file (YAML or JSON)
func SaveConfig(config *Config, path string) error {
	var data []byte
	var err error

	// Determine format based on file extension
	ext := filepath.Ext(path)

	switch ext {
	case ".yaml", ".yml":
		data, err = yaml.Marshal(config)
		if err != nil {
			return fmt.Errorf("failed to marshal YAML config: %w", err)
		}
	case ".json":
		data, err = json.MarshalIndent(config, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON config: %w", err)
		}
	default:
		// Default to YAML
		data, err = yaml.Marshal(config)
		if err != nil {
			return fmt.Errorf("failed to marshal YAML config: %w", err)
		}
	}

	// Write file
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// ToRegistry converts a Config into a rules.Registry
// This allows loading rules from configuration files
func (c *Config) ToRegistry(parserRegistry ParserRegistry) (*rules.Registry, error) {
	registry := rules.NewRegistry()

	// Apply default settings
	defaultEnabled := true
	if c.Settings.DefaultEnabled {
		defaultEnabled = c.Settings.DefaultEnabled
	}

	defaultPriority := 50
	if c.Settings.DefaultPriority > 0 {
		defaultPriority = c.Settings.DefaultPriority
	}

	// Convert each rule config to a SearchRule
	for i, ruleConfig := range c.Rules {
		rule, err := ruleConfig.ToSearchRule(parserRegistry, defaultEnabled, defaultPriority)
		if err != nil {
			return nil, fmt.Errorf("failed to convert rule %d (%s): %w", i, ruleConfig.Name, err)
		}

		if err := registry.Register(rule); err != nil {
			return nil, fmt.Errorf("failed to register rule %s: %w", ruleConfig.Name, err)
		}
	}

	return registry, nil
}

// ToSearchRule converts a RuleConfig to a rules.SearchRule
func (rc *RuleConfig) ToSearchRule(parserRegistry ParserRegistry, defaultEnabled bool, defaultPriority int) (*rules.SearchRule, error) {
	// Validate required fields
	if rc.Name == "" {
		return nil, fmt.Errorf("rule name is required")
	}

	// Build the rule
	builder := rules.NewRuleBuilder(rc.Name)

	// Set description
	if rc.Description != "" {
		builder.Description(rc.Description)
	}

	// Set priority
	priority := defaultPriority
	if rc.Priority > 0 {
		priority = rc.Priority
	}
	builder.Priority(priority)

	// Set enabled state
	enabled := defaultEnabled
	if rc.Enabled != nil {
		enabled = *rc.Enabled
	}
	builder.Enabled(enabled)

	// Set tags
	if len(rc.Tags) > 0 {
		builder.Tags(rc.Tags...)
	}

	// Set match conditions
	if rc.Match.FilePattern != "" {
		builder.FilePattern(rc.Match.FilePattern)
	}

	if rc.Match.PathPattern != "" {
		builder.PathPattern(rc.Match.PathPattern)
	}

	if rc.Match.RequiredContent != "" {
		builder.RequiredContent(rc.Match.RequiredContent)
	}

	if rc.Match.MaxFileSize > 0 {
		builder.MaxFileSize(rc.Match.MaxFileSize)
	}

	// Get parser function from registry
	parser, err := parserRegistry.GetParser(rc.Parser.Type, rc.Parser.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to get parser: %w", err)
	}
	builder.Parser(parser)

	// Build and return
	return builder.Build()
}

// FromRegistry converts a rules.Registry to a Config
// This allows exporting rules to configuration files
func FromRegistry(registry *rules.Registry) *Config {
	config := &Config{
		Version: "1.0",
		Rules:   make([]RuleConfig, 0),
		Settings: SettingsConfig{
			DefaultEnabled:  true,
			DefaultPriority: 50,
		},
	}

	// Convert each rule
	for _, rule := range registry.List() {
		ruleConfig := RuleConfig{
			Name:        rule.Name,
			Description: rule.Description,
			Priority:    rule.Priority,
			Enabled:     &rule.Enabled,
			Tags:        rule.Tags,
			Match: MatchConfig{
				FilePattern: rule.Condition.FilePattern,
				MaxFileSize: rule.Condition.MaxFileSize,
			},
			// Note: Parser type and config cannot be easily reverse-engineered
			// from the ParserFunc, so we leave it empty
			Parser: ParserConfig{
				Type: "unknown",
			},
		}

		// Add regex patterns as strings if they exist
		if rule.Condition.PathPattern != nil {
			ruleConfig.Match.PathPattern = rule.Condition.PathPattern.String()
		}

		if rule.Condition.RequiredContent != nil {
			ruleConfig.Match.RequiredContent = rule.Condition.RequiredContent.String()
		}

		config.Rules = append(config.Rules, ruleConfig)
	}

	return config
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Version == "" {
		return fmt.Errorf("config version is required")
	}

	if len(c.Rules) == 0 {
		return fmt.Errorf("at least one rule is required")
	}

	// Check for duplicate rule names
	names := make(map[string]bool)
	for i, rule := range c.Rules {
		if rule.Name == "" {
			return fmt.Errorf("rule %d: name is required", i)
		}

		if names[rule.Name] {
			return fmt.Errorf("duplicate rule name: %s", rule.Name)
		}
		names[rule.Name] = true

		// Validate match conditions
		if rule.Match.FilePattern == "" && rule.Match.PathPattern == "" {
			return fmt.Errorf("rule %s: at least one match condition (file_pattern or path_pattern) is required", rule.Name)
		}

		// Validate regex patterns
		if rule.Match.PathPattern != "" {
			if _, err := regexp.Compile(rule.Match.PathPattern); err != nil {
				return fmt.Errorf("rule %s: invalid path_pattern: %w", rule.Name, err)
			}
		}

		if rule.Match.RequiredContent != "" {
			if _, err := regexp.Compile(rule.Match.RequiredContent); err != nil {
				return fmt.Errorf("rule %s: invalid required_content: %w", rule.Name, err)
			}
		}

		// Validate parser
		if rule.Parser.Type == "" {
			return fmt.Errorf("rule %s: parser type is required", rule.Name)
		}
	}

	return nil
}

// ParserRegistry is an interface for getting parser functions by type
type ParserRegistry interface {
	// GetParser returns a parser function for the given type and configuration
	GetParser(parserType string, config map[string]interface{}) (rules.ParserFunc, error)
}
