package publisher

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/ryosukesatoh/daily-feed/internal/retry"
	"github.com/ryosukesatoh/daily-feed/internal/summarizer"
)

type discordEmbedFooter struct {
	Text string `json:"text"`
}

type discordEmbedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline,omitempty"`
}

type discordEmbed struct {
	Title       string              `json:"title,omitempty"`
	URL         string              `json:"url,omitempty"`
	Description string              `json:"description,omitempty"`
	Color       int                 `json:"color,omitempty"`
	Fields      []discordEmbedField `json:"fields,omitempty"`
	Footer      *discordEmbedFooter `json:"footer,omitempty"`
	Timestamp   string              `json:"timestamp,omitempty"`
}

type discordWebhookPayload struct {
	Embeds []discordEmbed `json:"embeds"`
}

// DiscordPublisher publishes digests to a Discord channel via webhook.
type DiscordPublisher struct {
	webhookURL  string
	client      *http.Client
	retryConfig retry.Config
}

// NewDiscordPublisher creates a new DiscordPublisher.
func NewDiscordPublisher(webhookURL string) *DiscordPublisher {
	return &DiscordPublisher{
		webhookURL: webhookURL,
		client:     &http.Client{Timeout: 30 * time.Second},
		retryConfig: retry.Config{
			MaxRetries: 3,
			BaseDelay:  1 * time.Second,
		},
	}
}

// Publish sends the digest to Discord as a series of rich embeds.
func (d *DiscordPublisher) Publish(ctx context.Context, digest *summarizer.Digest) error {
	embeds := d.buildEmbeds(digest)
	batches := batchEmbeds(embeds)

	for i, batch := range batches {
		err := retry.WithBackoff(ctx, d.retryConfig, func(ctx context.Context) error {
			return d.sendWebhook(ctx, batch)
		})
		
		if err != nil {
			return fmt.Errorf("discord: failed to send batch %d: %w", i+1, err)
		}
		
		// Delay between batches to avoid rate limits.
		if i < len(batches)-1 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(500 * time.Millisecond):
			}
		}
	}
	return nil
}

// buildEmbeds creates the overview embed and one embed per paper.
func (d *DiscordPublisher) buildEmbeds(digest *summarizer.Digest) []discordEmbed {
	embeds := make([]discordEmbed, 0, len(digest.Summaries)+1)

	// Overview embed
	overview := discordEmbed{
		Title:       fmt.Sprintf("Daily Feed: %s", digest.GetTopicsString()),
		Description: truncate(digest.Overview, 4096),
		Color:       0x5865F2, // Discord blurple
		Footer:      &discordEmbedFooter{Text: digest.Date.Format("2006-01-02")},
		Timestamp:   digest.Date.Format(time.RFC3339),
	}
	embeds = append(embeds, overview)

	// Per-paper embeds
	for i, ps := range digest.Summaries {
		e := discordEmbed{
			Title:       truncate(fmt.Sprintf("%d. %s", i+1, ps.Paper.Title), 256),
			URL:         ps.Paper.URL,
			Description: truncate(ps.Summary, 4096),
			Color:       0x5865F2,
		}

		if len(ps.KeyPoints) > 0 {
			e.Fields = []discordEmbedField{
				{
					Name:  "Key Points",
					Value: truncate(formatKeyPoints(ps.KeyPoints), 1024),
				},
			}
		}

		// Footer with authors and category
		var footerParts []string
		if len(ps.Paper.Authors) > 0 {
			footerParts = append(footerParts, strings.Join(ps.Paper.Authors, ", "))
		}
		if ps.Paper.Category != "" {
			footerParts = append(footerParts, ps.Paper.Category)
		}
		if len(footerParts) > 0 {
			e.Footer = &discordEmbedFooter{Text: truncate(strings.Join(footerParts, " | "), 2048)}
		}

		embeds = append(embeds, e)
	}

	return embeds
}

// batchEmbeds splits embeds into batches respecting Discord limits:
// max 10 embeds per message, max 6000 total characters per message.
func batchEmbeds(embeds []discordEmbed) [][]discordEmbed {
	var batches [][]discordEmbed
	var current []discordEmbed
	currentChars := 0

	for _, e := range embeds {
		ec := embedCharCount(e)

		if len(current) > 0 && (len(current) >= 10 || currentChars+ec > 6000) {
			batches = append(batches, current)
			current = nil
			currentChars = 0
		}

		current = append(current, e)
		currentChars += ec
	}

	if len(current) > 0 {
		batches = append(batches, current)
	}

	return batches
}

// sendWebhook posts a batch of embeds to the Discord webhook.
func (d *DiscordPublisher) sendWebhook(ctx context.Context, embeds []discordEmbed) error {
	payload := discordWebhookPayload{Embeds: embeds}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, d.webhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	// Use HTTP status codes to determine if error is retryable
	if !retry.HTTPStatusRetryable(resp.StatusCode) && (resp.StatusCode < 200 || resp.StatusCode >= 300) {
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	return nil
}

// truncate shortens s to max characters, preferring a sentence boundary.
func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}

	cut := s[:max-1]
	// Try to cut at a sentence boundary.
	if idx := strings.LastIndexAny(cut, ".!?"); idx > max/2 {
		return cut[:idx+1]
	}
	return cut + "\u2026"
}

// formatKeyPoints formats key points as a bulleted list.
func formatKeyPoints(kps []string) string {
	var b strings.Builder
	for i, kp := range kps {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString("\u2022 ")
		b.WriteString(kp)
	}
	return b.String()
}

// embedCharCount returns the total character count of an embed for batching purposes.
func embedCharCount(e discordEmbed) int {
	n := len(e.Title) + len(e.Description)
	for _, f := range e.Fields {
		n += len(f.Name) + len(f.Value)
	}
	if e.Footer != nil {
		n += len(e.Footer.Text)
	}
	return n
}