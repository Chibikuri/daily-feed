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

// Fetcher retrieves recent academic papers for a given topic.
type Fetcher interface {
	Fetch(ctx context.Context, topic string, maxResults int) ([]Paper, error)
}
