package config

import (
	"os"
	"strings"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	tmpConfig := `
topic: test topic
publisher:
  type: stdout
summarizer:
  type: anthropic
  api_key: test_api_key
`
	tmpfile, err := os.CreateTemp("", "config_test_*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(tmpConfig)); err != nil {
		t.Fatalf("Failed to write temp config: %v", err)
	}
	tmpfile.Close()

	cfg, err := Load(tmpfile.Name())
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg.Topic != "test topic" {
		t.Errorf("Expected topic 'test topic', got '%s'", cfg.Topic)
	}
	topics := cfg.GetTopics()
	if len(topics) != 1 || topics[0] != "test topic" {
		t.Errorf("Expected topics ['test topic'], got %v", topics)
	}
	if cfg.Publisher.Type != "stdout" {
		t.Errorf("Expected publisher type 'stdout', got '%s'", cfg.Publisher.Type)
	}
}

func TestLoadConfigMultipleTopics(t *testing.T) {
	tmpConfig := `
topics: ["quantum computing", "artificial intelligence"]
publisher:
  type: stdout
summarizer:
  type: anthropic
  api_key: test_api_key
`
	tmpfile, err := os.CreateTemp("", "config_multi_test_*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(tmpConfig)); err != nil {
		t.Fatalf("Failed to write temp config: %v", err)
	}
	tmpfile.Close()

	cfg, err := Load(tmpfile.Name())
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	topics := cfg.GetTopics()
	expected := []string{"quantum computing", "artificial intelligence"}
	if len(topics) != len(expected) {
		t.Errorf("Expected %d topics, got %d", len(expected), len(topics))
	}
	for i, topic := range topics {
		if topic != expected[i] {
			t.Errorf("Expected topic[%d] '%s', got '%s'", i, expected[i], topic)
		}
	}
	
	topicsString := cfg.GetTopicsString()
	expectedString := "quantum computing, artificial intelligence"
	if topicsString != expectedString {
		t.Errorf("Expected topics string '%s', got '%s'", expectedString, topicsString)
	}
}

func TestTopicsPrecedence(t *testing.T) {
	// When both topic and topics are specified, topics should take precedence
	tmpConfig := `
topic: single topic
topics: ["first topic", "second topic"]
publisher:
  type: stdout
summarizer:
  type: anthropic
  api_key: test_api_key
`
	tmpfile, err := os.CreateTemp("", "config_precedence_test_*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(tmpConfig)); err != nil {
		t.Fatalf("Failed to write temp config: %v", err)
	}
	tmpfile.Close()

	cfg, err := Load(tmpfile.Name())
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	topics := cfg.GetTopics()
	expected := []string{"first topic", "second topic"}
	if len(topics) != len(expected) {
		t.Errorf("Expected %d topics, got %d", len(expected), len(topics))
	}
	for i, topic := range topics {
		if topic != expected[i] {
			t.Errorf("Expected topic[%d] '%s', got '%s'", i, expected[i], topic)
		}
	}
}

func TestDefaults(t *testing.T) {
	tmpConfig := `
topic: defaults test
summarizer:
  api_key: test_key
`
	tmpfile, err := os.CreateTemp("", "config_defaults_*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())
	if _, err := tmpfile.Write([]byte(tmpConfig)); err != nil {
		t.Fatalf("Failed to write temp config: %v", err)
	}
	tmpfile.Close()

	cfg, err := Load(tmpfile.Name())
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg.Language != "en" {
		t.Errorf("Expected default language 'en', got '%s'", cfg.Language)
	}
	if cfg.Schedule != "0 8 * * *" {
		t.Errorf("Expected default schedule '0 8 * * *', got '%s'", cfg.Schedule)
	}
	if cfg.MaxResults != 20 {
		t.Errorf("Expected default max_results 20, got %d", cfg.MaxResults)
	}
	if cfg.TopN != 5 {
		t.Errorf("Expected default top_n 5, got %d", cfg.TopN)
	}
	if cfg.Fetcher.Type != "arxiv" {
		t.Errorf("Expected default fetcher type 'arxiv', got '%s'", cfg.Fetcher.Type)
	}
	if cfg.Summarizer.Type != "anthropic" {
		t.Errorf("Expected default summarizer type 'anthropic', got '%s'", cfg.Summarizer.Type)
	}
	if cfg.Summarizer.Model != "claude-sonnet-4-20250514" {
		t.Errorf("Expected default model 'claude-sonnet-4-20250514', got '%s'", cfg.Summarizer.Model)
	}
	if cfg.Summarizer.MaxTokens != 4096 {
		t.Errorf("Expected default max_tokens 4096, got %d", cfg.Summarizer.MaxTokens)
	}
	if cfg.Publisher.Type != "stdout" {
		t.Errorf("Expected default publisher type 'stdout', got '%s'", cfg.Publisher.Type)
	}
	if cfg.Publisher.Web.Addr != ":8080" {
		t.Errorf("Expected default web addr ':8080', got '%s'", cfg.Publisher.Web.Addr)
	}
	if cfg.Publisher.Email.SMTPPort != 587 {
		t.Errorf("Expected default SMTP port 587, got %d", cfg.Publisher.Email.SMTPPort)
	}
}

func TestLanguageValidation(t *testing.T) {
	tests := []struct {
		name     string
		language string
		wantErr  bool
	}{
		{"valid english", "en", false},
		{"valid japanese", "ja", false},
		{"invalid language", "fr", true},
		{"invalid language", "zh", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpConfig := `
topic: test topic
language: ` + tt.language + `
publisher:
  type: stdout
summarizer:
  type: anthropic
  api_key: test_api_key
`
			tmpfile, err := os.CreateTemp("", "language_test_*.yaml")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(tmpfile.Name())

			if _, err := tmpfile.Write([]byte(tmpConfig)); err != nil {
				t.Fatalf("Failed to write temp config: %v", err)
			}
			tmpfile.Close()

			_, err = Load(tmpfile.Name())
			if tt.wantErr && err == nil {
				t.Errorf("Expected error for language %s, got none", tt.language)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Unexpected error for language %s: %v", tt.language, err)
			}
			if tt.wantErr && err != nil && !strings.Contains(err.Error(), "unsupported language") {
				t.Errorf("Expected 'unsupported language' error, got: %v", err)
			}
		})
	}
}

func TestConfigValidation(t *testing.T) {
	invalidConfig := `
publisher:
  type: stdout
summarizer:
  type: anthropic
  api_key: test_key
`
	tmpfile, err := os.CreateTemp("", "invalid_config_test_*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(invalidConfig)); err != nil {
		t.Fatalf("Failed to write temp config: %v", err)
	}
	tmpfile.Close()

	_, err = Load(tmpfile.Name())
	if err == nil {
		t.Fatal("Expected validation error for missing topic, got none")
	}
	if !strings.Contains(err.Error(), "at least one topic is required") {
		t.Errorf("Expected 'at least one topic is required' error, got: %v", err)
	}
}

func TestEmptyTopicsValidation(t *testing.T) {
	invalidConfig := `
topics: []
publisher:
  type: stdout
summarizer:
  type: anthropic
  api_key: test_key
`
	tmpfile, err := os.CreateTemp("", "empty_topics_test_*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(invalidConfig)); err != nil {
		t.Fatalf("Failed to write temp config: %v", err)
	}
	tmpfile.Close()

	_, err = Load(tmpfile.Name())
	if err == nil {
		t.Fatal("Expected validation error for empty topics array, got none")
	}
	if !strings.Contains(err.Error(), "at least one topic is required") {
		t.Errorf("Expected 'at least one topic is required' error, got: %v", err)
	}
}

func TestDiscordValidation(t *testing.T) {
	cfg := `
topic: test
summarizer:
  api_key: test_key
publisher:
  type: discord
`
	tmpfile, err := os.CreateTemp("", "discord_config_*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())
	if _, err := tmpfile.Write([]byte(cfg)); err != nil {
		t.Fatalf("Failed to write temp config: %v", err)
	}
	tmpfile.Close()

	_, err = Load(tmpfile.Name())
	if err == nil {
		t.Fatal("Expected validation error for missing discord webhook_url")
	}
	if !strings.Contains(err.Error(), "webhook_url is required") {
		t.Errorf("Expected webhook_url error, got: %v", err)
	}
}

func TestEmailValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  string
		wantErr string
	}{
		{
			name: "missing smtp_host",
			config: `
topic: test
summarizer:
  api_key: test_key
publisher:
  type: email
  email:
    from: sender@example.com
    to: [recipient@example.com]
`,
			wantErr: "smtp_host is required",
		},
		{
			name: "missing to",
			config: `
topic: test
summarizer:
  api_key: test_key
publisher:
  type: email
  email:
    smtp_host: smtp.example.com
    from: sender@example.com
`,
			wantErr: "to is required",
		},
		{
			name: "missing from",
			config: `
topic: test
summarizer:
  api_key: test_key
publisher:
  type: email
  email:
    smtp_host: smtp.example.com
    to: [recipient@example.com]
`,
			wantErr: "from is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpfile, err := os.CreateTemp("", "email_config_*.yaml")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(tmpfile.Name())
			if _, err := tmpfile.Write([]byte(tt.config)); err != nil {
				t.Fatalf("Failed to write temp config: %v", err)
			}
			tmpfile.Close()

			_, err = Load(tmpfile.Name())
			if err == nil {
				t.Fatalf("Expected validation error containing %q", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("Expected error containing %q, got: %v", tt.wantErr, err)
			}
		})
	}
}

func TestFileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/path/config.yaml")
	if err == nil {
		t.Fatal("Expected error for non-existent file")
	}
	if !strings.Contains(err.Error(), "failed to read") {
		t.Errorf("Expected 'failed to read' error, got: %v", err)
	}
}

func TestEnvVarExpansion(t *testing.T) {
	os.Setenv("TEST_VAR", "expanded_value")
	defer os.Unsetenv("TEST_VAR")

	input := "value: ${TEST_VAR}"
	expanded := expandEnvVars(input)
	expected := "value: expanded_value"

	if expanded != expected {
		t.Errorf("Expected '%s', got '%s'", expected, expanded)
	}
}

func TestEnvVarExpansionUnset(t *testing.T) {
	os.Unsetenv("UNSET_VAR_12345")

	input := "value: ${UNSET_VAR_12345}"
	expanded := expandEnvVars(input)

	if expanded != input {
		t.Errorf("Expected unset var to remain as-is, got '%s'", expanded)
	}
}