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

var engineerPriorityKeywords = map[string]int{
	"api":         4,
	"sdk":         4,
	"cli":         3,
	"developer":   3,
	"developers":  3,
	"devtool":     3,
	"agent":       3,
	"agents":      3,
	"coding":      3,
	"code":        2,
	"open source": 4,
	"github":      3,
	"copilot":     2,
	"inference":   2,
	"tooling":     2,
	"framework":   3,
	"library":     3,
	"integration": 2,
	"plugin":      2,
	"oss":         2,
	"assistant":   2,
	"vscode":      2,
}

var penaltyKeywords = []string{"opinion", "how to", "tutorial", "guide"}
var lowValueKeywords = []string{"video", "podcast", "newsletter", "what happened at"}
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
	selected := selectFallbackCandidates(candidates, limit)
	items := make([]model.DigestItem, 0, len(selected))
	for _, candidate := range selected {
		items = append(items, model.DigestItem{
			TitleEN:        candidate.Article.Title,
			SummaryZH:      fallbackSummary(candidate.Article),
			WhyItMattersZH: fallbackWhyItMatters(candidate.Article),
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
		if isLowValueArticle(article) {
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
	body := normalizeTitle(article.SummaryRaw)
	for keyword, value := range keywordScores {
		if strings.Contains(title, keyword) {
			score += value
		}
	}
	for keyword, value := range engineerPriorityKeywords {
		if strings.Contains(title, keyword) {
			score += value
			continue
		}
		if body != "" && strings.Contains(body, keyword) {
			score += value
		}
	}
	for _, keyword := range penaltyKeywords {
		if strings.Contains(title, keyword) {
			score -= 2
		}
	}
	if isLowValueArticle(article) {
		score -= 5
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

func fallbackWhyItMatters(article model.Article) string {
	title := normalizeTitle(article.Title)
	switch {
	case isEngineerRelevant(article):
		return "這則消息和 API、工具鏈、開發流程或開源生態直接相關，對軟體工程師的實際工作影響更大。"
	case strings.Contains(title, "model") || strings.Contains(title, "launch") || strings.Contains(title, "release"):
		return "這則消息涉及模型或產品能力變動，會直接影響你後續選型與追蹤重點。"
	case strings.Contains(title, "policy") || strings.Contains(title, "court") || strings.Contains(title, "regulation"):
		return "這則消息牽涉政策或法律動向，可能改變 AI 產品與市場的限制條件。"
	case strings.Contains(title, "funding") || strings.Contains(title, "acquisition") || strings.Contains(title, "partnership"):
		return "這則消息反映產業資源流向，通常會影響接下來的競爭格局。"
	case article.SourceType == "official":
		return "這是官方來源的直接更新，資訊可信度高，值得優先閱讀。"
	default:
		return "這則消息在當日候選中排名靠前，值得你快速了解原文內容。"
	}
}

func isLowValueArticle(article model.Article) bool {
	title := normalizeTitle(article.Title)
	url := strings.ToLower(article.URL)
	if strings.Contains(url, "/video/") {
		return true
	}
	for _, keyword := range lowValueKeywords {
		if strings.Contains(title, keyword) {
			return true
		}
	}
	return false
}

func isEngineerRelevant(article model.Article) bool {
	title := normalizeTitle(article.Title)
	body := normalizeTitle(article.SummaryRaw)
	for keyword := range engineerPriorityKeywords {
		if strings.Contains(title, keyword) {
			return true
		}
		if body != "" && strings.Contains(body, keyword) {
			return true
		}
	}
	return false
}

func hasAlternativeSource(candidates []Candidate, seenSources map[string]struct{}) bool {
	for _, candidate := range candidates {
		if _, ok := seenSources[candidate.Article.Source]; !ok {
			return true
		}
	}
	return false
}

func selectFallbackCandidates(candidates []Candidate, limit int) []Candidate {
	if limit <= 0 || len(candidates) == 0 {
		return nil
	}

	selected := make([]Candidate, 0, min(limit, len(candidates)))
	seenSources := make(map[string]struct{})
	needPractical := min(2, limit)

	for _, candidate := range candidates {
		if len(selected) >= needPractical {
			break
		}
		if !isEngineerRelevant(candidate.Article) {
			continue
		}
		if _, ok := seenSources[candidate.Article.Source]; ok && hasAlternativeSource(candidates, seenSources) {
			continue
		}
		selected = append(selected, candidate)
		seenSources[candidate.Article.Source] = struct{}{}
	}

	for _, candidate := range candidates {
		if len(selected) >= limit {
			break
		}
		if containsCandidate(selected, candidate.Article.URL) {
			continue
		}
		if _, ok := seenSources[candidate.Article.Source]; ok && hasAlternativeSource(candidates, seenSources) {
			continue
		}
		selected = append(selected, candidate)
		seenSources[candidate.Article.Source] = struct{}{}
	}

	for _, candidate := range candidates {
		if len(selected) >= limit {
			break
		}
		if containsCandidate(selected, candidate.Article.URL) {
			continue
		}
		selected = append(selected, candidate)
	}

	return selected
}

func containsCandidate(candidates []Candidate, url string) bool {
	for _, candidate := range candidates {
		if candidate.Article.URL == url {
			return true
		}
	}
	return false
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
