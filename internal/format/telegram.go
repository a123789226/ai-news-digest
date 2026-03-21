package format

import (
	"fmt"
	"strings"

	"github.com/boxiang/ai-news-digest/internal/model"
)

func TelegramDigest(items []model.DigestItem) string {
	parts := make([]string, 0, len(items)+1)
	parts = append(parts, "AI News Digest")
	for i, item := range items {
		parts = append(parts, fmt.Sprintf("[%d] %s\n\n中文摘要：%s\n\nWhy it matters：%s\n\nSource: %s\nLink: %s", i+1, item.TitleEN, item.SummaryZH, item.WhyItMattersZH, item.Source, item.URL))
	}
	return strings.Join(parts, "\n\n")
}
