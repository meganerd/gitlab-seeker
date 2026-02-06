# Rules Package

The `rules` package provides a flexible and extensible framework for defining search rules that can parse file content to extract Python version information.

## Overview

This package implements a rule engine for the GitLab Python Scanner. It allows you to define configurable search rules that specify:
- Which files to match (by pattern)
- How to parse their content
- Priority ordering for rule evaluation
- Confidence levels for results

## Core Components

### SearchRule

The main structure representing a search rule:

```go
type SearchRule struct {
    Name        string           // Unique identifier
    Description string           // Human-readable description
    Priority    int             // Lower number = higher priority (0 is highest)
    Condition   MatchCondition  // When to apply this rule
    Parser      ParserFunc      // How to extract version information
    Enabled     bool            // Whether rule is active
    Tags        []string        // Categorization tags
}
```

### MatchCondition

Defines when a rule should be applied:

```go
type MatchCondition struct {
    FilePattern     string         // Glob pattern: "*.toml", ".python-version"
    PathPattern     *regexp.Regexp // Optional regex for full path matching
    RequiredContent *regexp.Regexp // Optional content pre-check (optimization)
    MaxFileSize     int64          // Maximum file size to process (0 = unlimited)
}
```

### SearchResult

The output from applying a rule:

```go
type SearchResult struct {
    Found      bool              // Whether a version was found
    Version    string            // The detected Python version
    Source     string            // Where it was found
    Confidence float64           // Confidence level (0.0-1.0)
    RawValue   string            // Raw extracted value (for debugging)
    Metadata   map[string]string // Additional information
}
```

### ParserFunc

The function signature for parsers:

```go
type ParserFunc func(content []byte, filename string) (*SearchResult, error)
```

## Usage Examples

### Example 1: Simple File Pattern Rule

```go
rule := NewRuleBuilder("python-version-file").
    Description("Detects Python version from .python-version file").
    Priority(0). // Highest priority - explicit version file
    FilePattern(".python-version").
    Parser(func(content []byte, filename string) (*SearchResult, error) {
        version := strings.TrimSpace(string(content))
        return &SearchResult{
            Found:      true,
            Version:    version,
            Source:     filename,
            Confidence: 1.0,
            RawValue:   version,
        }, nil
    }).
    Tags("explicit", "version-file").
    Build()
```

### Example 2: TOML File with Content Filter

```go
rule := NewRuleBuilder("pyproject-toml").
    Description("Extracts Python version from pyproject.toml").
    Priority(10).
    FilePattern("pyproject.toml").
    RequiredContent(`python\s*=`). // Only parse if contains "python ="
    MaxFileSize(1024 * 1024). // Don't parse files > 1MB
    Parser(parsePyprojectToml).
    Tags("config", "toml").
    Build()
```

### Example 3: Path-Based Matching

```go
rule := NewRuleBuilder("dockerfile").
    Description("Extracts Python version from Dockerfile").
    Priority(20).
    FilePattern("Dockerfile*"). // Matches Dockerfile, Dockerfile.dev, etc.
    PathPattern(`^.*/Dockerfile.*$`). // Additional path validation
    RequiredContent(`FROM\s+python:`). // Only parse if contains Python image
    Parser(parseDockerfile).
    Tags("docker", "inferred").
    Build()
```

### Example 4: Using MustBuild for Static Rules

```go
// MustBuild panics on error - useful for package-level initialization
var defaultRules = []*SearchRule{
    NewRuleBuilder("python-version").
        FilePattern(".python-version").
        Parser(parsePythonVersionFile).
        MustBuild(),
    
    NewRuleBuilder("runtime-txt").
        FilePattern("runtime.txt").
        Parser(parseRuntimeTxt).
        MustBuild(),
}
```

## Rule Methods

### Matches(filename, filepath string) bool

Checks if the rule should be applied to a given file:

```go
rule := NewRuleBuilder("test-rule").
    FilePattern("*.py").
    PathPattern(".*test.*").
    Parser(mockParser).
    MustBuild()

if rule.Matches("test_foo.py", "/project/tests/test_foo.py") {
    // Rule matches this file
}
```

### Apply(ctx context.Context, content []byte, filename string) (*SearchResult, error)

Executes the parser on file content:

```go
ctx := context.Background()
content, err := os.ReadFile(".python-version")
if err != nil {
    log.Fatal(err)
}

result, err := rule.Apply(ctx, content, ".python-version")
if err != nil {
    log.Fatal(err)
}

if result.Found {
    fmt.Printf("Found Python %s in %s\n", result.Version, result.Source)
}
```

### Validate() error

Validates that the rule is properly configured:

```go
err := rule.Validate()
if err != nil {
    log.Fatalf("Invalid rule: %v", err)
}
```

### Clone() *SearchRule

Creates a deep copy of the rule:

```go
customRule := defaultRule.Clone()
customRule.Priority = 5
customRule.Enabled = false
```

## RuleBuilder

The `RuleBuilder` provides a fluent interface for constructing rules:

```go
rule, err := NewRuleBuilder("my-rule").
    Description("Custom rule").
    Priority(10).
    FilePattern("*.txt").
    PathPattern(".*config.*").
    RequiredContent("python").
    MaxFileSize(10240).
    Parser(myParser).
    Tags("custom", "text").
    Enabled(true).
    Build()

if err != nil {
    log.Fatal(err)
}
```

### Builder Methods

- `Description(string)` - Sets the rule description
- `Priority(int)` - Sets priority (default: 50)
- `FilePattern(string)` - Sets file glob pattern
- `PathPattern(string)` - Sets path regex pattern
- `RequiredContent(string)` - Sets content pre-check regex
- `MaxFileSize(int64)` - Sets maximum file size
- `Parser(ParserFunc)` - Sets the parser function
- `Tags(...string)` - Sets categorization tags
- `Enabled(bool)` - Sets enabled state (default: true)
- `Build()` - Validates and returns the rule (returns error)
- `MustBuild()` - Validates and returns the rule (panics on error)

## Pattern Matching

### File Patterns (Glob-style)

Supports simple glob patterns:
- `*.py` - All Python files
- `test_*` - Files starting with "test_"
- `*file*` - Files containing "file"
- `?.txt` - Single character wildcard
- `.python-version` - Exact match

### Path Patterns (Regex)

Use Go regular expressions for path matching:
- `^.*/Dockerfile$` - Any Dockerfile in any directory
- `.*test.*` - Paths containing "test"
- `^/project/src/.*\.py$` - Python files in /project/src

### Content Patterns (Regex)

Pre-filter files before parsing:
- `python\s*=` - Contains "python ="
- `FROM\s+python:` - Dockerfile with Python base image
- `python_requires` - setup.py with python_requires

## Priority System

Rules are evaluated in priority order:
- **0-9**: Highest priority (explicit version files)
- **10-29**: High priority (configuration files)
- **30-49**: Medium priority (build/dependency files)
- **50-69**: Lower priority (inferred versions)
- **70+**: Lowest priority (fallback methods)

Example priority scheme:
```go
const (
    PriorityExplicit   = 0  // .python-version
    PriorityConfig     = 10 // pyproject.toml, setup.py
    PriorityDependency = 30 // Pipfile, requirements.txt
    PriorityInferred   = 50 // Dockerfile, CI config
    PriorityFallback   = 70 // Heuristics, file analysis
)
```

## Confidence Levels

Indicate how confident we are in the result:
- **1.0**: Explicit version file (.python-version)
- **0.9**: Configuration with exact version (pyproject.toml)
- **0.7**: Build files with version constraints (setup.py)
- **0.5**: Inferred from tools (Dockerfile, CI)
- **0.3**: Heuristic detection
- **0.0**: Uncertain/ambiguous

## File Size Limits

Prevent parsing large files:
```go
rule := NewRuleBuilder("setup-py").
    FilePattern("setup.py").
    MaxFileSize(1024 * 100). // 100KB max
    Parser(parseSetupPy).
    MustBuild()
```

Files exceeding the limit return an error without parsing.

## Testing

The package includes comprehensive tests:
- Pattern matching (glob and regex)
- Rule validation
- Rule application
- Builder functionality
- Cloning and immutability

Run tests:
```bash
go test ./internal/rules/...
```

With coverage:
```bash
go test ./internal/rules/... -cover
```

Coverage: **90.4%** of statements

## Best Practices

### 1. Use Descriptive Names

```go
// Good
NewRuleBuilder("pyproject-toml-poetry")

// Bad
NewRuleBuilder("rule1")
```

### 2. Set Appropriate Priorities

Order rules from most specific to most general:
```go
.python-version     Priority: 0  (most specific)
pyproject.toml      Priority: 10
Dockerfile          Priority: 50
file analysis       Priority: 70 (most general)
```

### 3. Use Content Pre-filtering

Avoid parsing irrelevant files:
```go
NewRuleBuilder("pyproject-toml").
    RequiredContent(`\[tool\.poetry\.dependencies\]`).
    // Only parse pyproject.toml files with Poetry config
```

### 4. Handle Parse Errors Gracefully

```go
Parser(func(content []byte, filename string) (*SearchResult, error) {
    version, err := extractVersion(content)
    if err != nil {
        // Return no match instead of error for malformed content
        return &SearchResult{Found: false}, nil
    }
    return &SearchResult{Found: true, Version: version}, nil
})
```

### 5. Populate Metadata

Include debugging information:
```go
return &SearchResult{
    Found:      true,
    Version:    "3.11.0",
    Source:     filename,
    Confidence: 1.0,
    RawValue:   rawContent,
    Metadata: map[string]string{
        "parser":    "pyproject-toml",
        "section":   "tool.poetry.dependencies",
        "line_number": "42",
    },
}
```

## Integration Example

Here's how rules would be used in the scanner:

```go
// 1. Define rules
rules := []*SearchRule{
    NewRuleBuilder("python-version").
        Priority(0).
        FilePattern(".python-version").
        Parser(parsePythonVersion).
        MustBuild(),
    
    NewRuleBuilder("pyproject-toml").
        Priority(10).
        FilePattern("pyproject.toml").
        RequiredContent(`python\s*=`).
        Parser(parsePyproject).
        MustBuild(),
}

// 2. For each file in project
for _, file := range projectFiles {
    for _, rule := range rules {
        if !rule.Matches(file.Name, file.Path) {
            continue
        }
        
        content, err := fetchFile(file)
        if err != nil {
            continue
        }
        
        result, err := rule.Apply(ctx, content, file.Name)
        if err != nil {
            log.Printf("Rule %s error: %v", rule.Name, err)
            continue
        }
        
        if result.Found {
            fmt.Printf("Found: Python %s (from %s, confidence %.1f)\n",
                result.Version, result.Source, result.Confidence)
            break // Stop after first match
        }
    }
}
```

## Future Enhancements

Potential improvements (out of scope for this task):
- Rule composition (AND/OR conditions)
- Async parser execution
- Caching of parsed results
- Rule statistics/metrics
- Dynamic rule loading from config files
- Rule dependencies (one rule requires another)

## References

- Task: `gitlab-python-scanner-11`
- Next Task: `gitlab-python-scanner-12` (Parser Registry)
- Related: `gitlab-python-scanner-10` (File Fetching)
