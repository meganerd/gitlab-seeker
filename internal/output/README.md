# Output Package

The output package provides real-time streaming of scan results to the console, file logging, and statistics tracking.

## Features

### Console Streaming
- **Real-time Streaming**: Stream scan results to console as they are discovered
- **Thread-Safe**: Safe for concurrent use from multiple goroutines
- **Formatted Output**: Clean, readable output format matching project requirements
- **Flexible Writers**: Support custom writers for testing or redirection

### File Logging
- **Multiple Formats**: Text and JSON (JSONL/NDJSON) output formats
- **Persistent Storage**: Save scan results to log files for later analysis
- **Thread-Safe**: Safe for concurrent logging from multiple goroutines
- **Append Mode**: Option to append to existing log files
- **Structured Data**: JSON format enables easy parsing and analysis

### Statistics
- **Progress Tracking**: Track scan progress and aggregate statistics
- **Version Distribution**: Track count of each Python version detected
- **Error Tracking**: Record and report scanning errors

## Usage

### Basic Console Streaming

```go
package main

import (
    "github.com/gbjohnso/gitlab-python-scanner/internal/output"
)

func main() {
    // Create a console streamer
    streamer := output.NewConsoleStreamer()
    stats := output.NewScanStatistics()

    // Print header
    streamer.PrintHeader("https://gitlab.com/myorg", 10)

    // Stream results as they come in
    result := &output.ScanResult{
        ProjectName:     "my-project",
        PythonVersion:   "3.11.5",
        DetectionSource: ".python-version",
        Index:           1,
        TotalProjects:   10,
    }
    
    streamer.StreamResult(result)
    stats.RecordResult(result)

    // Print summary at the end
    streamer.PrintSummary(stats)
}
```

### Concurrent Streaming

The `ConsoleStreamer` is thread-safe and can be used from multiple goroutines:

```go
streamer := output.NewConsoleStreamer()
stats := output.NewScanStatistics()

var wg sync.WaitGroup

// Scan projects concurrently
for i, project := range projects {
    wg.Add(1)
    go func(idx int, proj Project) {
        defer wg.Done()
        
        // Scan the project (your logic here)
        version, source := detectPythonVersion(proj)
        
        // Stream result immediately
        result := &output.ScanResult{
            ProjectName:     proj.Name,
            PythonVersion:   version,
            DetectionSource: source,
            Index:           idx + 1,
            TotalProjects:   len(projects),
        }
        
        streamer.StreamResult(result)
        stats.RecordResult(result)
    }(i, project)
}

wg.Wait()
streamer.PrintSummary(stats)
```

### File Logging - Text Format

```go
package main

import (
    "log"
    "github.com/gbjohnso/gitlab-python-scanner/internal/output"
)

func main() {
    // Create a file logger with text format
    logger, err := output.NewFileLogger("scan_results.log", output.FormatText)
    if err != nil {
        log.Fatal(err)
    }
    defer logger.Close()

    // Write header
    logger.WriteHeader("https://gitlab.com/myorg", 10)

    // Log results
    result := &output.ScanResult{
        ProjectName:     "my-project",
        PythonVersion:   "3.11.5",
        DetectionSource: ".python-version",
        Index:           1,
        TotalProjects:   10,
    }
    logger.LogResult(result)

    // Write summary
    stats := output.NewScanStatistics()
    stats.RecordResult(result)
    logger.WriteSummary(stats)
}
```

**Example text output:**
```
=== GitLab Python Scanner Log ===
Timestamp: 2024-02-06T10:30:00Z
GitLab URL: https://gitlab.com/myorg
Total Projects: 10
=====================================

[2024-02-06T10:30:01Z] [1/10] my-project: Python 3.11.5 (from .python-version)
[2024-02-06T10:30:02Z] [2/10] frontend-app: Python not detected

=== Scan Summary ===
Timestamp: 2024-02-06T10:30:05Z
Total Projects: 2
Python Projects: 1
Non-Python Projects: 1

Python Version Distribution:
  3.11.5: 1
====================
```

### File Logging - JSON Format (JSONL/NDJSON)

```go
// Create a JSON format logger
logger, err := output.NewFileLogger("scan_results.jsonl", output.FormatJSON)
if err != nil {
    log.Fatal(err)
}
defer logger.Close()

// Write header
logger.WriteHeader("https://gitlab.com/myorg", 10)

// Log results - each result is a JSON line
logger.LogResult(&output.ScanResult{
    ProjectName:     "my-project",
    ProjectPath:     "/projects/my-project",
    PythonVersion:   "3.11.5",
    DetectionSource: ".python-version",
    Index:           1,
    TotalProjects:   10,
})

// Write summary
stats := output.NewScanStatistics()
logger.WriteSummary(stats)
```

**Example JSON output (JSONL):**
```json
{"type":"scan_started","timestamp":"2024-02-06T10:30:00Z","gitlab_url":"https://gitlab.com/myorg","total_projects":10}
{"timestamp":"2024-02-06T10:30:01Z","project_name":"my-project","project_path":"/projects/my-project","python_version":"3.11.5","detection_source":".python-version","index":1,"total_projects":10}
{"timestamp":"2024-02-06T10:30:02Z","project_name":"frontend-app","project_path":"/projects/frontend-app","index":2,"total_projects":10}
{"type":"scan_completed","timestamp":"2024-02-06T10:30:05Z","total_projects":2,"python_projects":1,"non_python_projects":1,"error_count":0,"version_counts":{"3.11.5":1}}
```

### Combined Console and File Output

```go
// Create both outputs
console := output.NewConsoleStreamer()
logger, err := output.NewFileLogger("scan.log", output.FormatText)
if err != nil {
    log.Fatal(err)
}
defer logger.Close()

stats := output.NewScanStatistics()

// Write to both
console.PrintHeader(gitlabURL, totalProjects)
logger.WriteHeader(gitlabURL, totalProjects)

for _, result := range scanResults {
    console.StreamResult(result)  // Real-time console output
    logger.LogResult(result)      // Persistent log file
    stats.RecordResult(result)
}

console.PrintSummary(stats)
logger.WriteSummary(stats)
```

### Append to Existing Log

```go
// Append to existing log file instead of overwriting
logger, err := output.NewFileLoggerAppend("scan.log", output.FormatText)
if err != nil {
    log.Fatal(err)
}
defer logger.Close()
```

## Types

### LogEntry

Represents a single log entry in JSON format:

```go
type LogEntry struct {
    Timestamp       time.Time // When the entry was logged
    ProjectName     string    // Name of the project
    ProjectPath     string    // Full path of the project (optional)
    PythonVersion   string    // Detected Python version (optional)
    DetectionSource string    // Where version was detected (optional)
    Error           string    // Error message if scan failed (optional)
    Index           int       // Sequential index of this result
    TotalProjects   int       // Total number of projects being scanned
}
```

### LogFormat

Defines the log file output format:

```go
type LogFormat string

const (
    FormatJSON LogFormat = "json"  // JSONL/NDJSON format (one JSON object per line)
    FormatText LogFormat = "text"  // Human-readable text format
)
```

### ScanResult

Represents a single scan result for a project:

```go
type ScanResult struct {
    ProjectName       string // Name of the project
    ProjectPath       string // Full path of the project
    PythonVersion     string // Detected Python version (e.g., "3.11.5")
    DetectionSource   string // Where the version was detected (e.g., ".python-version")
    Error             error  // Any error encountered during scanning
    Index             int    // Sequential index of this result
    TotalProjects     int    // Total number of projects being scanned
}
```

### ScanStatistics

Tracks aggregate statistics across all scan results:

```go
type ScanStatistics struct {
    TotalProjects      int            // Total number of projects scanned
    PythonProjects     int            // Number of projects with Python detected
    NonPythonProjects  int            // Number of projects without Python
    ErrorCount         int            // Number of errors encountered
    VersionCounts      map[string]int // Count of each Python version detected
}
```

## Output Format

### Successful Detection
```
[1/42] project-alpha: Python 3.11.5 (from .python-version)
```

### Python Not Detected
```
[2/42] frontend-app: Python not detected
```

### Error During Scan
```
[3/42] failed-project: Error - network timeout
```

### Header
```
Found 42 projects in organization
```

### Summary
```
Scan complete: 42 projects, 28 Python projects, 14 non-Python
```

### Summary with Errors
```
Scan complete: 50 projects, 30 Python projects, 15 non-Python
Errors encountered: 5
```

## Testing

The package includes comprehensive tests with 90.3% coverage:

```bash
go test ./internal/output/... -v -cover
```

Run examples:

```bash
go test ./internal/output/... -v -run Example
```

## Thread Safety

Both `ConsoleStreamer` and `FileLogger` are thread-safe:

- **ConsoleStreamer.StreamResult()** uses a mutex to ensure thread-safe concurrent writes
- **FileLogger.LogResult()** uses a mutex to ensure thread-safe concurrent writes

This allows you to call them from multiple goroutines without worrying about garbled output or file corruption.

Example of concurrent usage:

```go
streamer := output.NewConsoleStreamer()
logger, _ := output.NewFileLogger("scan.log", output.FormatText)
defer logger.Close()

var wg sync.WaitGroup
for _, project := range projects {
    wg.Add(1)
    go func(proj Project) {
        defer wg.Done()
        
        result := scanProject(proj)
        
        // Both calls are thread-safe
        streamer.StreamResult(result)
        logger.LogResult(result)
    }(project)
}
wg.Wait()
```

## Custom Writers

For testing or custom output destinations, use `NewConsoleStreamerWithWriter`:

```go
import (
    "bytes"
    "testing"
)

func TestMyScanner(t *testing.T) {
    buf := &bytes.Buffer{}
    streamer := output.NewConsoleStreamerWithWriter(buf)
    
    // ... use streamer in tests ...
    
    output := buf.String()
    // ... verify output ...
}
```
