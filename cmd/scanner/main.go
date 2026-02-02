package main

import (
	"flag"
	"fmt"
	"os"
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

	// TODO: Implement scanning logic
	fmt.Printf("GitLab Python Version Scanner\n")
	fmt.Printf("==============================\n\n")
	fmt.Printf("Scanning: %s\n", config.GitLabURL)
	if config.LogFile != "" {
		fmt.Printf("Logging to: %s\n", config.LogFile)
	}
	fmt.Println("\nScanning not yet implemented - see tasks with: bd ready")
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
