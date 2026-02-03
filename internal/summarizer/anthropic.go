package summarizer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/ryosukesatoh/daily-feed/internal/fetcher"
)

// AnthropicSummarizer uses the Anthropic Messages API to summarize papers.
type AnthropicSummarizer struct {
	apiKey    string
	model     string
	maxTokens int
	topN      int
	topic     string
	client    *http.Client
}

func NewAnthropicSummarizer(apiKey, model string, maxTokens, topN int, topic string) *AnthropicSummarizer {
	return &AnthropicSummarizer{
		apiKey:    apiKey,
		model:     model,
		maxTokens: maxTokens,
		topN:      topN,
		topic:     topic,
		client:    &http.Client{Timeout: 120 * time.Second},
	}
}

// Anthropic API request/response types

type anthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	Messages  []anthropicMessage `json:"messages"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicResponse struct {
	Content []anthropicContent `json:"content"`
	Error   *anthropicError    `json:"error,omitempty"`
}

type anthropicContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type anthropicError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// digestJSON is the expected JSON structure from the LLM.
type digestJSON struct {
	Overview  string        `json:"overview"`
	Summaries []summaryJSON `json:"summaries"`
}

type summaryJSON struct {
	Index     int      `json:"index"`
	Summary   string   `json:"summary"`
	KeyPoints []string `json:"key_points"`
}

func (s *AnthropicSummarizer) Summarize(ctx context.Context, papers []fetcher.Paper) (*Digest, error) {
	if len(papers) == 0 {
		return &Digest{
			Topic:    s.topic,
			Date:     time.Now(),
			Overview: "No papers found for the given topic.",
		}, nil
	}

	prompt := s.buildPrompt(papers)

	body, err := s.callAPI(ctx, prompt)
	if err != nil {
		return nil, err
	}

	return s.parseResponse(body, papers)
}

func (s *AnthropicSummarizer) buildPrompt(papers []fetcher.Paper) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("You are an expert research analyst. I have %d recent papers about \"%s\".\n\n", len(papers), s.topic))

	for i, p := range papers {
		sb.WriteString(fmt.Sprintf("--- Paper %d ---\n", i+1))
		sb.WriteString(fmt.Sprintf("Title: %s\n", p.Title))
		sb.WriteString(fmt.Sprintf("Authors: %s\n", strings.Join(p.Authors, ", ")))
		sb.WriteString(fmt.Sprintf("Category: %s\n", p.Category))
		sb.WriteString(fmt.Sprintf("Abstract: %s\n\n", p.Abstract))
	}

	sb.WriteString(fmt.Sprintf(`Please analyze these papers and:
1. Rank them by importance and relevance to "%s"
2. Select the top %d most important papers
3. For each selected paper, provide a clear summary and 3-5 key points
4. Write a brief overall digest overview

Respond in JSON with this exact structure:
{
  "overview": "A 2-3 sentence overview of the most important trends and findings",
  "summaries": [
    {
      "index": 1,
      "summary": "2-3 sentence summary of the paper",
      "key_points": ["point 1", "point 2", "point 3"]
    }
  ]
}

The "index" field should be the 1-based paper number from the list above.
Respond ONLY with valid JSON, no markdown fences or additional text.`, s.topic, s.topN))

	return sb.String()
}

func (s *AnthropicSummarizer) callAPI(ctx context.Context, prompt string) (string, error) {
	reqBody := anthropicRequest{
		Model:     s.model,
		MaxTokens: s.maxTokens,
		Messages: []anthropicMessage{
			{Role: "user", Content: prompt},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("anthropic: failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.anthropic.com/v1/messages", bytes.NewReader(jsonData))
	if err != nil {
		return "", fmt.Errorf("anthropic: failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", s.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("anthropic: request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("anthropic: failed to read response: %w", err)
	}

	var apiResp anthropicResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return "", fmt.Errorf("anthropic: failed to parse response: %w", err)
	}

	if apiResp.Error != nil {
		return "", fmt.Errorf("anthropic: API error: %s - %s", apiResp.Error.Type, apiResp.Error.Message)
	}

	if len(apiResp.Content) == 0 {
		return "", fmt.Errorf("anthropic: empty response")
	}

	return apiResp.Content[0].Text, nil
}

func (s *AnthropicSummarizer) parseResponse(body string, papers []fetcher.Paper) (*Digest, error) {
	// Strip markdown fences if present
	body = strings.TrimSpace(body)
	body = strings.TrimPrefix(body, "```json")
	body = strings.TrimPrefix(body, "```")
	body = strings.TrimSuffix(body, "```")
	body = strings.TrimSpace(body)

	var dj digestJSON
	if err := json.Unmarshal([]byte(body), &dj); err != nil {
		return nil, fmt.Errorf("anthropic: failed to parse LLM JSON: %w\nraw response: %s", err, body)
	}

	digest := &Digest{
		Topic:    s.topic,
		Date:     time.Now(),
		Overview: dj.Overview,
	}

	for _, sj := range dj.Summaries {
		idx := sj.Index - 1 // Convert from 1-based to 0-based
		if idx < 0 || idx >= len(papers) {
			continue
		}
		digest.Summaries = append(digest.Summaries, PaperSummary{
			Paper:     papers[idx],
			Summary:   sj.Summary,
			KeyPoints: sj.KeyPoints,
		})
	}

	return digest, nil
}
