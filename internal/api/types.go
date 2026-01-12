package api

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type MessageRequest struct {
	Messages    []Message `json:"messages"`
	System      string    `json:"system,omitempty"`
	Model       string    `json:"model"`
	MaxTokens   int       `json:"max_tokens"`
	Temperature float64   `json:"temperature"`
}

type MessageResponse struct {
	Content    string `json:"content"`
	StopReason string `json:"stop_reason"`
	Usage      Usage  `json:"usage"`
}

type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}
