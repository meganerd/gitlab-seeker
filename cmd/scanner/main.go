package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gbjohnso/gitlab-python-scanner/internal/config"
	"github.com/gbjohnso/gitlab-python-scanner/internal/gitlab"
	"github.com/gbjohnso/gitlab-python-scanner/internal/output"
	"github.com/gbjohnso/gitlab-python-scanner/internal/parsers"
	"github.com/gbjohnso/gitlab-python-scanner/internal/rules"
	"github.com/gbjohnso/gitlab-python-scanner/internal/scanner"
)

// Config holds the application configuration for Python version scanning
type Config struct {
	GitLabURL   string
	Token       string
	LogFile     string
	Concurrency int
	Timeout     int
}

// SearchConfig holds the configuration for content string search
type SearchConfig struct {
	GitLabURL     string
	Token         string
	LogFile       string
	Concurrency   int
	Timeout       int
	SearchTerm    string
	IsRegex       bool
	FilePatterns  []string
	CaseSensitive bool
	ContextLines  int
	ConfigFile    string
}

// multiFlag allows a flag to be specified multiple times
type multiFlag []string

func (m *multiFlag) String() string { return strings.Join(*m, ", ") }
func (m *multiFlag) Set(value string) error {
	*m = append(*m, value)
	return nil
}

func main() {
	// Check for explicit "search" subcommand (kept for backward compat)
	if len(os.Args) > 1 && os.Args[1] == "search" {
		searchConfig := parseSearchFlags(os.Args[2:])
		runSearchMode(searchConfig)
		return
	}

	// Skip "scan" subcommand if provided explicitly
	args := os.Args[1:]
	if len(os.Args) > 1 && os.Args[1] == "scan" {
		args = os.Args[2:]
	}

	// Parse unified flags (includes both scan and search flags)
	searchConfig := parseSearchFlags(args)

	// If --search or --config is provided, run in search mode
	if searchConfig.SearchTerm != "" || searchConfig.ConfigFile != "" {
		runSearchMode(searchConfig)
		return
	}

	// Otherwise run in scan mode (Python version detection)
	scanConfig := &Config{
		GitLabURL:   searchConfig.GitLabURL,
		Token:       searchConfig.Token,
		LogFile:     searchConfig.LogFile,
		Concurrency: searchConfig.Concurrency,
		Timeout:     searchConfig.Timeout,
	}

	if err := validateConfig(scanConfig); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		flag.Usage()
		os.Exit(1)
	}

	fmt.Printf("GitLab Python Version Scanner\n")
	fmt.Printf("==============================\n\n")
	fmt.Printf("Scanning: %s\n", scanConfig.GitLabURL)
	if scanConfig.LogFile != "" {
		fmt.Printf("Logging to: %s\n", scanConfig.LogFile)
	}
	fmt.Println()

	client, err := createClient(scanConfig.GitLabURL, scanConfig.Token, scanConfig.Timeout)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating GitLab client: %v\n", err)
		os.Exit(1)
	}

	printClientInfo(client)

	if err := runScan(client, scanConfig); err != nil {
		fmt.Fprintf(os.Stderr, "Scan failed: %v\n", err)
		os.Exit(1)
	}
}

// runSearchMode validates and executes a content search
func runSearchMode(searchConfig *SearchConfig) {
	if err := validateSearchConfig(searchConfig); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// If a config file is provided, load searches from it
	var searchConfigs []*SearchConfig
	if searchConfig.ConfigFile != "" {
		loaded, err := loadSearchesFromConfig(searchConfig)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			os.Exit(1)
		}
		searchConfigs = loaded
	} else {
		searchConfigs = []*SearchConfig{searchConfig}
	}

	fmt.Printf("GitLab Content Search\n")
	fmt.Printf("=====================\n\n")
	fmt.Printf("Searching: %s\n", searchConfig.GitLabURL)
	if len(searchConfigs) == 1 {
		fmt.Printf("Search term: %q\n", searchConfigs[0].SearchTerm)
	} else {
		fmt.Printf("Searches: %d from config file\n", len(searchConfigs))
	}
	if searchConfig.LogFile != "" {
		fmt.Printf("Logging to: %s\n", searchConfig.LogFile)
	}
	fmt.Println()

	client, err := createClient(searchConfig.GitLabURL, searchConfig.Token, searchConfig.Timeout)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating GitLab client: %v\n", err)
		os.Exit(1)
	}

	printClientInfo(client)

	for _, sc := range searchConfigs {
		if len(searchConfigs) > 1 {
			fmt.Printf("\n--- Search: %q ---\n", sc.SearchTerm)
		}
		if err := runContentSearch(client, sc); err != nil {
			fmt.Fprintf(os.Stderr, "Search failed: %v\n", err)
			os.Exit(1)
		}
	}
}

// loadSearchesFromConfig loads search definitions from a YAML/JSON config file
func loadSearchesFromConfig(base *SearchConfig) ([]*SearchConfig, error) {
	cfg, err := config.LoadConfig(base.ConfigFile)
	if err != nil {
		return nil, err
	}

	if len(cfg.Searches) == 0 {
		return nil, fmt.Errorf("config file contains no search definitions")
	}

	var configs []*SearchConfig
	for _, s := range cfg.Searches {
		enabled := true
		if s.Enabled != nil {
			enabled = *s.Enabled
		}
		if !enabled {
			continue
		}

		configs = append(configs, &SearchConfig{
			GitLabURL:     base.GitLabURL,
			Token:         base.Token,
			LogFile:       base.LogFile,
			Concurrency:   base.Concurrency,
			Timeout:       base.Timeout,
			SearchTerm:    s.SearchTerm,
			IsRegex:       s.IsRegex,
			FilePatterns:  s.FilePatterns,
			CaseSensitive: s.CaseSensitive,
			ContextLines:  s.ContextLines,
		})
	}

	if len(configs) == 0 {
		return nil, fmt.Errorf("no enabled searches found in config file")
	}

	return configs, nil
}

// createClient creates and tests a GitLab client connection
func createClient(gitlabURL, token string, timeout int) (*gitlab.Client, error) {
	gitlabConfig := &gitlab.Config{
		GitLabURL: gitlabURL,
		Token:     token,
		Timeout:   time.Duration(timeout) * time.Second,
	}

	client, err := gitlab.NewClient(gitlabConfig)
	if err != nil {
		return nil, err
	}

	fmt.Println("Testing GitLab connection...")
	if err := client.TestConnection(); err != nil {
		return nil, err
	}
	fmt.Println("✓ Successfully connected to GitLab")
	fmt.Println()

	return client, nil
}

// printClientInfo prints the client connection details
func printClientInfo(client *gitlab.Client) {
	fmt.Printf("GitLab Base URL: %s\n", client.GetBaseURL())
	fmt.Printf("Organization: %s\n", client.GetOrganization())
	fmt.Println()
}

// runContentSearch orchestrates the content search process
func runContentSearch(client *gitlab.Client, config *SearchConfig) error {
	ctx := context.Background()

	fmt.Println("Fetching projects...")
	projects, err := client.ListAllProjects(ctx)
	if err != nil {
		return fmt.Errorf("failed to list projects: %w", err)
	}

	if len(projects) == 0 {
		fmt.Println("No projects found")
		return nil
	}

	streamer := output.NewConsoleStreamer()
	stats := output.NewContentScanStatistics()

	var logger *output.FileLogger
	if config.LogFile != "" {
		logger, err = output.NewFileLogger(config.LogFile, output.FormatJSON)
		if err != nil {
			return fmt.Errorf("failed to create log file: %w", err)
		}
		defer logger.Close()
	}

	if err := streamer.PrintContentHeader(config.GitLabURL, len(projects), config.SearchTerm); err != nil {
		return fmt.Errorf("failed to print header: %w", err)
	}

	contentScanner := scanner.NewContentScanner(client, scanner.ContentSearchConfig{
		SearchTerm:    config.SearchTerm,
		IsRegex:       config.IsRegex,
		FilePatterns:  config.FilePatterns,
		CaseSensitive: config.CaseSensitive,
		ContextLines:  config.ContextLines,
	})

	semaphore := make(chan struct{}, config.Concurrency)
	var wg sync.WaitGroup

	for i, project := range projects {
		wg.Add(1)
		go func(index int, proj *gitlab.Project) {
			defer wg.Done()

			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			result := contentScanner.ScanProject(ctx, proj, index+1, len(projects))

			stats.RecordResult(result)

			if err := streamer.StreamContentResult(result); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to stream result: %v\n", err)
			}

			if logger != nil {
				if err := logger.LogContentResult(result); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to log result: %v\n", err)
				}
			}
		}(i, project)
	}

	wg.Wait()

	if err := streamer.PrintContentSummary(stats); err != nil {
		return fmt.Errorf("failed to print summary: %w", err)
	}

	return nil
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

func parseScanFlags(args []string) *Config {
	config := &Config{}

	fs := flag.NewFlagSet("scan", flag.ExitOnError)
	fs.StringVar(&config.GitLabURL, "url", "", "GitLab URL including org/group (e.g., gitlab.com/myorg)")
	fs.StringVar(&config.Token, "token", os.Getenv("GITLAB_TOKEN"), "GitLab API token (or set GITLAB_TOKEN env var)")
	fs.StringVar(&config.LogFile, "log", "", "Path to log file (optional)")
	fs.IntVar(&config.Concurrency, "concurrency", 5, "Number of concurrent scans")
	fs.IntVar(&config.Timeout, "timeout", 30, "API timeout in seconds")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Scan GitLab projects to detect Python versions.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		fs.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExample:\n")
		fmt.Fprintf(os.Stderr, "  %s --url gitlab.com/myorg --token abc123\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s --url gitlab.com/myorg --token abc123 --log results.log\n", os.Args[0])
	}

	fs.Parse(args)
	return config
}

func parseSearchFlags(args []string) *SearchConfig {
	config := &SearchConfig{}
	var filePatterns multiFlag

	fs := flag.NewFlagSet("scanner", flag.ExitOnError)
	fs.StringVar(&config.GitLabURL, "url", "", "GitLab URL including org/group (e.g., gitlab.com/myorg)")
	fs.StringVar(&config.Token, "token", os.Getenv("GITLAB_TOKEN"), "GitLab API token (or set GITLAB_TOKEN env var)")
	fs.StringVar(&config.LogFile, "log", "", "Path to log file (optional)")
	fs.IntVar(&config.Concurrency, "concurrency", 5, "Number of concurrent operations")
	fs.IntVar(&config.Timeout, "timeout", 30, "API timeout in seconds")
	fs.StringVar(&config.SearchTerm, "search", "", "String or pattern to search for (enables search mode)")
	fs.BoolVar(&config.IsRegex, "regex", false, "Treat search term as a regex pattern")
	fs.Var(&filePatterns, "file", "Filename glob pattern to restrict search (repeatable, e.g., --file '*.py')")
	fs.BoolVar(&config.CaseSensitive, "case-sensitive", false, "Enable case-sensitive search (default: case-insensitive)")
	fs.IntVar(&config.ContextLines, "context", 0, "Lines of context around each match")
	fs.StringVar(&config.ConfigFile, "config", "", "Path to YAML/JSON config file with search definitions")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "GitLab project scanner and content search tool.\n\n")
		fmt.Fprintf(os.Stderr, "Without --search: scans projects for Python versions.\n")
		fmt.Fprintf(os.Stderr, "With --search:    searches for strings across project files.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		fs.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s --url gitlab.com/myorg --token abc123\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s --url gitlab.com/myorg --token abc123 --search \"API_KEY\"\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s --url gitlab.com/myorg --search \"password\\s*=\" --regex --file \"*.py\"\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s --url gitlab.com/myorg --config content-search.yaml\n", os.Args[0])
	}

	fs.Parse(args)
	config.FilePatterns = filePatterns
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

func validateSearchConfig(config *SearchConfig) error {
	if config.GitLabURL == "" {
		return fmt.Errorf("--url is required")
	}
	if config.Token == "" {
		return fmt.Errorf("--token is required (or set GITLAB_TOKEN environment variable)")
	}
	if config.SearchTerm == "" && config.ConfigFile == "" {
		return fmt.Errorf("--search or --config is required")
	}
	return nil
}
