package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Server implements an MCP server for Telegram Bot API operations
type Server struct {
	mcpServer    *server.MCPServer
	client       *http.Client
	botToken     string
	chatID       string
	lastUpdateID int64
	updateMu     sync.Mutex
}

// NewServer creates a new Telegram MCP server
func NewServer() *Server {
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	chatID := os.Getenv("TELEGRAM_CHAT_ID")

	if botToken == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN environment variable is required")
	}
	if chatID == "" {
		log.Fatal("TELEGRAM_CHAT_ID environment variable is required")
	}

	s := &Server{
		client:   &http.Client{},
		botToken: botToken,
		chatID:   chatID,
	}

	s.mcpServer = server.NewMCPServer(
		"telegram",
		"1.0.0",
		server.WithToolCapabilities(false),
	)

	s.registerTools()

	return s
}

// MCPServer returns the underlying MCP server
func (s *Server) MCPServer() *server.MCPServer {
	return s.mcpServer
}

// registerTools registers all Telegram tools
func (s *Server) registerTools() {
	// Send text message
	s.mcpServer.AddTool(
		mcp.NewTool("send_message",
			mcp.WithDescription("Send a text message to the configured Telegram chat"),
			mcp.WithString("text", mcp.Required(), mcp.Description("The message text to send")),
			mcp.WithString("parse_mode", mcp.Description("Optional. Parse mode: 'HTML', 'Markdown', or 'MarkdownV2'. Default is 'HTML'")),
			mcp.WithBoolean("disable_notification", mcp.Description("Optional. Send message silently without notification")),
		),
		s.handleSendMessage,
	)

	// Send message with inline keyboard
	s.mcpServer.AddTool(
		mcp.NewTool("send_message_with_keyboard",
			mcp.WithDescription("Send a message with inline keyboard buttons to the Telegram chat"),
			mcp.WithString("text", mcp.Required(), mcp.Description("The message text to send")),
			mcp.WithString("buttons", mcp.Required(), mcp.Description("JSON array of button rows, e.g. [[{\"text\":\"Button 1\",\"callback_data\":\"data1\"}]]")),
			mcp.WithString("parse_mode", mcp.Description("Optional. Parse mode: 'HTML', 'Markdown', or 'MarkdownV2'. Default is 'HTML'")),
		),
		s.handleSendMessageWithKeyboard,
	)

	// Send photo
	s.mcpServer.AddTool(
		mcp.NewTool("send_photo",
			mcp.WithDescription("Send a photo to the configured Telegram chat. Supports local files, HTTP URLs, and file_ids."),
			mcp.WithString("photo_url", mcp.Required(), mcp.Description("Local file path (e.g., /path/to/image.png or file:///path/to/image.png), HTTP URL, or Telegram file_id")),
			mcp.WithString("caption", mcp.Description("Optional. Photo caption (max 1024 characters)")),
			mcp.WithString("parse_mode", mcp.Description("Optional. Parse mode for caption: 'HTML', 'Markdown', or 'MarkdownV2'")),
		),
		s.handleSendPhoto,
	)

	// Get chat info
	s.mcpServer.AddTool(
		mcp.NewTool("get_chat",
			mcp.WithDescription("Get information about the configured Telegram chat"),
		),
		s.handleGetChat,
	)

	// Edit message
	s.mcpServer.AddTool(
		mcp.NewTool("edit_message",
			mcp.WithDescription("Edit a previously sent message text"),
			mcp.WithNumber("message_id", mcp.Required(), mcp.Description("Identifier of the message to edit")),
			mcp.WithString("text", mcp.Required(), mcp.Description("New text of the message")),
			mcp.WithString("parse_mode", mcp.Description("Optional. Parse mode: 'HTML', 'Markdown', or 'MarkdownV2'")),
		),
		s.handleEditMessage,
	)

	// Delete message
	s.mcpServer.AddTool(
		mcp.NewTool("delete_message",
			mcp.WithDescription("Delete a message from the Telegram chat"),
			mcp.WithNumber("message_id", mcp.Required(), mcp.Description("Identifier of the message to delete")),
		),
		s.handleDeleteMessage,
	)

	// Get bot info
	s.mcpServer.AddTool(
		mcp.NewTool("get_me",
			mcp.WithDescription("Get information about the Telegram bot"),
		),
		s.handleGetMe,
	)

	// Get updates (incoming messages)
	s.mcpServer.AddTool(
		mcp.NewTool("get_updates",
			mcp.WithDescription("Get new incoming messages from the Telegram chat. Uses long polling to wait for messages."),
			mcp.WithNumber("timeout", mcp.Description("Long polling timeout in seconds (1-50). Default is 30. Telegram will hold the connection until a message arrives or timeout expires.")),
			mcp.WithNumber("limit", mcp.Description("Maximum number of updates to return (1-100). Default is 10.")),
		),
		s.handleGetUpdates,
	)

	// Wait for reply after sending a message
	s.mcpServer.AddTool(
		mcp.NewTool("send_and_wait_reply",
			mcp.WithDescription("Send a message and wait for a reply from the user. Uses long polling to wait up to the specified timeout."),
			mcp.WithString("text", mcp.Required(), mcp.Description("The message text to send")),
			mcp.WithString("parse_mode", mcp.Description("Optional. Parse mode: 'HTML', 'Markdown', or 'MarkdownV2'. Default is 'HTML'")),
			mcp.WithNumber("wait_timeout", mcp.Description("How long to wait for a reply in seconds (1-600). Default is 300 (5 minutes). Maximum is 600 (10 minutes).")),
		),
		s.handleSendAndWaitReply,
	)
}

// handleSendMessage sends a text message
func (s *Server) handleSendMessage(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	text := req.GetString("text", "")
	if text == "" {
		return mcp.NewToolResultError("text parameter required"), nil
	}

	parseMode := req.GetString("parse_mode", "HTML")
	if parseMode == "" {
		parseMode = "HTML"
	}

	disableNotification := req.GetBool("disable_notification", false)

	payload := map[string]interface{}{
		"chat_id":              s.chatID,
		"text":                 text,
		"parse_mode":           parseMode,
		"disable_notification": disableNotification,
	}

	result, err := s.callTelegramAPI("sendMessage", payload)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to send message: %v", err)), nil
	}

	// Extract message_id from result
	var response struct {
		OK     bool `json:"ok"`
		Result struct {
			MessageID int `json:"message_id"`
		} `json:"result"`
	}
	if err := json.Unmarshal(result, &response); err == nil && response.OK {
		return mcp.NewToolResultText(fmt.Sprintf("Message sent successfully. Message ID: %d", response.Result.MessageID)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Message sent: %s", string(result))), nil
}

// handleSendMessageWithKeyboard sends a message with inline keyboard
func (s *Server) handleSendMessageWithKeyboard(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	text := req.GetString("text", "")
	if text == "" {
		return mcp.NewToolResultError("text parameter required"), nil
	}

	buttonsJSON := req.GetString("buttons", "")
	if buttonsJSON == "" {
		return mcp.NewToolResultError("buttons parameter required"), nil
	}

	parseMode := req.GetString("parse_mode", "HTML")
	if parseMode == "" {
		parseMode = "HTML"
	}

	// Parse buttons JSON
	var buttons [][]map[string]interface{}
	if err := json.Unmarshal([]byte(buttonsJSON), &buttons); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid buttons JSON: %v", err)), nil
	}

	payload := map[string]interface{}{
		"chat_id":    s.chatID,
		"text":       text,
		"parse_mode": parseMode,
		"reply_markup": map[string]interface{}{
			"inline_keyboard": buttons,
		},
	}

	result, err := s.callTelegramAPI("sendMessage", payload)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to send message: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Message with keyboard sent: %s", string(result))), nil
}

// handleSendPhoto sends a photo
func (s *Server) handleSendPhoto(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	photoURL := req.GetString("photo_url", "")
	if photoURL == "" {
		return mcp.NewToolResultError("photo_url parameter required"), nil
	}

	caption := req.GetString("caption", "")
	parseMode := req.GetString("parse_mode", "")

	// Check if it's a local file path
	filePath := photoURL
	// Remove file:// prefix if present
	if strings.HasPrefix(filePath, "file://") {
		filePath = strings.TrimPrefix(filePath, "file://")
	}

	// Check if file exists locally
	if _, err := os.Stat(filePath); err == nil {
		// It's a local file, upload it
		result, err := s.uploadPhotoFile(filePath, caption, parseMode)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to upload photo: %v", err)), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("Photo uploaded: %s", string(result))), nil
	}

	// Not a local file, treat as URL
	if !strings.HasPrefix(photoURL, "http://") && !strings.HasPrefix(photoURL, "https://") {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid photo path: %s (file not found and not a valid HTTP URL)", photoURL)), nil
	}

	// Send via URL
	payload := map[string]interface{}{
		"chat_id": s.chatID,
		"photo":   photoURL,
	}

	if caption != "" {
		payload["caption"] = caption
	}

	if parseMode != "" {
		payload["parse_mode"] = parseMode
	}

	result, err := s.callTelegramAPI("sendPhoto", payload)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to send photo: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Photo sent: %s", string(result))), nil
}

// handleGetChat gets chat information
func (s *Server) handleGetChat(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	payload := map[string]interface{}{
		"chat_id": s.chatID,
	}

	result, err := s.callTelegramAPI("getChat", payload)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get chat info: %v", err)), nil
	}

	return mcp.NewToolResultText(string(result)), nil
}

// handleEditMessage edits a message
func (s *Server) handleEditMessage(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	messageID := req.GetFloat("message_id", 0)
	if messageID == 0 {
		return mcp.NewToolResultError("message_id parameter required"), nil
	}

	text := req.GetString("text", "")
	if text == "" {
		return mcp.NewToolResultError("text parameter required"), nil
	}

	payload := map[string]interface{}{
		"chat_id":    s.chatID,
		"message_id": int(messageID),
		"text":       text,
	}

	parseMode := req.GetString("parse_mode", "")
	if parseMode != "" {
		payload["parse_mode"] = parseMode
	}

	result, err := s.callTelegramAPI("editMessageText", payload)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to edit message: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Message edited: %s", string(result))), nil
}

// handleDeleteMessage deletes a message
func (s *Server) handleDeleteMessage(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	messageID := req.GetFloat("message_id", 0)
	if messageID == 0 {
		return mcp.NewToolResultError("message_id parameter required"), nil
	}

	payload := map[string]interface{}{
		"chat_id":    s.chatID,
		"message_id": int(messageID),
	}

	result, err := s.callTelegramAPI("deleteMessage", payload)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to delete message: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Message deleted: %s", string(result))), nil
}

// handleGetMe gets bot information
func (s *Server) handleGetMe(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	result, err := s.callTelegramAPI("getMe", nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get bot info: %v", err)), nil
	}

	return mcp.NewToolResultText(string(result)), nil
}

// callTelegramAPI makes a request to the Telegram Bot API
func (s *Server) callTelegramAPI(method string, payload map[string]interface{}) ([]byte, error) {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/%s", s.botToken, method)

	var body io.Reader
	if payload != nil {
		jsonData, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal payload: %w", err)
		}
		body = strings.NewReader(string(jsonData))
	}

	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(responseBody))
	}

	return responseBody, nil
}

// uploadPhotoFile uploads a local photo file to Telegram
func (s *Server) uploadPhotoFile(filePath, caption, parseMode string) ([]byte, error) {
	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Create multipart form
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add chat_id
	if err := writer.WriteField("chat_id", s.chatID); err != nil {
		return nil, fmt.Errorf("failed to write chat_id field: %w", err)
	}

	// Add caption if provided
	if caption != "" {
		if err := writer.WriteField("caption", caption); err != nil {
			return nil, fmt.Errorf("failed to write caption field: %w", err)
		}
	}

	// Add parse_mode if provided
	if parseMode != "" {
		if err := writer.WriteField("parse_mode", parseMode); err != nil {
			return nil, fmt.Errorf("failed to write parse_mode field: %w", err)
		}
	}

	// Add file
	filename := filepath.Base(filePath)
	part, err := writer.CreateFormFile("photo", filename)
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}

	if _, err := io.Copy(part, file); err != nil {
		return nil, fmt.Errorf("failed to copy file data: %w", err)
	}

	// Close writer
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close writer: %w", err)
	}

	// Create request
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendPhoto", s.botToken)
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Send request
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(responseBody))
	}

	return responseBody, nil
}

// TelegramUpdate represents an update from Telegram
type TelegramUpdate struct {
	UpdateID int64            `json:"update_id"`
	Message  *TelegramMessage `json:"message,omitempty"`
}

// TelegramMessage represents a message in Telegram
type TelegramMessage struct {
	MessageID int64            `json:"message_id"`
	From      *TelegramUser    `json:"from,omitempty"`
	Chat      *TelegramChat    `json:"chat"`
	Date      int64            `json:"date"`
	Text      string           `json:"text,omitempty"`
	Photo     []interface{}    `json:"photo,omitempty"`
	Document  interface{}      `json:"document,omitempty"`
	ReplyTo   *TelegramMessage `json:"reply_to_message,omitempty"`
}

// TelegramUser represents a Telegram user
type TelegramUser struct {
	ID        int64  `json:"id"`
	IsBot     bool   `json:"is_bot"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name,omitempty"`
	Username  string `json:"username,omitempty"`
}

// TelegramChat represents a Telegram chat
type TelegramChat struct {
	ID        int64  `json:"id"`
	Type      string `json:"type"`
	Title     string `json:"title,omitempty"`
	Username  string `json:"username,omitempty"`
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
}

// GetUpdatesResponse represents the response from getUpdates
type GetUpdatesResponse struct {
	OK     bool             `json:"ok"`
	Result []TelegramUpdate `json:"result"`
}

// handleGetUpdates gets new messages using long polling
func (s *Server) handleGetUpdates(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	timeout := int(req.GetFloat("timeout", 30))
	if timeout < 1 {
		timeout = 1
	}
	if timeout > 50 {
		timeout = 50 // Telegram max is 50 seconds
	}

	limit := int(req.GetFloat("limit", 10))
	if limit < 1 {
		limit = 1
	}
	if limit > 100 {
		limit = 100
	}

	s.updateMu.Lock()
	offset := s.lastUpdateID
	s.updateMu.Unlock()

	payload := map[string]interface{}{
		"timeout":         timeout,
		"limit":           limit,
		"allowed_updates": []string{"message"},
	}
	if offset > 0 {
		payload["offset"] = offset + 1
	}

	// Create a client with extended timeout for long polling
	client := &http.Client{
		Timeout: time.Duration(timeout+10) * time.Second,
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/getUpdates", s.botToken)
	jsonData, _ := json.Marshal(payload)

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(jsonData)))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create request: %v", err)), nil
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(httpReq)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Request failed: %v", err)), nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to read response: %v", err)), nil
	}

	var updatesResp GetUpdatesResponse
	if err := json.Unmarshal(body, &updatesResp); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to parse response: %v", err)), nil
	}

	if !updatesResp.OK {
		return mcp.NewToolResultError(fmt.Sprintf("Telegram API error: %s", string(body))), nil
	}

	// Update the offset
	if len(updatesResp.Result) > 0 {
		s.updateMu.Lock()
		lastUpdate := updatesResp.Result[len(updatesResp.Result)-1]
		if lastUpdate.UpdateID > s.lastUpdateID {
			s.lastUpdateID = lastUpdate.UpdateID
		}
		s.updateMu.Unlock()
	}

	// Filter messages from the configured chat
	var messages []map[string]interface{}
	for _, update := range updatesResp.Result {
		if update.Message != nil && fmt.Sprintf("%d", update.Message.Chat.ID) == s.chatID {
			msg := map[string]interface{}{
				"message_id": update.Message.MessageID,
				"date":       update.Message.Date,
				"text":       update.Message.Text,
			}
			if update.Message.From != nil {
				msg["from"] = map[string]interface{}{
					"id":         update.Message.From.ID,
					"first_name": update.Message.From.FirstName,
					"last_name":  update.Message.From.LastName,
					"username":   update.Message.From.Username,
					"is_bot":     update.Message.From.IsBot,
				}
			}
			if update.Message.ReplyTo != nil {
				msg["reply_to_message_id"] = update.Message.ReplyTo.MessageID
			}
			messages = append(messages, msg)
		}
	}

	result, _ := json.MarshalIndent(map[string]interface{}{
		"count":    len(messages),
		"messages": messages,
	}, "", "  ")

	return mcp.NewToolResultText(string(result)), nil
}

// handleSendAndWaitReply sends a message and waits for a reply
func (s *Server) handleSendAndWaitReply(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	text := req.GetString("text", "")
	if text == "" {
		return mcp.NewToolResultError("text parameter required"), nil
	}

	parseMode := req.GetString("parse_mode", "HTML")
	if parseMode == "" {
		parseMode = "HTML"
	}

	waitTimeout := int(req.GetFloat("wait_timeout", 300))
	if waitTimeout < 1 {
		waitTimeout = 1
	}
	if waitTimeout > 600 {
		waitTimeout = 600 // Max 10 minutes
	}

	// First, send the message
	sendPayload := map[string]interface{}{
		"chat_id":    s.chatID,
		"text":       text,
		"parse_mode": parseMode,
	}

	sendResult, err := s.callTelegramAPI("sendMessage", sendPayload)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to send message: %v", err)), nil
	}

	var sendResp struct {
		OK     bool `json:"ok"`
		Result struct {
			MessageID int64 `json:"message_id"`
		} `json:"result"`
	}
	if err := json.Unmarshal(sendResult, &sendResp); err != nil || !sendResp.OK {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to parse send response: %s", string(sendResult))), nil
	}

	sentMessageID := sendResp.Result.MessageID

	// Clear any pending updates first to ensure we only get new messages
	s.updateMu.Lock()
	// Fetch current updates to get the latest offset
	clearPayload := map[string]interface{}{
		"timeout": 0,
		"limit":   100,
	}
	s.updateMu.Unlock()

	clearResult, _ := s.callTelegramAPI("getUpdates", clearPayload)
	var clearResp GetUpdatesResponse
	if err := json.Unmarshal(clearResult, &clearResp); err == nil && clearResp.OK && len(clearResp.Result) > 0 {
		s.updateMu.Lock()
		s.lastUpdateID = clearResp.Result[len(clearResp.Result)-1].UpdateID
		s.updateMu.Unlock()
	}

	// Now wait for a reply using long polling
	// Telegram max timeout is 50 seconds, so we loop
	startTime := time.Now()
	deadline := startTime.Add(time.Duration(waitTimeout) * time.Second)

	for time.Now().Before(deadline) {
		// Calculate remaining time
		remaining := time.Until(deadline)
		pollTimeout := 50 // Max Telegram allows
		if remaining < time.Duration(pollTimeout)*time.Second {
			pollTimeout = int(remaining.Seconds())
			if pollTimeout < 1 {
				pollTimeout = 1
			}
		}

		s.updateMu.Lock()
		offset := s.lastUpdateID
		s.updateMu.Unlock()

		payload := map[string]interface{}{
			"timeout":         pollTimeout,
			"limit":           10,
			"allowed_updates": []string{"message"},
		}
		if offset > 0 {
			payload["offset"] = offset + 1
		}

		// Create client with extended timeout
		client := &http.Client{
			Timeout: time.Duration(pollTimeout+10) * time.Second,
		}

		url := fmt.Sprintf("https://api.telegram.org/bot%s/getUpdates", s.botToken)
		jsonData, _ := json.Marshal(payload)

		httpReq, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(jsonData)))
		if err != nil {
			continue
		}
		httpReq.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(httpReq)
		if err != nil {
			// Check if context was cancelled
			if ctx.Err() != nil {
				return mcp.NewToolResultText(fmt.Sprintf(`{"sent_message_id": %d, "reply": null, "status": "cancelled", "waited_seconds": %.0f}`, sentMessageID, time.Since(startTime).Seconds())), nil
			}
			continue
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			continue
		}

		var updatesResp GetUpdatesResponse
		if err := json.Unmarshal(body, &updatesResp); err != nil || !updatesResp.OK {
			continue
		}

		// Update offset
		if len(updatesResp.Result) > 0 {
			s.updateMu.Lock()
			lastUpdate := updatesResp.Result[len(updatesResp.Result)-1]
			if lastUpdate.UpdateID > s.lastUpdateID {
				s.lastUpdateID = lastUpdate.UpdateID
			}
			s.updateMu.Unlock()
		}

		// Check for reply from the configured chat
		for _, update := range updatesResp.Result {
			if update.Message == nil {
				continue
			}
			// Check if it's from our chat
			if fmt.Sprintf("%d", update.Message.Chat.ID) != s.chatID {
				continue
			}
			// Skip bot messages
			if update.Message.From != nil && update.Message.From.IsBot {
				continue
			}

			// Found a human reply!
			reply := map[string]interface{}{
				"message_id": update.Message.MessageID,
				"text":       update.Message.Text,
				"date":       update.Message.Date,
			}
			if update.Message.From != nil {
				reply["from"] = map[string]interface{}{
					"id":         update.Message.From.ID,
					"first_name": update.Message.From.FirstName,
					"last_name":  update.Message.From.LastName,
					"username":   update.Message.From.Username,
				}
			}

			result, _ := json.MarshalIndent(map[string]interface{}{
				"sent_message_id": sentMessageID,
				"reply":           reply,
				"status":          "received",
				"waited_seconds":  time.Since(startTime).Seconds(),
			}, "", "  ")

			return mcp.NewToolResultText(string(result)), nil
		}
	}

	// Timeout reached, no reply
	result, _ := json.MarshalIndent(map[string]interface{}{
		"sent_message_id": sentMessageID,
		"reply":           nil,
		"status":          "timeout",
		"waited_seconds":  waitTimeout,
	}, "", "  ")

	return mcp.NewToolResultText(string(result)), nil
}
