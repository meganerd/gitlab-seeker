package parsers

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/gbjohnso/gitlab-python-scanner/internal/rules"
)

// PyprojectToml represents the structure of a pyproject.toml file
// It supports multiple formats: Poetry, PDM, and PEP 621
type PyprojectToml struct {
	// PEP 621 format
	Project *ProjectSection `toml:"project"`

	// Poetry format
	Tool *ToolSection `toml:"tool"`
}

// ProjectSection represents the [project] section (PEP 621)
type ProjectSection struct {
	Name              string                 `toml:"name"`
	RequiresPython    string                 `toml:"requires-python"`
	Dependencies      []string               `toml:"dependencies"`
	OptionalDeps      map[string][]string    `toml:"optional-dependencies"`
	DynamicFields     []string               `toml:"dynamic"`
}

// ToolSection represents the [tool] section
type ToolSection struct {
	Poetry *PoetrySection `toml:"poetry"`
	PDM    *PDMSection    `toml:"pdm"`
}

// PoetrySection represents Poetry configuration
type PoetrySection struct {
	Name         string                 `toml:"name"`
	Dependencies map[string]interface{} `toml:"dependencies"`
	DevDeps      map[string]interface{} `toml:"dev-dependencies"`
	Group        map[string]*DepGroup   `toml:"group"`
}

// DepGroup represents a Poetry dependency group
type DepGroup struct {
	Dependencies map[string]interface{} `toml:"dependencies"`
}

// PDMSection represents PDM configuration
type PDMSection struct {
	DevDeps map[string][]string `toml:"dev-dependencies"`
}

// ParsePyprojectToml is a ParserFunc that extracts Python version and dependencies
// from pyproject.toml files.
//
// Supports:
// - Poetry format: [tool.poetry.dependencies] with python = "^3.11"
// - PDM format: [project] with requires-python = ">=3.11"
// - PEP 621 format: [project] with requires-python = ">=3.11"
//
// Returns:
// - SearchResult with Python version if found
// - Confidence: 0.9 for explicit version constraints
// - Metadata includes: format type, dependency count, raw constraint
func ParsePyprojectToml(content []byte, filename string) (*rules.SearchResult, error) {
	var pyproject PyprojectToml

	// Parse the TOML content
	if err := toml.Unmarshal(content, &pyproject); err != nil {
		// Return no match instead of error for malformed TOML
		// This allows the scanner to continue with other files
		return &rules.SearchResult{Found: false}, nil
	}

	result := &rules.SearchResult{
		Found:    false,
		Source:   filename,
		Metadata: make(map[string]string),
	}

	// Try to extract Python version from different sections
	// Priority: PEP 621 > Poetry > PDM
	
	// 1. Try PEP 621 format ([project] section)
	if pyproject.Project != nil && pyproject.Project.RequiresPython != "" {
		version, err := extractVersionFromConstraint(pyproject.Project.RequiresPython)
		if err == nil && version != "" {
			result.Found = true
			result.Version = version
			result.RawValue = pyproject.Project.RequiresPython
			result.Confidence = 0.9
			result.Metadata["format"] = "PEP621"
			result.Metadata["constraint"] = pyproject.Project.RequiresPython
			
			if len(pyproject.Project.Dependencies) > 0 {
				result.Metadata["dependency_count"] = fmt.Sprintf("%d", len(pyproject.Project.Dependencies))
			}
			
			return result, nil
		}
	}

	// 2. Try Poetry format ([tool.poetry.dependencies])
	if pyproject.Tool != nil && pyproject.Tool.Poetry != nil {
		if pythonDep, ok := pyproject.Tool.Poetry.Dependencies["python"]; ok {
			constraint := ""
			
			// Handle different formats: string or map
			switch v := pythonDep.(type) {
			case string:
				constraint = v
			case map[string]interface{}:
				if ver, ok := v["version"].(string); ok {
					constraint = ver
				}
			}
			
			if constraint != "" {
				version, err := extractVersionFromConstraint(constraint)
				if err == nil && version != "" {
					result.Found = true
					result.Version = version
					result.RawValue = constraint
					result.Confidence = 0.9
					result.Metadata["format"] = "Poetry"
					result.Metadata["constraint"] = constraint
					
					// Count dependencies (excluding python itself)
					depCount := len(pyproject.Tool.Poetry.Dependencies) - 1
					if depCount > 0 {
						result.Metadata["dependency_count"] = fmt.Sprintf("%d", depCount)
					}
					
					return result, nil
				}
			}
		}
	}

	// 3. Try PDM format (uses [project] section like PEP 621)
	// PDM uses the same format as PEP 621, so it's already handled above

	// No Python version found
	return result, nil
}

// extractVersionFromConstraint extracts a Python version from a version constraint
// Handles common formats:
// - "^3.11" -> "3.11"
// - ">=3.11" -> "3.11"
// - "~=3.11.0" -> "3.11.0"
// - ">=3.10,<3.12" -> "3.10"
// - "3.11.*" -> "3.11"
// - "==3.11.5" -> "3.11.5"
func extractVersionFromConstraint(constraint string) (string, error) {
	// Clean up whitespace
	constraint = strings.TrimSpace(constraint)
	
	if constraint == "" {
		return "", fmt.Errorf("empty constraint")
	}

	// Pattern to match version numbers
	// Matches: 3.11, 3.11.0, 3.11.5, etc.
	versionPattern := regexp.MustCompile(`(\d+\.\d+(?:\.\d+)?)`)
	
	// Common constraint formats:
	// ^3.11, >=3.11, ~=3.11.0, ==3.11.5, 3.11.*, etc.
	
	// Find the first version number in the constraint
	matches := versionPattern.FindStringSubmatch(constraint)
	if len(matches) > 1 {
		return matches[1], nil
	}

	return "", fmt.Errorf("no version found in constraint: %s", constraint)
}

// GetPyprojectTomlRule returns a SearchRule for pyproject.toml parsing
// This is a convenience function for creating the rule
func GetPyprojectTomlRule() *rules.SearchRule {
	return rules.NewRuleBuilder("pyproject-toml").
		Description("Extracts Python version and dependencies from pyproject.toml (Poetry, PDM, PEP 621)").
		Priority(10). // High priority - explicit configuration file
		FilePattern("pyproject.toml").
		RequiredContent(`(requires-python|python\s*=)`). // Pre-filter: only parse if contains python version
		MaxFileSize(1024 * 1024). // Don't parse files > 1MB
		Parser(ParsePyprojectToml).
		Tags("config", "toml", "dependencies", "poetry", "pdm", "pep621").
		MustBuild()
}
