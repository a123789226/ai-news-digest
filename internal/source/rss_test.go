package source

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/boxiang/ai-news-digest/internal/config"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func TestRSSProviderFetch(t *testing.T) {
	payload, err := os.ReadFile(filepath.Join("testdata", "sample_rss.xml"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	fetcher := NewHTTPFetcher(&http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/rss+xml"}},
				Body:       io.NopCloser(strings.NewReader(string(payload))),
				Request:    req,
			}, nil
		}),
	})

	provider := NewRSSProvider(fetcher, config.SourceConfig{
		Name:    "OpenAI",
		Type:    "official",
		Mode:    "rss",
		URL:     "https://example.com/openai-feed.xml",
		Enabled: true,
	})

	articles, err := provider.Fetch(context.Background())
	if err != nil {
		t.Fatalf("fetch rss: %v", err)
	}
	if len(articles) != 1 {
		t.Fatalf("expected 1 article, got %d", len(articles))
	}
	if articles[0].Title != "OpenAI launches new model" {
		t.Fatalf("unexpected title: %s", articles[0].Title)
	}
	if articles[0].URL != "https://example.com/openai-model" {
		t.Fatalf("unexpected url: %s", articles[0].URL)
	}
}

func TestRSSProviderIncludeKeywords(t *testing.T) {
	payload := `
<rss version="2.0">
  <channel>
    <item>
      <title>Meta launches new tools</title>
      <link>https://example.com/meta-tools</link>
      <description>General product update</description>
      <category>Product News</category>
      <pubDate>Sat, 21 Mar 2026 00:00:00 GMT</pubDate>
    </item>
    <item>
      <title>Meta expands AI safety tooling</title>
      <link>https://example.com/meta-ai</link>
      <description>New AI systems for support and safety</description>
      <category>AI</category>
      <pubDate>Sat, 21 Mar 2026 00:10:00 GMT</pubDate>
    </item>
  </channel>
</rss>`

	provider := NewRSSProvider(NewHTTPFetcher(&http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/rss+xml"}},
				Body:       io.NopCloser(strings.NewReader(payload)),
				Request:    req,
			}, nil
		}),
	}), config.SourceConfig{
		Name:            "Meta AI",
		Type:            "official",
		Mode:            "rss",
		URL:             "https://example.com/meta-feed.xml",
		Enabled:         true,
		IncludeKeywords: []string{"AI", "Llama"},
	})

	articles, err := provider.Fetch(context.Background())
	if err != nil {
		t.Fatalf("fetch rss: %v", err)
	}
	if len(articles) != 1 {
		t.Fatalf("expected 1 filtered article, got %d", len(articles))
	}
	if articles[0].Title != "Meta expands AI safety tooling" {
		t.Fatalf("unexpected title: %s", articles[0].Title)
	}
}
