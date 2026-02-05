package summarizer

import (
	"context"
	"time"
	"github.com/ryosukesatoh/daily-feed/internal/fetcher"
)

// Digest represents a summary of research papers for a given topic
type Digest struct {
	Topic      string         `json:"topic"`
	Date       time.Time      `json:"date"`
	Overview   string         `json:"overview"`
	Summaries  []PaperSummary `json:"summaries"`
}

// PaperSummary contains summary information for a specific paper
type PaperSummary struct {
	Paper     fetcher.Paper `json:"paper"`
	Summary   string        `json:"summary"`
	KeyPoints []string      `json:"key_points"`
}

// Summarizer is an interface for summarizing research papers
type Summarizer interface {
	Summarize(ctx context.Context, papers []fetcher.Paper) (*Digest, error)
}