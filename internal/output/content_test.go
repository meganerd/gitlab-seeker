package output

import (
	"bytes"
	"strings"
	"testing"
)

func TestContentScanStatistics_RecordResult(t *testing.T) {
	stats := NewContentScanStatistics()

	// Record a project with matches
	stats.RecordResult(&ContentScanResult{
		ProjectName: "proj1",
		Matches: []ContentMatchEntry{
			{FilePath: "main.py", LineNumber: 10, LineContent: "API_KEY = 'abc'", MatchedText: "API_KEY"},
			{FilePath: "config.py", LineNumber: 5, LineContent: "API_KEY = 'def'", MatchedText: "API_KEY"},
		},
	})

	// Record a project with no matches
	stats.RecordResult(&ContentScanResult{
		ProjectName: "proj2",
		Matches:     nil,
	})

	// Record a project with an error
	stats.RecordResult(&ContentScanResult{
		ProjectName: "proj3",
		Error:       errForTest("connection failed"),
	})

	if stats.TotalProjects != 3 {
		t.Errorf("TotalProjects = %d, want 3", stats.TotalProjects)
	}
	if stats.ProjectsWithHits != 1 {
		t.Errorf("ProjectsWithHits = %d, want 1", stats.ProjectsWithHits)
	}
	if stats.ProjectsNoHits != 1 {
		t.Errorf("ProjectsNoHits = %d, want 1", stats.ProjectsNoHits)
	}
	if stats.TotalMatches != 2 {
		t.Errorf("TotalMatches = %d, want 2", stats.TotalMatches)
	}
	if stats.ErrorCount != 1 {
		t.Errorf("ErrorCount = %d, want 1", stats.ErrorCount)
	}
	if stats.MatchesByFile["main.py"] != 1 {
		t.Errorf("MatchesByFile[main.py] = %d, want 1", stats.MatchesByFile["main.py"])
	}
}

func TestConsoleStreamer_StreamContentResult(t *testing.T) {
	tests := []struct {
		name     string
		result   *ContentScanResult
		contains []string
	}{
		{
			name: "with matches",
			result: &ContentScanResult{
				ProjectName:   "my-project",
				Index:         1,
				TotalProjects: 10,
				Matches: []ContentMatchEntry{
					{FilePath: "src/app.py", LineNumber: 42, LineContent: "password = 'secret'"},
				},
			},
			contains: []string{"[1/10]", "my-project", "1 match", "src/app.py:42"},
		},
		{
			name: "no matches",
			result: &ContentScanResult{
				ProjectName:   "empty-project",
				Index:         2,
				TotalProjects: 10,
			},
			contains: []string{"[2/10]", "empty-project", "no matches"},
		},
		{
			name: "error",
			result: &ContentScanResult{
				ProjectName:   "broken-project",
				Index:         3,
				TotalProjects: 10,
				Error:         errForTest("API timeout"),
			},
			contains: []string{"[3/10]", "broken-project", "Error", "API timeout"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			streamer := NewConsoleStreamerWithWriter(&buf)

			err := streamer.StreamContentResult(tt.result)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			output := buf.String()
			for _, s := range tt.contains {
				if !strings.Contains(output, s) {
					t.Errorf("output missing %q, got: %s", s, output)
				}
			}
		})
	}
}

func TestConsoleStreamer_PrintContentSummary(t *testing.T) {
	var buf bytes.Buffer
	streamer := NewConsoleStreamerWithWriter(&buf)

	stats := NewContentScanStatistics()
	stats.TotalProjects = 50
	stats.ProjectsWithHits = 12
	stats.TotalMatches = 47

	err := streamer.PrintContentSummary(stats)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "50 projects scanned") {
		t.Errorf("missing project count in: %s", output)
	}
	if !strings.Contains(output, "12 with matches") {
		t.Errorf("missing hits count in: %s", output)
	}
	if !strings.Contains(output, "47 total matches") {
		t.Errorf("missing total matches in: %s", output)
	}
}

// errForTest is a simple error type for testing
type errForTest string

func (e errForTest) Error() string { return string(e) }
