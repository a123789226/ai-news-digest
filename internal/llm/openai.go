package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/boxiang/ai-news-digest/internal/model"
	"github.com/boxiang/ai-news-digest/internal/pipeline"
)

type Selector struct {
	apiKey string
	client *http.Client
	model  string
}

func NewSelectorFromEnv() *Selector {
	modelName := os.Getenv("GEMINI_MODEL")
	if modelName == "" {
		modelName = "gemini-2.5-flash"
	}
	return &Selector{
		apiKey: os.Getenv("GEMINI_API_KEY"),
		client: &http.Client{Timeout: 30 * time.Second},
		model:  modelName,
	}
}

func (s *Selector) SelectAndSummarize(ctx context.Context, candidates []pipeline.Candidate) ([]model.DigestItem, error) {
	if s.apiKey == "" {
		return nil, fmt.Errorf("missing GEMINI_API_KEY")
	}
	if len(candidates) == 0 {
		return nil, nil
	}

	payload := buildRequest(s.model, candidates)
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", s.model, s.apiKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		details, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("gemini status: %d: %s", resp.StatusCode, strings.TrimSpace(string(details)))
	}

	var result responseEnvelope
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	jsonText := result.OutputText()
	if jsonText == "" {
		return nil, fmt.Errorf("empty model output")
	}
	jsonText = extractJSONObject(jsonText)

	var parsed digestResponse
	if err := json.Unmarshal([]byte(jsonText), &parsed); err != nil {
		return nil, fmt.Errorf("parse digest json: %w", err)
	}
	return parsed.Items, nil
}

type requestEnvelope struct {
	Contents         []geminiContent  `json:"contents"`
	GenerationConfig generationConfig `json:"generationConfig"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type generationConfig struct {
	ResponseMIMEType string `json:"responseMimeType"`
}

type responseEnvelope struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

func (r responseEnvelope) OutputText() string {
	for _, candidate := range r.Candidates {
		for _, part := range candidate.Content.Parts {
			if strings.TrimSpace(part.Text) != "" {
				return part.Text
			}
		}
	}
	return ""
}

type digestResponse struct {
	Items []model.DigestItem `json:"items"`
}

func buildRequest(modelName string, candidates []pipeline.Candidate) requestEnvelope {
	candidateJSON, _ := json.Marshal(candidates[:min(len(candidates), 10)])
	prompt := "Select the top 1 to 3 AI news items from the candidates below. Return strict JSON with this exact shape: {\"items\":[{\"TitleEN\":\"...\",\"SummaryZH\":\"...\",\"WhyItMattersZH\":\"...\",\"Source\":\"...\",\"URL\":\"...\"}]}. Preserve the original English title. Write SummaryZH and WhyItMattersZH in Traditional Chinese. Avoid generic wording. Do not invent facts. If fewer than 3 strong items exist, return fewer. Candidates: " + string(candidateJSON)
	return requestEnvelope{
		Contents: []geminiContent{{
			Parts: []geminiPart{{Text: prompt}},
		}},
		GenerationConfig: generationConfig{
			ResponseMIMEType: "application/json",
		},
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func extractJSONObject(text string) string {
	text = strings.TrimSpace(text)
	if strings.HasPrefix(text, "```") {
		lines := strings.Split(text, "\n")
		if len(lines) >= 3 {
			text = strings.Join(lines[1:len(lines)-1], "\n")
			text = strings.TrimSpace(text)
		}
	}

	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start >= 0 && end > start {
		return text[start : end+1]
	}

	return text
}
