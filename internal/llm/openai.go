package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
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
	modelName := os.Getenv("OPENAI_MODEL")
	if modelName == "" {
		modelName = "gpt-4.1-mini"
	}
	return &Selector{
		apiKey: os.Getenv("OPENAI_API_KEY"),
		client: &http.Client{Timeout: 30 * time.Second},
		model:  modelName,
	}
}

func (s *Selector) SelectAndSummarize(ctx context.Context, candidates []pipeline.Candidate) ([]model.DigestItem, error) {
	if s.apiKey == "" {
		return nil, fmt.Errorf("missing OPENAI_API_KEY")
	}
	if len(candidates) == 0 {
		return nil, nil
	}

	payload := buildRequest(s.model, candidates)
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.openai.com/v1/responses", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("openai status: %d", resp.StatusCode)
	}

	var result responseEnvelope
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	jsonText := result.OutputText()
	if jsonText == "" {
		return nil, fmt.Errorf("empty model output")
	}

	var parsed digestResponse
	if err := json.Unmarshal([]byte(jsonText), &parsed); err != nil {
		return nil, fmt.Errorf("parse digest json: %w", err)
	}
	return parsed.Items, nil
}

type requestEnvelope struct {
	Model string       `json:"model"`
	Input []inputItem  `json:"input"`
	Text  responseText `json:"text"`
}

type inputItem struct {
	Role    string `json:"role"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
}

type responseText struct {
	Format responseFormat `json:"format"`
}

type responseFormat struct {
	Type string `json:"type"`
}

type responseEnvelope struct {
	Output []struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	} `json:"output"`
}

func (r responseEnvelope) OutputText() string {
	for _, output := range r.Output {
		for _, content := range output.Content {
			if content.Type == "output_text" || content.Type == "text" {
				return content.Text
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
	prompt := "Select the top 1 to 3 AI news items. Return strict JSON with shape {\"items\":[{\"TitleEN\":string,\"SummaryZH\":string,\"WhyItMattersZH\":string,\"Source\":string,\"URL\":string}]}. Preserve the original English title. Write SummaryZH and WhyItMattersZH in Traditional Chinese. Do not invent facts. Candidates: " + string(candidateJSON)
	content := struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}{Type: "input_text", Text: prompt}
	return requestEnvelope{
		Model: modelName,
		Input: []inputItem{{Role: "user", Content: []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		}{content}}},
		Text: responseText{Format: responseFormat{Type: "json_object"}},
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
