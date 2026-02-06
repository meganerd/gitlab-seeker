package parsers

import (
	"github.com/gbjohnso/gitlab-python-scanner/internal/rules"
)

// DefaultRegistry returns a new registry with all built-in parsers registered.
// This is the recommended way to get a registry for general use.
func DefaultRegistry() *rules.Registry {
	registry := rules.NewRegistry()
	
	// Register all built-in parsers
	registry.MustRegister(GetPyprojectTomlRule())
	
	// Add more built-in parsers here as they are implemented:
	// registry.MustRegister(GetRequirementsTxtRule())
	// registry.MustRegister(GetPythonVersionFileRule())
	// registry.MustRegister(GetDockerfileRule())
	
	return registry
}

// RegisterBuiltInParsers adds all built-in parsers to the given registry.
// This allows you to add built-in parsers to an existing registry.
func RegisterBuiltInParsers(registry *rules.Registry) error {
	parsers := []func() *rules.SearchRule{
		GetPyprojectTomlRule,
		// Add more built-in parsers here as they are implemented
	}
	
	for _, getRule := range parsers {
		if err := registry.Register(getRule()); err != nil {
			return err
		}
	}
	
	return nil
}
