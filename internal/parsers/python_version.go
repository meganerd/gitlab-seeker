package parsers

import (
	"bufio"
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/gbjohnso/gitlab-python-scanner/internal/rules"
)

// ============================================================================
// .python-version Parser
// ============================================================================

// ParsePythonVersionFile extracts Python version from .python-version files.
// This is the most explicit and reliable source of Python version information.
//
// Format examples:
//   3.11
//   3.11.5
//   python-3.11.5
//
// Returns:
// - Confidence: 1.0 (most reliable source)
func ParsePythonVersionFile(content []byte, filename string) (*rules.SearchResult, error) {
	versionStr := strings.TrimSpace(string(content))
	
	// Remove common prefixes
	versionStr = strings.TrimPrefix(versionStr, "python-")
	versionStr = strings.TrimPrefix(versionStr, "Python-")
	versionStr = strings.TrimPrefix(versionStr, "py")
	
	// Extract version number
	version, err := extractPythonVersion(versionStr)
	if err != nil || version == "" {
		return &rules.SearchResult{Found: false}, nil
	}
	
	return &rules.SearchResult{
		Found:      true,
		Version:    version,
		Source:     filename,
		Confidence: 1.0,
		RawValue:   versionStr,
		Metadata:   map[string]string{"source_type": "explicit_version_file"},
	}, nil
}

// GetPythonVersionFileRule returns a SearchRule for .python-version files
func GetPythonVersionFileRule() *rules.SearchRule {
	return rules.NewRuleBuilder("python-version-file").
		Description("Extracts Python version from .python-version file").
		Priority(1). // Highest priority - most explicit source
		FilePattern(".python-version").
		MaxFileSize(1024). // Small files only
		Parser(ParsePythonVersionFile).
		Tags("explicit", "version-file").
		MustBuild()
}

// ============================================================================
// runtime.txt Parser (Heroku)
// ============================================================================

// ParseRuntimeTxt extracts Python version from runtime.txt files.
// Common in Heroku deployments.
//
// Format examples:
//   python-3.11.5
//   python-3.11
//
// Returns:
// - Confidence: 0.95 (very explicit, Heroku-specific)
func ParseRuntimeTxt(content []byte, filename string) (*rules.SearchResult, error) {
	versionStr := strings.TrimSpace(string(content))
	
	// runtime.txt typically has format: python-3.11.5
	versionStr = strings.TrimPrefix(versionStr, "python-")
	versionStr = strings.TrimPrefix(versionStr, "Python-")
	
	version, err := extractPythonVersion(versionStr)
	if err != nil || version == "" {
		return &rules.SearchResult{Found: false}, nil
	}
	
	return &rules.SearchResult{
		Found:      true,
		Version:    version,
		Source:     filename,
		Confidence: 0.95,
		RawValue:   string(content),
		Metadata:   map[string]string{"source_type": "heroku_runtime"},
	}, nil
}

// GetRuntimeTxtRule returns a SearchRule for runtime.txt files
func GetRuntimeTxtRule() *rules.SearchRule {
	return rules.NewRuleBuilder("runtime-txt").
		Description("Extracts Python version from runtime.txt (Heroku)").
		Priority(2).
		FilePattern("runtime.txt").
		RequiredContent(`python-?\d+\.\d+`).
		MaxFileSize(1024).
		Parser(ParseRuntimeTxt).
		Tags("explicit", "heroku", "deployment").
		MustBuild()
}

// ============================================================================
// setup.py Parser
// ============================================================================

// ParseSetupPy extracts Python version from setup.py files.
// Looks for python_requires argument in setup() call.
//
// Format examples:
//   python_requires='>=3.11'
//   python_requires=">=3.10,<4.0"
//
// Returns:
// - Confidence: 0.9 (explicit configuration)
func ParseSetupPy(content []byte, filename string) (*rules.SearchResult, error) {
	contentStr := string(content)
	
	// Look for python_requires in setup() call
	// Pattern: python_requires=['"]([^'"]+)['"]
	pattern := regexp.MustCompile(`python_requires\s*=\s*['"]([^'"]+)['"]`)
	matches := pattern.FindStringSubmatch(contentStr)
	
	if len(matches) < 2 {
		return &rules.SearchResult{Found: false}, nil
	}
	
	constraint := matches[1]
	version, err := extractVersionFromConstraint(constraint)
	if err != nil || version == "" {
		return &rules.SearchResult{Found: false}, nil
	}
	
	return &rules.SearchResult{
		Found:      true,
		Version:    version,
		Source:     filename,
		Confidence: 0.9,
		RawValue:   constraint,
		Metadata: map[string]string{
			"source_type": "setup_py",
			"constraint":  constraint,
		},
	}, nil
}

// GetSetupPyRule returns a SearchRule for setup.py files
func GetSetupPyRule() *rules.SearchRule {
	return rules.NewRuleBuilder("setup-py").
		Description("Extracts Python version from setup.py").
		Priority(8).
		FilePattern("setup.py").
		RequiredContent(`python_requires`).
		MaxFileSize(1024 * 1024). // 1MB
		Parser(ParseSetupPy).
		Tags("config", "python", "packaging").
		MustBuild()
}

// ============================================================================
// Pipfile Parser
// ============================================================================

// PipfileStruct represents the structure of a Pipfile
type PipfileStruct struct {
	Requires *PipfileRequires `toml:"requires"`
}

// PipfileRequires represents the [requires] section of a Pipfile
type PipfileRequires struct {
	PythonVersion     string `toml:"python_version"`
	PythonFullVersion string `toml:"python_full_version"`
}

// ParsePipfile extracts Python version from Pipfile.
//
// Format examples:
//   [requires]
//   python_version = "3.11"
//
// Returns:
// - Confidence: 0.9 (explicit configuration)
func ParsePipfile(content []byte, filename string) (*rules.SearchResult, error) {
	var pipfile PipfileStruct
	
	if err := toml.Unmarshal(content, &pipfile); err != nil {
		return &rules.SearchResult{Found: false}, nil
	}
	
	if pipfile.Requires == nil {
		return &rules.SearchResult{Found: false}, nil
	}
	
	// Check python_full_version first (more specific)
	versionStr := pipfile.Requires.PythonFullVersion
	if versionStr == "" {
		versionStr = pipfile.Requires.PythonVersion
	}
	
	if versionStr == "" {
		return &rules.SearchResult{Found: false}, nil
	}
	
	version, err := extractPythonVersion(versionStr)
	if err != nil || version == "" {
		return &rules.SearchResult{Found: false}, nil
	}
	
	return &rules.SearchResult{
		Found:      true,
		Version:    version,
		Source:     filename,
		Confidence: 0.9,
		RawValue:   versionStr,
		Metadata: map[string]string{
			"source_type": "pipfile",
			"format":      "pipenv",
		},
	}, nil
}

// GetPipfileRule returns a SearchRule for Pipfile
func GetPipfileRule() *rules.SearchRule {
	return rules.NewRuleBuilder("pipfile").
		Description("Extracts Python version from Pipfile").
		Priority(9).
		FilePattern("Pipfile").
		RequiredContent(`python_version|python_full_version`).
		MaxFileSize(1024 * 1024). // 1MB
		Parser(ParsePipfile).
		Tags("config", "pipenv", "dependencies").
		MustBuild()
}

// ============================================================================
// requirements.txt Parser
// ============================================================================

// ParseRequirementsTxt extracts Python version from requirements.txt comments.
// This is less reliable but still useful.
//
// Format examples:
//   # Python 3.11
//   # Requires Python >= 3.11
//
// Returns:
// - Confidence: 0.6 (inferred from comments)
func ParseRequirementsTxt(content []byte, filename string) (*rules.SearchResult, error) {
	scanner := bufio.NewScanner(bytes.NewReader(content))
	
	// Common comment patterns that indicate Python version
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`#\s*[Pp]ython\s+(\d+\.\d+(?:\.\d+)?)`),
		regexp.MustCompile(`#\s*[Rr]equires\s+[Pp]ython\s*[><=]+\s*(\d+\.\d+(?:\.\d+)?)`),
		regexp.MustCompile(`#\s*[Pp]y\s*[><=]+\s*(\d+\.\d+(?:\.\d+)?)`),
	}
	
	for scanner.Scan() {
		line := scanner.Text()
		
		for _, pattern := range patterns {
			matches := pattern.FindStringSubmatch(line)
			if len(matches) > 1 {
				version := matches[1]
				return &rules.SearchResult{
					Found:      true,
					Version:    version,
					Source:     filename,
					Confidence: 0.6,
					RawValue:   line,
					Metadata: map[string]string{
						"source_type": "requirements_comment",
					},
				}, nil
			}
		}
	}
	
	return &rules.SearchResult{Found: false}, nil
}

// GetRequirementsTxtRule returns a SearchRule for requirements.txt
func GetRequirementsTxtRule() *rules.SearchRule {
	return rules.NewRuleBuilder("requirements-txt").
		Description("Extracts Python version from requirements.txt comments").
		Priority(15). // Lower priority - less reliable
		FilePattern("requirements*.txt").
		RequiredContent(`[Pp]ython`).
		MaxFileSize(1024 * 1024). // 1MB
		Parser(ParseRequirementsTxt).
		Tags("dependencies", "comments", "inferred").
		MustBuild()
}

// ============================================================================
// .gitlab-ci.yml Parser
// ============================================================================

// ParseGitLabCI extracts Python version from .gitlab-ci.yml files.
// Looks for image specifications with Python versions.
//
// Format examples:
//   image: python:3.11
//   image: python:3.11-slim
//   image: python:3.11.5-alpine
//
// Returns:
// - Confidence: 0.75 (CI configuration)
func ParseGitLabCI(content []byte, filename string) (*rules.SearchResult, error) {
	contentStr := string(content)
	
	// Pattern to match Python docker images
	// image: python:3.11, image: python:3.11-slim, etc.
	pattern := regexp.MustCompile(`image:\s*python:(\d+\.\d+(?:\.\d+)?)`)
	matches := pattern.FindStringSubmatch(contentStr)
	
	if len(matches) < 2 {
		return &rules.SearchResult{Found: false}, nil
	}
	
	version := matches[1]
	
	return &rules.SearchResult{
		Found:      true,
		Version:    version,
		Source:     filename,
		Confidence: 0.75,
		RawValue:   matches[0],
		Metadata: map[string]string{
			"source_type": "gitlab_ci",
			"image":       matches[0],
		},
	}, nil
}

// GetGitLabCIRule returns a SearchRule for .gitlab-ci.yml
func GetGitLabCIRule() *rules.SearchRule {
	return rules.NewRuleBuilder("gitlab-ci").
		Description("Extracts Python version from .gitlab-ci.yml").
		Priority(12).
		FilePattern(".gitlab-ci.yml").
		RequiredContent(`image:\s*python:`).
		MaxFileSize(1024 * 1024). // 1MB
		Parser(ParseGitLabCI).
		Tags("ci", "gitlab", "docker").
		MustBuild()
}

// ============================================================================
// Dockerfile Parser
// ============================================================================

// ParseDockerfile extracts Python version from Dockerfile FROM statements.
//
// Format examples:
//   FROM python:3.11
//   FROM python:3.11-slim
//   FROM python:3.11.5-alpine
//
// Returns:
// - Confidence: 0.8 (deployment configuration)
func ParseDockerfile(content []byte, filename string) (*rules.SearchResult, error) {
	scanner := bufio.NewScanner(bytes.NewReader(content))
	
	// Pattern to match FROM python:version
	pattern := regexp.MustCompile(`^FROM\s+python:(\d+\.\d+(?:\.\d+)?)`)
	
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		matches := pattern.FindStringSubmatch(line)
		
		if len(matches) > 1 {
			version := matches[1]
			return &rules.SearchResult{
				Found:      true,
				Version:    version,
				Source:     filename,
				Confidence: 0.8,
				RawValue:   line,
				Metadata: map[string]string{
					"source_type": "dockerfile",
					"from_image":  line,
				},
			}, nil
		}
	}
	
	return &rules.SearchResult{Found: false}, nil
}

// GetDockerfileRule returns a SearchRule for Dockerfile
func GetDockerfileRule() *rules.SearchRule {
	return rules.NewRuleBuilder("dockerfile").
		Description("Extracts Python version from Dockerfile").
		Priority(11).
		FilePattern("Dockerfile*").
		RequiredContent(`FROM\s+python:`).
		MaxFileSize(1024 * 1024). // 1MB
		Parser(ParseDockerfile).
		Tags("docker", "deployment", "container").
		MustBuild()
}

// ============================================================================
// tox.ini Parser
// ============================================================================

// ToxIniStruct represents a simple structure for tox.ini
type ToxIniStruct struct {
	Tox *ToxSection `toml:"tox"`
}

// ToxSection represents the [tox] section
type ToxSection struct {
	EnvList string `toml:"envlist"`
}

// ParseToxIni extracts Python version from tox.ini files.
//
// Format examples:
//   [tox]
//   envlist = py311,py312
//
// Returns:
// - Confidence: 0.7 (testing configuration)
func ParseToxIni(content []byte, filename string) (*rules.SearchResult, error) {
	// Try TOML parsing first (newer format)
	var toxIni ToxIniStruct
	if err := toml.Unmarshal(content, &toxIni); err == nil {
		if toxIni.Tox != nil && toxIni.Tox.EnvList != "" {
			version := extractPythonVersionFromToxEnv(toxIni.Tox.EnvList)
			if version != "" {
				return &rules.SearchResult{
					Found:      true,
					Version:    version,
					Source:     filename,
					Confidence: 0.7,
					RawValue:   toxIni.Tox.EnvList,
					Metadata: map[string]string{
						"source_type": "tox_ini",
						"envlist":     toxIni.Tox.EnvList,
					},
				}, nil
			}
		}
	}
	
	// Fall back to INI-style parsing
	contentStr := string(content)
	pattern := regexp.MustCompile(`envlist\s*=\s*([^\n]+)`)
	matches := pattern.FindStringSubmatch(contentStr)
	
	if len(matches) < 2 {
		return &rules.SearchResult{Found: false}, nil
	}
	
	envlist := matches[1]
	version := extractPythonVersionFromToxEnv(envlist)
	if version == "" {
		return &rules.SearchResult{Found: false}, nil
	}
	
	return &rules.SearchResult{
		Found:      true,
		Version:    version,
		Source:     filename,
		Confidence: 0.7,
		RawValue:   envlist,
		Metadata: map[string]string{
			"source_type": "tox_ini",
			"envlist":     envlist,
		},
	}, nil
}

// extractPythonVersionFromToxEnv extracts version from tox envlist
// Examples: py311 -> 3.11, py312 -> 3.12, py39 -> 3.9
func extractPythonVersionFromToxEnv(envlist string) string {
	// Pattern to match py39, py310, py311, etc.
	pattern := regexp.MustCompile(`py(\d)(\d+)`)
	matches := pattern.FindStringSubmatch(envlist)
	
	if len(matches) < 3 {
		return ""
	}
	
	// Convert py311 -> 3.11
	major := matches[1]
	minor := matches[2]
	
	return fmt.Sprintf("%s.%s", major, minor)
}

// GetToxIniRule returns a SearchRule for tox.ini
func GetToxIniRule() *rules.SearchRule {
	return rules.NewRuleBuilder("tox-ini").
		Description("Extracts Python version from tox.ini").
		Priority(13).
		FilePattern("tox.ini").
		RequiredContent(`envlist`).
		MaxFileSize(1024 * 1024). // 1MB
		Parser(ParseToxIni).
		Tags("testing", "tox", "config").
		MustBuild()
}

// ============================================================================
// Helper Functions
// ============================================================================

// extractPythonVersion extracts a clean Python version from a string
// Handles: 3.11, 3.11.5, python-3.11, etc.
func extractPythonVersion(versionStr string) (string, error) {
	versionStr = strings.TrimSpace(versionStr)
	
	if versionStr == "" {
		return "", fmt.Errorf("empty version string")
	}
	
	// Pattern to match version numbers: 3.11, 3.11.5, etc.
	pattern := regexp.MustCompile(`^(\d+\.\d+(?:\.\d+)?)`)
	matches := pattern.FindStringSubmatch(versionStr)
	
	if len(matches) < 2 {
		return "", fmt.Errorf("no version found in: %s", versionStr)
	}
	
	return matches[1], nil
}
