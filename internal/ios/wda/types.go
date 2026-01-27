// Package wda provides a client for WebDriverAgent REST API.
package wda

// Session represents a WDA session.
type Session struct {
	SessionID    string            `json:"sessionId"`
	Capabilities map[string]any    `json:"capabilities,omitempty"`
}

// Response is the generic WDA response wrapper.
type Response struct {
	Value     any    `json:"value"`
	SessionID string `json:"sessionId,omitempty"`
	Status    int    `json:"status,omitempty"`
}

// Element represents a UI element.
type Element struct {
	ElementID string `json:"ELEMENT"`
}

// ElementResponse is the response from find element.
type ElementResponse struct {
	Value     Element `json:"value"`
	SessionID string  `json:"sessionId"`
}

// ElementsResponse is the response from find elements.
type ElementsResponse struct {
	Value     []Element `json:"value"`
	SessionID string    `json:"sessionId"`
}

// ElementAttribute contains element attribute info.
type ElementAttribute struct {
	Type              string `json:"type"`
	Value             string `json:"value"`
	Name              string `json:"name"`
	Label             string `json:"label"`
	Enabled           bool   `json:"enabled"`
	Visible           bool   `json:"visible"`
	AccessibilityID   string `json:"accessibilityId,omitempty"`
	Rect              Rect   `json:"rect"`
}

// Rect represents element bounds.
type Rect struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

// SourceResponse is the response from source endpoint.
type SourceResponse struct {
	Value     string `json:"value"`
	SessionID string `json:"sessionId"`
}

// WindowSize represents screen dimensions.
type WindowSize struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

// WindowSizeResponse wraps window size.
type WindowSizeResponse struct {
	Value     WindowSize `json:"value"`
	SessionID string     `json:"sessionId"`
}

// FindElementRequest is the request body for finding elements.
type FindElementRequest struct {
	Using string `json:"using"`
	Value string `json:"value"`
}

// TouchAction represents a touch action for gestures.
type TouchAction struct {
	Action  string  `json:"action"`
	Options *TouchOptions `json:"options,omitempty"`
}

// TouchOptions contains coordinates and other touch parameters.
type TouchOptions struct {
	X        int `json:"x,omitempty"`
	Y        int `json:"y,omitempty"`
	Element  string `json:"element,omitempty"`
	Duration int    `json:"duration,omitempty"`
}

// SwipeOptions contains parameters for swipe gesture.
type SwipeOptions struct {
	StartX   int `json:"startX"`
	StartY   int `json:"startY"`
	EndX     int `json:"endX"`
	EndY     int `json:"endY"`
	Duration int `json:"duration"` // milliseconds
}

// TypeRequest is the request for typing text.
type TypeRequest struct {
	Value []string `json:"value"`
}

// StatusInfo contains WDA server status.
type StatusInfo struct {
	Build struct {
		ProductBundleIdentifier string `json:"productBundleIdentifier"`
		Time                    string `json:"time"`
	} `json:"build"`
	OS struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"os"`
	State string `json:"state"`
}

// StatusResponse wraps status info.
type StatusResponse struct {
	Value     StatusInfo `json:"value"`
	SessionID string     `json:"sessionId"`
}
