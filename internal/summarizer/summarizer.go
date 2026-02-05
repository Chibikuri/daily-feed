package summarizer

import (
	"fmt"
	"github.com/ryosukesatoh/daily-feed/internal/config"
	"context"
)

// New creates a new summarizer based on the configuration
func New(cfg *config.Config) (Summarizer, error) {
	switch cfg.Summarizer.Type {
	case "anthropic":
		return NewAnthropicSummarizer(
			cfg.Summarizer.APIKey, 
			cfg.Summarizer.Model, 
			4096, // maxTokens default
			3,   // topN default 
			cfg.Topic,
		), nil
	default:
		return nil, ErrUnsupportedSummarizerType
	}
}

// ErrUnsupportedSummarizerType is returned when an unsupported summarizer type is specified
var ErrUnsupportedSummarizerType = fmt.Errorf("unsupported summarizer type")