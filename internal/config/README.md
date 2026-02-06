# Configuration File Support

The GitLab Python Scanner supports loading search rules from YAML or JSON configuration files. This allows users to define custom search rules without modifying code.

## Features

- **YAML and JSON Support**: Load rules from `.yaml`, `.yml`, or `.json` files
- **Flexible Parser System**: Built-in parsers for common formats plus custom regex support
- **Validation**: Automatic validation of configuration structure and patterns
- **Extensible**: Easy to add new parser types through the parser registry

## Configuration File Structure

### Basic Structure

```yaml
version: "1.0"

# Global settings (optional)
settings:
  default_enabled: true
  default_priority: 50

# Search rules
rules:
  - name: rule-name
    description: Human-readable description
    priority: 10  # Lower = higher priority
    enabled: true
    tags:
      - tag1
      - tag2
    match:
      file_pattern: "*.txt"
      path_pattern: "^/some/path/.*"  # Optional regex
      required_content: "pattern"  # Optional regex
      max_file_size: 1024  # In bytes
    parser:
      type: parser_type
      config:
        key: value
```

### JSON Format

The same structure can be used in JSON format:

```json
{
  "version": "1.0",
  "settings": {
    "default_enabled": true,
    "default_priority": 50
  },
  "rules": [
    {
      "name": "rule-name",
      "description": "Description",
      "priority": 10,
      "enabled": true,
      "tags": ["tag1"],
      "match": {
        "file_pattern": "*.txt"
      },
      "parser": {
        "type": "simple_version"
      }
    }
  ]
}
```

## Built-in Parser Types

### 1. simple_version

Reads the entire file content as a version string.

**Configuration:**
```yaml
parser:
  type: simple_version
  config:
    confidence: 1.0  # Optional, default 1.0
    trim_whitespace: true  # Optional, default true
```

**Example:**
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

### 2. regex

Uses regex pattern matching to extract version information.

**Configuration:**
```yaml
parser:
  type: regex
  config:
    pattern: 'python-(?P<version>\d+\.\d+\.\d+)'  # Required
    version_group: version  # Optional, default "version"
    confidence: 0.8  # Optional, default 0.5
```

**Example:**
```yaml
- name: dockerfile-python
  description: Extract Python version from Dockerfile
  priority: 30
  match:
    file_pattern: "Dockerfile*"
  parser:
    type: regex
    config:
      pattern: 'FROM python:(?P<version>\d+\.\d+(?:\.\d+)?)'
      version_group: version
      confidence: 0.7
```

### 3. pyproject_toml

Parses `pyproject.toml` files to extract Python version requirements.

**Configuration:**
```yaml
parser:
  type: pyproject_toml
  config: {}  # No additional config needed
```

**Example:**
```yaml
- name: pyproject-toml
  description: Parse pyproject.toml for Python version
  priority: 20
  match:
    file_pattern: "pyproject.toml"
  parser:
    type: pyproject_toml
```

## Match Conditions

### file_pattern
Glob pattern or exact filename to match.

Examples:
- `".python-version"` - Exact match
- `"*.toml"` - All TOML files
- `"Dockerfile*"` - Dockerfile and variants

### path_pattern (Optional)
Regex pattern for matching full file paths.

Examples:
- `"^Dockerfile$"` - Exact path match
- `".*/.gitlab-ci.yml"` - GitLab CI files anywhere

### required_content (Optional)
Regex pattern that must exist in the file. Used for optimization.

Examples:
- `"\\[tool\\.poetry\\]"` - Must contain poetry section
- `"python"` - Must mention python

### max_file_size (Optional)
Maximum file size to process in bytes. Prevents parsing large files.

Example:
- `1024` - 1 KB max
- `1048576` - 1 MB max

## Usage Examples

### Loading a Configuration

```go
import "github.com/gbjohnso/gitlab-python-scanner/internal/config"

// Load config file
cfg, err := config.LoadConfig("rules.yaml")
if err != nil {
    log.Fatal(err)
}

// Validate config
if err := cfg.Validate(); err != nil {
    log.Fatal(err)
}

// Convert to rule registry
parserRegistry := config.NewDefaultParserRegistry()
registry, err := cfg.ToRegistry(parserRegistry)
if err != nil {
    log.Fatal(err)
}

// Use the registry
result, err := registry.ExecuteFirstMatch(ctx, content, filename, filepath)
```

### Saving Configuration

```go
import "github.com/gbjohnso/gitlab-python-scanner/internal/config"

// Create config
cfg := &config.Config{
    Version: "1.0",
    Rules: []config.RuleConfig{
        // ... define rules
    },
}

// Save as YAML
err := config.SaveConfig(cfg, "rules.yaml")

// Save as JSON
err := config.SaveConfig(cfg, "rules.json")
```

## Example Configuration Files

See the `examples/` directory for complete configuration examples:

- `examples/basic-rules.yaml` - Basic Python version detection rules
- `examples/advanced-rules.yaml` - Advanced configuration with custom parsers
- `examples/docker-rules.yaml` - Dockerfile-specific rules

## Extending with Custom Parsers

You can register custom parser types:

```go
registry := config.NewDefaultParserRegistry()

// Register custom parser
registry.RegisterParser("my_parser", func(config map[string]interface{}) (rules.ParserFunc, error) {
    // Create and return a parser function
    return func(content []byte, filename string) (*rules.SearchResult, error) {
        // Custom parsing logic
        return &rules.SearchResult{
            Found: true,
            Version: "1.0.0",
            Confidence: 1.0,
        }, nil
    }, nil
})
```

## Validation

The configuration loader automatically validates:

- ✅ Config version is present
- ✅ At least one rule is defined
- ✅ Rule names are unique
- ✅ Required fields are present
- ✅ Regex patterns compile correctly
- ✅ Parser types are registered

## Migration from Code-based Rules

To migrate from code-based rules to configuration files:

1. Export existing rules using `config.FromRegistry(registry)`
2. Save to a YAML or JSON file
3. Review and customize the generated config
4. Load the config file instead of building rules in code

Example:

```go
// Export existing registry
cfg := config.FromRegistry(existingRegistry)

// Save to file
config.SaveConfig(cfg, "rules.yaml")
```

## Best Practices

1. **Start Simple**: Begin with basic file patterns and simple parsers
2. **Use Priority**: Set lower priority for more specific rules
3. **Enable Selectively**: Disable rules you don't need
4. **Test Patterns**: Validate regex patterns before deploying
5. **Document Rules**: Use descriptive names and add comments (YAML only)
6. **Version Control**: Keep config files in version control
7. **Environment-Specific**: Use different configs for different environments

## Troubleshooting

### Config won't load
- Check file format (YAML vs JSON)
- Validate YAML/JSON syntax
- Ensure all required fields are present

### Rules not matching
- Check file_pattern syntax (glob vs exact)
- Verify path_pattern regex compiles
- Test required_content pattern

### Parser errors
- Verify parser type is registered
- Check parser-specific config requirements
- Validate regex patterns in parser config

## API Reference

### Types

- `Config` - Complete configuration structure
- `RuleConfig` - Individual rule configuration
- `MatchConfig` - Match condition configuration
- `ParserConfig` - Parser configuration
- `SettingsConfig` - Global settings

### Functions

- `LoadConfig(path string) (*Config, error)` - Load config from file
- `SaveConfig(config *Config, path string) error` - Save config to file
- `(c *Config) Validate() error` - Validate configuration
- `(c *Config) ToRegistry(ParserRegistry) (*rules.Registry, error)` - Convert to registry
- `FromRegistry(*rules.Registry) *Config` - Export registry to config

### Parser Registry

- `NewDefaultParserRegistry() *DefaultParserRegistry` - Create default registry
- `RegisterParser(type, factory)` - Register custom parser
- `GetParser(type, config) (ParserFunc, error)` - Get parser instance
- `ListParserTypes() []string` - List registered parser types
