package publisher

import (
	"bytes"
	"os"
	"testing"

	"github.com/ryosukesatoh/daily-feed/internal/config"
)

func TestStdoutPublisher(t *testing.T) {
	cfg := &config.Config{
		Publisher: config.PublisherConfig{
			Type: "stdout",
		},
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	p, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create publisher: %v", err)
	}

	err = p.Publish([]byte("test content"))
	if err != nil {
		t.Errorf("Publish failed: %v", err)
	}

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if output == "" {
		t.Error("No output from stdout publisher")
	}
}

func TestInvalidPublisherType(t *testing.T) {
	cfg := &config.Config{
		Publisher: config.PublisherConfig{
			Type: "invalid_type",
		},
	}

	_, err := New(cfg)
	if err == nil {
		t.Fatal("Expected error for invalid publisher type")
	}
}