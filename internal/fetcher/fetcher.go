package fetcher

import (
	"context"
	"time"
)

// Paper represents a single academic publication.
type Paper struct {
	Title     string
	Authors   []string
	Abstract  string
	URL       string
	Published time.Time
	Category  string
}

// Fetcher retrieves recent academic papers for given topics.
type Fetcher interface {
	Fetch(ctx context.Context, topic string, maxResults int) ([]Paper, error)
	FetchMultiple(ctx context.Context, topics []string, maxResults int) ([]Paper, error)
}