package codeindex

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// OllamaClient communicates with local Ollama instance for embeddings.
type OllamaClient struct {
	baseURL    string
	model      string
	httpClient *http.Client
}

// NewOllamaClient creates a new Ollama client.
func NewOllamaClient(baseURL, model string) *OllamaClient {
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	if model == "" {
		model = "nomic-embed-text" // Good default for code embeddings
	}

	return &OllamaClient{
		baseURL: baseURL,
		model:   model,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

// EmbeddingRequest represents the Ollama API embedding request.
type EmbeddingRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

// EmbeddingResponse represents the Ollama API embedding response.
type EmbeddingResponse struct {
	Embedding []float64 `json:"embedding"`
}

// GenerateEmbedding generates an embedding vector for the given text.
func (c *OllamaClient) GenerateEmbedding(ctx context.Context, text string) ([]float64, error) {
	req := EmbeddingRequest{
		Model:  c.model,
		Prompt: text,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama API error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var embedResp EmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&embedResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if len(embedResp.Embedding) == 0 {
		return nil, fmt.Errorf("empty embedding returned")
	}

	return embedResp.Embedding, nil
}

// GenerateBatchEmbeddings generates embeddings for multiple texts.
func (c *OllamaClient) GenerateBatchEmbeddings(ctx context.Context, texts []string) ([][]float64, error) {
	embeddings := make([][]float64, len(texts))

	for i, text := range texts {
		embed, err := c.GenerateEmbedding(ctx, text)
		if err != nil {
			return nil, fmt.Errorf("generate embedding for text %d: %w", i, err)
		}
		embeddings[i] = embed
	}

	return embeddings, nil
}

// CheckHealth checks if Ollama is running and the model is available.
func (c *OllamaClient) CheckHealth(ctx context.Context) error {
	// Try to generate a small test embedding
	_, err := c.GenerateEmbedding(ctx, "test")
	if err != nil {
		return fmt.Errorf("ollama health check failed: %w (ensure ollama is running and model '%s' is pulled)", err, c.model)
	}
	return nil
}

// GenerateRequest represents the Ollama API generate request.
type GenerateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

// GenerateResponse represents the Ollama API generate response.
type GenerateResponse struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

// Generate generates text using an LLM model.
// Uses a different model than embeddings (defaults to llama3.2 or qwen2.5).
func (c *OllamaClient) Generate(ctx context.Context, prompt string) (string, error) {
	// Use a small, fast model for reranking
	// Try common models in order of preference
	rerankModel := "qwen2.5:1.5b" // Small and fast

	req := GenerateRequest{
		Model:  rerankModel,
		Prompt: prompt,
		Stream: false,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/generate", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("ollama API error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var genResp GenerateResponse
	if err := json.NewDecoder(resp.Body).Decode(&genResp); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	return genResp.Response, nil
}
