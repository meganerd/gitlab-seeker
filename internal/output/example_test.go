package output_test

import (
	"fmt"
	"time"

	"github.com/gbjohnso/gitlab-python-scanner/internal/output"
)

// ExampleConsoleStreamer demonstrates real-time streaming of scan results
func ExampleConsoleStreamer() {
	// Create a console streamer
	streamer := output.NewConsoleStreamer()
	stats := output.NewScanStatistics()

	// Print header
	totalProjects := 5
	streamer.PrintHeader("https://gitlab.com/myorg", totalProjects)

	// Simulate scanning projects with results streaming in real-time
	projects := []struct {
		name    string
		version string
		source  string
	}{
		{"project-alpha", "3.11.5", ".python-version"},
		{"legacy-app", "2.7.18", "setup.py"},
		{"frontend-app", "", ""},
		{"backend-api", "3.10.0", "pyproject.toml"},
		{"data-pipeline", "3.9.16", "Pipfile"},
	}

	for i, proj := range projects {
		result := &output.ScanResult{
			ProjectName:     proj.name,
			PythonVersion:   proj.version,
			DetectionSource: proj.source,
			Index:           i + 1,
			TotalProjects:   totalProjects,
		}

		// Stream result immediately (in real app, this happens as each project is scanned)
		streamer.StreamResult(result)
		stats.RecordResult(result)

		// Simulate scanning delay
		time.Sleep(10 * time.Millisecond)
	}

	// Print summary
	streamer.PrintSummary(stats)

	// Output:
	// Found 5 projects in organization
	//
	// [1/5] project-alpha: Python 3.11.5 (from .python-version)
	// [2/5] legacy-app: Python 2.7.18 (from setup.py)
	// [3/5] frontend-app: Python not detected
	// [4/5] backend-api: Python 3.10.0 (from pyproject.toml)
	// [5/5] data-pipeline: Python 3.9.16 (from Pipfile)
	//
	// Scan complete: 5 projects, 4 Python projects, 1 non-Python
}

// ExampleConsoleStreamer_concurrent demonstrates concurrent streaming
func ExampleConsoleStreamer_concurrent() {
	streamer := output.NewConsoleStreamer()
	
	// In a real scenario, you would use goroutines to scan projects concurrently
	// The mutex in ConsoleStreamer ensures thread-safe output
	fmt.Println("Concurrent scanning (results may arrive in any order):")
	
	result1 := &output.ScanResult{
		ProjectName:     "fast-project",
		PythonVersion:   "3.11.0",
		DetectionSource: "pyproject.toml",
		Index:           1,
		TotalProjects:   2,
	}
	
	result2 := &output.ScanResult{
		ProjectName:     "slow-project",
		PythonVersion:   "3.10.5",
		DetectionSource: ".python-version",
		Index:           2,
		TotalProjects:   2,
	}
	
	// These would normally be called from different goroutines
	streamer.StreamResult(result1)
	streamer.StreamResult(result2)
}
