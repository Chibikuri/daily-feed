package publisher

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"

	"github.com/ryosukesatoh/daily-feed/internal/summarizer"
)

// WebPublisher serves the latest digest as an HTML page over HTTP.
type WebPublisher struct {
	addr   string
	server *http.Server
	mu     sync.RWMutex
	latest *summarizer.Digest
}

func NewWebPublisher(addr string) *WebPublisher {
	wp := &WebPublisher{addr: addr}
	mux := http.NewServeMux()
	mux.HandleFunc("/", wp.handleIndex)
	wp.server = &http.Server{
		Addr:    addr,
		Handler: mux,
	}
	return wp
}

// Start begins serving HTTP in the background. Call Shutdown to stop.
func (wp *WebPublisher) Start() error {
	ln, err := net.Listen("tcp", wp.addr)
	if err != nil {
		return fmt.Errorf("web: failed to listen on %s: %w", wp.addr, err)
	}
	go func() {
		log.Printf("Web publisher listening on %s", wp.addr)
		if err := wp.server.Serve(ln); err != nil && err != http.ErrServerClosed {
			log.Printf("Web publisher error: %v", err)
		}
	}()
	return nil
}

// Shutdown gracefully shuts down the HTTP server.
func (wp *WebPublisher) Shutdown(ctx context.Context) error {
	return wp.server.Shutdown(ctx)
}

func (wp *WebPublisher) Publish(_ context.Context, digest *summarizer.Digest) error {
	wp.mu.Lock()
	wp.latest = digest
	wp.mu.Unlock()
	log.Printf("Web publisher updated with new digest for %q", digest.Topic)
	return nil
}

func (wp *WebPublisher) handleIndex(w http.ResponseWriter, r *http.Request) {
	wp.mu.RLock()
	digest := wp.latest
	wp.mu.RUnlock()

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if digest == nil {
		fmt.Fprint(w, `<!DOCTYPE html><html><body><h1>Daily Feed</h1><p>No digest available yet. Check back later.</p></body></html>`)
		return
	}

	fmt.Fprint(w, buildHTMLBody(digest))
}
