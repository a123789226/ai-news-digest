package source

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/boxiang/ai-news-digest/internal/config"
	"github.com/boxiang/ai-news-digest/internal/model"
)

type Provider interface {
	Name() string
	Fetch(ctx context.Context) ([]model.Article, error)
}

type HTTPFetcher struct {
	client *http.Client
}

func NewHTTPFetcher(client *http.Client) *HTTPFetcher {
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second}
	}
	return &HTTPFetcher{client: client}
}

func (f *HTTPFetcher) Get(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("User-Agent", "ai-news-digest/0.1")

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}
	return body, nil
}

func BuildProviders(fetcher *HTTPFetcher, configs []config.SourceConfig) []Provider {
	providers := make([]Provider, 0, len(configs))
	for _, cfg := range configs {
		if !cfg.Enabled {
			continue
		}
		switch cfg.Mode {
		case "rss":
			providers = append(providers, NewRSSProvider(fetcher, cfg))
		}
	}
	return providers
}

func Collect(ctx context.Context, providers []Provider, logger *log.Logger) ([]model.Article, []error) {
	articles := make([]model.Article, 0)
	errs := make([]error, 0)
	for _, provider := range providers {
		result, err := provider.Fetch(ctx)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", provider.Name(), err))
			continue
		}
		logger.Printf("fetched %d articles from %s", len(result), provider.Name())
		articles = append(articles, result...)
	}
	return articles, errs
}
