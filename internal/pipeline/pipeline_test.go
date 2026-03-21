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

func TestPrepareCandidatesPrioritizesEngineerRelevantNews(t *testing.T) {
	now := time.Date(2026, 3, 21, 1, 0, 0, 0, time.UTC)
	articles := []model.Article{
		{
			Source:      "TechCrunch AI",
			SourceType:  "media",
			Title:       "AI startup raises new funding round",
			URL:         "https://example.com/funding",
			PublishedAt: now.Add(-2 * time.Hour),
			SummaryRaw:  "Funding announcement for expansion.",
		},
		{
			Source:      "OpenAI",
			SourceType:  "official",
			Title:       "OpenAI launches new API and SDK for agent workflows",
			URL:         "https://example.com/api-sdk",
			PublishedAt: now.Add(-3 * time.Hour),
			SummaryRaw:  "New API and SDK improve developer integration.",
		},
	}

	candidates := PrepareCandidates(articles, now)
	if len(candidates) != 2 {
		t.Fatalf("expected 2 candidates, got %d", len(candidates))
	}
	if candidates[0].Article.URL != "https://example.com/api-sdk" {
		t.Fatalf("expected engineering-relevant news first, got %s", candidates[0].Article.URL)
	}
}

func TestSelectFallbackCandidatesPrefersTwoPracticalItems(t *testing.T) {
	candidates := []Candidate{
		{Article: model.Article{Title: "Company raises major funding", SummaryRaw: "Expansion plans", Source: "TechCrunch AI", URL: "https://example.com/funding"}},
		{Article: model.Article{Title: "New SDK for agent workflows", SummaryRaw: "Developer tooling update", Source: "OpenAI", URL: "https://example.com/sdk"}, Score: 10},
		{Article: model.Article{Title: "CLI adds background tasks", SummaryRaw: "Useful for engineers", Source: "Anthropic", URL: "https://example.com/cli"}, Score: 9},
	}

	selected := selectFallbackCandidates(candidates, 3)
	if len(selected) != 3 {
		t.Fatalf("expected 3 selected candidates, got %d", len(selected))
	}
	if !isEngineerRelevant(selected[0].Article) || !isEngineerRelevant(selected[1].Article) {
		t.Fatalf("expected first two selections to be engineer-relevant")
	}
}

func TestSelectFallbackCandidatesFillsWithPlatformThenNews(t *testing.T) {
	candidates := []Candidate{
		{Article: model.Article{Title: "New SDK for agent workflows", SummaryRaw: "Developer tooling update", Source: "OpenAI", SourceType: "practical", URL: "https://example.com/sdk"}},
		{Article: model.Article{Title: "GPT model pricing update", SummaryRaw: "API pricing changes", Source: "OpenAI News", SourceType: "official", URL: "https://example.com/pricing"}},
		{Article: model.Article{Title: "AI company raises major funding", SummaryRaw: "Industry move", Source: "TechCrunch AI", SourceType: "media", URL: "https://example.com/funding"}},
	}

	selected := selectFallbackCandidates(candidates, 3)
	if len(selected) != 3 {
		t.Fatalf("expected 3 selected candidates, got %d", len(selected))
	}
	if classifyTier(selected[0].Article) != tierPractical {
		t.Fatalf("expected first item to be practical")
	}
	if classifyTier(selected[1].Article) != tierPlatform {
		t.Fatalf("expected second item to be platform")
	}
	if classifyTier(selected[2].Article) != tierNews {
		t.Fatalf("expected third item to be news")
	}
}
