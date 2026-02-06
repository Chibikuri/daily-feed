package main

import (
	"os"
	"testing"

	"github.com/ryosukesatoh/daily-feed/internal/config"
)

func TestMultipleTopicsIntegration(t *testing.T) {
	// Test single topic configuration (legacy)
	singleTopicConfig := `
topic: "quantum computing"
language: "en"
publisher:
  type: "stdout"
summarizer:
  type: "anthropic"
  api_key: "test_key"
`
	tmpfile, err := createTempConfig(t, singleTopicConfig)
	if err != nil {
		t.Fatalf("Failed to create temp config: %v", err)
	}
	defer tmpfile.cleanup()

	cfg, err := config.Load(tmpfile.path)
	if err != nil {
		t.Fatalf("Failed to load single topic config: %v", err)
	}

	topics := cfg.GetTopics()
	if len(topics) != 1 || topics[0] != "quantum computing" {
		t.Errorf("Expected single topic 'quantum computing', got %v", topics)
	}

	// Test multiple topics configuration
	multiTopicConfig := `
topics: ["quantum computing", "artificial intelligence"]
language: "en"
publisher:
  type: "stdout"
summarizer:
  type: "anthropic"
  api_key: "test_key"
`
	tmpfile2, err := createTempConfig(t, multiTopicConfig)
	if err != nil {
		t.Fatalf("Failed to create temp config: %v", err)
	}
	defer tmpfile2.cleanup()

	cfg2, err := config.Load(tmpfile2.path)
	if err != nil {
		t.Fatalf("Failed to load multi topic config: %v", err)
	}

	topics2 := cfg2.GetTopics()
	expected := []string{"quantum computing", "artificial intelligence"}
	if len(topics2) != len(expected) {
		t.Errorf("Expected %d topics, got %d", len(expected), len(topics2))
	}
	for i, topic := range topics2 {
		if topic != expected[i] {
			t.Errorf("Expected topic[%d] '%s', got '%s'", i, expected[i], topic)
		}
	}

	topicsString := cfg2.GetTopicsString()
	expectedString := "quantum computing, artificial intelligence"
	if topicsString != expectedString {
		t.Errorf("Expected topics string '%s', got '%s'", expectedString, topicsString)
	}
}

type tempConfig struct {
	path    string
	cleanup func()
}

func createTempConfig(t *testing.T, content string) (*tempConfig, error) {
	tmpfile, err := os.CreateTemp("", "integration_test_*.yaml")
	if err != nil {
		return nil, err
	}

	if _, err := tmpfile.Write([]byte(content)); err != nil {
		os.Remove(tmpfile.Name())
		return nil, err
	}
	tmpfile.Close()

	return &tempConfig{
		path: tmpfile.Name(),
		cleanup: func() {
			os.Remove(tmpfile.Name())
		},
	}, nil
}