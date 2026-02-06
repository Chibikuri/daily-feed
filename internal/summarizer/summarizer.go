package summarizer

import (
	"context"
	"strings"
	"time"

	"github.com/ryosukesatoh/daily-feed/internal/fetcher"
)

// PaperSummary holds a paper and its generated summary.
type PaperSummary struct {
	Paper     fetcher.Paper
	Summary   string
	KeyPoints []string
}

// Digest is the final output of the summarization pipeline.
type Digest struct {
	Topic     string    // Legacy single topic for backward compatibility
	Topics    []string  // Multiple topics
	Date      time.Time
	Summaries []PaperSummary
	Overview  string // High-level overview of all papers
}

// GetTopicsString returns a comma-separated string of all topics for display purposes.
func (d *Digest) GetTopicsString() string {
	if len(d.Topics) > 0 {
		return strings.Join(d.Topics, ", ")
	}
	return d.Topic
}

// Summarizer takes a list of papers and produces a digest with summaries.
type Summarizer interface {
	Summarize(ctx context.Context, papers []fetcher.Paper) (*Digest, error)
}