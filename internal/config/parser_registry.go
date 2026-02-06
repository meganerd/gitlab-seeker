package config

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/gbjohnso/gitlab-python-scanner/internal/parsers"
	"github.com/gbjohnso/gitlab-python-scanner/internal/rules"
)

// DefaultParserRegistry implements ParserRegistry with built-in parsers
type DefaultParserRegistry struct {
	parsers map[string]ParserFactory
}

// ParserFactory is a function that creates a parser from configuration
type ParserFactory func(config map[string]interface{}) (rules.ParserFunc, error)

// NewDefaultParserRegistry creates a new parser registry with built-in parsers
func NewDefaultParserRegistry() *DefaultParserRegistry {
	registry := &DefaultParserRegistry{
		parsers: make(map[string]ParserFactory),
	}

	// Register built-in parsers
	registry.RegisterParser("pyproject_toml", func(config map[string]interface{}) (rules.ParserFunc, error) {
		// Use the pyproject.toml parser
		rule := parsers.GetPyprojectTomlRule()
		return rule.Parser, nil
	})

	registry.RegisterParser("regex", createRegexParser)
	registry.RegisterParser("simple_version", createSimpleVersionParser)
	registry.RegisterParser("string_search", createStringSearchParser)

	return registry
}

// RegisterParser adds a parser factory to the registry
func (r *DefaultParserRegistry) RegisterParser(parserType string, factory ParserFactory) {
	r.parsers[parserType] = factory
}

// GetParser returns a parser function for the given type and configuration
func (r *DefaultParserRegistry) GetParser(parserType string, config map[string]interface{}) (rules.ParserFunc, error) {
	factory, exists := r.parsers[parserType]
	if !exists {
		return nil, fmt.Errorf("unknown parser type: %s", parserType)
	}

	return factory(config)
}

// createRegexParser creates a parser that uses regex to extract version information
func createRegexParser(config map[string]interface{}) (rules.ParserFunc, error) {
	// Get regex pattern from config
	patternStr, ok := config["pattern"].(string)
	if !ok {
		return nil, fmt.Errorf("regex parser requires 'pattern' string in config")
	}

	pattern, err := regexp.Compile(patternStr)
	if err != nil {
		return nil, fmt.Errorf("invalid regex pattern: %w", err)
	}

	// Get optional group name or index for version extraction
	versionGroup := "version"
	if group, ok := config["version_group"].(string); ok {
		versionGroup = group
	}

	// Get optional confidence value
	confidence := 0.5
	if conf, ok := config["confidence"].(float64); ok {
		confidence = conf
	}

	// Return parser function
	return func(content []byte, filename string) (*rules.SearchResult, error) {
		matches := pattern.FindSubmatch(content)
		if matches == nil {
			return &rules.SearchResult{Found: false}, nil
		}

		// Extract version from named group or first capture group
		var version string
		if groupIndex := pattern.SubexpIndex(versionGroup); groupIndex >= 0 && groupIndex < len(matches) {
			version = string(matches[groupIndex])
		} else if len(matches) > 1 {
			version = string(matches[1])
		} else {
			version = string(matches[0])
		}

		if version == "" {
			return &rules.SearchResult{Found: false}, nil
		}

		return &rules.SearchResult{
			Found:      true,
			Version:    strings.TrimSpace(version),
			Source:     filename,
			Confidence: confidence,
			RawValue:   version,
		}, nil
	}, nil
}

// createSimpleVersionParser creates a parser that extracts a simple version string
func createSimpleVersionParser(config map[string]interface{}) (rules.ParserFunc, error) {
	// Get optional confidence value
	confidence := 1.0
	if conf, ok := config["confidence"].(float64); ok {
		confidence = conf
	}

	// Get optional trim behavior
	trimWhitespace := true
	if trim, ok := config["trim_whitespace"].(bool); ok {
		trimWhitespace = trim
	}

	// Return parser function that reads the entire file as a version string
	return func(content []byte, filename string) (*rules.SearchResult, error) {
		version := string(content)
		
		if trimWhitespace {
			version = strings.TrimSpace(version)
		}

		if version == "" {
			return &rules.SearchResult{Found: false}, nil
		}

		return &rules.SearchResult{
			Found:      true,
			Version:    version,
			Source:     filename,
			Confidence: confidence,
			RawValue:   version,
		}, nil
	}, nil
}

// createStringSearchParser creates a parser that searches for arbitrary strings/regex in file content
func createStringSearchParser(config map[string]interface{}) (rules.ParserFunc, error) {
	searchTerm, ok := config["search_term"].(string)
	if !ok || searchTerm == "" {
		return nil, fmt.Errorf("string_search parser requires 'search_term' string in config")
	}

	isRegex := false
	if v, ok := config["is_regex"].(bool); ok {
		isRegex = v
	}

	caseSensitive := false
	if v, ok := config["case_sensitive"].(bool); ok {
		caseSensitive = v
	}

	maxMatches := 0
	if v, ok := config["max_matches"].(float64); ok {
		maxMatches = int(v)
	}

	parser := &parsers.StringSearchParser{
		SearchTerm:    searchTerm,
		IsRegex:       isRegex,
		CaseSensitive: caseSensitive,
		MaxMatches:    maxMatches,
	}

	return parser.AsParserFunc(), nil
}

// ListParserTypes returns a list of all registered parser types
func (r *DefaultParserRegistry) ListParserTypes() []string {
	types := make([]string, 0, len(r.parsers))
	for parserType := range r.parsers {
		types = append(types, parserType)
	}
	return types
}
