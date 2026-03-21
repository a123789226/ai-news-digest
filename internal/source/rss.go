package source

import (
	"context"
	"encoding/xml"
	"fmt"
	"html"
	"regexp"
	"strings"
	"time"

	"github.com/boxiang/ai-news-digest/internal/config"
	"github.com/boxiang/ai-news-digest/internal/model"
)

type RSSProvider struct {
	fetcher *HTTPFetcher
	config  config.SourceConfig
}

func NewRSSProvider(fetcher *HTTPFetcher, cfg config.SourceConfig) *RSSProvider {
	return &RSSProvider{fetcher: fetcher, config: cfg}
}

func (p *RSSProvider) Name() string {
	return p.config.Name
}

func (p *RSSProvider) Fetch(ctx context.Context) ([]model.Article, error) {
	payload, err := p.fetcher.Get(ctx, p.config.URL)
	if err != nil {
		return nil, err
	}

	var rss rssDocument
	if err := xml.Unmarshal(payload, &rss); err != nil {
		return nil, fmt.Errorf("parse rss: %w", err)
	}

	articles := make([]model.Article, 0, len(rss.Channel.Items)+len(rss.Entries))
	for _, item := range rss.Channel.Items {
		publishedAt, _ := parsePublished(item.PubDate, item.Updated)
		article := model.Article{
			Source:      p.config.Name,
			SourceType:  p.config.Type,
			Title:       strings.TrimSpace(item.Title),
			URL:         strings.TrimSpace(item.Link),
			PublishedAt: publishedAt,
			SummaryRaw:  strings.TrimSpace(firstNonEmpty(item.Description, item.Content)),
		}
		if !matchesIncludeKeywords(article, p.config.IncludeKeywords, item.Categories) {
			continue
		}
		articles = append(articles, article)
	}
	for _, entry := range rss.Entries {
		publishedAt, _ := parsePublished(entry.Published, entry.Updated)
		link := entry.Link.Href
		if link == "" && len(entry.Links) > 0 {
			link = entry.Links[0].Href
		}
		article := model.Article{
			Source:      p.config.Name,
			SourceType:  p.config.Type,
			Title:       strings.TrimSpace(entry.Title),
			URL:         strings.TrimSpace(link),
			PublishedAt: publishedAt,
			SummaryRaw:  strings.TrimSpace(firstNonEmpty(entry.Summary, entry.Content)),
		}
		if !matchesIncludeKeywords(article, p.config.IncludeKeywords, entry.Categories) {
			continue
		}
		articles = append(articles, article)
	}
	return articles, nil
}

type rssDocument struct {
	Channel rssChannel `xml:"channel"`
	Entries []atomItem `xml:"entry"`
}

type rssChannel struct {
	Items []rssItem `xml:"item"`
}

type rssItem struct {
	Title       string   `xml:"title"`
	Link        string   `xml:"link"`
	Description string   `xml:"description"`
	Content     string   `xml:"encoded"`
	PubDate     string   `xml:"pubDate"`
	Updated     string   `xml:"updated"`
	Categories  []string `xml:"category"`
}

type atomItem struct {
	Title      string     `xml:"title"`
	Summary    string     `xml:"summary"`
	Content    string     `xml:"content"`
	Published  string     `xml:"published"`
	Updated    string     `xml:"updated"`
	Link       atomLink   `xml:"link"`
	Links      []atomLink `xml:"link"`
	Categories []string   `xml:"category>term"`
}

type atomLink struct {
	Href string `xml:"href,attr"`
}

var htmlTagPattern = regexp.MustCompile(`<[^>]+>`)

func parsePublished(values ...string) (time.Time, error) {
	layouts := []string{time.RFC1123Z, time.RFC1123, time.RFC3339, time.RFC822Z, time.RFC822, "2006-01-02T15:04:05-07:00"}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		for _, layout := range layouts {
			if ts, err := time.Parse(layout, value); err == nil {
				return ts.UTC(), nil
			}
		}
	}
	return time.Time{}, fmt.Errorf("unsupported timestamp")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = cleanText(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func matchesIncludeKeywords(article model.Article, keywords []string, categories []string) bool {
	if len(keywords) == 0 {
		return true
	}

	var builder strings.Builder
	builder.WriteString(strings.ToLower(article.Title))
	builder.WriteString(" ")
	builder.WriteString(strings.ToLower(article.SummaryRaw))
	builder.WriteString(" ")
	builder.WriteString(strings.ToLower(article.URL))
	for _, category := range categories {
		builder.WriteString(" ")
		builder.WriteString(strings.ToLower(category))
	}

	haystack := builder.String()
	for _, keyword := range keywords {
		if strings.Contains(haystack, strings.ToLower(keyword)) {
			return true
		}
	}

	return false
}

func cleanText(value string) string {
	value = html.UnescapeString(value)
	value = htmlTagPattern.ReplaceAllString(value, " ")
	value = strings.Join(strings.Fields(value), " ")
	return strings.TrimSpace(value)
}
