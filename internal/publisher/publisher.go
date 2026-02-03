package publisher

import (
	"context"

	"github.com/ryosukesatoh/daily-feed/internal/summarizer"
)

// Publisher publishes a digest to some output destination.
type Publisher interface {
	Publish(ctx context.Context, digest *summarizer.Digest) error
}
