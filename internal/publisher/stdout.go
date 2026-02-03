package publisher

import (
	"context"
	"fmt"
	"strings"

	"github.com/ryosukesatoh/daily-feed/internal/summarizer"
)

// StdoutPublisher prints the digest to stdout.
type StdoutPublisher struct{}

func NewStdoutPublisher() *StdoutPublisher {
	return &StdoutPublisher{}
}

func (p *StdoutPublisher) Publish(_ context.Context, digest *summarizer.Digest) error {
	fmt.Println(strings.Repeat("=", 72))
	fmt.Printf("Daily Feed Digest: %s\n", digest.Topic)
	fmt.Printf("Date: %s\n", digest.Date.Format("2006-01-02 15:04"))
	fmt.Println(strings.Repeat("=", 72))
	fmt.Println()

	fmt.Println("Overview:")
	fmt.Println(digest.Overview)
	fmt.Println()

	for i, s := range digest.Summaries {
		fmt.Println(strings.Repeat("-", 72))
		fmt.Printf("%d. %s\n", i+1, s.Paper.Title)
		fmt.Printf("   Authors: %s\n", strings.Join(s.Paper.Authors, ", "))
		fmt.Printf("   URL: %s\n", s.Paper.URL)
		fmt.Printf("   Category: %s\n", s.Paper.Category)
		fmt.Println()
		fmt.Printf("   %s\n", s.Summary)
		fmt.Println()
		if len(s.KeyPoints) > 0 {
			fmt.Println("   Key Points:")
			for _, kp := range s.KeyPoints {
				fmt.Printf("   - %s\n", kp)
			}
		}
		fmt.Println()
	}

	fmt.Println(strings.Repeat("=", 72))
	return nil
}
