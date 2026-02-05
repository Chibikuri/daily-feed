package summarizer

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ryosukesatoh/daily-feed/internal/fetcher"
)

func samplePapers() []fetcher.Paper {
	return []fetcher.Paper{
		{
			Title:    "Paper One",
			Authors:  []string{"Alice", "Bob"},
			Abstract: "Abstract one about AI.",
			URL:      "http://example.com/1",
			Category: "cs.AI",
		},
		{
			Title:    "Paper Two",
			Authors:  []string{"Charlie"},
			Abstract: "Abstract two about ML.",
			URL:      "http://example.com/2",
			Category: "cs.LG",
		},
	}
}

func TestParseResponseValidJSON(t *testing.T) {
	s := &AnthropicSummarizer{topic: "AI", topN: 5}
	papers := samplePapers()

	body := `{
		"overview": "Test overview of papers.",
		"summaries": [
			{"index": 1, "summary": "Summary of paper one.", "key_points": ["point A", "point B"]},
			{"index": 2, "summary": "Summary of paper two.", "key_points": ["point C"]}
		]
	}`

	digest, err := s.parseResponse(body, papers)
	if err != nil {
		t.Fatalf("parseResponse returned error: %v", err)
	}

	if digest.Topic != "AI" {
		t.Errorf("Expected topic 'AI', got %q", digest.Topic)
	}
	if digest.Overview != "Test overview of papers." {
		t.Errorf("Expected overview, got %q", digest.Overview)
	}
	if len(digest.Summaries) != 2 {
		t.Fatalf("Expected 2 summaries, got %d", len(digest.Summaries))
	}
	if digest.Summaries[0].Paper.Title != "Paper One" {
		t.Errorf("Expected first summary to map to 'Paper One', got %q", digest.Summaries[0].Paper.Title)
	}
	if digest.Summaries[1].Summary != "Summary of paper two." {
		t.Errorf("Expected second summary text, got %q", digest.Summaries[1].Summary)
	}
	if len(digest.Summaries[0].KeyPoints) != 2 {
		t.Errorf("Expected 2 key points for first summary, got %d", len(digest.Summaries[0].KeyPoints))
	}
}

func TestParseResponseMarkdownFences(t *testing.T) {
	s := &AnthropicSummarizer{topic: "AI", topN: 5}
	papers := samplePapers()

	body := "```json\n" + `{"overview": "Overview.", "summaries": [{"index": 1, "summary": "S1.", "key_points": []}]}` + "\n```"

	digest, err := s.parseResponse(body, papers)
	if err != nil {
		t.Fatalf("parseResponse with markdown fences returned error: %v", err)
	}
	if digest.Overview != "Overview." {
		t.Errorf("Expected 'Overview.', got %q", digest.Overview)
	}
}

func TestParseResponseOutOfBoundsIndex(t *testing.T) {
	s := &AnthropicSummarizer{topic: "AI", topN: 5}
	papers := samplePapers()

	body := `{
		"overview": "Overview.",
		"summaries": [
			{"index": 1, "summary": "Valid.", "key_points": []},
			{"index": 99, "summary": "Invalid index.", "key_points": []},
			{"index": 0, "summary": "Zero index.", "key_points": []}
		]
	}`

	digest, err := s.parseResponse(body, papers)
	if err != nil {
		t.Fatalf("parseResponse returned error: %v", err)
	}
	// Only index 1 is valid (0-based: 0). Index 99 and 0 should be skipped.
	if len(digest.Summaries) != 1 {
		t.Errorf("Expected 1 valid summary (skipping out-of-bounds), got %d", len(digest.Summaries))
	}
}

func TestParseResponseInvalidJSON(t *testing.T) {
	s := &AnthropicSummarizer{topic: "AI", topN: 5}
	papers := samplePapers()

	_, err := s.parseResponse("not json at all", papers)
	if err == nil {
		t.Fatal("Expected error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "failed to parse LLM JSON") {
		t.Errorf("Expected 'failed to parse LLM JSON' error, got: %v", err)
	}
}

func TestBuildPromptFormat(t *testing.T) {
	s := &AnthropicSummarizer{topic: "machine learning", topN: 3}
	papers := samplePapers()

	prompt := s.buildPrompt(papers)

	if !strings.Contains(prompt, "2 recent papers") {
		t.Error("Expected prompt to mention number of papers")
	}
	if !strings.Contains(prompt, `"machine learning"`) {
		t.Error("Expected prompt to contain topic")
	}
	if !strings.Contains(prompt, "--- Paper 1 ---") {
		t.Error("Expected prompt to contain '--- Paper 1 ---'")
	}
	if !strings.Contains(prompt, "--- Paper 2 ---") {
		t.Error("Expected prompt to contain '--- Paper 2 ---'")
	}
	if !strings.Contains(prompt, "Title: Paper One") {
		t.Error("Expected prompt to contain paper title")
	}
	if !strings.Contains(prompt, "Authors: Alice, Bob") {
		t.Error("Expected prompt to contain authors")
	}
	if !strings.Contains(prompt, "top 3") {
		t.Error("Expected prompt to reference topN value")
	}
}

func TestSummarizeEmptyPapers(t *testing.T) {
	s := &AnthropicSummarizer{topic: "AI", topN: 5}

	digest, err := s.Summarize(context.Background(), nil)
	if err != nil {
		t.Fatalf("Summarize with empty papers returned error: %v", err)
	}
	if digest.Topic != "AI" {
		t.Errorf("Expected topic 'AI', got %q", digest.Topic)
	}
	if digest.Overview != "No papers found for the given topic." {
		t.Errorf("Expected default overview, got %q", digest.Overview)
	}
	if len(digest.Summaries) != 0 {
		t.Errorf("Expected 0 summaries, got %d", len(digest.Summaries))
	}
}

func TestSummarizeWithMockAPI(t *testing.T) {
	responseJSON := digestJSON{
		Overview: "AI research overview.",
		Summaries: []summaryJSON{
			{Index: 1, Summary: "Summary of paper one.", KeyPoints: []string{"point A"}},
		},
	}
	apiResponse := anthropicResponse{
		Content: []anthropicContent{
			{Type: "text", Text: mustMarshal(t, responseJSON)},
		},
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-api-key") != "test-key" {
			t.Errorf("Expected x-api-key 'test-key', got %q", r.Header.Get("x-api-key"))
		}
		if r.Header.Get("anthropic-version") != "2023-06-01" {
			t.Errorf("Expected anthropic-version header")
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(apiResponse)
	}))
	defer ts.Close()

	s := &AnthropicSummarizer{
		apiKey:    "test-key",
		model:     "test-model",
		maxTokens: 1024,
		topN:      5,
		topic:     "AI",
		client:    ts.Client(),
	}

	// Override the API endpoint by replacing callAPI with our test server.
	// Since callAPI is hardcoded to api.anthropic.com, we test via parseResponse
	// and verify the full pipeline through the mock server indirectly.
	// We test Summarize by pointing the client at our test server via a custom transport.
	transport := &rewriteTransport{
		base:    ts.Client().Transport,
		testURL: ts.URL,
	}
	s.client = &http.Client{Transport: transport}

	papers := samplePapers()[:1]
	digest, err := s.Summarize(context.Background(), papers)
	if err != nil {
		t.Fatalf("Summarize returned error: %v", err)
	}

	if digest.Overview != "AI research overview." {
		t.Errorf("Expected overview 'AI research overview.', got %q", digest.Overview)
	}
	if len(digest.Summaries) != 1 {
		t.Fatalf("Expected 1 summary, got %d", len(digest.Summaries))
	}
	if digest.Summaries[0].Paper.Title != "Paper One" {
		t.Errorf("Expected paper title 'Paper One', got %q", digest.Summaries[0].Paper.Title)
	}
}

func TestSummarizeAPIError(t *testing.T) {
	apiResponse := anthropicResponse{
		Error: &anthropicError{
			Type:    "invalid_request_error",
			Message: "bad request",
		},
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(apiResponse)
	}))
	defer ts.Close()

	s := &AnthropicSummarizer{
		apiKey:    "test-key",
		model:     "test-model",
		maxTokens: 1024,
		topN:      5,
		topic:     "AI",
		client:    &http.Client{Transport: &rewriteTransport{testURL: ts.URL}},
	}

	_, err := s.Summarize(context.Background(), samplePapers()[:1])
	if err == nil {
		t.Fatal("Expected error for API error response")
	}
	if !strings.Contains(err.Error(), "API error") {
		t.Errorf("Expected 'API error' in error message, got: %v", err)
	}
}

// rewriteTransport redirects all requests to the test server URL.
type rewriteTransport struct {
	base    http.RoundTripper
	testURL string
}

func (t *rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.URL.Scheme = "http"
	req.URL.Host = strings.TrimPrefix(t.testURL, "http://")
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}
	return base.RoundTrip(req)
}

func mustMarshal(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}
	return string(b)
}
