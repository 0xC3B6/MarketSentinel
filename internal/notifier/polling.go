package notifier

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// CommandHandler is called when a user command is received.
type CommandHandler func(command string) string

// telegramUpdate represents a Telegram update from long polling.
type telegramUpdate struct {
	UpdateID int `json:"update_id"`
	Message  *struct {
		Text string `json:"text"`
	} `json:"message"`
}

// StartPolling begins long-polling for Telegram commands. Blocks until ctx is cancelled.
func (t *TelegramNotifier) StartPolling(ctx context.Context, handler CommandHandler) {
	offset := 0
	client := &http.Client{Timeout: 35 * time.Second}

	for {
		select {
		case <-ctx.Done():
			log.Println("[INFO] Telegram polling stopped")
			return
		default:
		}

		apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/getUpdates?offset=%d&timeout=30", t.BotToken, offset)
		req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
		if err != nil {
			log.Printf("[ERROR] create polling request: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}

		resp, err := client.Do(req)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("[WARN] polling request failed: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			log.Printf("[WARN] read polling response: %v", err)
			continue
		}

		var result struct {
			OK     bool             `json:"ok"`
			Result []telegramUpdate `json:"result"`
		}
		if err := json.Unmarshal(body, &result); err != nil {
			log.Printf("[WARN] decode polling response: %v", err)
			continue
		}

		for _, update := range result.Result {
			offset = update.UpdateID + 1
			if update.Message == nil || update.Message.Text == "" {
				continue
			}
			text := strings.TrimSpace(update.Message.Text)
			log.Printf("[INFO] received command: %s", text)
			reply := handler(text)
			if reply != "" {
				if err := t.Send(reply); err != nil {
					log.Printf("[ERROR] send reply: %v", err)
				}
			}
		}
	}
}
