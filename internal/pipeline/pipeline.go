package pipeline

import (
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/boxiang/ai-news-digest/internal/model"
)

type Candidate struct {
	Article         model.Article
	Score           int
	Corroborations  int
	NormalizedTitle string
}

var keywordScores = map[string]int{
	"release":     2,
	"launch":      2,
	"model":       2,
	"api":         2,
	"funding":     2,
	"policy":      2,
	"research":    2,
	"open source": 2,
}

var penaltyKeywords = []string{"opinion", "how to", "tutorial", "guide"}
var titleCleaner = regexp.MustCompile(`[^a-z0-9\s]+`)

func PrepareCandidates(articles []model.Article, now time.Time) []Candidate {
	recent := filterRecent(articles, now)
	merged := dedupe(recent)
	candidates := make([]Candidate, 0, len(merged))
	for _, article := range merged {
		normalized := normalizeTitle(article.Title)
		candidates = append(candidates, Candidate{
			Article:         article,
			Score:           scoreArticle(article),
			NormalizedTitle: normalized,
		})
	}
	applyCorroborations(candidates)
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].Score == candidates[j].Score {
			return candidates[i].Article.PublishedAt.After(candidates[j].Article.PublishedAt)
		}
		return candidates[i].Score > candidates[j].Score
	})
	return candidates
}

func FallbackDigestItems(candidates []Candidate, limit int) []model.DigestItem {
	if limit > len(candidates) {
		limit = len(candidates)
	}
	items := make([]model.DigestItem, 0, limit)
	for _, candidate := range candidates[:limit] {
		items = append(items, model.DigestItem{
			TitleEN:        candidate.Article.Title,
			SummaryZH:      fallbackSummary(candidate.Article),
			WhyItMattersZH: "這則消息來自高分來源，值得你快速查看原文。",
			Source:         candidate.Article.Source,
			URL:            candidate.Article.URL,
		})
	}
	return items
}

func filterRecent(articles []model.Article, now time.Time) []model.Article {
	cutoff := now.Add(-24 * time.Hour)
	filtered := make([]model.Article, 0, len(articles))
	for _, article := range articles {
		if article.URL == "" || article.Title == "" || article.PublishedAt.IsZero() {
			continue
		}
		if article.PublishedAt.Before(cutoff) || article.PublishedAt.After(now.Add(5*time.Minute)) {
			continue
		}
		filtered = append(filtered, article)
	}
	return filtered
}

func dedupe(articles []model.Article) []model.Article {
	seenURLs := map[string]struct{}{}
	seenTitles := map[string]model.Article{}
	result := make([]model.Article, 0, len(articles))
	for _, article := range articles {
		if _, ok := seenURLs[article.URL]; ok {
			continue
		}
		seenURLs[article.URL] = struct{}{}
		normalized := normalizeTitle(article.Title)
		if existing, ok := seenTitles[normalized]; ok {
			if preferredArticle(article, existing) {
				for i := range result {
					if result[i].URL == existing.URL {
						result[i] = article
						break
					}
				}
				seenTitles[normalized] = article
			}
			continue
		}
		seenTitles[normalized] = article
		result = append(result, article)
	}
	return result
}

func preferredArticle(current, existing model.Article) bool {
	if current.SourceType != existing.SourceType {
		return sourceRank(current.SourceType) > sourceRank(existing.SourceType)
	}
	return current.PublishedAt.After(existing.PublishedAt)
}

func sourceRank(sourceType string) int {
	switch sourceType {
	case "official":
		return 3
	case "media":
		return 2
	case "social":
		return 1
	default:
		return 0
	}
}

func normalizeTitle(title string) string {
	title = strings.ToLower(title)
	title = titleCleaner.ReplaceAllString(title, " ")
	title = strings.Join(strings.Fields(title), " ")
	return title
}

func scoreArticle(article model.Article) int {
	score := 0
	score += sourceRank(article.SourceType) * 3
	title := normalizeTitle(article.Title)
	for keyword, value := range keywordScores {
		if strings.Contains(title, keyword) {
			score += value
		}
	}
	for _, keyword := range penaltyKeywords {
		if strings.Contains(title, keyword) {
			score -= 2
		}
	}
	return score
}

func applyCorroborations(candidates []Candidate) {
	for i := range candidates {
		for j := range candidates {
			if i == j {
				continue
			}
			if overlapScore(candidates[i].NormalizedTitle, candidates[j].NormalizedTitle) >= 0.6 {
				candidates[i].Corroborations++
			}
		}
		if candidates[i].Corroborations > 0 {
			candidates[i].Score += min(candidates[i].Corroborations, 2)
		}
	}
}

func overlapScore(a, b string) float64 {
	if a == "" || b == "" {
		return 0
	}
	as := strings.Fields(a)
	bs := strings.Fields(b)
	if len(as) == 0 || len(bs) == 0 {
		return 0
	}
	set := map[string]struct{}{}
	for _, token := range as {
		set[token] = struct{}{}
	}
	shared := 0
	for _, token := range bs {
		if _, ok := set[token]; ok {
			shared++
		}
	}
	denominator := len(as)
	if len(bs) < denominator {
		denominator = len(bs)
	}
	return float64(shared) / float64(denominator)
}

func fallbackSummary(article model.Article) string {
	summary := strings.TrimSpace(article.SummaryRaw)
	if summary == "" {
		return "請查看原文取得完整內容。"
	}
	runes := []rune(summary)
	if len(runes) > 120 {
		return string(runes[:120]) + "..."
	}
	return summary
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
