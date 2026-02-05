package publisher

import (
	"context"
	"fmt"
	"github.com/ryosukesatoh/daily-feed/internal/config"
	"github.com/ryosukesatoh/daily-feed/internal/summarizer"
)

// Publisher is an interface for publishing summaries
type Publisher interface {
	Publish(ctx context.Context, summary *summarizer.Digest) error
}

// ErrUnsupportedPublisherType is returned when an unsupported publisher type is specified
var ErrUnsupportedPublisherType = fmt.Errorf("unsupported publisher type")

// New creates a new publisher based on the configuration
func New(cfg *config.Config) (Publisher, error) {
	switch cfg.Publisher.Type {
	case "discord":
		return NewDiscordPublisher(cfg.Publisher.Discord.WebhookURL), nil
	case "stdout":
		return NewStdoutPublisher(), nil
	case "email":
		return NewEmailPublisher(
			cfg.Publisher.Email.SMTPHost,
			cfg.Publisher.Email.SMTPPort,
			cfg.Publisher.Email.Username,
			cfg.Publisher.Email.Password,
			cfg.Publisher.Email.From,
			cfg.Publisher.Email.To,
		), nil
	case "web":
		return NewWebPublisher(cfg.Publisher.Web.Addr), nil
	default:
		return nil, ErrUnsupportedPublisherType
	}
}