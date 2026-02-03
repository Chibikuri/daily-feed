package publisher

import (
	"context"
	"fmt"
	"net/smtp"
	"strings"

	"github.com/ryosukesatoh/daily-feed/internal/summarizer"
)

// EmailPublisher sends the digest as an HTML email via SMTP.
type EmailPublisher struct {
	host     string
	port     int
	username string
	password string
	from     string
	to       []string
}

func NewEmailPublisher(host string, port int, username, password, from string, to []string) *EmailPublisher {
	return &EmailPublisher{
		host:     host,
		port:     port,
		username: username,
		password: password,
		from:     from,
		to:       to,
	}
}

func (p *EmailPublisher) Publish(_ context.Context, digest *summarizer.Digest) error {
	subject := fmt.Sprintf("Daily Feed: %s - %s", digest.Topic, digest.Date.Format("2006-01-02"))
	body := buildHTMLBody(digest)

	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=\"UTF-8\"\r\n\r\n%s",
		p.from,
		strings.Join(p.to, ","),
		subject,
		body,
	)

	addr := fmt.Sprintf("%s:%d", p.host, p.port)
	auth := smtp.PlainAuth("", p.username, p.password, p.host)

	if err := smtp.SendMail(addr, auth, p.from, p.to, []byte(msg)); err != nil {
		return fmt.Errorf("email: failed to send: %w", err)
	}

	return nil
}

func buildHTMLBody(digest *summarizer.Digest) string {
	var sb strings.Builder

	sb.WriteString(`<!DOCTYPE html><html><head><style>
body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; max-width: 700px; margin: 0 auto; padding: 20px; color: #333; }
h1 { color: #1a1a2e; border-bottom: 2px solid #e94560; padding-bottom: 10px; }
h2 { color: #16213e; }
.overview { background: #f0f0f0; padding: 15px; border-radius: 8px; margin-bottom: 20px; }
.paper { border: 1px solid #ddd; border-radius: 8px; padding: 15px; margin-bottom: 15px; }
.paper h3 { margin-top: 0; color: #0f3460; }
.meta { color: #666; font-size: 0.9em; margin-bottom: 10px; }
.key-points { margin-top: 10px; }
.key-points li { margin-bottom: 5px; }
</style></head><body>`)

	sb.WriteString(fmt.Sprintf("<h1>Daily Feed: %s</h1>", digest.Topic))
	sb.WriteString(fmt.Sprintf("<p><em>%s</em></p>", digest.Date.Format("January 2, 2006")))

	sb.WriteString(fmt.Sprintf(`<div class="overview"><h2>Overview</h2><p>%s</p></div>`, digest.Overview))

	for i, s := range digest.Summaries {
		sb.WriteString(`<div class="paper">`)
		sb.WriteString(fmt.Sprintf(`<h3>%d. <a href="%s">%s</a></h3>`, i+1, s.Paper.URL, s.Paper.Title))
		sb.WriteString(fmt.Sprintf(`<div class="meta">%s | %s</div>`, strings.Join(s.Paper.Authors, ", "), s.Paper.Category))
		sb.WriteString(fmt.Sprintf("<p>%s</p>", s.Summary))

		if len(s.KeyPoints) > 0 {
			sb.WriteString(`<div class="key-points"><strong>Key Points:</strong><ul>`)
			for _, kp := range s.KeyPoints {
				sb.WriteString(fmt.Sprintf("<li>%s</li>", kp))
			}
			sb.WriteString("</ul></div>")
		}
		sb.WriteString("</div>")
	}

	sb.WriteString("</body></html>")
	return sb.String()
}
