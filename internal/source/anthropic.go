package source

import (
	"context"
	"fmt"
	"html"
	"regexp"
	"strings"
	"time"

	"github.com/boxiang/ai-news-digest/internal/config"
	"github.com/boxiang/ai-news-digest/internal/model"
)

type AnthropicNewsProvider struct {
	fetcher *HTTPFetcher
	config  config.SourceConfig
}

func NewAnthropicNewsProvider(fetcher *HTTPFetcher, cfg config.SourceConfig) *AnthropicNewsProvider {
	return &AnthropicNewsProvider{fetcher: fetcher, config: cfg}
}

func (p *AnthropicNewsProvider) Name() string {
	return p.config.Name
}

var (
	anthropicLinkPattern = regexp.MustCompile(`<a[^>]+href="(/news/[^"#?]+)"[^>]*>(.*?)</a>`)
	htmlTagStripper      = regexp.MustCompile(`<[^>]+>`)
	anthropicTextPattern = regexp.MustCompile(`^((?:Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)\s+\d{1,2},\s+\d{4})\s+([A-Za-z ]+?)\s+(.+)$`)
)

func (p *AnthropicNewsProvider) Fetch(ctx context.Context) ([]model.Article, error) {
	payload, err := p.fetcher.Get(ctx, p.config.URL)
	if err != nil {
		return nil, err
	}

	matches := anthropicLinkPattern.FindAllStringSubmatch(string(payload), -1)
	articles := make([]model.Article, 0, len(matches))
	seen := make(map[string]struct{})

	for _, match := range matches {
		if len(match) < 3 {
			continue
		}

		path := match[1]
		if _, ok := seen[path]; ok {
			continue
		}

		text := cleanAnthropicAnchorText(match[2])
		if text == "" {
			continue
		}

		parsed, ok := parseAnthropicNewsText(text)
		if !ok {
			continue
		}

		url := "https://www.anthropic.com" + path
		articles = append(articles, model.Article{
			Source:      p.config.Name,
			SourceType:  p.config.Type,
			Title:       parsed.title,
			URL:         url,
			PublishedAt: parsed.publishedAt,
			SummaryRaw:  parsed.category,
		})
		seen[path] = struct{}{}

		if reachedLimit(articles, p.config.MaxItems) {
			break
		}
	}

	if len(articles) == 0 {
		return nil, fmt.Errorf("no news items parsed")
	}

	return articles, nil
}

type anthropicParsedItem struct {
	title       string
	category    string
	publishedAt time.Time
}

func cleanAnthropicAnchorText(value string) string {
	value = html.UnescapeString(value)
	value = htmlTagStripper.ReplaceAllString(value, " ")
	value = strings.Join(strings.Fields(value), " ")
	return strings.TrimSpace(value)
}

func parseAnthropicNewsText(text string) (anthropicParsedItem, bool) {
	match := anthropicTextPattern.FindStringSubmatch(text)
	if len(match) < 4 {
		return anthropicParsedItem{}, false
	}

	publishedAt, err := time.Parse("Jan 2, 2006", strings.TrimSpace(match[1]))
	if err != nil {
		return anthropicParsedItem{}, false
	}

	return anthropicParsedItem{
		title:       strings.TrimSpace(match[3]),
		category:    strings.TrimSpace(match[2]),
		publishedAt: publishedAt.UTC(),
	}, true
}
