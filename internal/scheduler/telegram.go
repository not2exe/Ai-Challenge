package scheduler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// TelegramSender sends messages via Telegram Bot API.
type TelegramSender struct {
	botToken string
	chatID   string
	client   *http.Client
}

// NewTelegramSender creates a new Telegram sender.
func NewTelegramSender(botToken, chatID string) *TelegramSender {
	return &TelegramSender{
		botToken: botToken,
		chatID:   chatID,
		client:   &http.Client{Timeout: 30 * time.Second},
	}
}

type telegramSendRequest struct {
	ChatID    string `json:"chat_id"`
	Text      string `json:"text"`
	ParseMode string `json:"parse_mode"`
}

type telegramResponse struct {
	OK          bool   `json:"ok"`
	Description string `json:"description,omitempty"`
}

// SendMessage sends a message to the configured chat.
// Tries plain text (no parse mode) to avoid Markdown formatting errors from LLM output.
func (t *TelegramSender) SendMessage(text string) error {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", t.botToken)

	payload := telegramSendRequest{
		ChatID:    t.chatID,
		Text:      text,
		ParseMode: "HTML",
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal telegram request: %w", err)
	}

	resp, err := t.client.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to send telegram message: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read telegram response: %w", err)
	}

	var tgResp telegramResponse
	if err := json.Unmarshal(respBody, &tgResp); err != nil {
		return fmt.Errorf("failed to parse telegram response: %w", err)
	}

	if !tgResp.OK {
		return fmt.Errorf("telegram API error: %s", tgResp.Description)
	}

	return nil
}
