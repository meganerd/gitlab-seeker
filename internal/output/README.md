# Output Package

The output package provides real-time streaming of scan results to the console and statistics tracking.

## Features

- **Real-time Streaming**: Stream scan results to console as they are discovered
- **Thread-Safe**: Safe for concurrent use from multiple goroutines
- **Formatted Output**: Clean, readable output format matching project requirements
- **Statistics Tracking**: Track scan progress and aggregate statistics
- **Flexible Writers**: Support custom writers for testing or redirection

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

## Types

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

The package includes comprehensive tests with 100% coverage:

```bash
go test ./internal/output/... -v -cover
```

Run examples:

```bash
go test ./internal/output/... -v -run Example
```

## Thread Safety

The `ConsoleStreamer.StreamResult()` method uses a mutex to ensure thread-safe concurrent writes. This allows you to call it from multiple goroutines without worrying about garbled output.

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
