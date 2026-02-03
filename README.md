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
| `schedule` | `0 8 * * *` | Cron expression for digest schedule |
| `max_results` | `20` | Max papers to fetch from arXiv |
| `top_n` | `5` | Papers to include in the digest |
| `run_on_start` | `true` | Run a digest immediately on startup |
| `publisher.type` | `stdout` | Output method: `stdout`, `email`, or `web` |

## Usage

```sh
export ANTHROPIC_API_KEY=sk-ant-...
./daily-feed -config config.yaml
```

### Publisher modes

- **stdout** — prints the digest to the terminal
- **email** — sends an HTML email via SMTP
- **web** — serves the latest digest at `http://localhost:8080`
