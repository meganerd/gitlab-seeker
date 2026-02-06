package parsers

import (
	"github.com/gbjohnso/gitlab-python-scanner/internal/rules"
)

// DefaultRegistry returns a new registry with all built-in parsers registered.
// This is the recommended way to get a registry for general use.
func DefaultRegistry() *rules.Registry {
	registry := rules.NewRegistry()
	
	// Register all built-in parsers (in priority order)
	registry.MustRegister(GetPythonVersionFileRule()) // Priority 1
	registry.MustRegister(GetRuntimeTxtRule())         // Priority 2
	registry.MustRegister(GetSetupPyRule())            // Priority 8
	registry.MustRegister(GetPipfileRule())            // Priority 9
	registry.MustRegister(GetPyprojectTomlRule())      // Priority 10
	registry.MustRegister(GetDockerfileRule())         // Priority 11
	registry.MustRegister(GetGitLabCIRule())           // Priority 12
	registry.MustRegister(GetToxIniRule())             // Priority 13
	registry.MustRegister(GetRequirementsTxtRule())    // Priority 15
	
	return registry
}

// RegisterBuiltInParsers adds all built-in parsers to the given registry.
// This allows you to add built-in parsers to an existing registry.
func RegisterBuiltInParsers(registry *rules.Registry) error {
	parsers := []func() *rules.SearchRule{
		GetPythonVersionFileRule,
		GetRuntimeTxtRule,
		GetSetupPyRule,
		GetPipfileRule,
		GetPyprojectTomlRule,
		GetDockerfileRule,
		GetGitLabCIRule,
		GetToxIniRule,
		GetRequirementsTxtRule,
	}
	
	for _, getRule := range parsers {
		if err := registry.Register(getRule()); err != nil {
			return err
		}
	}
	
	return nil
}
