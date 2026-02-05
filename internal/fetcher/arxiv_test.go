package fetcher

import (
	"context"
	"testing"

	"github.com/ryosukesatoh/daily-feed/internal/config"
)

func TestArxivFetcher(t *testing.T) {
	cfg := &config.Config{
		Topic: "machine learning",
		Fetcher: config.FetcherConfig{
			Type: "arxiv",
		},
		MaxResults: 5,
	}

	f, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create fetcher: %v", err)
	}

	if f == nil {
		t.Fatal("Fetcher is nil")
	}

	results, err := f.Fetch(context.Background(), cfg.Topic, cfg.MaxResults)
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}

	if len(results) > cfg.MaxResults {
		t.Errorf("Expected max %d results, got %d", cfg.MaxResults, len(results))
	}
}