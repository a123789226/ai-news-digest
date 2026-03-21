package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/boxiang/ai-news-digest/internal/config"
	"github.com/boxiang/ai-news-digest/internal/deliver"
	"github.com/boxiang/ai-news-digest/internal/format"
	"github.com/boxiang/ai-news-digest/internal/llm"
	"github.com/boxiang/ai-news-digest/internal/pipeline"
	"github.com/boxiang/ai-news-digest/internal/source"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	logger := log.New(os.Stdout, "", log.LstdFlags|log.Lmsgprefix)

	appConfig, err := config.Load("configs/sources.yaml")
	if err != nil {
		logger.Fatalf("load config: %v", err)
	}

	fetcher := source.NewHTTPFetcher(nil)
	providers := source.BuildProviders(fetcher, appConfig.Sources)

	articles, fetchErrs := source.Collect(ctx, providers, logger)
	for _, fetchErr := range fetchErrs {
		logger.Printf("source error: %v", fetchErr)
	}

	candidates := pipeline.PrepareCandidates(articles, time.Now().UTC())
	if len(candidates) == 0 {
		logger.Println("no candidates after filtering")
		return
	}

	selector := llm.NewSelectorFromEnv()
	items, err := selector.SelectAndSummarize(ctx, candidates)
	if err != nil {
		logger.Printf("llm fallback triggered: %v", err)
		items = pipeline.FallbackDigestItems(candidates, 3)
	}
	if len(items) == 0 {
		logger.Println("no digest items to send")
		return
	}

	message := format.TelegramDigest(items)
	sender, err := deliver.NewTelegramSenderFromEnv()
	if err != nil {
		logger.Fatalf("telegram config: %v", err)
	}
	if err := sender.Send(ctx, message); err != nil {
		logger.Fatalf("telegram send: %v", err)
	}
}
