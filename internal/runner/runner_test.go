package runner

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ryosukesatoh/daily-feed/internal/fetcher"
	"github.com/ryosukesatoh/daily-feed/internal/publisher"
	"github.com/ryosukesatoh/daily-feed/internal/summarizer"
)

// Mock implementations

type mockFetcher struct {
	papers []fetcher.Paper
	err    error
}

func (m *mockFetcher) Fetch(ctx context.Context, topic string, maxResults int) ([]fetcher.Paper, error) {
	return m.papers, m.err
}

type mockSummarizer struct {
	digest *summarizer.Digest
	err    error
}

func (m *mockSummarizer) Summarize(ctx context.Context, papers []fetcher.Paper) (*summarizer.Digest, error) {
	return m.digest, m.err
}

type mockPublisher struct {
	published bool
	err       error
}

func (m *mockPublisher) Publish(ctx context.Context, digest *summarizer.Digest) error {
	m.published = true
	return m.err
}

func samplePapers() []fetcher.Paper {
	return []fetcher.Paper{
		{
			Title:    "Test Paper",
			Authors:  []string{"Author"},
			Abstract: "Abstract text.",
			URL:      "http://example.com/1",
			Category: "cs.AI",
		},
	}
}

func sampleDigest() *summarizer.Digest {
	return &summarizer.Digest{
		Topic:    "test topic",
		Date:     time.Now(),
		Overview: "Test overview.",
		Summaries: []summarizer.PaperSummary{
			{
				Paper:     samplePapers()[0],
				Summary:   "Paper summary.",
				KeyPoints: []string{"key point"},
			},
		},
	}
}

func TestRunSuccess(t *testing.T) {
	pub := &mockPublisher{}
	r := New(
		"test topic",
		10,
		&mockFetcher{papers: samplePapers()},
		&mockSummarizer{digest: sampleDigest()},
		[]publisher.Publisher{pub},
	)

	err := r.Run(context.Background())
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !pub.published {
		t.Error("Expected publisher to be called")
	}
}

func TestRunFetchError(t *testing.T) {
	r := New(
		"test topic",
		10,
		&mockFetcher{err: errors.New("fetch failed")},
		&mockSummarizer{digest: sampleDigest()},
		nil,
	)

	err := r.Run(context.Background())
	if err == nil {
		t.Fatal("Expected error from fetch failure")
	}
}

func TestRunSummarizeError(t *testing.T) {
	r := New(
		"test topic",
		10,
		&mockFetcher{papers: samplePapers()},
		&mockSummarizer{err: errors.New("summarize failed")},
		nil,
	)

	err := r.Run(context.Background())
	if err == nil {
		t.Fatal("Expected error from summarize failure")
	}
}

func TestRunPublishFailureDoesNotFail(t *testing.T) {
	failPub := &mockPublisher{err: errors.New("publish failed")}
	successPub := &mockPublisher{}

	r := New(
		"test topic",
		10,
		&mockFetcher{papers: samplePapers()},
		&mockSummarizer{digest: sampleDigest()},
		[]publisher.Publisher{failPub, successPub},
	)

	err := r.Run(context.Background())
	if err != nil {
		t.Fatalf("Run should not fail when publisher fails, got: %v", err)
	}
	if !failPub.published {
		t.Error("Expected failing publisher to be called")
	}
	if !successPub.published {
		t.Error("Expected second publisher to be called even after first fails")
	}
}
