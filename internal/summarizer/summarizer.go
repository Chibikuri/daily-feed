package summarizer

import (
	"context"
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
	Topic     string
	Date      time.Time
	Summaries []PaperSummary
	Overview  string // High-level overview of all papers
}

// Summarizer takes a list of papers and produces a digest with summaries.
type Summarizer interface {
	Summarize(ctx context.Context, papers []fetcher.Paper) (*Digest, error)
}
