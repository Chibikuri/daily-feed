package runner

import (
	"context"
	"fmt"
	"log"

	"github.com/ryosukesatoh/daily-feed/internal/fetcher"
	"github.com/ryosukesatoh/daily-feed/internal/publisher"
	"github.com/ryosukesatoh/daily-feed/internal/summarizer"
)

// Runner orchestrates the fetch -> summarize -> publish pipeline.
type Runner struct {
	topic      string
	maxResults int
	fetcher    fetcher.Fetcher
	summarizer summarizer.Summarizer
	publishers []publisher.Publisher
}

func New(topic string, maxResults int, f fetcher.Fetcher, s summarizer.Summarizer, pubs []publisher.Publisher) *Runner {
	return &Runner{
		topic:      topic,
		maxResults: maxResults,
		fetcher:    f,
		summarizer: s,
		publishers: pubs,
	}
}

// Run executes the full pipeline once.
func (r *Runner) Run(ctx context.Context) error {
	log.Printf("Starting pipeline for topic %q (max_results=%d)", r.topic, r.maxResults)

	// Step 1: Fetch papers
	log.Println("Fetching papers...")
	papers, err := r.fetcher.Fetch(ctx, r.topic, r.maxResults)
	if err != nil {
		return fmt.Errorf("runner: fetch failed: %w", err)
	}
	log.Printf("Fetched %d papers", len(papers))

	// Step 2: Summarize
	log.Println("Summarizing papers...")
	digest, err := r.summarizer.Summarize(ctx, papers)
	if err != nil {
		return fmt.Errorf("runner: summarize failed: %w", err)
	}
	log.Printf("Generated digest with %d summaries", len(digest.Summaries))

	// Step 3: Publish
	for _, pub := range r.publishers {
		log.Printf("Publishing via %T...", pub)
		if err := pub.Publish(ctx, digest); err != nil {
			log.Printf("WARNING: publish via %T failed: %v", pub, err)
		}
	}

	log.Println("Pipeline complete")
	return nil
}
