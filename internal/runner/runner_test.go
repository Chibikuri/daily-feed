package runner

import (
	"testing"

	"github.com/ryosukesatoh/daily-feed/internal/config"
	"github.com/ryosukesatoh/daily-feed/internal/fetcher"
	"github.com/ryosukesatoh/daily-feed/internal/publisher"
	"github.com/ryosukesatoh/daily-feed/internal/summarizer"
)

func TestRunnerInitialization(t *testing.T) {
	cfg := &config.Config{
		Topic: "test topic",
		RunOnStart: true,
		Fetcher: config.FetcherConfig{Type: "arxiv"},
		Summarizer: config.SummarizerConfig{
			Type: "anthropic", 
			APIKey: "test_key",
		},
		Publisher: config.PublisherConfig{Type: "stdout"},
	}

	f, _ := fetcher.New(cfg)
	s, _ := summarizer.New(cfg)
	p, _ := publisher.New(cfg)

	r, err := New(cfg, f, s, p)
	if err != nil {
		t.Fatalf("Failed to create runner: %v", err)
	}

	if r == nil {
		t.Error("Runner should not be nil")
	}
}

func TestRunnerExecution(t *testing.T) {
	cfg := &config.Config{
		Topic: "test topic",
		RunOnStart: true,
		Fetcher: config.FetcherConfig{Type: "arxiv"},
		Summarizer: config.SummarizerConfig{
			Type: "anthropic", 
			APIKey: "test_key",
		},
		Publisher: config.PublisherConfig{Type: "stdout"},
	}

	f, _ := fetcher.New(cfg)
	s, _ := summarizer.New(cfg)
	p, _ := publisher.New(cfg)

	r, _ := New(cfg, f, s, p)

	err := r.Run()
	if err != nil {
		t.Fatalf("Runner execution failed: %v", err)
	}
}