package output

import (
	"bytes"
	"errors"
	"strings"
	"sync"
	"testing"
)

func TestNewConsoleStreamer(t *testing.T) {
	streamer := NewConsoleStreamer()
	if streamer == nil {
		t.Fatal("NewConsoleStreamer() returned nil")
	}
	if streamer.writer == nil {
		t.Error("streamer.writer should not be nil")
	}
}

func TestConsoleStreamer_StreamResult_Success(t *testing.T) {
	buf := &bytes.Buffer{}
	streamer := NewConsoleStreamerWithWriter(buf)

	result := &ScanResult{
		ProjectName:     "my-project",
		PythonVersion:   "3.11.5",
		DetectionSource: ".python-version",
		Index:           1,
		TotalProjects:   10,
	}

	err := streamer.StreamResult(result)
	if err != nil {
		t.Fatalf("StreamResult() error = %v", err)
	}

	output := buf.String()
	expected := "[1/10] my-project: Python 3.11.5 (from .python-version)\n"
	if output != expected {
		t.Errorf("StreamResult() output = %q, want %q", output, expected)
	}
}

func TestConsoleStreamer_StreamResult_NotDetected(t *testing.T) {
	buf := &bytes.Buffer{}
	streamer := NewConsoleStreamerWithWriter(buf)

	result := &ScanResult{
		ProjectName:   "frontend-app",
		PythonVersion: "",
		Index:         2,
		TotalProjects: 10,
	}

	err := streamer.StreamResult(result)
	if err != nil {
		t.Fatalf("StreamResult() error = %v", err)
	}

	output := buf.String()
	expected := "[2/10] frontend-app: Python not detected\n"
	if output != expected {
		t.Errorf("StreamResult() output = %q, want %q", output, expected)
	}
}

func TestConsoleStreamer_StreamResult_Error(t *testing.T) {
	buf := &bytes.Buffer{}
	streamer := NewConsoleStreamerWithWriter(buf)

	testErr := errors.New("network timeout")
	result := &ScanResult{
		ProjectName:   "failed-project",
		Error:         testErr,
		Index:         3,
		TotalProjects: 10,
	}

	err := streamer.StreamResult(result)
	if err != nil {
		t.Fatalf("StreamResult() error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "[3/10]") {
		t.Error("Output should contain index [3/10]")
	}
	if !strings.Contains(output, "failed-project") {
		t.Error("Output should contain project name")
	}
	if !strings.Contains(output, "Error") {
		t.Error("Output should contain 'Error'")
	}
	if !strings.Contains(output, "network timeout") {
		t.Error("Output should contain error message")
	}
}

func TestConsoleStreamer_StreamResult_Concurrent(t *testing.T) {
	buf := &bytes.Buffer{}
	streamer := NewConsoleStreamerWithWriter(buf)

	const numGoroutines = 10
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Stream results concurrently
	for i := 1; i <= numGoroutines; i++ {
		go func(index int) {
			defer wg.Done()
			result := &ScanResult{
				ProjectName:     "project-" + string(rune('0'+index)),
				PythonVersion:   "3.11.0",
				DetectionSource: "pyproject.toml",
				Index:           index,
				TotalProjects:   numGoroutines,
			}
			err := streamer.StreamResult(result)
			if err != nil {
				t.Errorf("StreamResult() error = %v", err)
			}
		}(i)
	}

	wg.Wait()

	// Verify that we got all the results (order doesn't matter due to concurrency)
	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != numGoroutines {
		t.Errorf("Expected %d lines, got %d", numGoroutines, len(lines))
	}

	// Check that each line is valid
	for _, line := range lines {
		if !strings.Contains(line, "Python 3.11.0") {
			t.Errorf("Line should contain version: %s", line)
		}
		if !strings.Contains(line, "pyproject.toml") {
			t.Errorf("Line should contain source: %s", line)
		}
	}
}

func TestConsoleStreamer_PrintHeader(t *testing.T) {
	buf := &bytes.Buffer{}
	streamer := NewConsoleStreamerWithWriter(buf)

	err := streamer.PrintHeader("https://gitlab.com/myorg", 42)
	if err != nil {
		t.Fatalf("PrintHeader() error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "42 projects") {
		t.Error("Header should contain project count")
	}
}

func TestConsoleStreamer_PrintSummary(t *testing.T) {
	buf := &bytes.Buffer{}
	streamer := NewConsoleStreamerWithWriter(buf)

	stats := &ScanStatistics{
		TotalProjects:     42,
		PythonProjects:    28,
		NonPythonProjects: 14,
		ErrorCount:        0,
	}

	err := streamer.PrintSummary(stats)
	if err != nil {
		t.Fatalf("PrintSummary() error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "42 projects") {
		t.Error("Summary should contain total project count")
	}
	if !strings.Contains(output, "28 Python projects") {
		t.Error("Summary should contain Python project count")
	}
	if !strings.Contains(output, "14 non-Python") {
		t.Error("Summary should contain non-Python project count")
	}
}

func TestConsoleStreamer_PrintSummary_WithErrors(t *testing.T) {
	buf := &bytes.Buffer{}
	streamer := NewConsoleStreamerWithWriter(buf)

	stats := &ScanStatistics{
		TotalProjects:     50,
		PythonProjects:    30,
		NonPythonProjects: 15,
		ErrorCount:        5,
	}

	err := streamer.PrintSummary(stats)
	if err != nil {
		t.Fatalf("PrintSummary() error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Errors encountered: 5") {
		t.Error("Summary should contain error count when errors > 0")
	}
}

func TestScanStatistics_RecordResult_Python(t *testing.T) {
	stats := NewScanStatistics()

	result := &ScanResult{
		PythonVersion: "3.11.5",
	}

	stats.RecordResult(result)

	if stats.TotalProjects != 1 {
		t.Errorf("TotalProjects = %d, want 1", stats.TotalProjects)
	}
	if stats.PythonProjects != 1 {
		t.Errorf("PythonProjects = %d, want 1", stats.PythonProjects)
	}
	if stats.NonPythonProjects != 0 {
		t.Errorf("NonPythonProjects = %d, want 0", stats.NonPythonProjects)
	}
	if stats.VersionCounts["3.11.5"] != 1 {
		t.Errorf("VersionCounts[3.11.5] = %d, want 1", stats.VersionCounts["3.11.5"])
	}
}

func TestScanStatistics_RecordResult_NonPython(t *testing.T) {
	stats := NewScanStatistics()

	result := &ScanResult{
		PythonVersion: "",
	}

	stats.RecordResult(result)

	if stats.TotalProjects != 1 {
		t.Errorf("TotalProjects = %d, want 1", stats.TotalProjects)
	}
	if stats.PythonProjects != 0 {
		t.Errorf("PythonProjects = %d, want 0", stats.PythonProjects)
	}
	if stats.NonPythonProjects != 1 {
		t.Errorf("NonPythonProjects = %d, want 1", stats.NonPythonProjects)
	}
}

func TestScanStatistics_RecordResult_Error(t *testing.T) {
	stats := NewScanStatistics()

	result := &ScanResult{
		Error: errors.New("test error"),
	}

	stats.RecordResult(result)

	if stats.TotalProjects != 1 {
		t.Errorf("TotalProjects = %d, want 1", stats.TotalProjects)
	}
	if stats.ErrorCount != 1 {
		t.Errorf("ErrorCount = %d, want 1", stats.ErrorCount)
	}
	if stats.PythonProjects != 0 {
		t.Errorf("PythonProjects = %d, want 0", stats.PythonProjects)
	}
	if stats.NonPythonProjects != 0 {
		t.Errorf("NonPythonProjects = %d, want 0", stats.NonPythonProjects)
	}
}

func TestScanStatistics_RecordResult_MultipleVersions(t *testing.T) {
	stats := NewScanStatistics()

	// Record multiple results
	results := []*ScanResult{
		{PythonVersion: "3.11.5"},
		{PythonVersion: "3.10.0"},
		{PythonVersion: "3.11.5"},
		{PythonVersion: "2.7.18"},
		{PythonVersion: ""},
	}

	for _, result := range results {
		stats.RecordResult(result)
	}

	if stats.TotalProjects != 5 {
		t.Errorf("TotalProjects = %d, want 5", stats.TotalProjects)
	}
	if stats.PythonProjects != 4 {
		t.Errorf("PythonProjects = %d, want 4", stats.PythonProjects)
	}
	if stats.NonPythonProjects != 1 {
		t.Errorf("NonPythonProjects = %d, want 1", stats.NonPythonProjects)
	}
	if stats.VersionCounts["3.11.5"] != 2 {
		t.Errorf("VersionCounts[3.11.5] = %d, want 2", stats.VersionCounts["3.11.5"])
	}
	if stats.VersionCounts["3.10.0"] != 1 {
		t.Errorf("VersionCounts[3.10.0] = %d, want 1", stats.VersionCounts["3.10.0"])
	}
	if stats.VersionCounts["2.7.18"] != 1 {
		t.Errorf("VersionCounts[2.7.18] = %d, want 1", stats.VersionCounts["2.7.18"])
	}
}
