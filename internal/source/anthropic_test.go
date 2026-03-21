package source

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/boxiang/ai-news-digest/internal/config"
)

func TestAnthropicNewsProviderFetch(t *testing.T) {
	payload := `
<!DOCTYPE html><html><body>
<a href="/news/anthropic-partner-network">Mar 12, 2026 Announcements Anthropic invests $100 million into the Claude Partner Network</a>
<a href="/news/anthropic-institute">Mar 11, 2026 Announcements Introducing The Anthropic Institute</a>
<a href="/careers">Careers</a>
</body></html>`

	provider := NewAnthropicNewsProvider(NewHTTPFetcher(&http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"text/html"}},
				Body:       io.NopCloser(strings.NewReader(payload)),
				Request:    req,
			}, nil
		}),
	}), config.SourceConfig{
		Name:     "Anthropic",
		Type:     "official",
		Mode:     "anthropic_news",
		URL:      "https://www.anthropic.com/news",
		Enabled:  true,
		MaxItems: 10,
	})

	articles, err := provider.Fetch(context.Background())
	if err != nil {
		t.Fatalf("fetch anthropic news: %v", err)
	}
	if len(articles) != 2 {
		t.Fatalf("expected 2 articles, got %d", len(articles))
	}
	if articles[0].Title != "Anthropic invests $100 million into the Claude Partner Network" {
		t.Fatalf("unexpected first title: %s", articles[0].Title)
	}
	if articles[0].URL != "https://www.anthropic.com/news/anthropic-partner-network" {
		t.Fatalf("unexpected first url: %s", articles[0].URL)
	}
}
