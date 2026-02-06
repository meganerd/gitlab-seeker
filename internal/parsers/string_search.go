package parsers

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/gbjohnso/gitlab-python-scanner/internal/output"
	"github.com/gbjohnso/gitlab-python-scanner/internal/rules"
)

// StringSearchParser searches file content for arbitrary strings or regex patterns
type StringSearchParser struct {
	SearchTerm    string // The literal string or regex pattern to search for
	IsRegex       bool   // Whether SearchTerm is a regex
	CaseSensitive bool   // Whether the search is case-sensitive
	ContextLines  int    // Number of context lines before/after each match
	MaxMatches    int    // Maximum matches to return (0 = unlimited)

	compiled *regexp.Regexp // Compiled regex (set on first use)
}

// Search finds all occurrences of the search term in the given content
func (p *StringSearchParser) Search(content []byte, filename string) ([]output.ContentMatchEntry, error) {
	if p.SearchTerm == "" {
		return nil, fmt.Errorf("search term cannot be empty")
	}

	if err := p.ensureCompiled(); err != nil {
		return nil, err
	}

	lines := strings.Split(string(content), "\n")
	var matches []output.ContentMatchEntry

	for i, line := range lines {
		var matched bool
		var matchedText string

		if p.compiled != nil {
			loc := p.compiled.FindStringIndex(line)
			if loc != nil {
				matched = true
				matchedText = line[loc[0]:loc[1]]
			}
		} else {
			searchIn := line
			searchFor := p.SearchTerm
			if !p.CaseSensitive {
				searchIn = strings.ToLower(searchIn)
				searchFor = strings.ToLower(searchFor)
			}
			idx := strings.Index(searchIn, searchFor)
			if idx >= 0 {
				matched = true
				matchedText = line[idx : idx+len(p.SearchTerm)]
			}
		}

		if matched {
			matches = append(matches, output.ContentMatchEntry{
				FilePath:    filename,
				LineNumber:  i + 1,
				LineContent: strings.TrimRight(line, "\r"),
				MatchedText: matchedText,
			})

			if p.MaxMatches > 0 && len(matches) >= p.MaxMatches {
				break
			}
		}
	}

	return matches, nil
}

// AsParserFunc returns a rules.ParserFunc adapter for use in the existing rule engine
func (p *StringSearchParser) AsParserFunc() rules.ParserFunc {
	return func(content []byte, filename string) (*rules.SearchResult, error) {
		matches, err := p.Search(content, filename)
		if err != nil {
			return nil, err
		}

		if len(matches) == 0 {
			return &rules.SearchResult{Found: false}, nil
		}

		return &rules.SearchResult{
			Found:    true,
			Version:  matches[0].MatchedText,
			Source:   filename,
			RawValue: matches[0].LineContent,
			Metadata: map[string]string{
				"match_count": fmt.Sprintf("%d", len(matches)),
				"line_number": fmt.Sprintf("%d", matches[0].LineNumber),
			},
		}, nil
	}
}

// ensureCompiled compiles the regex pattern if needed
func (p *StringSearchParser) ensureCompiled() error {
	if !p.IsRegex {
		return nil
	}
	if p.compiled != nil {
		return nil
	}

	pattern := p.SearchTerm
	if !p.CaseSensitive {
		pattern = "(?i)" + pattern
	}

	var err error
	p.compiled, err = regexp.Compile(pattern)
	if err != nil {
		return fmt.Errorf("invalid regex pattern %q: %w", p.SearchTerm, err)
	}
	return nil
}
