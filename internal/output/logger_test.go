package output

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestNewFileLogger(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	logger, err := NewFileLogger(logPath, FormatText)
	if err != nil {
		t.Fatalf("Failed to create file logger: %v", err)
	}
	defer logger.Close()

	if logger.file == nil {
		t.Error("Expected file to be initialized")
	}

	if logger.format != FormatText {
		t.Errorf("Expected format %s, got %s", FormatText, logger.format)
	}

	// Verify file was created
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Error("Expected log file to be created")
	}
}

func TestNewFileLoggerAppend(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	// Write initial content
	if err := os.WriteFile(logPath, []byte("initial content\n"), 0644); err != nil {
		t.Fatalf("Failed to create initial file: %v", err)
	}

	// Open in append mode
	logger, err := NewFileLoggerAppend(logPath, FormatText)
	if err != nil {
		t.Fatalf("Failed to create file logger: %v", err)
	}
	defer logger.Close()

	// Write new content
	result := &ScanResult{
		ProjectName:   "test-project",
		PythonVersion: "3.11.5",
		DetectionSource: ".python-version",
		Index:         1,
		TotalProjects: 1,
	}
	if err := logger.LogResult(result); err != nil {
		t.Fatalf("Failed to log result: %v", err)
	}

	logger.Close()

	// Verify both old and new content exist
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "initial content") {
		t.Error("Expected initial content to be preserved")
	}
	if !strings.Contains(contentStr, "test-project") {
		t.Error("Expected new content to be appended")
	}
}

func TestFileLogger_LogResult_Text_Success(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	logger, err := NewFileLogger(logPath, FormatText)
	if err != nil {
		t.Fatalf("Failed to create file logger: %v", err)
	}
	defer logger.Close()

	result := &ScanResult{
		ProjectName:     "test-project",
		ProjectPath:     "/projects/test-project",
		PythonVersion:   "3.11.5",
		DetectionSource: ".python-version",
		Index:           1,
		TotalProjects:   10,
	}

	if err := logger.LogResult(result); err != nil {
		t.Fatalf("Failed to log result: %v", err)
	}

	logger.Close()

	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "test-project") {
		t.Error("Expected project name in log")
	}
	if !strings.Contains(contentStr, "3.11.5") {
		t.Error("Expected Python version in log")
	}
	if !strings.Contains(contentStr, ".python-version") {
		t.Error("Expected detection source in log")
	}
	if !strings.Contains(contentStr, "[1/10]") {
		t.Error("Expected progress indicator in log")
	}
}

func TestFileLogger_LogResult_Text_NotDetected(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	logger, err := NewFileLogger(logPath, FormatText)
	if err != nil {
		t.Fatalf("Failed to create file logger: %v", err)
	}
	defer logger.Close()

	result := &ScanResult{
		ProjectName:   "frontend-app",
		Index:         2,
		TotalProjects: 10,
	}

	if err := logger.LogResult(result); err != nil {
		t.Fatalf("Failed to log result: %v", err)
	}

	logger.Close()

	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "frontend-app") {
		t.Error("Expected project name in log")
	}
	if !strings.Contains(contentStr, "Python not detected") {
		t.Error("Expected 'Python not detected' message in log")
	}
}

func TestFileLogger_LogResult_Text_Error(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	logger, err := NewFileLogger(logPath, FormatText)
	if err != nil {
		t.Fatalf("Failed to create file logger: %v", err)
	}
	defer logger.Close()

	result := &ScanResult{
		ProjectName:   "failed-project",
		Error:         errors.New("network timeout"),
		Index:         3,
		TotalProjects: 10,
	}

	if err := logger.LogResult(result); err != nil {
		t.Fatalf("Failed to log result: %v", err)
	}

	logger.Close()

	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "failed-project") {
		t.Error("Expected project name in log")
	}
	if !strings.Contains(contentStr, "Error - network timeout") {
		t.Error("Expected error message in log")
	}
}

func TestFileLogger_LogResult_JSON_Success(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	logger, err := NewFileLogger(logPath, FormatJSON)
	if err != nil {
		t.Fatalf("Failed to create file logger: %v", err)
	}
	defer logger.Close()

	result := &ScanResult{
		ProjectName:     "test-project",
		ProjectPath:     "/projects/test-project",
		PythonVersion:   "3.11.5",
		DetectionSource: ".python-version",
		Index:           1,
		TotalProjects:   10,
	}

	if err := logger.LogResult(result); err != nil {
		t.Fatalf("Failed to log result: %v", err)
	}

	logger.Close()

	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	// Parse JSON
	var entry LogEntry
	if err := json.Unmarshal(content, &entry); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	if entry.ProjectName != "test-project" {
		t.Errorf("Expected project name 'test-project', got '%s'", entry.ProjectName)
	}
	if entry.PythonVersion != "3.11.5" {
		t.Errorf("Expected Python version '3.11.5', got '%s'", entry.PythonVersion)
	}
	if entry.DetectionSource != ".python-version" {
		t.Errorf("Expected detection source '.python-version', got '%s'", entry.DetectionSource)
	}
	if entry.Index != 1 {
		t.Errorf("Expected index 1, got %d", entry.Index)
	}
	if entry.TotalProjects != 10 {
		t.Errorf("Expected total projects 10, got %d", entry.TotalProjects)
	}
	if entry.Timestamp.IsZero() {
		t.Error("Expected timestamp to be set")
	}
}

func TestFileLogger_LogResult_JSON_NotDetected(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	logger, err := NewFileLogger(logPath, FormatJSON)
	if err != nil {
		t.Fatalf("Failed to create file logger: %v", err)
	}
	defer logger.Close()

	result := &ScanResult{
		ProjectName:   "frontend-app",
		Index:         2,
		TotalProjects: 10,
	}

	if err := logger.LogResult(result); err != nil {
		t.Fatalf("Failed to log result: %v", err)
	}

	logger.Close()

	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	// Parse JSON
	var entry LogEntry
	if err := json.Unmarshal(content, &entry); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	if entry.ProjectName != "frontend-app" {
		t.Errorf("Expected project name 'frontend-app', got '%s'", entry.ProjectName)
	}
	if entry.PythonVersion != "" {
		t.Errorf("Expected empty Python version, got '%s'", entry.PythonVersion)
	}
}

func TestFileLogger_LogResult_JSON_Error(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	logger, err := NewFileLogger(logPath, FormatJSON)
	if err != nil {
		t.Fatalf("Failed to create file logger: %v", err)
	}
	defer logger.Close()

	result := &ScanResult{
		ProjectName:   "failed-project",
		Error:         errors.New("network timeout"),
		Index:         3,
		TotalProjects: 10,
	}

	if err := logger.LogResult(result); err != nil {
		t.Fatalf("Failed to log result: %v", err)
	}

	logger.Close()

	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	// Parse JSON
	var entry LogEntry
	if err := json.Unmarshal(content, &entry); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	if entry.ProjectName != "failed-project" {
		t.Errorf("Expected project name 'failed-project', got '%s'", entry.ProjectName)
	}
	if entry.Error != "network timeout" {
		t.Errorf("Expected error 'network timeout', got '%s'", entry.Error)
	}
}

func TestFileLogger_LogResult_Concurrent(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	logger, err := NewFileLogger(logPath, FormatText)
	if err != nil {
		t.Fatalf("Failed to create file logger: %v", err)
	}
	defer logger.Close()

	// Simulate concurrent writes
	var wg sync.WaitGroup
	numGoroutines := 10

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			result := &ScanResult{
				ProjectName:   "project-" + string(rune('A'+index)),
				PythonVersion: "3.11.5",
				DetectionSource: ".python-version",
				Index:         index + 1,
				TotalProjects: numGoroutines,
			}
			if err := logger.LogResult(result); err != nil {
				t.Errorf("Failed to log result: %v", err)
			}
		}(i)
	}

	wg.Wait()
	logger.Close()

	// Verify file has all entries
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	if len(lines) != numGoroutines {
		t.Errorf("Expected %d lines, got %d", numGoroutines, len(lines))
	}
}

func TestFileLogger_WriteHeader_Text(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	logger, err := NewFileLogger(logPath, FormatText)
	if err != nil {
		t.Fatalf("Failed to create file logger: %v", err)
	}
	defer logger.Close()

	if err := logger.WriteHeader("https://gitlab.com/myorg", 42); err != nil {
		t.Fatalf("Failed to write header: %v", err)
	}

	logger.Close()

	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "GitLab Python Scanner Log") {
		t.Error("Expected header title in log")
	}
	if !strings.Contains(contentStr, "https://gitlab.com/myorg") {
		t.Error("Expected GitLab URL in log")
	}
	if !strings.Contains(contentStr, "42") {
		t.Error("Expected total projects in log")
	}
}

func TestFileLogger_WriteHeader_JSON(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	logger, err := NewFileLogger(logPath, FormatJSON)
	if err != nil {
		t.Fatalf("Failed to create file logger: %v", err)
	}
	defer logger.Close()

	if err := logger.WriteHeader("https://gitlab.com/myorg", 42); err != nil {
		t.Fatalf("Failed to write header: %v", err)
	}

	logger.Close()

	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	// Parse JSON
	var header map[string]interface{}
	if err := json.Unmarshal(content, &header); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	if header["type"] != "scan_started" {
		t.Errorf("Expected type 'scan_started', got '%v'", header["type"])
	}
	if header["gitlab_url"] != "https://gitlab.com/myorg" {
		t.Errorf("Expected GitLab URL 'https://gitlab.com/myorg', got '%v'", header["gitlab_url"])
	}
	if int(header["total_projects"].(float64)) != 42 {
		t.Errorf("Expected total projects 42, got %v", header["total_projects"])
	}
}

func TestFileLogger_WriteSummary_Text(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	logger, err := NewFileLogger(logPath, FormatText)
	if err != nil {
		t.Fatalf("Failed to create file logger: %v", err)
	}
	defer logger.Close()

	stats := &ScanStatistics{
		TotalProjects:     42,
		PythonProjects:    28,
		NonPythonProjects: 14,
		ErrorCount:        2,
		VersionCounts: map[string]int{
			"3.11.5": 15,
			"3.10.0": 10,
			"2.7.18": 3,
		},
	}

	if err := logger.WriteSummary(stats); err != nil {
		t.Fatalf("Failed to write summary: %v", err)
	}

	logger.Close()

	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "Scan Summary") {
		t.Error("Expected summary title in log")
	}
	if !strings.Contains(contentStr, "Total Projects: 42") {
		t.Error("Expected total projects in summary")
	}
	if !strings.Contains(contentStr, "Python Projects: 28") {
		t.Error("Expected Python projects in summary")
	}
	if !strings.Contains(contentStr, "Non-Python Projects: 14") {
		t.Error("Expected non-Python projects in summary")
	}
	if !strings.Contains(contentStr, "Errors: 2") {
		t.Error("Expected error count in summary")
	}
	if !strings.Contains(contentStr, "3.11.5: 15") {
		t.Error("Expected version distribution in summary")
	}
}

func TestFileLogger_WriteSummary_JSON(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	logger, err := NewFileLogger(logPath, FormatJSON)
	if err != nil {
		t.Fatalf("Failed to create file logger: %v", err)
	}
	defer logger.Close()

	stats := &ScanStatistics{
		TotalProjects:     42,
		PythonProjects:    28,
		NonPythonProjects: 14,
		ErrorCount:        2,
		VersionCounts: map[string]int{
			"3.11.5": 15,
			"3.10.0": 10,
		},
	}

	if err := logger.WriteSummary(stats); err != nil {
		t.Fatalf("Failed to write summary: %v", err)
	}

	logger.Close()

	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	// Parse JSON
	var summary map[string]interface{}
	if err := json.Unmarshal(content, &summary); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	if summary["type"] != "scan_completed" {
		t.Errorf("Expected type 'scan_completed', got '%v'", summary["type"])
	}
	if int(summary["total_projects"].(float64)) != 42 {
		t.Errorf("Expected total projects 42, got %v", summary["total_projects"])
	}
	if int(summary["python_projects"].(float64)) != 28 {
		t.Errorf("Expected Python projects 28, got %v", summary["python_projects"])
	}
	if int(summary["error_count"].(float64)) != 2 {
		t.Errorf("Expected error count 2, got %v", summary["error_count"])
	}
}

func TestFileLogger_Close(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	logger, err := NewFileLogger(logPath, FormatText)
	if err != nil {
		t.Fatalf("Failed to create file logger: %v", err)
	}

	if err := logger.Close(); err != nil {
		t.Fatalf("Failed to close logger: %v", err)
	}

	// Closing again should not error
	if err := logger.Close(); err != nil {
		t.Fatalf("Second close failed: %v", err)
	}
}

func TestFileLogger_Sync(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	logger, err := NewFileLogger(logPath, FormatText)
	if err != nil {
		t.Fatalf("Failed to create file logger: %v", err)
	}
	defer logger.Close()

	result := &ScanResult{
		ProjectName:   "test-project",
		PythonVersion: "3.11.5",
		DetectionSource: ".python-version",
		Index:         1,
		TotalProjects: 1,
	}

	if err := logger.LogResult(result); err != nil {
		t.Fatalf("Failed to log result: %v", err)
	}

	// Sync should flush the data
	if err := logger.Sync(); err != nil {
		t.Fatalf("Failed to sync: %v", err)
	}

	// Verify data was written
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	if !strings.Contains(string(content), "test-project") {
		t.Error("Expected synced data to be present")
	}
}

func TestFileLogger_InvalidPath(t *testing.T) {
	// Try to create logger with invalid path
	_, err := NewFileLogger("/invalid/path/that/does/not/exist/test.log", FormatText)
	if err == nil {
		t.Error("Expected error for invalid path")
	}
}

func TestFileLogger_MultipleEntries_JSONL(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	logger, err := NewFileLogger(logPath, FormatJSON)
	if err != nil {
		t.Fatalf("Failed to create file logger: %v", err)
	}
	defer logger.Close()

	// Write multiple entries
	results := []*ScanResult{
		{
			ProjectName:     "project-1",
			PythonVersion:   "3.11.5",
			DetectionSource: ".python-version",
			Index:           1,
			TotalProjects:   3,
		},
		{
			ProjectName:   "project-2",
			Index:         2,
			TotalProjects: 3,
		},
		{
			ProjectName:   "project-3",
			Error:         errors.New("test error"),
			Index:         3,
			TotalProjects: 3,
		},
	}

	for _, result := range results {
		if err := logger.LogResult(result); err != nil {
			t.Fatalf("Failed to log result: %v", err)
		}
	}

	logger.Close()

	// Read and parse JSONL
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	if len(lines) != 3 {
		t.Errorf("Expected 3 JSON lines, got %d", len(lines))
	}

	// Parse each line
	for i, line := range lines {
		var entry LogEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			t.Fatalf("Failed to parse JSON line %d: %v", i+1, err)
		}
	}
}
