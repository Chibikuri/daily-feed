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
	topic     string   // Legacy single topic for backward compatibility
	topics    []string // Multiple topics
	language  string
	client    *http.Client
}

func NewAnthropicSummarizer(apiKey, model string, maxTokens, topN int, topic, language string) *AnthropicSummarizer {
	return &AnthropicSummarizer{
		apiKey:    apiKey,
		model:     model,
		maxTokens: maxTokens,
		topN:      topN,
		topic:     topic,
		topics:    []string{topic}, // Initialize topics with single topic for backward compatibility
		language:  language,
		client:    &http.Client{Timeout: 120 * time.Second},
	}
}

func NewAnthropicSummarizerMultiTopic(apiKey, model string, maxTokens, topN int, topics []string, language string) *AnthropicSummarizer {
	// For backward compatibility, set the first topic as the legacy topic
	var topic string
	if len(topics) > 0 {
		topic = topics[0]
	}
	
	return &AnthropicSummarizer{
		apiKey:    apiKey,
		model:     model,
		maxTokens: maxTokens,
		topN:      topN,
		topic:     topic,
		topics:    topics,
		language:  language,
		client:    &http.Client{Timeout: 120 * time.Second},
	}
}

// GetTopics returns the topics, prioritizing the new topics field over the legacy topic field.
func (s *AnthropicSummarizer) GetTopics() []string {
	if len(s.topics) > 0 {
		return s.topics
	}
	if s.topic != "" {
		return []string{s.topic}
	}
	return []string{}
}

// GetTopicsString returns a comma-separated string of all topics for display purposes.
func (s *AnthropicSummarizer) GetTopicsString() string {
	return strings.Join(s.GetTopics(), ", ")
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

// retryWithBackoff executes a function with exponential backoff retry logic
func (s *AnthropicSummarizer) retryWithBackoff(ctx context.Context, operation func(context.Context) error) error {
	maxRetries := 3
	baseDelay := 2 * time.Second
	
	for attempt := 0; attempt <= maxRetries; attempt++ {
		err := operation(ctx)
		if err == nil {
			return nil
		}
		
		// Don't retry on the last attempt
		if attempt == maxRetries {
			return fmt.Errorf("anthropic: operation failed after %d attempts: %w", maxRetries+1, err)
		}
		
		// Calculate exponential backoff delay: 2s, 4s, 8s
		delay := baseDelay * time.Duration(1<<attempt)
		
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
			// Continue to next attempt
		}
	}
	
	return nil // Should never reach here
}

func (s *AnthropicSummarizer) Summarize(ctx context.Context, papers []fetcher.Paper) (*Digest, error) {
	topics := s.GetTopics()
	topicsString := s.GetTopicsString()
	
	if len(papers) == 0 {
		noResultsText := fmt.Sprintf("No papers found for the given topic(s): %s.", topicsString)
		if s.language == "ja" {
			noResultsText = fmt.Sprintf("指定されたトピック「%s」に関する論文は見つかりませんでした。", topicsString)
		}
		return &Digest{
			Topic:    s.topic, // For backward compatibility
			Topics:   topics,
			Date:     time.Now(),
			Overview: noResultsText,
		}, nil
	}

	prompt := s.buildPrompt(papers)

	var body string
	err := s.retryWithBackoff(ctx, func(ctx context.Context) error {
		var err error
		body, err = s.callAPI(ctx, prompt)
		return err
	})
	
	if err != nil {
		return nil, err
	}

	return s.parseResponse(body, papers, topics)
}

func (s *AnthropicSummarizer) buildPrompt(papers []fetcher.Paper) string {
	var sb strings.Builder
	topics := s.GetTopics()
	topicsString := s.GetTopicsString()

	if s.language == "ja" {
		if len(topics) > 1 {
			sb.WriteString(fmt.Sprintf("あなたは専門的な研究アナリストです。「%s」に関する%d件の最近の論文があります。\n\n", topicsString, len(papers)))
		} else {
			sb.WriteString(fmt.Sprintf("あなたは専門的な研究アナリストです。「%s」に関する%d件の最近の論文があります。\n\n", topicsString, len(papers)))
		}
	} else {
		if len(topics) > 1 {
			sb.WriteString(fmt.Sprintf("You are an expert research analyst. I have %d recent papers about \"%s\".\n\n", len(papers), topicsString))
		} else {
			sb.WriteString(fmt.Sprintf("You are an expert research analyst. I have %d recent papers about \"%s\".\n\n", len(papers), topicsString))
		}
	}

	for i, p := range papers {
		sb.WriteString(fmt.Sprintf("--- Paper %d ---\n", i+1))
		if s.language == "ja" {
			sb.WriteString(fmt.Sprintf("タイトル: %s\n", p.Title))
			sb.WriteString(fmt.Sprintf("著者: %s\n", strings.Join(p.Authors, ", ")))
			sb.WriteString(fmt.Sprintf("カテゴリ: %s\n", p.Category))
			sb.WriteString(fmt.Sprintf("要旨: %s\n\n", p.Abstract))
		} else {
			sb.WriteString(fmt.Sprintf("Title: %s\n", p.Title))
			sb.WriteString(fmt.Sprintf("Authors: %s\n", strings.Join(p.Authors, ", ")))
			sb.WriteString(fmt.Sprintf("Category: %s\n", p.Category))
			sb.WriteString(fmt.Sprintf("Abstract: %s\n\n", p.Abstract))
		}
	}

	if s.language == "ja" {
		if len(topics) > 1 {
			sb.WriteString(fmt.Sprintf(`これらの論文を分析し、以下を行ってください：
1. 「%s」における重要性と関連性でランク付けする
2. 最も重要な上位%d件の論文を選択する
3. 選択した各論文について、明確な要約と3-5つのキーポイントを提供する
4. 全体の簡潔な概要を書く（複数のトピック領域にわたる主要なトレンドと発見を含む）

以下の正確な構造でJSONで応答してください：
{
  "overview": "複数のトピック領域における最も重要なトレンドと発見についての2-3文の概要",
  "summaries": [
    {
      "index": 1,
      "summary": "論文の2-3文の要約",
      "key_points": ["ポイント1", "ポイント2", "ポイント3"]
    }
  ]
}

"index"フィールドは上記リストの1ベースの論文番号である必要があります。
有効なJSONのみで応答し、マークダウンフェンスや追加のテキストは含めないでください。`, topicsString, s.topN))
		} else {
			sb.WriteString(fmt.Sprintf(`これらの論文を分析し、以下を行ってください：
1. 「%s」における重要性と関連性でランク付けする
2. 最も重要な上位%d件の論文を選択する
3. 選択した各論文について、明確な要約と3-5つのキーポイントを提供する
4. 全体の簡潔な概要を書く

以下の正確な構造でJSONで応答してください：
{
  "overview": "最も重要なトレンドと発見についての2-3文の概要",
  "summaries": [
    {
      "index": 1,
      "summary": "論文の2-3文の要約",
      "key_points": ["ポイント1", "ポイント2", "ポイント3"]
    }
  ]
}

"index"フィールドは上記リストの1ベースの論文番号である必要があります。
有効なJSONのみで応答し、マークダウンフェンスや追加のテキストは含めないでください。`, topicsString, s.topN))
		}
	} else {
		if len(topics) > 1 {
			sb.WriteString(fmt.Sprintf(`Please analyze these papers and:
1. Rank them by importance and relevance to "%s"
2. Select the top %d most important papers
3. For each selected paper, provide a clear summary and 3-5 key points
4. Write a brief overall digest overview that captures key trends and findings across multiple topic areas

Respond in JSON with this exact structure:
{
  "overview": "A 2-3 sentence overview of the most important trends and findings across multiple topics",
  "summaries": [
    {
      "index": 1,
      "summary": "2-3 sentence summary of the paper",
      "key_points": ["point 1", "point 2", "point 3"]
    }
  ]
}

The "index" field should be the 1-based paper number from the list above.
Respond ONLY with valid JSON, no markdown fences or additional text.`, topicsString, s.topN))
		} else {
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
Respond ONLY with valid JSON, no markdown fences or additional text.`, topicsString, s.topN))
		}
	}

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

func (s *AnthropicSummarizer) parseResponse(body string, papers []fetcher.Paper, topics []string) (*Digest, error) {
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
		Topic:    s.topic, // For backward compatibility
		Topics:   topics,
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