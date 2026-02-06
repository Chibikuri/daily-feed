# Daily Feed

A Go application that fetches recent academic papers from arXiv, selects the most important ones, summarizes them using Anthropic Claude, and publishes a daily digest.

## Requirements

- Go 1.25+
- Anthropic API key

## Installation

```sh
go build -o daily-feed ./cmd/daily-feed
```

## Configuration

Copy and edit the default config:

```sh
cp config.yaml config.local.yaml
# edit config.local.yaml
```

Secrets support environment variable expansion (`${ANTHROPIC_API_KEY}`).

| Field | Default | Description |
|---|---|---|
| `topic` | *(optional)* | Search topic for papers (legacy single topic support) |
| `topics` | *(optional)* | Array of search topics for papers (new multiple topics support) |
| `language` | `en` | Summary language: `en` (English) or `ja` (Japanese) |
| `schedule` | `0 8 * * *` | Cron expression for digest schedule |
| `max_results` | `20` | Max papers to fetch from arXiv |
| `top_n` | `5` | Papers to include in the digest |
| `run_on_start` | `true` | Run a digest immediately on startup |
| `publisher.type` | `stdout` | Output method: `stdout`, `email`, or `web` |

**Note**: Either `topic` or `topics` is required. If both are specified, `topics` takes precedence. Use `topic` for single topic searches (legacy format) or `topics` for multiple topic searches.

### Multiple Topics Support

You can now specify multiple topics to get a comprehensive digest covering multiple research areas:

```yaml
# Multiple topics (recommended)
topics: ["quantum computing", "artificial intelligence", "machine learning"]

# Single topic (legacy format - still supported)
topic: "quantum computing"
```

When using multiple topics, the system will:
1. Fetch papers related to any of the specified topics
2. Rank and select the most important papers across all topics
3. Generate a summary that highlights trends and findings across multiple research areas

### Language Support

The application supports generating summaries in multiple languages:

- **English** (`en`) - Default language for summaries and analysis
- **Japanese** (`ja`) - Summaries and analysis in Japanese

To generate Japanese summaries, set the language in your config:

```yaml
topics: ["quantum computing", "artificial intelligence"]
language: "ja"  # Use Japanese for summaries
```

Or use the provided Japanese config template:

```sh
./daily-feed -config config.ja.yaml
```

## Usage

```sh
export ANTHROPIC_API_KEY=sk-ant-...
./daily-feed -config config.yaml
```

### Example Configurations

**Multiple Topics (English)**:
```sh
./daily-feed -config config.multi-topic.yaml
```

**Single Topic (Japanese)**:
```sh
./daily-feed -config config.ja.yaml
```

### Publisher modes

- **stdout** — prints the digest to the terminal
- **email** — sends an HTML email via SMTP
- **web** — serves the latest digest at `http://localhost:8080`
- **discord** — posts digest to Discord channel via webhook

## Examples

### Basic usage with multiple topics:
```yaml
topics: ["quantum computing", "artificial intelligence"]
language: "en"
publisher:
  type: "stdout"
```

### Advanced configuration with email publishing:
```yaml
topics: ["machine learning", "natural language processing", "computer vision"]
language: "en"
max_results: 30
top_n: 8
publisher:
  type: "email"
  email:
    smtp_host: "smtp.gmail.com"
    smtp_port: 587
    username: "your-email@gmail.com" 
    password: "${EMAIL_APP_PASSWORD}"
    from: "daily-feed@yourcompany.com"
    to: ["team@yourcompany.com"]
```