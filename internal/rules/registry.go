package rules

import (
	"context"
	"fmt"
	"sort"
	"sync"
)

// Registry manages a collection of SearchRules and provides
// rule execution, lookup, and management capabilities.
type Registry struct {
	mu    sync.RWMutex
	rules map[string]*SearchRule
}

// NewRegistry creates a new empty rule registry
func NewRegistry() *Registry {
	return &Registry{
		rules: make(map[string]*SearchRule),
	}
}

// Register adds a rule to the registry.
// If a rule with the same name exists, it will be replaced.
// Returns an error if the rule is invalid.
func (r *Registry) Register(rule *SearchRule) error {
	if rule == nil {
		return fmt.Errorf("cannot register nil rule")
	}

	if err := rule.Validate(); err != nil {
		return fmt.Errorf("invalid rule: %w", err)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.rules[rule.Name] = rule
	return nil
}

// MustRegister adds a rule to the registry and panics on error.
// Useful for registering built-in rules at initialization.
func (r *Registry) MustRegister(rule *SearchRule) {
	if err := r.Register(rule); err != nil {
		panic(fmt.Sprintf("failed to register rule: %v", err))
	}
}

// Unregister removes a rule from the registry by name.
// Returns true if the rule was found and removed, false otherwise.
func (r *Registry) Unregister(name string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.rules[name]; exists {
		delete(r.rules, name)
		return true
	}
	return false
}

// Get retrieves a rule by name.
// Returns nil if the rule is not found.
func (r *Registry) Get(name string) *SearchRule {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.rules[name]
}

// List returns all registered rules, sorted by priority (ascending).
// Lower priority numbers come first (higher priority).
func (r *Registry) List() []*SearchRule {
	r.mu.RLock()
	defer r.mu.RUnlock()

	rules := make([]*SearchRule, 0, len(r.rules))
	for _, rule := range r.rules {
		rules = append(rules, rule)
	}

	// Sort by priority (lower number = higher priority)
	sort.Slice(rules, func(i, j int) bool {
		return rules[i].Priority < rules[j].Priority
	})

	return rules
}

// ListEnabled returns only enabled rules, sorted by priority.
func (r *Registry) ListEnabled() []*SearchRule {
	r.mu.RLock()
	defer r.mu.RUnlock()

	rules := make([]*SearchRule, 0, len(r.rules))
	for _, rule := range r.rules {
		if rule.Enabled {
			rules = append(rules, rule)
		}
	}

	// Sort by priority
	sort.Slice(rules, func(i, j int) bool {
		return rules[i].Priority < rules[j].Priority
	})

	return rules
}

// Enable enables a rule by name.
// Returns true if the rule was found and updated, false otherwise.
func (r *Registry) Enable(name string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	if rule, exists := r.rules[name]; exists {
		rule.Enabled = true
		return true
	}
	return false
}

// Disable disables a rule by name.
// Returns true if the rule was found and updated, false otherwise.
func (r *Registry) Disable(name string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	if rule, exists := r.rules[name]; exists {
		rule.Enabled = false
		return true
	}
	return false
}

// Count returns the total number of registered rules.
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.rules)
}

// Clear removes all rules from the registry.
func (r *Registry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.rules = make(map[string]*SearchRule)
}

// FindMatchingRules returns all enabled rules that match the given file,
// sorted by priority (highest priority first).
func (r *Registry) FindMatchingRules(filename, filepath string) []*SearchRule {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var matches []*SearchRule
	for _, rule := range r.rules {
		if rule.Enabled && rule.Matches(filename, filepath) {
			matches = append(matches, rule)
		}
	}

	// Sort by priority (lower number = higher priority)
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Priority < matches[j].Priority
	})

	return matches
}

// ExecutionResult represents the result of executing rules against a file
type ExecutionResult struct {
	// File is the name of the file that was processed
	File string

	// Results contains all successful search results, ordered by priority
	Results []*SearchResult

	// BestResult is the highest confidence result, or nil if no matches
	BestResult *SearchResult

	// RulesApplied is the number of rules that were executed
	RulesApplied int

	// Errors contains any errors that occurred during execution
	Errors []error
}

// ExecutionOptions configures how rules are executed
type ExecutionOptions struct {
	// StopOnFirstMatch stops execution after the first successful match
	// Useful when you only need one result (e.g., Python version detection)
	StopOnFirstMatch bool

	// MaxResults limits the number of results to return
	// 0 means no limit
	MaxResults int

	// MinConfidence filters out results below this confidence threshold
	// 0.0 means all results are returned
	MinConfidence float64

	// Tags filters rules to only those with at least one matching tag
	// Empty slice means no tag filtering
	Tags []string
}

// DefaultExecutionOptions returns sensible defaults for rule execution
func DefaultExecutionOptions() ExecutionOptions {
	return ExecutionOptions{
		StopOnFirstMatch: false,
		MaxResults:       0,
		MinConfidence:    0.0,
		Tags:             nil,
	}
}

// Execute applies all matching rules to the given file content.
// Rules are executed in priority order (highest priority first).
// Returns an ExecutionResult with all successful matches and any errors.
func (r *Registry) Execute(ctx context.Context, content []byte, filename, filepath string, opts ExecutionOptions) *ExecutionResult {
	result := &ExecutionResult{
		File:    filename,
		Results: make([]*SearchResult, 0),
		Errors:  make([]error, 0),
	}

	// Find all matching rules
	matchingRules := r.FindMatchingRules(filename, filepath)

	// Filter by tags if specified
	if len(opts.Tags) > 0 {
		matchingRules = filterByTags(matchingRules, opts.Tags)
	}

	// Execute each matching rule
	for _, rule := range matchingRules {
		// Check context cancellation
		select {
		case <-ctx.Done():
			result.Errors = append(result.Errors, fmt.Errorf("execution cancelled: %w", ctx.Err()))
			return result
		default:
		}

		result.RulesApplied++

		// Apply the rule
		searchResult, err := rule.Apply(ctx, content, filename)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("rule %s: %w", rule.Name, err))
			continue
		}

		// Skip if not found or below confidence threshold
		if searchResult == nil || !searchResult.Found {
			continue
		}

		if opts.MinConfidence > 0 && searchResult.Confidence < opts.MinConfidence {
			continue
		}

		// Add to results
		result.Results = append(result.Results, searchResult)

		// Update best result (highest confidence)
		if result.BestResult == nil || searchResult.Confidence > result.BestResult.Confidence {
			result.BestResult = searchResult
		}

		// Stop if requested
		if opts.StopOnFirstMatch {
			break
		}

		// Check max results limit
		if opts.MaxResults > 0 && len(result.Results) >= opts.MaxResults {
			break
		}
	}

	return result
}

// ExecuteFirstMatch is a convenience method that returns only the first (highest priority) match.
// Returns nil if no matches were found.
func (r *Registry) ExecuteFirstMatch(ctx context.Context, content []byte, filename, filepath string) (*SearchResult, error) {
	opts := ExecutionOptions{
		StopOnFirstMatch: true,
		MaxResults:       1,
	}

	result := r.Execute(ctx, content, filename, filepath, opts)

	// Check for errors
	if len(result.Errors) > 0 {
		// Return the first error
		return nil, result.Errors[0]
	}

	// Return the best result (or nil if no matches)
	return result.BestResult, nil
}

// ExecuteBestMatch returns the result with the highest confidence score.
// Returns nil if no matches were found.
func (r *Registry) ExecuteBestMatch(ctx context.Context, content []byte, filename, filepath string) (*SearchResult, error) {
	opts := DefaultExecutionOptions()
	result := r.Execute(ctx, content, filename, filepath, opts)

	// Check for errors
	if len(result.Errors) > 0 {
		// Return the first error
		return nil, result.Errors[0]
	}

	// Return the best result (or nil if no matches)
	return result.BestResult, nil
}

// filterByTags returns rules that have at least one tag matching the filter
func filterByTags(rules []*SearchRule, tags []string) []*SearchRule {
	if len(tags) == 0 {
		return rules
	}

	tagSet := make(map[string]bool)
	for _, tag := range tags {
		tagSet[tag] = true
	}

	filtered := make([]*SearchRule, 0)
	for _, rule := range rules {
		for _, ruleTag := range rule.Tags {
			if tagSet[ruleTag] {
				filtered = append(filtered, rule)
				break
			}
		}
	}

	return filtered
}

// Clone creates a new registry with deep copies of all rules
func (r *Registry) Clone() *Registry {
	r.mu.RLock()
	defer r.mu.RUnlock()

	clone := NewRegistry()
	for name, rule := range r.rules {
		clone.rules[name] = rule.Clone()
	}

	return clone
}

// Statistics returns information about the registry
type RegistryStatistics struct {
	TotalRules    int
	EnabledRules  int
	DisabledRules int
	RulesByPriority map[int]int
	RulesByTag    map[string]int
}

// GetStatistics returns statistical information about the registry
func (r *Registry) GetStatistics() RegistryStatistics {
	r.mu.RLock()
	defer r.mu.RUnlock()

	stats := RegistryStatistics{
		RulesByPriority: make(map[int]int),
		RulesByTag:      make(map[string]int),
	}

	for _, rule := range r.rules {
		stats.TotalRules++
		
		if rule.Enabled {
			stats.EnabledRules++
		} else {
			stats.DisabledRules++
		}

		stats.RulesByPriority[rule.Priority]++

		for _, tag := range rule.Tags {
			stats.RulesByTag[tag]++
		}
	}

	return stats
}
