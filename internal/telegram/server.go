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

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Server implements an MCP server for Telegram Bot API operations
type Server struct {
	mcpServer *server.MCPServer
	client    *http.Client
	botToken  string
	chatID    string
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
