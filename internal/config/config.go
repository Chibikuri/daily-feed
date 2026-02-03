package config

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Topic      string           `yaml:"topic"`
	Schedule   string           `yaml:"schedule"`
	MaxResults int              `yaml:"max_results"`
	TopN       int              `yaml:"top_n"`
	RunOnStart bool             `yaml:"run_on_start"`
	Fetcher    FetcherConfig    `yaml:"fetcher"`
	Summarizer SummarizerConfig `yaml:"summarizer"`
	Publisher  PublisherConfig  `yaml:"publisher"`
}

type FetcherConfig struct {
	Type string `yaml:"type"`
}

type SummarizerConfig struct {
	Type      string `yaml:"type"`
	Model     string `yaml:"model"`
	APIKey    string `yaml:"api_key"`
	MaxTokens int    `yaml:"max_tokens"`
}

type PublisherConfig struct {
	Type    string        `yaml:"type"`
	Email   EmailConfig   `yaml:"email"`
	Web     WebConfig     `yaml:"web"`
	Discord DiscordConfig `yaml:"discord"`
}

type DiscordConfig struct {
	WebhookURL string `yaml:"webhook_url"`
}

type EmailConfig struct {
	SMTPHost string   `yaml:"smtp_host"`
	SMTPPort int      `yaml:"smtp_port"`
	Username string   `yaml:"username"`
	Password string   `yaml:"password"`
	From     string   `yaml:"from"`
	To       []string `yaml:"to"`
}

type WebConfig struct {
	Addr string `yaml:"addr"`
}

var envVarRegex = regexp.MustCompile(`\$\{([^}]+)\}`)

// expandEnvVars replaces ${VAR_NAME} patterns with environment variable values.
func expandEnvVars(s string) string {
	return envVarRegex.ReplaceAllStringFunc(s, func(match string) string {
		varName := strings.TrimSuffix(strings.TrimPrefix(match, "${"), "}")
		if val, ok := os.LookupEnv(varName); ok {
			return val
		}
		return match
	})
}

func setDefaults(cfg *Config) {
	if cfg.Schedule == "" {
		cfg.Schedule = "0 8 * * *"
	}
	if cfg.MaxResults == 0 {
		cfg.MaxResults = 20
	}
	if cfg.TopN == 0 {
		cfg.TopN = 5
	}
	if cfg.Fetcher.Type == "" {
		cfg.Fetcher.Type = "arxiv"
	}
	if cfg.Summarizer.Type == "" {
		cfg.Summarizer.Type = "anthropic"
	}
	if cfg.Summarizer.Model == "" {
		cfg.Summarizer.Model = "claude-sonnet-4-20250514"
	}
	if cfg.Summarizer.MaxTokens == 0 {
		cfg.Summarizer.MaxTokens = 4096
	}
	if cfg.Publisher.Type == "" {
		cfg.Publisher.Type = "stdout"
	}
	if cfg.Publisher.Web.Addr == "" {
		cfg.Publisher.Web.Addr = ":8080"
	}
	if cfg.Publisher.Email.SMTPPort == 0 {
		cfg.Publisher.Email.SMTPPort = 587
	}
}

func validate(cfg *Config) error {
	if cfg.Topic == "" {
		return fmt.Errorf("config: topic is required")
	}
	if cfg.Fetcher.Type != "arxiv" {
		return fmt.Errorf("config: unsupported fetcher type %q (supported: arxiv)", cfg.Fetcher.Type)
	}
	if cfg.Summarizer.Type != "anthropic" {
		return fmt.Errorf("config: unsupported summarizer type %q (supported: anthropic)", cfg.Summarizer.Type)
	}
	if cfg.Summarizer.APIKey == "" {
		return fmt.Errorf("config: summarizer.api_key is required (set ANTHROPIC_API_KEY env var)")
	}
	switch cfg.Publisher.Type {
	case "stdout", "email", "web", "discord":
	default:
		return fmt.Errorf("config: unsupported publisher type %q (supported: stdout, email, web, discord)", cfg.Publisher.Type)
	}
	if cfg.Publisher.Type == "discord" {
		if cfg.Publisher.Discord.WebhookURL == "" {
			return fmt.Errorf("config: publisher.discord.webhook_url is required for discord publisher")
		}
	}
	if cfg.Publisher.Type == "email" {
		if cfg.Publisher.Email.SMTPHost == "" {
			return fmt.Errorf("config: publisher.email.smtp_host is required for email publisher")
		}
		if len(cfg.Publisher.Email.To) == 0 {
			return fmt.Errorf("config: publisher.email.to is required for email publisher")
		}
		if cfg.Publisher.Email.From == "" {
			return fmt.Errorf("config: publisher.email.from is required for email publisher")
		}
	}
	return nil
}

// Load reads the config file, expands environment variables, applies defaults,
// and validates the configuration.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("config: failed to read %s: %w", path, err)
	}

	expanded := expandEnvVars(string(data))

	var cfg Config
	if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
		return nil, fmt.Errorf("config: failed to parse %s: %w", path, err)
	}

	setDefaults(&cfg)

	if err := validate(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
