package scanner

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gbjohnso/gitlab-python-scanner/internal/gitlab"
	"github.com/gbjohnso/gitlab-python-scanner/internal/output"
	"github.com/gbjohnso/gitlab-python-scanner/internal/parsers"
)

// ContentSearchConfig holds configuration for a content search operation
type ContentSearchConfig struct {
	SearchTerm    string   // The string or regex to search for
	IsRegex       bool     // Whether SearchTerm is a regex
	FilePatterns  []string // Filename glob patterns to restrict to (empty = all files)
	CaseSensitive bool     // Case sensitivity
	ContextLines  int      // Context lines around matches
	MaxMatches    int      // Max matches per project (0 = unlimited)
	MaxFileSize   int64    // Skip files larger than this (bytes, 0 = 1MB default)
}

// ContentScanner orchestrates searching across a project's files
type ContentScanner struct {
	client *gitlab.Client
	parser *parsers.StringSearchParser
	config ContentSearchConfig
}

// NewContentScanner creates a new content scanner
func NewContentScanner(client *gitlab.Client, config ContentSearchConfig) *ContentScanner {
	if config.MaxFileSize == 0 {
		config.MaxFileSize = 1024 * 1024 // 1MB default
	}

	return &ContentScanner{
		client: client,
		config: config,
		parser: &parsers.StringSearchParser{
			SearchTerm:    config.SearchTerm,
			IsRegex:       config.IsRegex,
			CaseSensitive: config.CaseSensitive,
			ContextLines:  config.ContextLines,
			MaxMatches:    config.MaxMatches,
		},
	}
}

// ScanProject searches a single project for the configured search term
func (cs *ContentScanner) ScanProject(ctx context.Context, project *gitlab.Project, index, total int) *output.ContentScanResult {
	result := &output.ContentScanResult{
		ProjectName:   project.Name,
		ProjectPath:   project.PathWithNamespace,
		SearchTerm:    cs.config.SearchTerm,
		Index:         index,
		TotalProjects: total,
	}

	var matches []output.ContentMatchEntry
	var err error

	if cs.config.IsRegex {
		matches, err = cs.searchLocal(ctx, project)
	} else {
		matches, err = cs.searchViaAPI(ctx, project)
	}

	if err != nil {
		result.Error = err
		return result
	}

	result.Matches = matches
	return result
}

// searchViaAPI uses the GitLab Search API for literal string search (most efficient)
func (cs *ContentScanner) searchViaAPI(ctx context.Context, project *gitlab.Project) ([]output.ContentMatchEntry, error) {
	blobs, err := cs.client.SearchBlobs(ctx, project.ID, cs.config.SearchTerm, nil)
	if err != nil {
		return nil, fmt.Errorf("search API error: %w", err)
	}

	var matches []output.ContentMatchEntry
	for _, blob := range blobs {
		// Filter by file patterns if specified
		if len(cs.config.FilePatterns) > 0 && !cs.matchesFilePattern(blob.Filename) {
			continue
		}

		// Parse the blob data snippet into individual line matches
		lines := strings.Split(blob.Data, "\n")
		for i, line := range lines {
			line = strings.TrimRight(line, "\r")
			searchIn := line
			searchFor := cs.config.SearchTerm
			if !cs.config.CaseSensitive {
				searchIn = strings.ToLower(searchIn)
				searchFor = strings.ToLower(searchFor)
			}
			if strings.Contains(searchIn, searchFor) {
				idx := strings.Index(searchIn, searchFor)
				matches = append(matches, output.ContentMatchEntry{
					FilePath:    blob.Path,
					LineNumber:  blob.Startline + i,
					LineContent: line,
					MatchedText: line[idx : idx+len(cs.config.SearchTerm)],
				})

				if cs.config.MaxMatches > 0 && len(matches) >= cs.config.MaxMatches {
					return matches, nil
				}
			}
		}
	}

	return matches, nil
}

// searchLocal fetches files and searches locally (needed for regex)
func (cs *ContentScanner) searchLocal(ctx context.Context, project *gitlab.Project) ([]output.ContentMatchEntry, error) {
	files, err := cs.getFilesToSearch(ctx, project)
	if err != nil {
		return nil, err
	}

	var allMatches []output.ContentMatchEntry
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, 3) // Limit concurrent file fetches per project

	for _, file := range files {
		if cs.config.MaxMatches > 0 && len(allMatches) >= cs.config.MaxMatches {
			break
		}

		wg.Add(1)
		go func(f *gitlab.TreeFile) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			content, err := cs.client.GetRawFile(ctx, project.ID, f.Path, nil)
			if err != nil {
				return
			}

			// Skip files that are too large
			if int64(len(content)) > cs.config.MaxFileSize {
				return
			}

			matches, err := cs.parser.Search(content, f.Path)
			if err != nil {
				return
			}

			if len(matches) > 0 {
				mu.Lock()
				allMatches = append(allMatches, matches...)
				mu.Unlock()
			}
		}(file)
	}

	wg.Wait()

	// Trim to max if we overshot due to concurrency
	if cs.config.MaxMatches > 0 && len(allMatches) > cs.config.MaxMatches {
		allMatches = allMatches[:cs.config.MaxMatches]
	}

	return allMatches, nil
}

// getFilesToSearch determines which files to fetch and search
func (cs *ContentScanner) getFilesToSearch(ctx context.Context, project *gitlab.Project) ([]*gitlab.TreeFile, error) {
	if len(cs.config.FilePatterns) > 0 {
		// Specific file patterns: list tree and filter
		allFiles, err := cs.client.ListRepositoryTree(ctx, project.ID, &gitlab.ListTreeOptions{
			Recursive: true,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to list repository tree: %w", err)
		}

		var filtered []*gitlab.TreeFile
		for _, f := range allFiles {
			if cs.matchesFilePattern(f.Name) {
				filtered = append(filtered, f)
			}
		}
		return filtered, nil
	}

	// No file filter: list all files
	allFiles, err := cs.client.ListRepositoryTree(ctx, project.ID, &gitlab.ListTreeOptions{
		Recursive: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list repository tree: %w", err)
	}

	return allFiles, nil
}

// matchesFilePattern checks if a filename matches any of the configured file patterns
func (cs *ContentScanner) matchesFilePattern(filename string) bool {
	for _, pattern := range cs.config.FilePatterns {
		matched, err := filepath.Match(pattern, filename)
		if err == nil && matched {
			return true
		}
	}
	return false
}
