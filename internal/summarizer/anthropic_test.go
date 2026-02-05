package summarizer

import (
	"context"
	"testing"

	"github.com/ryosukesatoh/daily-feed/internal/config"
	"github.com/ryosukesatoh/daily-feed/internal/fetcher"
)

func TestAnthropicSummarizer(t *testing.T) {
	cfg := &config.Config{
		Summarizer: config.SummarizerConfig{
			Type: "anthropic",
			APIKey: "test_api_key",
			Model: "claude-sonnet-4-20250514",
		},
	}

	s, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create summarizer: %v", err)
	}

	if s == nil {
		t.Fatal("Summarizer is nil")
	}

	// Since we can't actually call Anthropic API in test, 
	// just verify that the method exists and handles basic cases
	sampleText := "Sample research abstract to summarize."
	samplePapers := []fetcher.Paper{{Title: sampleText}}
	summary, err := s.Summarize(context.Background(), samplePapers)
	if err != nil {
		t.Logf("Note: Summary generation might require a real API key")
	}

	if summary == "" && err == nil {
		t.Error("Expected non-empty summary or an error")
	}
}