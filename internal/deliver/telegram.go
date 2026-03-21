package deliver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

type TelegramSender struct {
	token  string
	chatID string
	client *http.Client
}

func NewTelegramSenderFromEnv() (*TelegramSender, error) {
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	chatID := os.Getenv("TELEGRAM_CHAT_ID")
	if token == "" || chatID == "" {
		return nil, fmt.Errorf("missing TELEGRAM_BOT_TOKEN or TELEGRAM_CHAT_ID")
	}
	return &TelegramSender{
		token:  token,
		chatID: chatID,
		client: &http.Client{Timeout: 15 * time.Second},
	}, nil
}

func (s *TelegramSender) Send(ctx context.Context, message string) error {
	payload := map[string]string{
		"chat_id": s.chatID,
		"text":    message,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal telegram payload: %w", err)
	}
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", s.token)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("new telegram request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("do telegram request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("telegram status: %d", resp.StatusCode)
	}
	return nil
}
