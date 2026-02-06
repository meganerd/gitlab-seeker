package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/gbjohnso/gitlab-python-scanner/internal/gitlab"
	"github.com/gbjohnso/gitlab-python-scanner/internal/output"
	"github.com/gbjohnso/gitlab-python-scanner/internal/parsers"
	"github.com/gbjohnso/gitlab-python-scanner/internal/rules"
)

// Config holds the application configuration
type Config struct {
	GitLabURL   string
	Token       string
	LogFile     string
	Concurrency int
	Timeout     int
}

func main() {
	// Parse command-line flags
	config := parseFlags()

	// Validate required parameters
	if err := validateConfig(config); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		flag.Usage()
		os.Exit(1)
	}

	fmt.Printf("GitLab Python Version Scanner\n")
	fmt.Printf("==============================\n\n")
	fmt.Printf("Scanning: %s\n", config.GitLabURL)
	if config.LogFile != "" {
		fmt.Printf("Logging to: %s\n", config.LogFile)
	}
	fmt.Println()

	// Create GitLab client
	gitlabConfig := &gitlab.Config{
		GitLabURL: config.GitLabURL,
		Token:     config.Token,
		Timeout:   time.Duration(config.Timeout) * time.Second,
	}

	client, err := gitlab.NewClient(gitlabConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating GitLab client: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("GitLab Base URL: %s\n", client.GetBaseURL())
	fmt.Printf("Organization: %s\n", client.GetOrganization())
	fmt.Println()

	// Test the connection
	fmt.Println("Testing GitLab connection...")
	if err := client.TestConnection(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✓ Successfully connected to GitLab")
	fmt.Println()

	// Run the scan
	if err := runScan(client, config); err != nil {
		fmt.Fprintf(os.Stderr, "Scan failed: %v\n", err)
		os.Exit(1)
	}
}

// runScan orchestrates the scanning process
func runScan(client *gitlab.Client, config *Config) error {
	ctx := context.Background()

	// List all projects
	fmt.Println("Fetching projects...")
	projects, err := client.ListAllProjects(ctx)
	if err != nil {
		return fmt.Errorf("failed to list projects: %w", err)
	}

	if len(projects) == 0 {
		fmt.Println("No projects found")
		return nil
	}

	// Initialize output handlers
	streamer := output.NewConsoleStreamer()
	stats := output.NewScanStatistics()

	var logger *output.FileLogger
	if config.LogFile != "" {
		logger, err = output.NewFileLogger(config.LogFile, output.FormatJSON)
		if err != nil {
			return fmt.Errorf("failed to create log file: %w", err)
		}
		defer logger.Close()

		if err := logger.WriteHeader(config.GitLabURL, len(projects)); err != nil {
			return fmt.Errorf("failed to write log header: %w", err)
		}
	}

	// Print header
	if err := streamer.PrintHeader(config.GitLabURL, len(projects)); err != nil {
		return fmt.Errorf("failed to print header: %w", err)
	}

	// Create rule registry for Python version detection
	registry := parsers.DefaultRegistry()

	// Set up concurrency control
	semaphore := make(chan struct{}, config.Concurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex

	// Scan each project concurrently
	for i, project := range projects {
		wg.Add(1)
		go func(index int, proj *gitlab.Project) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// Scan the project
			result := scanProject(ctx, client, registry, proj, index+1, len(projects))

			// Thread-safe result recording
			mu.Lock()
			stats.RecordResult(result)
			mu.Unlock()

			// Stream result to console
			if err := streamer.StreamResult(result); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to stream result: %v\n", err)
			}

			// Log result to file if logger is configured
			if logger != nil {
				if err := logger.LogResult(result); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to log result: %v\n", err)
				}
			}
		}(i, project)
	}

	// Wait for all scans to complete
	wg.Wait()

	// Print summary
	if err := streamer.PrintSummary(stats); err != nil {
		return fmt.Errorf("failed to print summary: %w", err)
	}

	// Write summary to log
	if logger != nil {
		if err := logger.WriteSummary(stats); err != nil {
			return fmt.Errorf("failed to write log summary: %w", err)
		}
	}

	return nil
}

// scanProject scans a single project for Python version information
func scanProject(ctx context.Context, client *gitlab.Client, registry *rules.Registry, project *gitlab.Project, index, total int) *output.ScanResult {
	result := &output.ScanResult{
		ProjectName:   project.Name,
		ProjectPath:   project.PathWithNamespace,
		Index:         index,
		TotalProjects: total,
	}

	// Get all enabled rules to determine which files to check
	enabledRules := registry.ListEnabled()
	if len(enabledRules) == 0 {
		result.Error = fmt.Errorf("no enabled rules found")
		return result
	}

	// Try each rule's file pattern until we find a match
	// Rules are already sorted by priority (highest first)
	for _, rule := range enabledRules {
		filename := rule.Condition.FilePattern

		// Try to fetch the file from the project
		content, err := client.GetRawFile(ctx, project.ID, filename, nil)
		if err != nil {
			// File not found or other error - try next rule
			continue
		}

		// Apply the rule to parse the file content
		searchResult, err := rule.Apply(ctx, content, filename)
		if err != nil {
			// Parse error - try next rule
			continue
		}

		// Check if we found a Python version
		if searchResult != nil && searchResult.Found && searchResult.Version != "" {
			result.PythonVersion = searchResult.Version
			result.DetectionSource = searchResult.Source
			return result
		}
	}

	// No Python version found
	return result
}

func parseFlags() *Config {
	config := &Config{}

	flag.StringVar(&config.GitLabURL, "url", "", "GitLab URL including org/group (e.g., gitlab.com/myorg)")
	flag.StringVar(&config.Token, "token", os.Getenv("GITLAB_TOKEN"), "GitLab API token (or set GITLAB_TOKEN env var)")
	flag.StringVar(&config.LogFile, "log", "", "Path to log file (optional)")
	flag.IntVar(&config.Concurrency, "concurrency", 5, "Number of concurrent scans")
	flag.IntVar(&config.Timeout, "timeout", 30, "API timeout in seconds")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Scan GitLab projects to detect Python versions.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExample:\n")
		fmt.Fprintf(os.Stderr, "  %s --url gitlab.com/myorg --token abc123\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s --url gitlab.com/myorg --token abc123 --log results.log\n", os.Args[0])
	}

	flag.Parse()
	return config
}

func validateConfig(config *Config) error {
	if config.GitLabURL == "" {
		return fmt.Errorf("--url is required")
	}
	if config.Token == "" {
		return fmt.Errorf("--token is required (or set GITLAB_TOKEN environment variable)")
	}
	return nil
}
