package publisher

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ryosukesatoh/daily-feed/internal/fetcher"
	"github.com/ryosukesatoh/daily-feed/internal/summarizer"
)

func sampleDigest() *summarizer.Digest {
	return &summarizer.Digest{
		Topic: "machine learning",
		Date:  time.Date(2025, 1, 15, 8, 0, 0, 0, time.UTC),
		Summaries: []summarizer.PaperSummary{
			{
				Paper: fetcher.Paper{
					Title:    "Test Paper One",
					Authors:  []string{"Alice", "Bob"},
					URL:      "http://example.com/1",
					Category: "cs.AI",
				},
				Summary:   "This is a summary of paper one.",
				KeyPoints: []string{"Point A", "Point B", "Point C"},
			},
			{
				Paper: fetcher.Paper{
					Title:    "Test Paper Two",
					Authors:  []string{"Charlie"},
					URL:      "http://example.com/2",
					Category: "cs.LG",
				},
				Summary:   "This is a summary of paper two.",
				KeyPoints: []string{"Point D"},
			},
		},
		Overview: "Overview of today's papers on machine learning.",
	}
}

func TestStdoutPublish(t *testing.T) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	pub := NewStdoutPublisher()
	err := pub.Publish(context.Background(), sampleDigest())

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("Publish returned error: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	for _, want := range []string{
		"machine learning",
		"Test Paper One",
		"Test Paper Two",
		"Alice, Bob",
		"Overview of today's papers",
		"Point A",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("Expected output to contain %q", want)
		}
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		name  string
		input string
		max   int
		check func(string) bool
		desc  string
	}{
		{
			name:  "short string unchanged",
			input: "hello",
			max:   10,
			check: func(s string) bool { return s == "hello" },
			desc:  "expected 'hello'",
		},
		{
			name:  "exact length unchanged",
			input: "hello",
			max:   5,
			check: func(s string) bool { return s == "hello" },
			desc:  "expected 'hello'",
		},
		{
			name:  "long string truncated with ellipsis",
			input: "This is a very long string that should be truncated.",
			max:   20,
			check: func(s string) bool { return len(s) < 52 && strings.HasSuffix(s, "\u2026") },
			desc:  "expected truncated string ending with ellipsis",
		},
		{
			name:  "truncation prefers sentence boundary",
			input: "A long enough first sentence. The rest is extra padding text here.",
			max:   40,
			check: func(s string) bool { return s == "A long enough first sentence." },
			desc:  "expected truncation at sentence boundary",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncate(tt.input, tt.max)
			if !tt.check(result) {
				t.Errorf("%s, got %q", tt.desc, result)
			}
		})
	}
}

func TestFormatKeyPoints(t *testing.T) {
	kps := []string{"First point", "Second point", "Third point"}
	result := formatKeyPoints(kps)

	if !strings.Contains(result, "\u2022 First point") {
		t.Error("Expected bullet point for 'First point'")
	}
	if !strings.Contains(result, "\u2022 Second point") {
		t.Error("Expected bullet point for 'Second point'")
	}
	lines := strings.Split(result, "\n")
	if len(lines) != 3 {
		t.Errorf("Expected 3 lines, got %d", len(lines))
	}
}

func TestFormatKeyPointsEmpty(t *testing.T) {
	result := formatKeyPoints(nil)
	if result != "" {
		t.Errorf("Expected empty string for nil key points, got %q", result)
	}
}

func TestEmbedCharCount(t *testing.T) {
	e := discordEmbed{
		Title:       "Title",       // 5
		Description: "Description", // 11
		Fields: []discordEmbedField{
			{Name: "Field", Value: "Value"}, // 5 + 5 = 10
		},
		Footer: &discordEmbedFooter{Text: "Footer"}, // 6
	}

	count := embedCharCount(e)
	expected := 5 + 11 + 5 + 5 + 6
	if count != expected {
		t.Errorf("Expected char count %d, got %d", expected, count)
	}
}

func TestEmbedCharCountNoFooter(t *testing.T) {
	e := discordEmbed{
		Title:       "Title",
		Description: "Desc",
	}

	count := embedCharCount(e)
	if count != 9 {
		t.Errorf("Expected char count 9, got %d", count)
	}
}

func TestBatchEmbedsUnder10(t *testing.T) {
	embeds := make([]discordEmbed, 5)
	for i := range embeds {
		embeds[i] = discordEmbed{Title: "T"}
	}

	batches := batchEmbeds(embeds)
	if len(batches) != 1 {
		t.Errorf("Expected 1 batch for 5 embeds, got %d", len(batches))
	}
	if len(batches[0]) != 5 {
		t.Errorf("Expected 5 embeds in batch, got %d", len(batches[0]))
	}
}

func TestBatchEmbedsOver10(t *testing.T) {
	embeds := make([]discordEmbed, 12)
	for i := range embeds {
		embeds[i] = discordEmbed{Title: "T"}
	}

	batches := batchEmbeds(embeds)
	if len(batches) != 2 {
		t.Errorf("Expected 2 batches for 12 embeds, got %d", len(batches))
	}
	if len(batches[0]) != 10 {
		t.Errorf("Expected 10 embeds in first batch, got %d", len(batches[0]))
	}
	if len(batches[1]) != 2 {
		t.Errorf("Expected 2 embeds in second batch, got %d", len(batches[1]))
	}
}

func TestBatchEmbedsCharLimit(t *testing.T) {
	// Each embed has 2000 chars. 3 embeds = 6000 chars, so the 4th should start a new batch.
	embeds := make([]discordEmbed, 4)
	for i := range embeds {
		embeds[i] = discordEmbed{Description: strings.Repeat("x", 2000)}
	}

	batches := batchEmbeds(embeds)
	if len(batches) != 2 {
		t.Errorf("Expected 2 batches due to char limit, got %d", len(batches))
	}
	if len(batches[0]) != 3 {
		t.Errorf("Expected 3 embeds in first batch, got %d", len(batches[0]))
	}
	if len(batches[1]) != 1 {
		t.Errorf("Expected 1 embed in second batch, got %d", len(batches[1]))
	}
}

func TestDiscordPublishWithMockWebhook(t *testing.T) {
	var receivedPayloads []discordWebhookPayload

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %q", r.Header.Get("Content-Type"))
		}

		body, _ := io.ReadAll(r.Body)
		var payload discordWebhookPayload
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Errorf("Failed to parse webhook payload: %v", err)
		}
		receivedPayloads = append(receivedPayloads, payload)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()

	pub := &DiscordPublisher{
		webhookURL: ts.URL,
		client:     ts.Client(),
	}

	err := pub.Publish(context.Background(), sampleDigest())
	if err != nil {
		t.Fatalf("Publish returned error: %v", err)
	}

	if len(receivedPayloads) == 0 {
		t.Fatal("No webhook payloads received")
	}

	// With 2 papers + 1 overview = 3 embeds, should be 1 batch
	total := 0
	for _, p := range receivedPayloads {
		total += len(p.Embeds)
	}
	if total != 3 {
		t.Errorf("Expected 3 total embeds (1 overview + 2 papers), got %d", total)
	}

	// Check overview embed
	overview := receivedPayloads[0].Embeds[0]
	if !strings.Contains(overview.Title, "machine learning") {
		t.Errorf("Expected overview title to contain topic, got %q", overview.Title)
	}
}

func TestDiscordPublishWebhookError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer ts.Close()

	pub := &DiscordPublisher{
		webhookURL: ts.URL,
		client:     ts.Client(),
	}

	err := pub.Publish(context.Background(), sampleDigest())
	if err == nil {
		t.Fatal("Expected error for webhook failure")
	}
	if !strings.Contains(err.Error(), "unexpected status 400") {
		t.Errorf("Expected 'unexpected status 400' error, got: %v", err)
	}
}
