package publisher

import (
	"fmt"
	"github.com/ryosukesatoh/daily-feed/internal/config"
)

// New creates a new publisher based on the configuration
func New(cfg *config.Config) (Publisher, error) {
	switch cfg.Publisher.Type {
	case "slack":
		return NewSlackPublisher(cfg.Publisher.WebhookURL), nil
	default:
		return nil, ErrUnsupportedPublisherType
	}
}

// ErrUnsupportedPublisherType is returned when an unsupported publisher type is specified
var ErrUnsupportedPublisherType = fmt.Errorf("unsupported publisher type")