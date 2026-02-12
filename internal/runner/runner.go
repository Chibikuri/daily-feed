package runner

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/ryosukesatoh/daily-feed/internal/fetcher"
	"github.com/ryosukesatoh/daily-feed/internal/publisher"
	"github.com/ryosukesatoh/daily-feed/internal/summarizer"
)

// Runner orchestrates the fetch -> summarize -> publish pipeline.
type Runner struct {
	topic      string   // Legacy single topic for backward compatibility
	topics     []string // Multiple topics
	maxResults int
	fetcher    fetcher.Fetcher
	summarizer summarizer.Summarizer
	publishers []publisher.Publisher
}

func New(topic string, maxResults int, f fetcher.Fetcher, s summarizer.Summarizer, pubs []publisher.Publisher) *Runner {
	return &Runner{
		topic:      topic,
		topics:     []string{topic}, // Initialize with single topic for backward compatibility
		maxResults: maxResults,
		fetcher:    f,
		summarizer: s,
		publishers: pubs,
	}
}

func NewMultiTopic(topics []string, maxResults int, f fetcher.Fetcher, s summarizer.Summarizer, pubs []publisher.Publisher) *Runner {
	// For backward compatibility, set the first topic as the legacy topic
	var topic string
	if len(topics) > 0 {
		topic = topics[0]
	}
	
	return &Runner{
		topic:      topic,
		topics:     topics,
		maxResults: maxResults,
		fetcher:    f,
		summarizer: s,
		publishers: pubs,
	}
}

// GetTopics returns the topics, prioritizing the new topics field over the legacy topic field.
func (r *Runner) GetTopics() []string {
	if len(r.topics) > 0 {
		return r.topics
	}
	if r.topic != "" {
		return []string{r.topic}
	}
	return []string{}
}

// GetTopicsString returns a comma-separated string of all topics for display purposes.
func (r *Runner) GetTopicsString() string {
	return strings.Join(r.GetTopics(), ", ")
}

// Run executes the full pipeline once.
func (r *Runner) Run(ctx context.Context) error {
	topics := r.GetTopics()
	topicsString := r.GetTopicsString()
	
	log.Printf("Starting pipeline for topic(s) %q (max_results=%d)", topicsString, r.maxResults)

	// Step 1: Fetch papers
	log.Println("Fetching papers...")
	var papers []fetcher.Paper
	var err error
	
	if len(topics) == 1 {
		papers, err = r.fetcher.Fetch(ctx, topics[0], r.maxResults)
	} else {
		papers, err = r.fetcher.FetchMultiple(ctx, topics, r.maxResults)
	}
	
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

	// Step 3: Publish - Continue with other publishers even if one fails
	var publishErrors []error
	for _, pub := range r.publishers {
		log.Printf("Publishing via %T...", pub)
		if err := pub.Publish(ctx, digest); err != nil {
			publishError := fmt.Errorf("publish via %T failed: %w", pub, err)
			publishErrors = append(publishErrors, publishError)
			log.Printf("WARNING: %v", publishError)
		} else {
			log.Printf("Successfully published via %T", pub)
		}
	}

	// If all publishers failed, return an error
	if len(publishErrors) == len(r.publishers) && len(r.publishers) > 0 {
		return fmt.Errorf("runner: all publishers failed: %v", publishErrors)
	}

	// If some publishers succeeded, log the failures but don't fail the pipeline
	if len(publishErrors) > 0 {
		log.Printf("Pipeline completed with %d publisher failures out of %d publishers", len(publishErrors), len(r.publishers))
	} else {
		log.Println("Pipeline completed successfully")
	}
	
	return nil
}