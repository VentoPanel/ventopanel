package notifier

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Noop is a no-op notifier used when credentials are not configured.
type Noop struct{}

func NewNoop() *Noop { return &Noop{} }

func (n *Noop) Notify(_ context.Context, _ string) error { return nil }

// Telegram sends messages via the Telegram Bot API.
type Telegram struct {
	token      string
	chatID     string
	httpClient *http.Client
}

func NewTelegram(token, chatID string) *Telegram {
	return &Telegram{
		token:  token,
		chatID: chatID,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (t *Telegram) Notify(ctx context.Context, message string) error {
	if t.token == "" || t.chatID == "" {
		return nil
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", t.token)
	body, err := json.Marshal(map[string]string{
		"chat_id":    t.chatID,
		"text":       message,
		"parse_mode": "HTML",
	})
	if err != nil {
		return fmt.Errorf("telegram: marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("telegram: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("telegram: send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("telegram: unexpected status %d", resp.StatusCode)
	}

	return nil
}

// WhatsApp sends messages to a generic webhook (e.g. Twilio, Evolution API).
type WhatsApp struct {
	webhookURL string
	httpClient *http.Client
}

func NewWhatsApp(webhookURL string) *WhatsApp {
	return &WhatsApp{
		webhookURL: webhookURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (w *WhatsApp) Notify(ctx context.Context, message string) error {
	if w.webhookURL == "" {
		return nil
	}

	body, err := json.Marshal(map[string]string{"text": message})
	if err != nil {
		return fmt.Errorf("whatsapp: marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, w.webhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("whatsapp: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := w.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("whatsapp: send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("whatsapp: unexpected status %d", resp.StatusCode)
	}

	return nil
}
