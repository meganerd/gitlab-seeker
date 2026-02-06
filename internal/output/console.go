package output

import (
	"fmt"
	"io"
	"os"
	"sync"
)

// ScanResult represents a single scan result for a project
type ScanResult struct {
	ProjectName       string // Name of the project
	ProjectPath       string // Full path of the project
	PythonVersion     string // Detected Python version (e.g., "3.11.5")
	DetectionSource   string // Where the version was detected (e.g., ".python-version")
	Error             error  // Any error encountered during scanning
	Index             int    // Sequential index of this result
	TotalProjects     int    // Total number of projects being scanned
}

// ConsoleStreamer handles real-time streaming of scan results to console
type ConsoleStreamer struct {
	writer io.Writer
	mu     sync.Mutex // Protects concurrent writes
}

// NewConsoleStreamer creates a new console streamer that writes to stdout
func NewConsoleStreamer() *ConsoleStreamer {
	return &ConsoleStreamer{
		writer: os.Stdout,
	}
}

// NewConsoleStreamerWithWriter creates a console streamer with a custom writer
// Useful for testing or redirecting output
func NewConsoleStreamerWithWriter(w io.Writer) *ConsoleStreamer {
	return &ConsoleStreamer{
		writer: w,
	}
}

// StreamResult writes a single scan result to the console in real-time
// This method is thread-safe and can be called concurrently from multiple goroutines
func (cs *ConsoleStreamer) StreamResult(result *ScanResult) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	// Handle error cases
	if result.Error != nil {
		_, err := fmt.Fprintf(cs.writer, "[%d/%d] %s: Error - %v\n",
			result.Index,
			result.TotalProjects,
			result.ProjectName,
			result.Error,
		)
		return err
	}

	// Handle Python not detected
	if result.PythonVersion == "" {
		_, err := fmt.Fprintf(cs.writer, "[%d/%d] %s: Python not detected\n",
			result.Index,
			result.TotalProjects,
			result.ProjectName,
		)
		return err
	}

	// Handle successful detection
	_, err := fmt.Fprintf(cs.writer, "[%d/%d] %s: Python %s (from %s)\n",
		result.Index,
		result.TotalProjects,
		result.ProjectName,
		result.PythonVersion,
		result.DetectionSource,
	)
	return err
}

// PrintHeader writes the initial header information to the console
func (cs *ConsoleStreamer) PrintHeader(gitlabURL string, totalProjects int) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	_, err := fmt.Fprintf(cs.writer, "\nFound %d projects in organization\n\n", totalProjects)
	return err
}

// PrintSummary writes the final summary statistics to the console
func (cs *ConsoleStreamer) PrintSummary(stats *ScanStatistics) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	_, err := fmt.Fprintf(cs.writer, "\nScan complete: %d projects, %d Python projects, %d non-Python\n",
		stats.TotalProjects,
		stats.PythonProjects,
		stats.NonPythonProjects,
	)
	
	if stats.ErrorCount > 0 {
		fmt.Fprintf(cs.writer, "Errors encountered: %d\n", stats.ErrorCount)
	}
	
	return err
}

// ScanStatistics holds summary statistics for a scan operation
type ScanStatistics struct {
	TotalProjects      int            // Total number of projects scanned
	PythonProjects     int            // Number of projects with Python detected
	NonPythonProjects  int            // Number of projects without Python
	ErrorCount         int            // Number of errors encountered
	VersionCounts      map[string]int // Count of each Python version detected
}

// NewScanStatistics creates a new statistics tracker
func NewScanStatistics() *ScanStatistics {
	return &ScanStatistics{
		VersionCounts: make(map[string]int),
	}
}

// RecordResult updates statistics based on a scan result
func (ss *ScanStatistics) RecordResult(result *ScanResult) {
	ss.TotalProjects++
	
	if result.Error != nil {
		ss.ErrorCount++
		return
	}
	
	if result.PythonVersion == "" {
		ss.NonPythonProjects++
	} else {
		ss.PythonProjects++
		ss.VersionCounts[result.PythonVersion]++
	}
}
