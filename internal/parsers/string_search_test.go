package parsers

import (
	"testing"
)

func TestStringSearchParser_LiteralSearch(t *testing.T) {
	parser := &StringSearchParser{
		SearchTerm:    "TODO",
		CaseSensitive: true,
	}

	content := []byte("line one\nTODO: fix this\nline three\nTODO: and this\n")
	matches, err := parser.Search(content, "test.go")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(matches) != 2 {
		t.Fatalf("expected 2 matches, got %d", len(matches))
	}

	if matches[0].LineNumber != 2 {
		t.Errorf("match[0] line = %d, want 2", matches[0].LineNumber)
	}
	if matches[0].MatchedText != "TODO" {
		t.Errorf("match[0] text = %q, want %q", matches[0].MatchedText, "TODO")
	}
	if matches[1].LineNumber != 4 {
		t.Errorf("match[1] line = %d, want 4", matches[1].LineNumber)
	}
}

func TestStringSearchParser_CaseInsensitive(t *testing.T) {
	parser := &StringSearchParser{
		SearchTerm:    "todo",
		CaseSensitive: false,
	}

	content := []byte("TODO: fix\ntodo: also\nToDo: mixed\nno match here\n")
	matches, err := parser.Search(content, "test.go")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(matches) != 3 {
		t.Fatalf("expected 3 matches, got %d", len(matches))
	}
}

func TestStringSearchParser_CaseSensitiveNoMatch(t *testing.T) {
	parser := &StringSearchParser{
		SearchTerm:    "TODO",
		CaseSensitive: true,
	}

	content := []byte("todo: lowercase\n")
	matches, err := parser.Search(content, "test.go")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(matches) != 0 {
		t.Fatalf("expected 0 matches, got %d", len(matches))
	}
}

func TestStringSearchParser_RegexSearch(t *testing.T) {
	parser := &StringSearchParser{
		SearchTerm:    `password\s*=\s*"[^"]+"`,
		IsRegex:       true,
		CaseSensitive: true,
	}

	content := []byte("username = \"admin\"\npassword = \"secret123\"\nhost = \"localhost\"\n")
	matches, err := parser.Search(content, "config.py")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matches))
	}
	if matches[0].LineNumber != 2 {
		t.Errorf("match line = %d, want 2", matches[0].LineNumber)
	}
	if matches[0].MatchedText != `password = "secret123"` {
		t.Errorf("match text = %q, want %q", matches[0].MatchedText, `password = "secret123"`)
	}
}

func TestStringSearchParser_RegexCaseInsensitive(t *testing.T) {
	parser := &StringSearchParser{
		SearchTerm:    `API_KEY`,
		IsRegex:       true,
		CaseSensitive: false,
	}

	content := []byte("api_key = 'abc'\nAPI_KEY = 'def'\nApi_Key = 'ghi'\n")
	matches, err := parser.Search(content, "test.py")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(matches) != 3 {
		t.Fatalf("expected 3 matches, got %d", len(matches))
	}
}

func TestStringSearchParser_MaxMatches(t *testing.T) {
	parser := &StringSearchParser{
		SearchTerm: "match",
		MaxMatches: 2,
	}

	content := []byte("match 1\nmatch 2\nmatch 3\nmatch 4\n")
	matches, err := parser.Search(content, "test.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(matches) != 2 {
		t.Fatalf("expected 2 matches (max), got %d", len(matches))
	}
}

func TestStringSearchParser_NoMatch(t *testing.T) {
	parser := &StringSearchParser{
		SearchTerm: "nonexistent",
	}

	content := []byte("nothing here\nstill nothing\n")
	matches, err := parser.Search(content, "test.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(matches) != 0 {
		t.Fatalf("expected 0 matches, got %d", len(matches))
	}
}

func TestStringSearchParser_EmptyContent(t *testing.T) {
	parser := &StringSearchParser{
		SearchTerm: "test",
	}

	matches, err := parser.Search([]byte(""), "empty.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(matches) != 0 {
		t.Fatalf("expected 0 matches, got %d", len(matches))
	}
}

func TestStringSearchParser_EmptySearchTerm(t *testing.T) {
	parser := &StringSearchParser{
		SearchTerm: "",
	}

	_, err := parser.Search([]byte("some content"), "test.txt")
	if err == nil {
		t.Fatal("expected error for empty search term")
	}
}

func TestStringSearchParser_InvalidRegex(t *testing.T) {
	parser := &StringSearchParser{
		SearchTerm: "[invalid",
		IsRegex:    true,
	}

	_, err := parser.Search([]byte("test"), "test.txt")
	if err == nil {
		t.Fatal("expected error for invalid regex")
	}
}

func TestStringSearchParser_AsParserFunc(t *testing.T) {
	parser := &StringSearchParser{
		SearchTerm: "found",
	}

	parserFunc := parser.AsParserFunc()

	// Test with matching content
	result, err := parserFunc([]byte("line 1\nfound it\nline 3\n"), "test.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Found {
		t.Error("expected Found=true")
	}
	if result.Metadata["match_count"] != "1" {
		t.Errorf("match_count = %q, want %q", result.Metadata["match_count"], "1")
	}

	// Test with non-matching content
	result, err = parserFunc([]byte("nothing here\n"), "test.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Found {
		t.Error("expected Found=false")
	}
}

func TestStringSearchParser_FilePath(t *testing.T) {
	parser := &StringSearchParser{
		SearchTerm: "test",
	}

	matches, err := parser.Search([]byte("this is a test"), "src/main.py")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matches))
	}
	if matches[0].FilePath != "src/main.py" {
		t.Errorf("FilePath = %q, want %q", matches[0].FilePath, "src/main.py")
	}
}
