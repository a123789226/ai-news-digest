package pipeline

import (
	"testing"
	"time"

	"github.com/boxiang/ai-news-digest/internal/model"
)

func TestPrepareCandidatesPrefersOfficialDuplicate(t *testing.T) {
	now := time.Date(2026, 3, 21, 1, 0, 0, 0, time.UTC)
	articles := []model.Article{
		{
			Source:      "TechCrunch AI",
			SourceType:  "media",
			Title:       "OpenAI launches new reasoning model",
			URL:         "https://example.com/media",
			PublishedAt: now.Add(-2 * time.Hour),
		},
		{
			Source:      "OpenAI",
			SourceType:  "official",
			Title:       "OpenAI launches new reasoning model",
			URL:         "https://example.com/official",
			PublishedAt: now.Add(-90 * time.Minute),
		},
	}

	candidates := PrepareCandidates(articles, now)
	if len(candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(candidates))
	}
	if candidates[0].Article.Source != "OpenAI" {
		t.Fatalf("expected official source to win, got %s", candidates[0].Article.Source)
	}
}

func TestFallbackDigestItemsLimit(t *testing.T) {
	candidates := []Candidate{
		{Article: model.Article{Title: "A", Source: "OpenAI", URL: "https://a"}},
		{Article: model.Article{Title: "B", Source: "Anthropic", URL: "https://b"}},
	}

	items := FallbackDigestItems(candidates, 1)
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].TitleEN != "A" {
		t.Fatalf("unexpected title: %s", items[0].TitleEN)
	}
}

func TestPrepareCandidatesFiltersLowValueVideo(t *testing.T) {
	now := time.Date(2026, 3, 21, 1, 0, 0, 0, time.UTC)
	articles := []model.Article{
		{
			Source:      "TechCrunch AI",
			SourceType:  "media",
			Title:       "What happened at Nvidia GTC",
			URL:         "https://example.com/video/gtc-recap",
			PublishedAt: now.Add(-2 * time.Hour),
		},
	}

	candidates := PrepareCandidates(articles, now)
	if len(candidates) != 0 {
		t.Fatalf("expected video recap to be filtered out, got %d candidates", len(candidates))
	}
}

func TestFallbackDigestItemsPrefersSourceDiversity(t *testing.T) {
	candidates := []Candidate{
		{Article: model.Article{Title: "A", Source: "TechCrunch AI", URL: "https://a"}},
		{Article: model.Article{Title: "B", Source: "TechCrunch AI", URL: "https://b"}},
		{Article: model.Article{Title: "C", Source: "OpenAI", URL: "https://c"}},
	}

	items := FallbackDigestItems(candidates, 2)
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if items[0].Source == items[1].Source {
		t.Fatalf("expected source diversity, got duplicate source %s", items[0].Source)
	}
}
