package format

import (
	"strings"
	"testing"

	"github.com/boxiang/ai-news-digest/internal/model"
)

func TestTelegramDigestIncludesAllSections(t *testing.T) {
	message := TelegramDigest([]model.DigestItem{{
		TitleEN:        "OpenAI releases model",
		SummaryZH:      "中文摘要",
		WhyItMattersZH: "重要原因",
		Source:         "OpenAI",
		URL:            "https://example.com",
	}})

	checks := []string{"AI News Digest", "OpenAI releases model", "中文摘要", "Why it matters", "https://example.com"}
	for _, check := range checks {
		if !strings.Contains(message, check) {
			t.Fatalf("message missing %q", check)
		}
	}
}
