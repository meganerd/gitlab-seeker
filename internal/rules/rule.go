package rules

import (
	"context"
	"fmt"
	"regexp"
)

// SearchResult represents the result of applying a search rule
type SearchResult struct {
	// Found indicates whether the rule successfully found a match
	Found bool

	// Version is the detected Python version (if found)
	Version string

	// Source is the file or location where the version was found
	Source string

	// Confidence indicates how confident we are in this result (0.0 to 1.0)
	// 1.0 = explicit version file, 0.5 = inferred from tool config, etc.
	Confidence float64

	// RawValue is the raw extracted value before parsing (for debugging)
	RawValue string

	// Metadata contains additional information about the match
	Metadata map[string]string
}

// ParserFunc is a function that parses file content to extract Python version information
// Parameters:
//   - content: The raw file content as bytes
//   - filename: The name of the file being parsed (for context)
// Returns:
//   - *SearchResult: The parsing result, or nil if no version found
//   - error: An error if parsing failed (nil if successful or no match)
type ParserFunc func(content []byte, filename string) (*SearchResult, error)

// MatchCondition defines when a rule should be applied
type MatchCondition struct {
	// FilePattern is a glob pattern or regex to match filenames
	// Examples: ".python-version", "*.toml", "pyproject.toml"
	FilePattern string

	// PathPattern is an optional regex to match full file paths
	// Examples: "^Dockerfile$", ".*/.gitlab-ci.yml"
	PathPattern *regexp.Regexp

	// RequiredContent is an optional regex that must match in the file
	// for the parser to be invoked (optimization to skip irrelevant files)
	RequiredContent *regexp.Regexp

	// MaxFileSize is the maximum file size to process (bytes)
	// If 0, no limit is applied. Prevents parsing huge files.
	MaxFileSize int64
}

// SearchRule defines a rule for searching and extracting Python version information
type SearchRule struct {
	// Name is a unique identifier for this rule
	// Examples: "python-version-file", "pyproject-toml", "dockerfile"
	Name string

	// Description provides human-readable information about this rule
	Description string

	// Priority determines the order in which rules are evaluated
	// Higher priority rules are checked first (0 is highest priority)
	// This allows explicit version files to be checked before inferred ones
	Priority int

	// Condition defines when this rule should be applied
	Condition MatchCondition

	// Parser is the function that extracts version information
	Parser ParserFunc

	// Enabled indicates whether this rule is active
	// Allows rules to be temporarily disabled without removing them
	Enabled bool

	// Tags provide categorization for rules
	// Examples: ["explicit", "config-file"], ["docker", "inferred"]
	Tags []string
}

// Matches checks if this rule should be applied to a given file
func (r *SearchRule) Matches(filename string, filepath string) bool {
	if !r.Enabled {
		return false
	}

	// Check file pattern (simple glob or exact match)
	if r.Condition.FilePattern != "" {
		matched, err := matchPattern(r.Condition.FilePattern, filename)
		if err != nil || !matched {
			return false
		}
	}

	// Check path pattern (regex)
	if r.Condition.PathPattern != nil {
		if !r.Condition.PathPattern.MatchString(filepath) {
			return false
		}
	}

	return true
}

// Apply executes the parser on the given file content
func (r *SearchRule) Apply(ctx context.Context, content []byte, filename string) (*SearchResult, error) {
	if !r.Enabled {
		return nil, fmt.Errorf("rule %s is disabled", r.Name)
	}

	// Check file size limit
	if r.Condition.MaxFileSize > 0 && int64(len(content)) > r.Condition.MaxFileSize {
		return nil, fmt.Errorf("file size %d exceeds maximum %d bytes", len(content), r.Condition.MaxFileSize)
	}

	// Check required content pattern
	if r.Condition.RequiredContent != nil {
		if !r.Condition.RequiredContent.Match(content) {
			// File doesn't contain required pattern, skip without error
			return &SearchResult{Found: false}, nil
		}
	}

	// Execute the parser
	result, err := r.Parser(content, filename)
	if err != nil {
		return nil, fmt.Errorf("parser error in rule %s: %w", r.Name, err)
	}

	// Populate source if not already set
	if result != nil && result.Found && result.Source == "" {
		result.Source = filename
	}

	return result, nil
}

// Validate checks if the rule is properly configured
func (r *SearchRule) Validate() error {
	if r.Name == "" {
		return fmt.Errorf("rule name cannot be empty")
	}

	if r.Parser == nil {
		return fmt.Errorf("rule %s: parser function is required", r.Name)
	}

	if r.Condition.FilePattern == "" && r.Condition.PathPattern == nil {
		return fmt.Errorf("rule %s: at least one match condition (FilePattern or PathPattern) is required", r.Name)
	}

	// Validate regex patterns compile correctly
	if r.Condition.PathPattern != nil {
		// Pattern is already compiled, just verify it exists
		if r.Condition.PathPattern.String() == "" {
			return fmt.Errorf("rule %s: PathPattern regex is empty", r.Name)
		}
	}

	if r.Condition.RequiredContent != nil {
		if r.Condition.RequiredContent.String() == "" {
			return fmt.Errorf("rule %s: RequiredContent regex is empty", r.Name)
		}
	}

	return nil
}

// Clone creates a deep copy of the rule
func (r *SearchRule) Clone() *SearchRule {
	clone := &SearchRule{
		Name:        r.Name,
		Description: r.Description,
		Priority:    r.Priority,
		Enabled:     r.Enabled,
		Parser:      r.Parser,
		Condition: MatchCondition{
			FilePattern:  r.Condition.FilePattern,
			MaxFileSize:  r.Condition.MaxFileSize,
		},
	}

	// Copy tags slice
	if len(r.Tags) > 0 {
		clone.Tags = make([]string, len(r.Tags))
		copy(clone.Tags, r.Tags)
	}

	// Regex patterns are immutable, so we can share them
	clone.Condition.PathPattern = r.Condition.PathPattern
	clone.Condition.RequiredContent = r.Condition.RequiredContent

	return clone
}

// matchPattern performs glob-like pattern matching
// Supports:
//   - Exact match: "pyproject.toml"
//   - Wildcard: "*.toml", "Dockerfile*"
//   - Simple glob patterns
func matchPattern(pattern, filename string) (bool, error) {
	// Exact match
	if pattern == filename {
		return true, nil
	}

	// Simple wildcard matching
	// Convert glob pattern to regex
	regexPattern := globToRegex(pattern)
	matched, err := regexp.MatchString(regexPattern, filename)
	if err != nil {
		return false, fmt.Errorf("invalid pattern %s: %w", pattern, err)
	}

	return matched, nil
}

// globToRegex converts a simple glob pattern to a regex pattern
func globToRegex(glob string) string {
	// Escape special regex characters except * and ?
	regex := regexp.QuoteMeta(glob)
	
	// Replace escaped wildcards with regex equivalents
	regex = regexp.MustCompile(`\\\*`).ReplaceAllString(regex, ".*")
	regex = regexp.MustCompile(`\\\?`).ReplaceAllString(regex, ".")
	
	// Anchor the pattern
	return "^" + regex + "$"
}

// RuleBuilder provides a fluent interface for constructing SearchRules
type RuleBuilder struct {
	rule *SearchRule
	err  error
}

// NewRuleBuilder creates a new rule builder with the given name
func NewRuleBuilder(name string) *RuleBuilder {
	return &RuleBuilder{
		rule: &SearchRule{
			Name:     name,
			Priority: 50, // Default middle priority
			Enabled:  true,
		},
	}
}

// Description sets the rule description
func (b *RuleBuilder) Description(desc string) *RuleBuilder {
	b.rule.Description = desc
	return b
}

// Priority sets the rule priority (lower number = higher priority)
func (b *RuleBuilder) Priority(priority int) *RuleBuilder {
	b.rule.Priority = priority
	return b
}

// FilePattern sets the file pattern to match
func (b *RuleBuilder) FilePattern(pattern string) *RuleBuilder {
	b.rule.Condition.FilePattern = pattern
	return b
}

// PathPattern sets a regex pattern for matching file paths
func (b *RuleBuilder) PathPattern(pattern string) *RuleBuilder {
	if b.err != nil {
		return b
	}

	regex, err := regexp.Compile(pattern)
	if err != nil {
		b.err = fmt.Errorf("invalid path pattern: %w", err)
		return b
	}

	b.rule.Condition.PathPattern = regex
	return b
}

// RequiredContent sets a regex pattern that must exist in the file
func (b *RuleBuilder) RequiredContent(pattern string) *RuleBuilder {
	if b.err != nil {
		return b
	}

	regex, err := regexp.Compile(pattern)
	if err != nil {
		b.err = fmt.Errorf("invalid required content pattern: %w", err)
		return b
	}

	b.rule.Condition.RequiredContent = regex
	return b
}

// MaxFileSize sets the maximum file size to process
func (b *RuleBuilder) MaxFileSize(bytes int64) *RuleBuilder {
	b.rule.Condition.MaxFileSize = bytes
	return b
}

// Parser sets the parser function
func (b *RuleBuilder) Parser(parser ParserFunc) *RuleBuilder {
	b.rule.Parser = parser
	return b
}

// Enabled sets whether the rule is enabled
func (b *RuleBuilder) Enabled(enabled bool) *RuleBuilder {
	b.rule.Enabled = enabled
	return b
}

// Tags sets the rule tags
func (b *RuleBuilder) Tags(tags ...string) *RuleBuilder {
	b.rule.Tags = tags
	return b
}

// Build constructs the final SearchRule and validates it
func (b *RuleBuilder) Build() (*SearchRule, error) {
	if b.err != nil {
		return nil, b.err
	}

	if err := b.rule.Validate(); err != nil {
		return nil, err
	}

	return b.rule, nil
}

// MustBuild builds the rule and panics on error (useful for static rule definitions)
func (b *RuleBuilder) MustBuild() *SearchRule {
	rule, err := b.Build()
	if err != nil {
		panic(fmt.Sprintf("failed to build rule: %v", err))
	}
	return rule
}
