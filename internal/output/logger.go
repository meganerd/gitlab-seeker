package output

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

// LogEntry represents a single log entry in the log file
type LogEntry struct {
	Timestamp       time.Time `json:"timestamp"`
	ProjectName     string    `json:"project_name"`
	ProjectPath     string    `json:"project_path,omitempty"`
	PythonVersion   string    `json:"python_version,omitempty"`
	DetectionSource string    `json:"detection_source,omitempty"`
	Error           string    `json:"error,omitempty"`
	Index           int       `json:"index"`
	TotalProjects   int       `json:"total_projects"`
}

// LogFormat defines the format for log file output
type LogFormat string

const (
	// FormatJSON outputs each result as a JSON line (JSONL/NDJSON)
	FormatJSON LogFormat = "json"
	// FormatText outputs each result as a formatted text line
	FormatText LogFormat = "text"
)

// FileLogger handles writing scan results to a log file
type FileLogger struct {
	file   *os.File
	format LogFormat
	mu     sync.Mutex // Protects concurrent writes
}

// NewFileLogger creates a new file logger that writes to the specified path
// The file is created if it doesn't exist, or truncated if it does
func NewFileLogger(path string, format LogFormat) (*FileLogger, error) {
	file, err := os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("failed to create log file: %w", err)
	}

	return &FileLogger{
		file:   file,
		format: format,
	}, nil
}

// NewFileLoggerAppend creates a new file logger that appends to an existing file
// The file is created if it doesn't exist
func NewFileLoggerAppend(path string, format LogFormat) (*FileLogger, error) {
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	return &FileLogger{
		file:   file,
		format: format,
	}, nil
}

// LogResult writes a single scan result to the log file
// This method is thread-safe and can be called concurrently from multiple goroutines
func (fl *FileLogger) LogResult(result *ScanResult) error {
	fl.mu.Lock()
	defer fl.mu.Unlock()

	entry := LogEntry{
		Timestamp:       time.Now(),
		ProjectName:     result.ProjectName,
		ProjectPath:     result.ProjectPath,
		PythonVersion:   result.PythonVersion,
		DetectionSource: result.DetectionSource,
		Index:           result.Index,
		TotalProjects:   result.TotalProjects,
	}

	if result.Error != nil {
		entry.Error = result.Error.Error()
	}

	switch fl.format {
	case FormatJSON:
		return fl.writeJSON(&entry)
	case FormatText:
		return fl.writeText(&entry)
	default:
		return fmt.Errorf("unknown log format: %s", fl.format)
	}
}

// writeJSON writes a log entry in JSON format (one JSON object per line)
func (fl *FileLogger) writeJSON(entry *LogEntry) error {
	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal log entry: %w", err)
	}

	_, err = fl.file.Write(append(data, '\n'))
	if err != nil {
		return fmt.Errorf("failed to write to log file: %w", err)
	}

	return nil
}

// writeText writes a log entry in text format
func (fl *FileLogger) writeText(entry *LogEntry) error {
	var line string

	if entry.Error != "" {
		line = fmt.Sprintf("[%s] [%d/%d] %s: Error - %s\n",
			entry.Timestamp.Format(time.RFC3339),
			entry.Index,
			entry.TotalProjects,
			entry.ProjectName,
			entry.Error,
		)
	} else if entry.PythonVersion == "" {
		line = fmt.Sprintf("[%s] [%d/%d] %s: Python not detected\n",
			entry.Timestamp.Format(time.RFC3339),
			entry.Index,
			entry.TotalProjects,
			entry.ProjectName,
		)
	} else {
		line = fmt.Sprintf("[%s] [%d/%d] %s: Python %s (from %s)\n",
			entry.Timestamp.Format(time.RFC3339),
			entry.Index,
			entry.TotalProjects,
			entry.ProjectName,
			entry.PythonVersion,
			entry.DetectionSource,
		)
	}

	_, err := fl.file.WriteString(line)
	if err != nil {
		return fmt.Errorf("failed to write to log file: %w", err)
	}

	return nil
}

// WriteHeader writes the initial header information to the log file
func (fl *FileLogger) WriteHeader(gitlabURL string, totalProjects int) error {
	fl.mu.Lock()
	defer fl.mu.Unlock()

	var header string
	timestamp := time.Now().Format(time.RFC3339)

	switch fl.format {
	case FormatJSON:
		// For JSON, write a header entry
		headerEntry := map[string]interface{}{
			"type":           "scan_started",
			"timestamp":      timestamp,
			"gitlab_url":     gitlabURL,
			"total_projects": totalProjects,
		}
		data, err := json.Marshal(headerEntry)
		if err != nil {
			return fmt.Errorf("failed to marshal header: %w", err)
		}
		header = string(data) + "\n"
	case FormatText:
		header = fmt.Sprintf("=== GitLab Python Scanner Log ===\n")
		header += fmt.Sprintf("Timestamp: %s\n", timestamp)
		header += fmt.Sprintf("GitLab URL: %s\n", gitlabURL)
		header += fmt.Sprintf("Total Projects: %d\n", totalProjects)
		header += fmt.Sprintf("=====================================\n\n")
	default:
		return fmt.Errorf("unknown log format: %s", fl.format)
	}

	_, err := fl.file.WriteString(header)
	if err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}

	return nil
}

// WriteSummary writes the final summary statistics to the log file
func (fl *FileLogger) WriteSummary(stats *ScanStatistics) error {
	fl.mu.Lock()
	defer fl.mu.Unlock()

	var summary string
	timestamp := time.Now().Format(time.RFC3339)

	switch fl.format {
	case FormatJSON:
		// For JSON, write a summary entry
		summaryEntry := map[string]interface{}{
			"type":               "scan_completed",
			"timestamp":          timestamp,
			"total_projects":     stats.TotalProjects,
			"python_projects":    stats.PythonProjects,
			"non_python_projects": stats.NonPythonProjects,
			"error_count":        stats.ErrorCount,
			"version_counts":     stats.VersionCounts,
		}
		data, err := json.Marshal(summaryEntry)
		if err != nil {
			return fmt.Errorf("failed to marshal summary: %w", err)
		}
		summary = string(data) + "\n"
	case FormatText:
		summary = fmt.Sprintf("\n=== Scan Summary ===\n")
		summary += fmt.Sprintf("Timestamp: %s\n", timestamp)
		summary += fmt.Sprintf("Total Projects: %d\n", stats.TotalProjects)
		summary += fmt.Sprintf("Python Projects: %d\n", stats.PythonProjects)
		summary += fmt.Sprintf("Non-Python Projects: %d\n", stats.NonPythonProjects)
		if stats.ErrorCount > 0 {
			summary += fmt.Sprintf("Errors: %d\n", stats.ErrorCount)
		}
		if len(stats.VersionCounts) > 0 {
			summary += fmt.Sprintf("\nPython Version Distribution:\n")
			for version, count := range stats.VersionCounts {
				summary += fmt.Sprintf("  %s: %d\n", version, count)
			}
		}
		summary += fmt.Sprintf("====================\n")
	default:
		return fmt.Errorf("unknown log format: %s", fl.format)
	}

	_, err := fl.file.WriteString(summary)
	if err != nil {
		return fmt.Errorf("failed to write summary: %w", err)
	}

	return nil
}

// Close closes the log file
// This should be called when logging is complete to ensure all data is flushed
func (fl *FileLogger) Close() error {
	fl.mu.Lock()
	defer fl.mu.Unlock()

	if fl.file != nil {
		err := fl.file.Close()
		fl.file = nil // Set to nil to prevent double-close
		return err
	}
	return nil
}

// Sync flushes any buffered data to the file
func (fl *FileLogger) Sync() error {
	fl.mu.Lock()
	defer fl.mu.Unlock()

	if fl.file != nil {
		return fl.file.Sync()
	}
	return nil
}
