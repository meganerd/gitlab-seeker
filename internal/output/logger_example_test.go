package output_test

import (
	"fmt"
	"log"

	"github.com/gbjohnso/gitlab-python-scanner/internal/output"
)

// ExampleFileLogger_text demonstrates basic usage of FileLogger with text format
func ExampleFileLogger_text() {
	// Create a text format logger
	logger, err := output.NewFileLogger("scan_results.log", output.FormatText)
	if err != nil {
		log.Fatal(err)
	}
	defer logger.Close()

	// Write header
	logger.WriteHeader("https://gitlab.com/myorg", 3)

	// Log results as they come in
	results := []*output.ScanResult{
		{
			ProjectName:     "backend-api",
			PythonVersion:   "3.11.5",
			DetectionSource: ".python-version",
			Index:           1,
			TotalProjects:   3,
		},
		{
			ProjectName:   "frontend-app",
			Index:         2,
			TotalProjects: 3,
		},
		{
			ProjectName:     "data-pipeline",
			PythonVersion:   "3.10.0",
			DetectionSource: "pyproject.toml",
			Index:           3,
			TotalProjects:   3,
		},
	}

	stats := output.NewScanStatistics()
	for _, result := range results {
		logger.LogResult(result)
		stats.RecordResult(result)
	}

	// Write summary
	logger.WriteSummary(stats)

	fmt.Println("Log file created: scan_results.log")
	// Output: Log file created: scan_results.log
}

// ExampleFileLogger_json demonstrates JSON format logging (JSONL/NDJSON)
func ExampleFileLogger_json() {
	// Create a JSON format logger
	logger, err := output.NewFileLogger("scan_results.jsonl", output.FormatJSON)
	if err != nil {
		log.Fatal(err)
	}
	defer logger.Close()

	// Write header
	logger.WriteHeader("https://gitlab.com/myorg", 2)

	// Log results
	logger.LogResult(&output.ScanResult{
		ProjectName:     "backend-api",
		ProjectPath:     "/projects/backend-api",
		PythonVersion:   "3.11.5",
		DetectionSource: ".python-version",
		Index:           1,
		TotalProjects:   2,
	})

	logger.LogResult(&output.ScanResult{
		ProjectName:   "frontend-app",
		ProjectPath:   "/projects/frontend-app",
		Index:         2,
		TotalProjects: 2,
	})

	// Write summary
	stats := output.NewScanStatistics()
	stats.TotalProjects = 2
	stats.PythonProjects = 1
	stats.NonPythonProjects = 1
	logger.WriteSummary(stats)

	fmt.Println("JSONL log file created: scan_results.jsonl")
	// Output: JSONL log file created: scan_results.jsonl
}

// ExampleFileLogger_concurrent demonstrates concurrent logging
func ExampleFileLogger_concurrent() {
	logger, err := output.NewFileLogger("concurrent_scan.log", output.FormatText)
	if err != nil {
		log.Fatal(err)
	}
	defer logger.Close()

	// The logger is thread-safe, so you can log from multiple goroutines
	// This is useful when scanning projects concurrently

	logger.WriteHeader("https://gitlab.com/myorg", 5)

	// In a real scenario, you would use goroutines here
	// For demonstration, we'll just show sequential calls
	for i := 1; i <= 5; i++ {
		result := &output.ScanResult{
			ProjectName:     fmt.Sprintf("project-%d", i),
			PythonVersion:   "3.11.5",
			DetectionSource: ".python-version",
			Index:           i,
			TotalProjects:   5,
		}
		logger.LogResult(result)
	}

	fmt.Println("Concurrent scan complete")
	// Output: Concurrent scan complete
}

// ExampleFileLogger_withConsole demonstrates using both console and file logging
func ExampleFileLogger_withConsole() {
	// Create both console streamer and file logger
	console := output.NewConsoleStreamer()
	logger, err := output.NewFileLogger("combined_output.log", output.FormatText)
	if err != nil {
		log.Fatal(err)
	}
	defer logger.Close()

	// Initialize
	gitlabURL := "https://gitlab.com/myorg"
	totalProjects := 2

	console.PrintHeader(gitlabURL, totalProjects)
	logger.WriteHeader(gitlabURL, totalProjects)

	// Log to both console and file
	stats := output.NewScanStatistics()
	results := []*output.ScanResult{
		{
			ProjectName:     "backend-api",
			PythonVersion:   "3.11.5",
			DetectionSource: ".python-version",
			Index:           1,
			TotalProjects:   totalProjects,
		},
		{
			ProjectName:   "frontend-app",
			Index:         2,
			TotalProjects: totalProjects,
		},
	}

	for _, result := range results {
		console.StreamResult(result)
		logger.LogResult(result)
		stats.RecordResult(result)
	}

	// Print/write summary
	console.PrintSummary(stats)
	logger.WriteSummary(stats)

	fmt.Println("Combined output complete")
	// Output:
	// Found 2 projects in organization
	//
	// [1/2] backend-api: Python 3.11.5 (from .python-version)
	// [2/2] frontend-app: Python not detected
	//
	// Scan complete: 2 projects, 1 Python projects, 1 non-Python
	// Combined output complete
}
