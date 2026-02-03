package fetcher

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// arXiv Atom feed XML structures

type arxivFeed struct {
	XMLName xml.Name     `xml:"feed"`
	Entries []arxivEntry `xml:"entry"`
}

type arxivEntry struct {
	Title     string         `xml:"title"`
	Summary   string         `xml:"summary"`
	Authors   []arxivAuthor  `xml:"author"`
	Links     []arxivLink    `xml:"link"`
	Published string         `xml:"published"`
	Category  []arxivCategory `xml:"category"`
}

type arxivAuthor struct {
	Name string `xml:"name"`
}

type arxivLink struct {
	Href string `xml:"href,attr"`
	Type string `xml:"type,attr"`
	Rel  string `xml:"rel,attr"`
}

type arxivCategory struct {
	Term string `xml:"term,attr"`
}

// ArxivFetcher fetches papers from the arXiv API.
type ArxivFetcher struct {
	client  *http.Client
	baseURL string
}

func NewArxivFetcher() *ArxivFetcher {
	return &ArxivFetcher{
		client:  &http.Client{Timeout: 30 * time.Second},
		baseURL: "http://export.arxiv.org/api/query",
	}
}

func (f *ArxivFetcher) Fetch(ctx context.Context, topic string, maxResults int) ([]Paper, error) {
	query := url.Values{}
	query.Set("search_query", fmt.Sprintf("all:%s", topic))
	query.Set("start", "0")
	query.Set("max_results", fmt.Sprintf("%d", maxResults))
	query.Set("sortBy", "submittedDate")
	query.Set("sortOrder", "descending")

	reqURL := fmt.Sprintf("%s?%s", f.baseURL, query.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("arxiv: failed to create request: %w", err)
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("arxiv: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("arxiv: unexpected status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("arxiv: failed to read response: %w", err)
	}

	var feed arxivFeed
	if err := xml.Unmarshal(body, &feed); err != nil {
		return nil, fmt.Errorf("arxiv: failed to parse XML: %w", err)
	}

	papers := make([]Paper, 0, len(feed.Entries))
	for _, entry := range feed.Entries {
		published, _ := time.Parse(time.RFC3339, entry.Published)

		authors := make([]string, len(entry.Authors))
		for i, a := range entry.Authors {
			authors[i] = strings.TrimSpace(a.Name)
		}

		var paperURL string
		for _, link := range entry.Links {
			if link.Rel == "alternate" || (link.Type == "text/html" && paperURL == "") {
				paperURL = link.Href
			}
		}
		if paperURL == "" && len(entry.Links) > 0 {
			paperURL = entry.Links[0].Href
		}

		var category string
		if len(entry.Category) > 0 {
			category = entry.Category[0].Term
		}

		papers = append(papers, Paper{
			Title:     strings.TrimSpace(entry.Title),
			Authors:   authors,
			Abstract:  strings.TrimSpace(entry.Summary),
			URL:       paperURL,
			Published: published,
			Category:  category,
		})
	}

	return papers, nil
}
