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

type AnthropicReleaseNotesProvider struct {
	fetcher *HTTPFetcher
	config  config.SourceConfig
}

func NewAnthropicReleaseNotesProvider(fetcher *HTTPFetcher, cfg config.SourceConfig) *AnthropicReleaseNotesProvider {
	return &AnthropicReleaseNotesProvider{fetcher: fetcher, config: cfg}
}

func (p *AnthropicReleaseNotesProvider) Name() string {
	return p.config.Name
}

var (
	releaseDatePattern = regexp.MustCompile(`<div>([A-Z][a-z]+ \d{1,2}, \d{4})</div>`)
	releaseItemPattern = regexp.MustCompile(`<li[^>]*>(.*?)</li>`)
	hrefPattern        = regexp.MustCompile(`href="([^"]+)"`)
)

func (p *AnthropicReleaseNotesProvider) Fetch(ctx context.Context) ([]model.Article, error) {
	payload, err := p.fetcher.Get(ctx, p.config.URL)
	if err != nil {
		return nil, err
	}

	htmlText := string(payload)
	dateMatches := releaseDatePattern.FindAllStringSubmatchIndex(htmlText, -1)
	if len(dateMatches) == 0 {
		return nil, fmt.Errorf("no release note dates parsed")
	}

	items := make([]model.Article, 0, p.config.MaxItems)
	for idx, match := range dateMatches {
		dateText := htmlText[match[2]:match[3]]
		publishedAt, err := time.Parse("January 2, 2006", dateText)
		if err != nil {
			continue
		}

		sectionStart := match[1]
		sectionEnd := len(htmlText)
		if idx+1 < len(dateMatches) {
			sectionEnd = dateMatches[idx+1][0]
		}
		section := htmlText[sectionStart:sectionEnd]

		itemMatches := releaseItemPattern.FindAllStringSubmatch(section, -1)
		for _, itemMatch := range itemMatches {
			if len(itemMatch) < 2 {
				continue
			}
			raw := itemMatch[1]
			title := cleanAnthropicReleaseText(raw)
			if title == "" {
				continue
			}

			url := p.config.URL + "#" + slugifyDate(dateText)
			if href := extractHref(raw); href != "" {
				url = absolutizeAnthropicURL(href)
			}

			items = append(items, model.Article{
				Source:      p.config.Name,
				SourceType:  p.config.Type,
				Title:       title,
				URL:         url,
				PublishedAt: publishedAt.UTC(),
				SummaryRaw:  "Anthropic API release note",
			})
			if reachedLimit(items, p.config.MaxItems) {
				return items, nil
			}
		}
	}

	if len(items) == 0 {
		return nil, fmt.Errorf("no release note items parsed")
	}

	return items, nil
}

func cleanAnthropicReleaseText(value string) string {
	value = html.UnescapeString(value)
	value = htmlTagStripper.ReplaceAllString(value, " ")
	value = strings.Join(strings.Fields(value), " ")
	return strings.TrimSpace(value)
}

func extractHref(value string) string {
	match := hrefPattern.FindStringSubmatch(value)
	if len(match) < 2 {
		return ""
	}
	return strings.TrimSpace(match[1])
}

func absolutizeAnthropicURL(href string) string {
	if strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://") {
		return href
	}
	if strings.HasPrefix(href, "/") {
		return "https://docs.anthropic.com" + href
	}
	return "https://docs.anthropic.com/" + href
}

func slugifyDate(value string) string {
	value = strings.ToLower(value)
	value = strings.ReplaceAll(value, ",", "")
	value = strings.ReplaceAll(value, " ", "-")
	return value
}
