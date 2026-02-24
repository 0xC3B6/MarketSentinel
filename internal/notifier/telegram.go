package notifier

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"
)

// TelegramNotifier sends messages via the Telegram Bot API.
type TelegramNotifier struct {
	BotToken string
	ChatID   string
	Client   *http.Client
}

// NewTelegramNotifier creates a notifier with optional proxy support.
func NewTelegramNotifier(botToken, chatID, proxyURL string) *TelegramNotifier {
	transport := &http.Transport{}
	if proxyURL != "" {
		if u, err := url.Parse(proxyURL); err == nil {
			transport.Proxy = http.ProxyURL(u)
		}
	}
	return &TelegramNotifier{
		BotToken: botToken,
		ChatID:   chatID,
		Client: &http.Client{
			Timeout:   30 * time.Second,
			Transport: transport,
		},
	}
}

// Send sends a message to the configured chat.
func (t *TelegramNotifier) Send(text string) error {
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", t.BotToken)
	payload := map[string]string{
		"chat_id":    t.ChatID,
		"text":       text,
		"parse_mode": "HTML",
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}
	resp, err := t.Client.Post(apiURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("send message: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("telegram API error: status %d, body: %s", resp.StatusCode, string(respBody))
	}
	return nil
}

// SendWithRetry sends a message with exponential backoff retry.
func (t *TelegramNotifier) SendWithRetry(ctx context.Context, text string, maxRetries int) error {
	var lastErr error
	for i := 0; i <= maxRetries; i++ {
		if err := t.Send(text); err != nil {
			lastErr = err
			backoff := time.Duration(1<<uint(i)) * time.Second
			log.Printf("[WARN] Telegram send failed (attempt %d/%d): %v, retrying in %v", i+1, maxRetries+1, err, backoff)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
				continue
			}
		}
		return nil
	}
	return fmt.Errorf("all %d retries exhausted: %w", maxRetries+1, lastErr)
}
