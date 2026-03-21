package source

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/boxiang/ai-news-digest/internal/config"
)

func TestAnthropicReleaseNotesProviderFetch(t *testing.T) {
	payload := `
<html><body>
<div>September 2, 2025</div>
<li>We've launched v2 of the <a href="/docs/en/agents-and-tools/tool-use/code-execution-tool">Code Execution Tool</a> in public beta.</li>
<li>We've moved our <a href="https://github.com/anthropics/anthropic-sdk-go">Go SDK</a> from beta to GA.</li>
<div>August 27, 2025</div>
<li>We've launched a beta of the PHP SDK.</li>
</body></html>`

	provider := NewAnthropicReleaseNotesProvider(NewHTTPFetcher(&http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"text/html"}},
				Body:       io.NopCloser(strings.NewReader(payload)),
				Request:    req,
			}, nil
		}),
	}), config.SourceConfig{
		Name:     "Anthropic API Release Notes",
		Type:     "practical",
		Mode:     "anthropic_release_notes",
		URL:      "https://docs.anthropic.com/en/release-notes/api",
		Enabled:  true,
		MaxItems: 3,
	})

	articles, err := provider.Fetch(context.Background())
	if err != nil {
		t.Fatalf("fetch release notes: %v", err)
	}
	if len(articles) != 3 {
		t.Fatalf("expected 3 release note items, got %d", len(articles))
	}
	if !strings.Contains(articles[0].Title, "Code Execution Tool") {
		t.Fatalf("unexpected first title: %s", articles[0].Title)
	}
	if articles[0].URL != "https://docs.anthropic.com/docs/en/agents-and-tools/tool-use/code-execution-tool" &&
		articles[0].URL != "https://docs.anthropic.com/en/agents-and-tools/tool-use/code-execution-tool" {
		t.Fatalf("unexpected first url: %s", articles[0].URL)
	}
	if articles[1].URL != "https://github.com/anthropics/anthropic-sdk-go" {
		t.Fatalf("unexpected second url: %s", articles[1].URL)
	}
}
