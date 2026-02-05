package summarizer

import (
	"fmt"
	"github.com/ryosukesatoh/daily-feed/internal/config"
)

// New creates a new summarizer based on the configuration
func New(cfg *config.Config) (Summarizer, error) {
	switch cfg.Summarizer.Type {
	case "anthropic":
		return NewAnthropicSummarizer(cfg.Summarizer.APIKey, cfg.Summarizer.Model), nil
	default:
		return nil, ErrUnsupportedSummarizerType
	}
}

// ErrUnsupportedSummarizerType is returned when an unsupported summarizer type is specified
var ErrUnsupportedSummarizerType = fmt.Errorf("unsupported summarizer type")