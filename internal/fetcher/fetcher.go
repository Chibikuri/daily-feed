package fetcher

import (
	"context"
	"fmt"
	"time"

	"github.com/ryosukesatoh/daily-feed/internal/config"
)

// Paper represents a research paper with its metadata
type Paper struct {
	Title     string
	Authors   []string
	Abstract  string
	URL       string
	Published time.Time
	Category  string
}

// Fetcher is an interface for fetching research papers from various sources
type Fetcher interface {
	Fetch(ctx context.Context, topic string, maxResults int) ([]Paper, error)
}

// New creates a new fetcher based on the configuration
func New(cfg *config.Config) (Fetcher, error) {
	switch cfg.Fetcher.Type {
	case "arxiv":
		return NewArxivFetcher(), nil
	default:
		return nil, ErrUnsupportedFetcherType
	}
}

// ErrUnsupportedFetcherType is returned when an unsupported fetcher type is specified
var ErrUnsupportedFetcherType = fmt.Errorf("unsupported fetcher type")