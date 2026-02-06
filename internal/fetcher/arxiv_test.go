package fetcher

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

const sampleAtomFeed = `<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <entry>
    <title>  Sample Paper Title  </title>
    <summary>  This is the abstract of the paper.  </summary>
    <author><name> Alice </name></author>
    <author><name> Bob </name></author>
    <link href="http://arxiv.org/abs/1234.5678" rel="alternate" type="text/html"/>
    <link href="http://arxiv.org/pdf/1234.5678" title="pdf" type="application/pdf"/>
    <published>2025-01-15T00:00:00Z</published>
    <category term="cs.AI"/>
  </entry>
  <entry>
    <title>Another Paper</title>
    <summary>Second abstract.</summary>
    <author><name>Charlie</name></author>
    <link href="http://arxiv.org/abs/2345.6789" rel="alternate" type="text/html"/>
    <published>2025-01-14T00:00:00Z</published>
    <category term="cs.LG"/>
    <category term="cs.CL"/>
  </entry>
</feed>`

func TestFetchParsesAtomFeed(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(sampleAtomFeed))
	}))
	defer ts.Close()

	f := &ArxivFetcher{
		client:  ts.Client(),
		baseURL: ts.URL,
	}

	papers, err := f.Fetch(context.Background(), "machine learning", 10)
	if err != nil {
		t.Fatalf("Fetch returned error: %v", err)
	}

	if len(papers) != 2 {
		t.Fatalf("Expected 2 papers, got %d", len(papers))
	}

	// First paper
	p := papers[0]
	if p.Title != "Sample Paper Title" {
		t.Errorf("Expected trimmed title 'Sample Paper Title', got %q", p.Title)
	}
	if p.Abstract != "This is the abstract of the paper." {
		t.Errorf("Expected trimmed abstract, got %q", p.Abstract)
	}
	if len(p.Authors) != 2 {
		t.Fatalf("Expected 2 authors, got %d", len(p.Authors))
	}
	if p.Authors[0] != "Alice" {
		t.Errorf("Expected author 'Alice', got %q", p.Authors[0])
	}
	if p.Authors[1] != "Bob" {
		t.Errorf("Expected author 'Bob', got %q", p.Authors[1])
	}
	if p.URL != "http://arxiv.org/abs/1234.5678" {
		t.Errorf("Expected alternate link URL, got %q", p.URL)
	}
	if p.Category != "cs.AI" {
		t.Errorf("Expected category 'cs.AI', got %q", p.Category)
	}
	if p.Published.Year() != 2025 || p.Published.Month() != 1 || p.Published.Day() != 15 {
		t.Errorf("Unexpected published date: %v", p.Published)
	}

	// Second paper
	p2 := papers[1]
	if p2.Title != "Another Paper" {
		t.Errorf("Expected 'Another Paper', got %q", p2.Title)
	}
	if p2.Category != "cs.LG" {
		t.Errorf("Expected first category 'cs.LG', got %q", p2.Category)
	}
}

func TestFetchQueryParameters(t *testing.T) {
	var receivedQuery string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?><feed xmlns="http://www.w3.org/2005/Atom"></feed>`))
	}))
	defer ts.Close()

	f := &ArxivFetcher{
		client:  ts.Client(),
		baseURL: ts.URL,
	}

	_, err := f.Fetch(context.Background(), "quantum computing", 5)
	if err != nil {
		t.Fatalf("Fetch returned error: %v", err)
	}

	if receivedQuery == "" {
		t.Fatal("No query parameters sent")
	}
	// Check key parameters are present
	for _, want := range []string{"search_query=all%3Aquantum+computing", "max_results=5", "sortBy=submittedDate", "sortOrder=descending"} {
		if !contains(receivedQuery, want) {
			t.Errorf("Expected query to contain %q, got %q", want, receivedQuery)
		}
	}
}

func TestFetchMultipleTopics(t *testing.T) {
	var receivedQuery string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(sampleAtomFeed))
	}))
	defer ts.Close()

	f := &ArxivFetcher{
		client:  ts.Client(),
		baseURL: ts.URL,
	}

	topics := []string{"quantum computing", "artificial intelligence"}
	papers, err := f.FetchMultiple(context.Background(), topics, 5)
	if err != nil {
		t.Fatalf("FetchMultiple returned error: %v", err)
	}

	if len(papers) != 2 {
		t.Fatalf("Expected 2 papers, got %d", len(papers))
	}

	// Check that the query contains both topics
	if !contains(receivedQuery, "quantum+computing") || !contains(receivedQuery, "artificial+intelligence") {
		t.Errorf("Expected query to contain both topics, got %q", receivedQuery)
	}
	
	// Check that OR logic is used
	if !contains(receivedQuery, "OR") {
		t.Errorf("Expected query to use OR logic, got %q", receivedQuery)
	}
}

func TestFetchMultipleTopicsEmpty(t *testing.T) {
	f := NewArxivFetcher()
	
	papers, err := f.FetchMultiple(context.Background(), []string{}, 5)
	if err != nil {
		t.Fatalf("FetchMultiple with empty topics returned error: %v", err)
	}
	
	if len(papers) != 0 {
		t.Errorf("Expected 0 papers for empty topics, got %d", len(papers))
	}
}

func TestFetchMultipleTopicsSingleTopic(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(sampleAtomFeed))
	}))
	defer ts.Close()

	f := &ArxivFetcher{
		client:  ts.Client(),
		baseURL: ts.URL,
	}

	// With single topic, should delegate to Fetch method
	topics := []string{"quantum computing"}
	papers, err := f.FetchMultiple(context.Background(), topics, 5)
	if err != nil {
		t.Fatalf("FetchMultiple with single topic returned error: %v", err)
	}

	if len(papers) != 2 {
		t.Fatalf("Expected 2 papers, got %d", len(papers))
	}
}

func TestFetchBadStatusCode(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	f := &ArxivFetcher{
		client:  ts.Client(),
		baseURL: ts.URL,
	}

	_, err := f.Fetch(context.Background(), "test", 5)
	if err == nil {
		t.Fatal("Expected error for 500 status code")
	}
	if !contains(err.Error(), "unexpected status 500") {
		t.Errorf("Expected 'unexpected status 500' error, got: %v", err)
	}
}

func TestFetchInvalidXML(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte("this is not xml"))
	}))
	defer ts.Close()

	f := &ArxivFetcher{
		client:  ts.Client(),
		baseURL: ts.URL,
	}

	_, err := f.Fetch(context.Background(), "test", 5)
	if err == nil {
		t.Fatal("Expected error for invalid XML")
	}
	if !contains(err.Error(), "failed to parse XML") {
		t.Errorf("Expected 'failed to parse XML' error, got: %v", err)
	}
}

func TestFetchEmptyFeed(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?><feed xmlns="http://www.w3.org/2005/Atom"></feed>`))
	}))
	defer ts.Close()

	f := &ArxivFetcher{
		client:  ts.Client(),
		baseURL: ts.URL,
	}

	papers, err := f.Fetch(context.Background(), "test", 5)
	if err != nil {
		t.Fatalf("Fetch returned error: %v", err)
	}
	if len(papers) != 0 {
		t.Errorf("Expected 0 papers, got %d", len(papers))
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}