# GitLab Python Version Scanner

A Go CLI tool to scan GitLab projects and detect Python versions across an organization using a flexible, rule-based search engine.

## Features

- **Flexible Rule Engine**: Configure custom search rules using YAML or JSON files
- **Built-in Parsers**: Pre-built parsers for common Python version sources
- **GitLab Integration**: Scans all projects in a GitLab group/organization
- **Multiple Detection Methods**: Detects Python versions from various file types
- **Configuration Files**: Load rules from external configuration files
- **Real-time Output**: Console output as projects are scanned
- **Concurrent Scanning**: Parallel project scanning for performance
- **Extensible Architecture**: Easy to add new parsers and rules

## Table of Contents

- [Installation](#installation)
- [Quick Start](#quick-start)
- [Rule Engine Architecture](#rule-engine-architecture)
- [Configuration Files](#configuration-files)
- [Built-in Parsers](#built-in-parsers)
- [Usage Examples](#usage-examples)
- [Python Detection](#python-detection)
- [Development](#development)

## Installation

### From Source

```bash
git clone https://github.com/gbjohnso/gitlab-python-scanner
cd gitlab-python-scanner
go build -o scanner ./cmd/scanner
```

### Build with Go

```bash
go build -o scanner ./cmd/scanner
```

## Quick Start

### Basic Scanning

```bash
# Scan all projects in an organization
./scanner --url https://gitlab.com/myorg --token YOUR_TOKEN

# With custom GitLab instance
./scanner --url https://gitlab.company.com/engineering --token YOUR_TOKEN

# Save results to log file
./scanner --url https://gitlab.com/myorg --token YOUR_TOKEN --log results.log
```

### Using Configuration Files

```bash
# Use custom rule configuration
./scanner --url https://gitlab.com/myorg --token YOUR_TOKEN --config rules.yaml

# Use example configuration
./scanner --url https://gitlab.com/myorg --token YOUR_TOKEN --config examples/basic-rules.yaml
```

### Environment Variables

```bash
export GITLAB_TOKEN=your_token_here
./scanner --url https://gitlab.com/myorg
```

## Rule Engine Architecture

The scanner uses a flexible rule-based engine that allows you to define custom search rules for detecting Python versions. Each rule specifies:

- **What files to match** (file patterns, path patterns)
- **How to parse content** (built-in or custom parsers)
- **Priority ordering** (which rules to try first)
- **Confidence levels** (how reliable the detection is)

### Core Components

#### SearchRule

A rule that defines when and how to extract Python version information:

```go
type SearchRule struct {
    Name        string          // Unique identifier
    Description string          // Human-readable description
    Priority    int            // Lower = higher priority (0 is highest)
    Condition   MatchCondition // When to apply this rule
    Parser      ParserFunc     // How to extract version
    Enabled     bool           // Whether rule is active
    Tags        []string       // Categorization tags
}
```

#### MatchCondition

Defines when a rule should be applied:

```go
type MatchCondition struct {
    FilePattern     string         // Glob: "*.toml", ".python-version"
    PathPattern     *regexp.Regexp // Optional regex for path
    RequiredContent *regexp.Regexp // Optional content pre-check
    MaxFileSize     int64          // Max file size to process
}
```

#### SearchResult

The output from applying a rule:

```go
type SearchResult struct {
    Found      bool              // Whether version was found
    Version    string            // Detected Python version
    Source     string            // Where it was found
    Confidence float64           // Confidence (0.0-1.0)
    RawValue   string            // Raw extracted value
    Metadata   map[string]string // Additional information
}
```

### Priority System

Rules are evaluated in priority order:

| Priority Range | Category | Examples |
|----------------|----------|----------|
| 0-9 | Highest - Explicit version files | `.python-version` |
| 10-29 | High - Configuration files | `pyproject.toml`, `setup.py` |
| 30-49 | Medium - Build/dependency files | `Pipfile`, `requirements.txt` |
| 50-69 | Lower - Inferred versions | `Dockerfile`, CI configs |
| 70+ | Lowest - Fallback methods | Heuristics |

### Confidence Levels

Indicate detection reliability:

- **1.0**: Explicit version file (`.python-version`)
- **0.9**: Configuration with exact version (`pyproject.toml`)
- **0.7**: Build files with constraints (`setup.py`)
- **0.5**: Inferred from tools (`Dockerfile`, CI)
- **0.3**: Heuristic detection

## Configuration Files

The scanner supports loading rules from YAML or JSON configuration files, allowing you to customize detection without modifying code.

### Basic Structure

```yaml
version: "1.0"

# Global settings (optional)
settings:
  default_enabled: true
  default_priority: 50

# Search rules
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
        trim_whitespace: true
```

### Match Conditions

#### file_pattern
Glob pattern or exact filename:
- `".python-version"` - Exact match
- `"*.toml"` - All TOML files
- `"Dockerfile*"` - Dockerfile variants

#### path_pattern (Optional)
Regex for full file paths:
- `"^Dockerfile$"` - Exact path
- `".*/.gitlab-ci.yml"` - GitLab CI files

#### required_content (Optional)
Regex that must exist in file (optimization):
- `"\\[tool\\.poetry\\]"` - Poetry section
- `"python"` - Must mention python

#### max_file_size (Optional)
Maximum file size in bytes:
- `1024` - 1 KB max
- `1048576` - 1 MB max

### Example Configurations

See the `examples/` directory:
- `examples/basic-rules.yaml` - Basic Python version detection
- `examples/basic-rules.json` - Same rules in JSON format

### Loading Configuration

```go
import "github.com/gbjohnso/gitlab-python-scanner/internal/config"

// Load config file
cfg, err := config.LoadConfig("rules.yaml")
if err != nil {
    log.Fatal(err)
}

// Validate
if err := cfg.Validate(); err != nil {
    log.Fatal(err)
}

// Convert to rule registry
parserRegistry := config.NewDefaultParserRegistry()
registry, err := cfg.ToRegistry(parserRegistry)
if err != nil {
    log.Fatal(err)
}
```

## Built-in Parsers

The scanner includes several pre-built parsers for common file formats.

### 1. simple_version

Reads entire file content as a version string.

**Configuration:**
```yaml
parser:
  type: simple_version
  config:
    confidence: 1.0
    trim_whitespace: true
```

**Example:**
```yaml
- name: python-version-file
  match:
    file_pattern: ".python-version"
  parser:
    type: simple_version
```

### 2. regex

Uses regex pattern matching to extract versions.

**Configuration:**
```yaml
parser:
  type: regex
  config:
    pattern: 'FROM python:(?P<version>\d+\.\d+(?:\.\d+)?)'
    version_group: version  # Optional, default "version"
    confidence: 0.7
```

**Example:**
```yaml
- name: dockerfile-python
  match:
    file_pattern: "Dockerfile*"
  parser:
    type: regex
    config:
      pattern: 'FROM python:(?P<version>\d+\.\d+(?:\.\d+)?)'
      confidence: 0.7
```

### 3. pyproject_toml

Parses `pyproject.toml` files for Python version requirements.

**Supported Formats:**
- **PEP 621** (Standard Python packaging)
- **Poetry** (Dependency management)
- **PDM** (Modern package manager)

**Configuration:**
```yaml
parser:
  type: pyproject_toml
  config: {}  # No additional config needed
```

**Example:**
```yaml
- name: pyproject-toml
  match:
    file_pattern: "pyproject.toml"
    required_content: "\\[tool\\.poetry\\]|\\[project\\]"
  parser:
    type: pyproject_toml
```

**Supported Version Constraints:**

| Format | Example | Extracted Version |
|--------|---------|-------------------|
| Caret | `^3.11` | `3.11` |
| Greater than | `>=3.10` | `3.10` |
| Exact | `==3.11.5` | `3.11.5` |
| Compatible | `~=3.11.0` | `3.11.0` |
| Range | `>=3.10,<3.12` | `3.10` |
| Wildcard | `3.11.*` | `3.11` |

**Example Files:**

Poetry format:
```toml
[tool.poetry.dependencies]
python = "^3.11"
fastapi = "^0.104.0"
```

PEP 621 format:
```toml
[project]
requires-python = ">=3.10"
dependencies = ["requests>=2.28.0"]
```

## Usage Examples

### Example 1: Simple Version File

```yaml
- name: python-version-file
  description: Parse .python-version file
  priority: 10
  match:
    file_pattern: ".python-version"
  parser:
    type: simple_version
    config:
      confidence: 1.0
```

### Example 2: Dockerfile with Pattern Matching

```yaml
- name: dockerfile-python
  description: Extract Python from Dockerfile
  priority: 50
  match:
    file_pattern: "Dockerfile*"
    required_content: "FROM.*python"
  parser:
    type: regex
    config:
      pattern: 'FROM python:(?P<version>\d+\.\d+(?:\.\d+)?)'
      confidence: 0.7
```

### Example 3: Setup.py Python Requires

```yaml
- name: setup-py-python-requires
  description: Extract python_requires from setup.py
  priority: 30
  match:
    file_pattern: "setup.py"
    required_content: "python_requires"
  parser:
    type: regex
    config:
      pattern: 'python_requires\s*=\s*[''"]>=?\s*(?P<version>\d+\.\d+(?:\.\d+)?)[''"]'
      confidence: 0.8
```

### Example 4: Requirements.txt Comments

```yaml
- name: requirements-txt-python-comment
  description: Look for Python version in comments
  priority: 40
  match:
    file_pattern: "requirements*.txt"
    required_content: "# [Pp]ython"
  parser:
    type: regex
    config:
      pattern: '#\s*[Pp]ython\s*(?:version|requires)?:?\s*(?P<version>\d+\.\d+(?:\.\d+)?)'
      confidence: 0.6
```

## Python Detection

The scanner detects Python versions from various file types:

### Highest Priority (Explicit)
1. **`.python-version`** - pyenv/asdf version file
2. **`runtime.txt`** - Heroku/platform runtime

### High Priority (Configuration)
3. **`pyproject.toml`** - Modern Python projects (Poetry, PEP 621, PDM)
4. **`setup.py`** - Traditional Python packages
5. **`setup.cfg`** - Setup configuration files

### Medium Priority (Dependencies)
6. **`Pipfile`** - Pipenv projects
7. **`requirements.txt`** - Common dependencies (with version comments)
8. **`tox.ini`** - Testing configuration

### Lower Priority (Inferred)
9. **`Dockerfile`** - Container definitions
10. **`.gitlab-ci.yml`** - CI/CD configuration
11. **`.github/workflows/*.yml`** - GitHub Actions

### Detection Process

1. **Rule Matching**: Each file is checked against rules in priority order
2. **Content Pre-filtering**: Files are pre-checked for required content (optimization)
3. **Parsing**: First matching rule's parser extracts the version
4. **Confidence Scoring**: Result includes confidence level
5. **Result Metadata**: Additional context (format, constraints, etc.)

## Development

### Project Structure

```
gitlab-project-search/
├── .beads/                      # Beads issue tracker database
├── cmd/
│   └── scanner/
│       └── main.go              # CLI entry point
├── internal/
│   ├── config/                  # Configuration file support
│   │   ├── config.go            # Config structure and loader
│   │   ├── loader.go            # YAML/JSON parsing
│   │   ├── parser_registry.go  # Parser type registry
│   │   ├── validation.go        # Config validation
│   │   └── README.md            # Config documentation
│   ├── parsers/                 # Built-in parser implementations
│   │   ├── pyproject.go         # pyproject.toml parser
│   │   ├── pyproject_test.go    # Comprehensive tests (95.7% coverage)
│   │   └── README.md            # Parser documentation
│   ├── rules/                   # Rule engine framework
│   │   ├── rule.go              # SearchRule and core types
│   │   ├── builder.go           # RuleBuilder fluent API
│   │   ├── registry.go          # Rule registry and execution
│   │   ├── rule_test.go         # Tests (90.4% coverage)
│   │   └── README.md            # Rules documentation
│   ├── gitlab/
│   │   ├── client.go            # GitLab API client
│   │   ├── auth.go              # Authentication
│   │   └── file_fetcher.go      # File content fetching
│   ├── scanner/
│   │   ├── scanner.go           # Main scanning logic
│   │   └── detector.go          # Version detection
│   └── output/
│       ├── console.go           # Console output
│       └── logger.go            # File logging
├── examples/
│   ├── basic-rules.yaml         # Example YAML configuration
│   └── basic-rules.json         # Example JSON configuration
├── go.mod
├── go.sum
├── README.md                    # This file
├── AGENTS.md                    # AI agent instructions
└── LICENSE
```

### Building

```bash
# Build the scanner
go build -o scanner ./cmd/scanner

# Build with specific flags
go build -ldflags="-s -w" -o scanner ./cmd/scanner
```

### Testing

```bash
# Run all tests
go test ./...

# Run with coverage
go test ./... -cover

# Generate coverage report
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out

# Run specific package tests
go test ./internal/rules -v
go test ./internal/parsers -v
go test ./internal/config -v
```

### Test Coverage

Current test coverage:
- **Rules Package**: 90.4% of statements
- **Parsers Package**: 95.7% of statements
- **Config Package**: Well-tested validation and loading

### Creating Custom Parsers

You can create custom parsers by implementing the `ParserFunc` signature:

```go
package main

import (
    "github.com/gbjohnso/gitlab-python-scanner/internal/rules"
    "regexp"
)

// Custom parser for a specific file format
func customParser(content []byte, filename string) (*rules.SearchResult, error) {
    // Your parsing logic here
    pattern := regexp.MustCompile(`version:\s*(\d+\.\d+)`)
    matches := pattern.FindSubmatch(content)
    
    if len(matches) < 2 {
        return &rules.SearchResult{Found: false}, nil
    }
    
    return &rules.SearchResult{
        Found:      true,
        Version:    string(matches[1]),
        Source:     filename,
        Confidence: 0.8,
        RawValue:   string(matches[0]),
        Metadata: map[string]string{
            "parser": "custom",
        },
    }, nil
}

// Create a rule using the custom parser
func main() {
    rule := rules.NewRuleBuilder("custom-rule").
        FilePattern("version.txt").
        Parser(customParser).
        Priority(20).
        Tags("custom").
        MustBuild()
    
    // Use the rule...
}
```

### Registering Custom Parser Types

To make custom parsers available in configuration files:

```go
package main

import (
    "github.com/gbjohnso/gitlab-python-scanner/internal/config"
    "github.com/gbjohnso/gitlab-python-scanner/internal/rules"
)

func main() {
    // Create parser registry
    registry := config.NewDefaultParserRegistry()
    
    // Register custom parser type
    registry.RegisterParser("my_custom_parser", func(cfg map[string]interface{}) (rules.ParserFunc, error) {
        // Extract config values
        pattern := cfg["pattern"].(string)
        
        // Return parser function
        return func(content []byte, filename string) (*rules.SearchResult, error) {
            // Use pattern to parse content
            // ...
            return result, nil
        }, nil
    })
    
    // Now "my_custom_parser" can be used in config files
    // Load config with custom registry
    cfg, _ := config.LoadConfig("rules.yaml")
    ruleRegistry, _ := cfg.ToRegistry(registry)
}
```

### Command Line Flags

| Flag | Description | Required | Default |
|------|-------------|----------|---------|
| `--url` | GitLab URL including org/group | Yes | - |
| `--token` | GitLab API token | Yes | - |
| `--config` | Path to rules config file (YAML/JSON) | No | Built-in rules |
| `--log` | Path to log file for output | No | - |
| `--concurrency` | Number of concurrent scans | No | 5 |
| `--timeout` | API timeout in seconds | No | 30 |

### Expected Output

```
GitLab Python Version Scanner
==============================

Scanning: https://gitlab.com/myorg
Config: examples/basic-rules.yaml
Logging to: results.log

GitLab Base URL: https://gitlab.com/api/v4
Organization: myorg

Testing GitLab connection...
✓ Successfully connected to GitLab

Found 42 projects in organization

[1/42] project-alpha
  ✓ Python 3.11.5 (from .python-version, confidence: 1.0)

[2/42] legacy-app
  ✓ Python 2.7.18 (from setup.py, confidence: 0.8)

[3/42] frontend-app
  ✗ Python not detected

[4/42] backend-api
  ✓ Python 3.10.0 (from pyproject.toml [Poetry], confidence: 0.9)

[5/42] data-pipeline
  ✓ Python 3.9.16 (from Dockerfile, confidence: 0.7)

...

Scan complete!
  Total projects: 42
  Python projects: 28
  Non-Python projects: 14
  Average confidence: 0.85
```

## Advanced Usage

### Using the Rule Registry Programmatically

```go
package main

import (
    "context"
    "log"
    
    "github.com/gbjohnso/gitlab-python-scanner/internal/rules"
    "github.com/gbjohnso/gitlab-python-scanner/internal/parsers"
)

func main() {
    // Create a rule registry
    registry := rules.NewRegistry()
    
    // Add built-in rules
    registry.AddRule(parsers.GetPyprojectTomlRule())
    
    // Add custom rules
    rule := rules.NewRuleBuilder("custom").
        FilePattern("*.py").
        Parser(myParser).
        MustBuild()
    registry.AddRule(rule)
    
    // Execute against content
    ctx := context.Background()
    content := []byte("python = \"^3.11\"")
    
    result, err := registry.ExecuteFirstMatch(ctx, content, "pyproject.toml", "pyproject.toml")
    if err != nil {
        log.Fatal(err)
    }
    
    if result.Found {
        log.Printf("Found Python %s (confidence: %.1f)", result.Version, result.Confidence)
    }
}
```

### Filtering and Prioritizing Rules

```go
// Get only enabled rules
enabledRules := registry.GetRules()

// Get rules by tag
configRules := registry.GetRulesByTag("config-file")

// Get rules by priority range (0-29 = high priority)
highPriorityRules := []*rules.SearchRule{}
for _, rule := range registry.GetRules() {
    if rule.Priority < 30 {
        highPriorityRules = append(highPriorityRules, rule)
    }
}
```

### Exporting Configuration

```go
package main

import (
    "github.com/gbjohnso/gitlab-python-scanner/internal/config"
    "github.com/gbjohnso/gitlab-python-scanner/internal/rules"
)

func main() {
    // Create registry with rules
    registry := rules.NewRegistry()
    // Add rules...
    
    // Export to config
    cfg := config.FromRegistry(registry)
    
    // Save as YAML
    if err := config.SaveConfig(cfg, "exported-rules.yaml"); err != nil {
        log.Fatal(err)
    }
    
    // Save as JSON
    if err := config.SaveConfig(cfg, "exported-rules.json"); err != nil {
        log.Fatal(err)
    }
}
```

## Troubleshooting

### Configuration won't load

**Problem**: Config file fails to load  
**Solution**: 
- Verify YAML/JSON syntax is valid
- Check file exists and is readable
- Ensure all required fields are present (version, rules)

### Rules not matching files

**Problem**: Files aren't being parsed  
**Solution**:
- Check `file_pattern` syntax (glob patterns)
- Verify `path_pattern` regex if used
- Test `required_content` regex
- Check file size isn't exceeding `max_file_size`

### Parser errors

**Problem**: Parser fails or returns no results  
**Solution**:
- Verify parser type is registered (`simple_version`, `regex`, `pyproject_toml`)
- Check parser config requirements
- Test regex patterns separately
- Review file content format matches parser expectations

### Low confidence results

**Problem**: Results have low confidence scores  
**Solution**:
- Use more specific rules (lower priority number)
- Prefer explicit version files (`.python-version`)
- Check if file format is ambiguous
- Consider adding custom parser for specific format

## Best Practices

### 1. Start with Built-in Rules

Use the provided `examples/basic-rules.yaml` as a starting point:

```bash
cp examples/basic-rules.yaml my-rules.yaml
# Edit my-rules.yaml to customize
./scanner --url ... --config my-rules.yaml
```

### 2. Use Priority Effectively

Order rules from most specific to most general:
- 0-9: Explicit version files
- 10-29: Configuration files
- 30-49: Build files
- 50+: Inferred/heuristic

### 3. Optimize with Content Pre-filtering

Add `required_content` to skip irrelevant files:

```yaml
match:
  file_pattern: "*.toml"
  required_content: "\\[tool\\.poetry\\]"  # Only parse Poetry files
```

### 4. Set Appropriate File Size Limits

Prevent parsing large files:

```yaml
match:
  file_pattern: "setup.py"
  max_file_size: 102400  # 100KB max
```

### 5. Document Custom Rules

Use descriptive names and comments:

```yaml
# Custom rule for internal company standards
- name: company-python-config
  description: Parse company-specific Python version file
  priority: 5
```

### 6. Test Rules Before Deployment

Validate configuration before using in production:

```bash
# Test loading config
go run test-config.go rules.yaml

# Run on small subset first
./scanner --url gitlab.com/test-group --config rules.yaml
```

## References

### Internal Documentation

- [Configuration Files](internal/config/README.md) - Detailed config documentation
- [Parsers](internal/parsers/README.md) - Built-in parser reference
- [Rules Engine](internal/rules/README.md) - Rule framework details

### External Resources

- [PEP 621 - Project Metadata](https://peps.python.org/pep-0621/)
- [Poetry Documentation](https://python-poetry.org/docs/)
- [PDM Documentation](https://pdm.fming.dev/)
- [GitLab API Documentation](https://docs.gitlab.com/ee/api/)

## Task Tracking with Beads

This project uses [Beads](https://github.com/beadnet/beads-mcp-server) for issue tracking. See [AGENTS.md](AGENTS.md) for details.

```bash
# See available work
bd ready

# Show all tasks
bd list

# View task details
bd show gitlab-python-scanner-19

# Update task status
bd update gitlab-python-scanner-19 --status in_progress
```

## Contributing

1. Check available tasks: `bd ready`
2. Claim a task: `bd update <task-id> --status in_progress --assignee <your-name>`
3. Make your changes
4. Run tests: `go test ./...`
5. Update documentation as needed
6. Complete task: `bd close <task-id> --reason "Implemented feature X"`

## License

MIT License - see LICENSE file

## Author

Created by gbjohnso

## Acknowledgments

- Built with [Go](https://golang.org/)
- Configuration parsing with [go-yaml](https://github.com/go-yaml/yaml)
- TOML parsing with [BurntSushi/toml](https://github.com/BurntSushi/toml)
- Task tracking with [Beads](https://github.com/beadnet/beads-mcp-server)

---

*This project uses AI-assisted development tracked with Beads*
