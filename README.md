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
| `topic` | *(required)* | Search topic for papers |
| `language` | `en` | Summary language: `en` (English) or `ja` (Japanese) |
| `schedule` | `0 8 * * *` | Cron expression for digest schedule |
| `max_results` | `20` | Max papers to fetch from arXiv |
| `top_n` | `5` | Papers to include in the digest |
| `run_on_start` | `true` | Run a digest immediately on startup |
| `publisher.type` | `stdout` | Output method: `stdout`, `email`, or `web` |

### Language Support

The application supports generating summaries in multiple languages:

- **English** (`en`) - Default language for summaries and analysis
- **Japanese** (`ja`) - Summaries and analysis in Japanese

To generate Japanese summaries, set the language in your config:

```yaml
topic: "quantum computing"
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

### Publisher modes

- **stdout** — prints the digest to the terminal
- **email** — sends an HTML email via SMTP
- **web** — serves the latest digest at `http://localhost:8080`