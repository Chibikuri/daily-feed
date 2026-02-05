package fetcher

import (
	"github.com/ryosukesatoh/daily-feed/internal/config"
)

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