package output

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// ContentMatchEntry represents a single string match found in a file
type ContentMatchEntry struct {
	FilePath    string // Full path of the file in the repository
	LineNumber  int    // 1-based line number of the match
	LineContent string // The full line containing the match
	MatchedText string // The specific text that matched
}

// ContentScanResult represents the content search results for a single project
type ContentScanResult struct {
	ProjectName   string              // Name of the project
	ProjectPath   string              // Full path of the project
	Matches       []ContentMatchEntry // All matches found in this project
	SearchTerm    string              // The string/pattern that was searched for
	Error         error               // Any error encountered during searching
	Index         int                 // Sequential index of this result
	TotalProjects int                 // Total number of projects being searched
}

// ContentScanStatistics holds summary statistics for a content search operation
type ContentScanStatistics struct {
	mu                sync.Mutex
	TotalProjects     int            // Total number of projects searched
	ProjectsWithHits  int            // Number of projects with at least one match
	ProjectsNoHits    int            // Number of projects with no matches
	TotalMatches      int            // Total number of matches across all projects
	ErrorCount        int            // Number of errors encountered
	MatchesByFile     map[string]int // Match count by filename
}

// NewContentScanStatistics creates a new content search statistics tracker
func NewContentScanStatistics() *ContentScanStatistics {
	return &ContentScanStatistics{
		MatchesByFile: make(map[string]int),
	}
}

// RecordResult updates statistics based on a content search result
func (cs *ContentScanStatistics) RecordResult(result *ContentScanResult) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	cs.TotalProjects++

	if result.Error != nil {
		cs.ErrorCount++
		return
	}

	if len(result.Matches) == 0 {
		cs.ProjectsNoHits++
	} else {
		cs.ProjectsWithHits++
		cs.TotalMatches += len(result.Matches)
		for _, m := range result.Matches {
			cs.MatchesByFile[m.FilePath]++
		}
	}
}

// StreamContentResult writes a single content search result to the console
func (cs *ConsoleStreamer) StreamContentResult(result *ContentScanResult) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if result.Error != nil {
		_, err := fmt.Fprintf(cs.writer, "[%d/%d] %s: Error - %v\n",
			result.Index, result.TotalProjects, result.ProjectName, result.Error)
		return err
	}

	if len(result.Matches) == 0 {
		_, err := fmt.Fprintf(cs.writer, "[%d/%d] %s: no matches\n",
			result.Index, result.TotalProjects, result.ProjectName)
		return err
	}

	_, err := fmt.Fprintf(cs.writer, "[%d/%d] %s: %d match(es) found\n",
		result.Index, result.TotalProjects, result.ProjectName, len(result.Matches))
	if err != nil {
		return err
	}

	for _, m := range result.Matches {
		_, err = fmt.Fprintf(cs.writer, "  %s:%d: %s\n", m.FilePath, m.LineNumber, m.LineContent)
		if err != nil {
			return err
		}
	}

	return nil
}

// PrintContentHeader writes the initial header for a content search
func (cs *ConsoleStreamer) PrintContentHeader(gitlabURL string, totalProjects int, searchTerm string) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	_, err := fmt.Fprintf(cs.writer, "\nSearching %d projects for %q\n\n", totalProjects, searchTerm)
	return err
}

// PrintContentSummary writes the final summary for a content search
func (cs *ConsoleStreamer) PrintContentSummary(stats *ContentScanStatistics) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	_, err := fmt.Fprintf(cs.writer, "\nSearch complete: %d projects scanned, %d with matches (%d total matches)\n",
		stats.TotalProjects, stats.ProjectsWithHits, stats.TotalMatches)

	if stats.ErrorCount > 0 {
		fmt.Fprintf(cs.writer, "Errors encountered: %d\n", stats.ErrorCount)
	}

	return err
}

// ContentLogEntry represents a single content search log entry
type ContentLogEntry struct {
	Timestamp   time.Time         `json:"timestamp"`
	ProjectName string            `json:"project_name"`
	ProjectPath string            `json:"project_path,omitempty"`
	SearchTerm  string            `json:"search_term"`
	Matches     []ContentMatchLog `json:"matches,omitempty"`
	MatchCount  int               `json:"match_count"`
	Error       string            `json:"error,omitempty"`
	Index       int               `json:"index"`
	Total       int               `json:"total_projects"`
}

// ContentMatchLog is the JSON-serializable form of a content match
type ContentMatchLog struct {
	FilePath    string `json:"file_path"`
	LineNumber  int    `json:"line_number"`
	LineContent string `json:"line_content"`
	MatchedText string `json:"matched_text"`
}

// LogContentResult writes a content search result to the log file
func (fl *FileLogger) LogContentResult(result *ContentScanResult) error {
	fl.mu.Lock()
	defer fl.mu.Unlock()

	entry := ContentLogEntry{
		Timestamp:   time.Now(),
		ProjectName: result.ProjectName,
		ProjectPath: result.ProjectPath,
		SearchTerm:  result.SearchTerm,
		MatchCount:  len(result.Matches),
		Index:       result.Index,
		Total:       result.TotalProjects,
	}

	if result.Error != nil {
		entry.Error = result.Error.Error()
	}

	for _, m := range result.Matches {
		entry.Matches = append(entry.Matches, ContentMatchLog{
			FilePath:    m.FilePath,
			LineNumber:  m.LineNumber,
			LineContent: m.LineContent,
			MatchedText: m.MatchedText,
		})
	}

	switch fl.format {
	case FormatJSON:
		data, err := json.Marshal(&entry)
		if err != nil {
			return fmt.Errorf("failed to marshal content log entry: %w", err)
		}
		_, err = fl.file.Write(append(data, '\n'))
		return err
	case FormatText:
		if entry.Error != "" {
			_, err := fmt.Fprintf(fl.file, "[%s] [%d/%d] %s: Error - %s\n",
				entry.Timestamp.Format(time.RFC3339), entry.Index, entry.Total, entry.ProjectName, entry.Error)
			return err
		}
		if entry.MatchCount == 0 {
			_, err := fmt.Fprintf(fl.file, "[%s] [%d/%d] %s: no matches\n",
				entry.Timestamp.Format(time.RFC3339), entry.Index, entry.Total, entry.ProjectName)
			return err
		}
		_, err := fmt.Fprintf(fl.file, "[%s] [%d/%d] %s: %d match(es)\n",
			entry.Timestamp.Format(time.RFC3339), entry.Index, entry.Total, entry.ProjectName, entry.MatchCount)
		if err != nil {
			return err
		}
		for _, m := range entry.Matches {
			fmt.Fprintf(fl.file, "  %s:%d: %s\n", m.FilePath, m.LineNumber, m.LineContent)
		}
		return nil
	default:
		return fmt.Errorf("unknown log format: %s", fl.format)
	}
}
