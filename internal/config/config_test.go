package config

import (
	"os"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Temporary config file for testing
	tmpConfig := `
topic: test topic
publisher:
  type: stdout
  email:
    to: ["test@example.com"]
    from: "sender@example.com"
    smtp_host: "smtp.example.com"
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

	// Test loading valid config
	cfg, err := Load(tmpfile.Name())
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify basic config values
	if cfg.Topic != "test topic" {
		t.Errorf("Expected topic 'test topic', got '%s'", cfg.Topic)
	}
	if cfg.Publisher.Type != "stdout" {
		t.Errorf("Expected publisher type 'stdout', got '%s'", cfg.Publisher.Type)
	}

	// Verify defaults are set
	if cfg.Schedule != "0 8 * * *" {
		t.Errorf("Expected default schedule '0 8 * * *', got '%s'", cfg.Schedule)
	}
	if cfg.MaxResults != 20 {
		t.Errorf("Expected default max_results 20, got %d", cfg.MaxResults)
	}
}

func TestConfigValidation(t *testing.T) {
	// Test config without topic should fail
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