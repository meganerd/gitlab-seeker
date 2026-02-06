package parsers

import (
	"bufio"
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"github.com/gbjohnso/gitlab-python-scanner/internal/rules"
)

// Requirement represents a single package requirement from requirements.txt
type Requirement struct {
	Name              string
	Specifier         string   // Version specifier (e.g., ">=2.28.0", "==1.2.3")
	Extras            []string // Optional extras (e.g., requests[security])
	Markers           string   // Environment markers (e.g., python_version>='3.8')
	Hashes            []string // Package hashes for verification
	IsEditable        bool     // -e or --editable flag
	IsRequirementFile bool     // -r or --requirement flag
	Comment           string   // Inline comment
}

// ParseRequirementsTxtDependencies extracts package dependencies from requirements.txt files.
//
// Supported formats:
//   - Simple: package-name
//   - With version: package-name==1.2.3
//   - Version specifiers: package>=1.0,<2.0
//   - Extras: package[extra1,extra2]>=1.0
//   - Environment markers: package>=1.0; python_version>='3.8'
//   - Hashes: package==1.0 --hash=sha256:abc123
//   - Editable installs: -e git+https://...
//   - Recursive requirements: -r other-requirements.txt
//   - Comments: # This is a comment
//   - Options: --index-url, --extra-index-url, etc.
//
// Returns:
// - SearchResult with dependency information in metadata
// - Confidence: 0.8 for explicit dependency declarations
func ParseRequirementsTxtDependencies(content []byte, filename string) (*rules.SearchResult, error) {
	scanner := bufio.NewScanner(bytes.NewReader(content))
	
	requirements := make([]Requirement, 0)
	var pythonVersion string
	var pythonVersionLine string
	
	// Patterns for Python version in comments
	pythonPatterns := []*regexp.Regexp{
		regexp.MustCompile(`#\s*[Pp]ython\s+(\d+\.\d+(?:\.\d+)?)`),
		regexp.MustCompile(`#\s*[Rr]equires\s+[Pp]ython\s*[><=]+\s*(\d+\.\d+(?:\.\d+)?)`),
		regexp.MustCompile(`#\s*[Pp]y\s*[><=]+\s*(\d+\.\d+(?:\.\d+)?)`),
	}
	
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		trimmedLine := strings.TrimSpace(line)
		
		// Skip empty lines
		if trimmedLine == "" {
			continue
		}
		
		// Check for Python version in comments (for backward compatibility)
		if pythonVersion == "" {
			for _, pattern := range pythonPatterns {
				matches := pattern.FindStringSubmatch(line)
				if len(matches) > 1 {
					pythonVersion = matches[1]
					pythonVersionLine = line
					break
				}
			}
		}
		
		// Skip pure comment lines
		if strings.HasPrefix(trimmedLine, "#") {
			continue
		}
		
		// Parse the requirement
		req, err := parseRequirementLine(trimmedLine)
		if err != nil {
			// Skip lines that can't be parsed (options, malformed, etc.)
			continue
		}
		
		if req != nil && !req.IsRequirementFile {
			requirements = append(requirements, *req)
		}
	}
	
	// Build result
	result := &rules.SearchResult{
		Source:   filename,
		Metadata: make(map[string]string),
	}
	
	// If we found a Python version in comments, include it
	if pythonVersion != "" {
		result.Found = true
		result.Version = pythonVersion
		result.RawValue = pythonVersionLine
		result.Confidence = 0.6 // Lower confidence for comment-based version
		result.Metadata["python_version_source"] = "comment"
	}
	
	// Add dependency information
	if len(requirements) > 0 {
		result.Found = true
		result.Metadata["dependency_count"] = fmt.Sprintf("%d", len(requirements))
		result.Metadata["has_dependencies"] = "true"
		
		// Store first few dependencies as examples (up to 5)
		maxExamples := 5
		if len(requirements) < maxExamples {
			maxExamples = len(requirements)
		}
		
		for i := 0; i < maxExamples; i++ {
			req := requirements[i]
			key := fmt.Sprintf("dependency_%d", i+1)
			value := req.Name
			if req.Specifier != "" {
				value += req.Specifier
			}
			result.Metadata[key] = value
		}
		
		// If we don't have a Python version from comments but we have dependencies,
		// set a minimal confidence level
		if result.Version == "" {
			result.Confidence = 0.5 // Dependencies found but no Python version
			result.Metadata["source_type"] = "dependencies_only"
		}
	}
	
	// If we found nothing, return not found
	if !result.Found {
		return &rules.SearchResult{Found: false}, nil
	}
	
	if result.Metadata["source_type"] == "" {
		result.Metadata["source_type"] = "requirements_txt"
	}
	
	return result, nil
}

// parseRequirementLine parses a single line from requirements.txt
func parseRequirementLine(line string) (*Requirement, error) {
	if line == "" {
		return nil, fmt.Errorf("empty line")
	}
	
	req := &Requirement{}
	
	// Check for editable install BEFORE removing comments
	// because editable URLs may contain # for fragments (e.g., #egg=package)
	if strings.HasPrefix(line, "-e ") || strings.HasPrefix(line, "--editable ") {
		req.IsEditable = true
		if strings.HasPrefix(line, "-e ") {
			line = strings.TrimSpace(line[3:]) // Remove "-e "
		} else {
			line = strings.TrimSpace(line[12:]) // Remove "--editable "
		}
		req.Name = line // For editable, store the full URL/path
		return req, nil
	}
	
	// Remove inline comments (but not for editable installs)
	commentIdx := strings.Index(line, "#")
	var comment string
	if commentIdx >= 0 {
		comment = strings.TrimSpace(line[commentIdx+1:])
		line = strings.TrimSpace(line[:commentIdx])
	}
	
	if line == "" {
		return nil, fmt.Errorf("empty line after comment removal")
	}
	
	req.Comment = comment
	
	// Check for recursive requirement files
	if strings.HasPrefix(line, "-r ") || strings.HasPrefix(line, "--requirement ") {
		req.IsRequirementFile = true
		return req, nil
	}
	
	// Check for options (--index-url, --extra-index-url, etc.)
	if strings.HasPrefix(line, "-") {
		return nil, fmt.Errorf("option line")
	}
	
	// Extract hashes if present
	if strings.Contains(line, "--hash") {
		parts := strings.Split(line, "--hash")
		line = strings.TrimSpace(parts[0])
		for i := 1; i < len(parts); i++ {
			hash := strings.TrimSpace(strings.Split(parts[i], " ")[0])
			hash = strings.TrimPrefix(hash, "=")
			if hash != "" {
				req.Hashes = append(req.Hashes, hash)
			}
		}
	}
	
	// Extract environment markers (after semicolon)
	if strings.Contains(line, ";") {
		parts := strings.SplitN(line, ";", 2)
		line = strings.TrimSpace(parts[0])
		req.Markers = strings.TrimSpace(parts[1])
	}
	
	// Parse package name, extras, and version specifier
	// Format: package-name[extra1,extra2]>=1.0,<2.0
	
	// Extract extras if present
	if strings.Contains(line, "[") {
		openIdx := strings.Index(line, "[")
		closeIdx := strings.Index(line, "]")
		if closeIdx > openIdx {
			extrasStr := line[openIdx+1 : closeIdx]
			req.Extras = strings.Split(extrasStr, ",")
			for i := range req.Extras {
				req.Extras[i] = strings.TrimSpace(req.Extras[i])
			}
			line = line[:openIdx] + line[closeIdx+1:]
		}
	}
	
	// Extract name and version specifier
	// Find where version specifier starts (first non-alphanumeric char except - and _)
	specifierPattern := regexp.MustCompile(`^([a-zA-Z0-9_-]+)(.*)$`)
	matches := specifierPattern.FindStringSubmatch(line)
	
	if len(matches) < 2 {
		return nil, fmt.Errorf("invalid requirement format")
	}
	
	req.Name = strings.TrimSpace(matches[1])
	if len(matches) > 2 {
		req.Specifier = strings.TrimSpace(matches[2])
	}
	
	if req.Name == "" {
		return nil, fmt.Errorf("empty package name")
	}
	
	return req, nil
}

// GetRequirementsTxtDependencyRule returns a SearchRule for requirements.txt dependency extraction
func GetRequirementsTxtDependencyRule() *rules.SearchRule {
	return rules.NewRuleBuilder("requirements-txt-dependencies").
		Description("Extracts package dependencies from requirements.txt files").
		Priority(15). // Lower priority - dependencies less critical than version
		FilePattern("requirements*.txt").
		MaxFileSize(5 * 1024 * 1024). // 5MB - requirements files can be larger
		Parser(ParseRequirementsTxtDependencies).
		Tags("dependencies", "requirements", "packages", "pip").
		MustBuild()
}
