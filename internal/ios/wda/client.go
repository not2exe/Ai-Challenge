package wda

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const defaultPort = 8100
const defaultTimeout = 30 * time.Second

// Client is a WebDriverAgent HTTP client.
type Client struct {
	baseURL    string
	httpClient *http.Client
	sessionID  string
}

// NewClient creates a new WDA client.
// If port is 0, uses default port 8100.
func NewClient(host string, port int) *Client {
	if port == 0 {
		port = defaultPort
	}
	if host == "" {
		host = "localhost"
	}

	return &Client{
		baseURL: fmt.Sprintf("http://%s:%d", host, port),
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
	}
}

// SetTimeout sets the HTTP client timeout.
func (c *Client) SetTimeout(d time.Duration) {
	c.httpClient.Timeout = d
}

// Status checks if WDA is running and returns status info.
func (c *Client) Status(ctx context.Context) (*StatusInfo, error) {
	resp, err := c.get(ctx, "/status")
	if err != nil {
		return nil, err
	}

	var statusResp StatusResponse
	if err := json.Unmarshal(resp, &statusResp); err != nil {
		return nil, fmt.Errorf("failed to parse status response: %w", err)
	}

	return &statusResp.Value, nil
}

// CreateSession creates a new WDA session.
func (c *Client) CreateSession(ctx context.Context) (*Session, error) {
	body := map[string]any{
		"capabilities": map[string]any{},
	}

	resp, err := c.post(ctx, "/session", body)
	if err != nil {
		return nil, err
	}

	var result struct {
		Value     Session `json:"value"`
		SessionID string  `json:"sessionId"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse session response: %w", err)
	}

	// Session ID can be in either place
	if result.Value.SessionID == "" {
		result.Value.SessionID = result.SessionID
	}
	c.sessionID = result.Value.SessionID

	return &result.Value, nil
}

// DeleteSession deletes the current session.
func (c *Client) DeleteSession(ctx context.Context) error {
	if c.sessionID == "" {
		return nil
	}

	_, err := c.delete(ctx, fmt.Sprintf("/session/%s", c.sessionID))
	if err != nil {
		return err
	}

	c.sessionID = ""
	return nil
}

// GetSessionID returns the current session ID.
func (c *Client) GetSessionID() string {
	return c.sessionID
}

// SetSessionID sets the session ID (for reusing existing sessions).
func (c *Client) SetSessionID(id string) {
	c.sessionID = id
}

// Source returns the page source (UI hierarchy) as XML.
func (c *Client) Source(ctx context.Context) (string, error) {
	if c.sessionID == "" {
		return "", fmt.Errorf("no active session")
	}

	resp, err := c.get(ctx, fmt.Sprintf("/session/%s/source", c.sessionID))
	if err != nil {
		return "", err
	}

	var sourceResp SourceResponse
	if err := json.Unmarshal(resp, &sourceResp); err != nil {
		return "", fmt.Errorf("failed to parse source response: %w", err)
	}

	return sourceResp.Value, nil
}

// SourceAccessible returns the accessibility tree as JSON.
func (c *Client) SourceAccessible(ctx context.Context) (string, error) {
	if c.sessionID == "" {
		return "", fmt.Errorf("no active session")
	}

	resp, err := c.get(ctx, fmt.Sprintf("/session/%s/wda/accessibleSource", c.sessionID))
	if err != nil {
		return "", err
	}

	return string(resp), nil
}

// WindowSize returns the window size.
func (c *Client) WindowSize(ctx context.Context) (*WindowSize, error) {
	if c.sessionID == "" {
		return nil, fmt.Errorf("no active session")
	}

	resp, err := c.get(ctx, fmt.Sprintf("/session/%s/window/size", c.sessionID))
	if err != nil {
		return nil, err
	}

	var sizeResp WindowSizeResponse
	if err := json.Unmarshal(resp, &sizeResp); err != nil {
		return nil, fmt.Errorf("failed to parse window size response: %w", err)
	}

	return &sizeResp.Value, nil
}

// FindElement finds a single element.
// using can be: "accessibility id", "class name", "name", "xpath", "predicate string", "class chain"
func (c *Client) FindElement(ctx context.Context, using, value string) (*Element, error) {
	if c.sessionID == "" {
		return nil, fmt.Errorf("no active session")
	}

	body := FindElementRequest{
		Using: using,
		Value: value,
	}

	resp, err := c.post(ctx, fmt.Sprintf("/session/%s/element", c.sessionID), body)
	if err != nil {
		return nil, err
	}

	var elemResp ElementResponse
	if err := json.Unmarshal(resp, &elemResp); err != nil {
		return nil, fmt.Errorf("failed to parse element response: %w", err)
	}

	return &elemResp.Value, nil
}

// FindElements finds multiple elements.
func (c *Client) FindElements(ctx context.Context, using, value string) ([]Element, error) {
	if c.sessionID == "" {
		return nil, fmt.Errorf("no active session")
	}

	body := FindElementRequest{
		Using: using,
		Value: value,
	}

	resp, err := c.post(ctx, fmt.Sprintf("/session/%s/elements", c.sessionID), body)
	if err != nil {
		return nil, err
	}

	var elemsResp ElementsResponse
	if err := json.Unmarshal(resp, &elemsResp); err != nil {
		return nil, fmt.Errorf("failed to parse elements response: %w", err)
	}

	return elemsResp.Value, nil
}

// GetElementAttribute gets an attribute of an element.
func (c *Client) GetElementAttribute(ctx context.Context, elementID, attribute string) (string, error) {
	if c.sessionID == "" {
		return "", fmt.Errorf("no active session")
	}

	resp, err := c.get(ctx, fmt.Sprintf("/session/%s/element/%s/attribute/%s", c.sessionID, elementID, attribute))
	if err != nil {
		return "", err
	}

	var result Response
	if err := json.Unmarshal(resp, &result); err != nil {
		return "", fmt.Errorf("failed to parse attribute response: %w", err)
	}

	if result.Value == nil {
		return "", nil
	}
	return fmt.Sprintf("%v", result.Value), nil
}

// GetElementRect gets the bounding rectangle of an element.
func (c *Client) GetElementRect(ctx context.Context, elementID string) (*Rect, error) {
	if c.sessionID == "" {
		return nil, fmt.Errorf("no active session")
	}

	resp, err := c.get(ctx, fmt.Sprintf("/session/%s/element/%s/rect", c.sessionID, elementID))
	if err != nil {
		return nil, err
	}

	var result struct {
		Value Rect `json:"value"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse rect response: %w", err)
	}

	return &result.Value, nil
}

// Click clicks on an element.
func (c *Client) Click(ctx context.Context, elementID string) error {
	if c.sessionID == "" {
		return fmt.Errorf("no active session")
	}

	_, err := c.post(ctx, fmt.Sprintf("/session/%s/element/%s/click", c.sessionID, elementID), map[string]any{})
	return err
}

// Tap taps at specific coordinates.
func (c *Client) Tap(ctx context.Context, x, y int) error {
	if c.sessionID == "" {
		return fmt.Errorf("no active session")
	}

	body := map[string]any{
		"x": x,
		"y": y,
	}

	_, err := c.post(ctx, fmt.Sprintf("/session/%s/wda/tap/0", c.sessionID), body)
	return err
}

// DoubleTap double taps at coordinates.
func (c *Client) DoubleTap(ctx context.Context, x, y int) error {
	if c.sessionID == "" {
		return fmt.Errorf("no active session")
	}

	body := map[string]any{
		"x": x,
		"y": y,
	}

	_, err := c.post(ctx, fmt.Sprintf("/session/%s/wda/doubleTap", c.sessionID), body)
	return err
}

// LongPress performs a long press at coordinates.
func (c *Client) LongPress(ctx context.Context, x, y int, duration float64) error {
	if c.sessionID == "" {
		return fmt.Errorf("no active session")
	}

	body := map[string]any{
		"x":        x,
		"y":        y,
		"duration": duration,
	}

	_, err := c.post(ctx, fmt.Sprintf("/session/%s/wda/touchAndHold", c.sessionID), body)
	return err
}

// Swipe performs a swipe gesture.
func (c *Client) Swipe(ctx context.Context, startX, startY, endX, endY int, duration float64) error {
	if c.sessionID == "" {
		return fmt.Errorf("no active session")
	}

	body := map[string]any{
		"fromX":    startX,
		"fromY":    startY,
		"toX":      endX,
		"toY":      endY,
		"duration": duration,
	}

	_, err := c.post(ctx, fmt.Sprintf("/session/%s/wda/dragfromtoforduration", c.sessionID), body)
	return err
}

// SendKeys types text into the currently focused element.
func (c *Client) SendKeys(ctx context.Context, text string) error {
	if c.sessionID == "" {
		return fmt.Errorf("no active session")
	}

	// WDA expects an array of characters
	chars := make([]string, len(text))
	for i, r := range text {
		chars[i] = string(r)
	}

	body := TypeRequest{
		Value: chars,
	}

	_, err := c.post(ctx, fmt.Sprintf("/session/%s/wda/keys", c.sessionID), body)
	return err
}

// ClearText clears text in an element.
func (c *Client) ClearText(ctx context.Context, elementID string) error {
	if c.sessionID == "" {
		return fmt.Errorf("no active session")
	}

	_, err := c.post(ctx, fmt.Sprintf("/session/%s/element/%s/clear", c.sessionID, elementID), map[string]any{})
	return err
}

// PressButton presses a hardware button.
// buttonName: "home", "volumeUp", "volumeDown"
func (c *Client) PressButton(ctx context.Context, buttonName string) error {
	if c.sessionID == "" {
		return fmt.Errorf("no active session")
	}

	body := map[string]any{
		"name": buttonName,
	}

	_, err := c.post(ctx, fmt.Sprintf("/session/%s/wda/pressButton", c.sessionID), body)
	return err
}

// Screenshot takes a screenshot and returns base64 encoded PNG.
func (c *Client) Screenshot(ctx context.Context) (string, error) {
	if c.sessionID == "" {
		return "", fmt.Errorf("no active session")
	}

	resp, err := c.get(ctx, fmt.Sprintf("/session/%s/screenshot", c.sessionID))
	if err != nil {
		return "", err
	}

	var result Response
	if err := json.Unmarshal(resp, &result); err != nil {
		return "", fmt.Errorf("failed to parse screenshot response: %w", err)
	}

	if result.Value == nil {
		return "", fmt.Errorf("no screenshot data returned")
	}
	return fmt.Sprintf("%v", result.Value), nil
}

// Helper methods for HTTP requests

func (c *Client) get(ctx context.Context, path string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	return c.doRequest(req)
}

func (c *Client) post(ctx context.Context, path string, body any) ([]byte, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	return c.doRequest(req)
}

func (c *Client) delete(ctx context.Context, path string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.baseURL+path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	return c.doRequest(req)
}

func (c *Client) doRequest(req *http.Request) ([]byte, error) {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		// Try to extract error message from response
		var errResp struct {
			Value struct {
				Message string `json:"message"`
				Error   string `json:"error"`
			} `json:"value"`
		}
		if json.Unmarshal(body, &errResp) == nil {
			msg := errResp.Value.Message
			if msg == "" {
				msg = errResp.Value.Error
			}
			if msg != "" {
				return nil, fmt.Errorf("WDA error: %s", msg)
			}
		}
		return nil, fmt.Errorf("WDA request failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	return body, nil
}
