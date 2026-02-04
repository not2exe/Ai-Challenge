package codeindex

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const (
	serverName    = "codeindex"
	serverVersion = "1.0.0"
)

// Server is the MCP server for code indexing and search.
type Server struct {
	mcpServer *server.MCPServer
	indexer   *Indexer
}

// NewServer creates a new Code Index MCP server.
func NewServer(indexer *Indexer) *Server {
	s := &Server{
		indexer: indexer,
	}

	s.mcpServer = server.NewMCPServer(
		serverName,
		serverVersion,
		server.WithToolCapabilities(false),
	)

	s.registerTools()
	return s
}

// MCPServer returns the underlying MCP server for serving.
func (s *Server) MCPServer() *server.MCPServer {
	return s.mcpServer
}

func (s *Server) registerTools() {
	// index_directory
	s.mcpServer.AddTool(
		mcp.NewTool("index_directory",
			mcp.WithDescription("Index all code files in a directory recursively. Creates embeddings using local Ollama."),
			mcp.WithString("path", mcp.Required(), mcp.Description("Path to directory to index")),
		),
		s.handleIndexDirectory,
	)

	// search_code
	s.mcpServer.AddTool(
		mcp.NewTool("search_code",
			mcp.WithDescription("Search indexed code by semantic similarity. Returns most relevant code chunks with filtering."),
			mcp.WithString("query", mcp.Required(), mcp.Description("Natural language query describing what code to find")),
			mcp.WithNumber("top_k", mcp.Description("Number of results to return (default: 5)")),
			mcp.WithNumber("min_similarity", mcp.Description("Minimum similarity threshold 0.0-1.0 (default: 0.3). Results below this are filtered out.")),
			mcp.WithBoolean("use_rerank", mcp.Description("Use LLM reranking for better relevance (slower but more accurate). Requires qwen2.5:1.5b model.")),
		),
		s.handleSearchCode,
	)

	// index_stats
	s.mcpServer.AddTool(
		mcp.NewTool("index_stats",
			mcp.WithDescription("Get statistics about the code index (number of chunks, files, model used)"),
		),
		s.handleIndexStats,
	)

	// check_health
	s.mcpServer.AddTool(
		mcp.NewTool("check_health",
			mcp.WithDescription("Check if Ollama is running and the embedding model is available"),
		),
		s.handleCheckHealth,
	)

	// reload_index
	s.mcpServer.AddTool(
		mcp.NewTool("reload_index",
			mcp.WithDescription("Reload the index from disk (useful after manual edits or external updates)"),
		),
		s.handleReloadIndex,
	)
}

func (s *Server) handleIndexDirectory(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path := req.GetString("path", "")
	if path == "" {
		return mcp.NewToolResultError("path is required"), nil
	}

	// Channel for progress messages
	progressMsg := ""
	progress := func(msg string) {
		progressMsg = msg
	}

	err := s.indexer.IndexDirectory(ctx, path, progress)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to index directory: %v", err)), nil
	}

	stats := s.indexer.Stats()
	result := map[string]interface{}{
		"success":      true,
		"message":      fmt.Sprintf("Successfully indexed directory: %s", path),
		"stats":        stats,
		"last_message": progressMsg,
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}

func (s *Server) handleSearchCode(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query := req.GetString("query", "")
	if query == "" {
		return mcp.NewToolResultError("query is required"), nil
	}

	topK := req.GetInt("top_k", 5)
	if topK <= 0 {
		topK = 5
	}

	minSimilarity := req.GetFloat("min_similarity", 0.3)
	if minSimilarity < 0 {
		minSimilarity = 0
	}
	if minSimilarity > 1 {
		minSimilarity = 1
	}

	useRerank := req.GetBool("use_rerank", false)

	// Get more results initially for filtering
	searchK := topK * 3
	if searchK < 15 {
		searchK = 15
	}

	results, err := s.indexer.Search(ctx, query, searchK)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("search failed: %v", err)), nil
	}

	// Apply reranking/filtering
	rerankerCfg := RerankerConfig{
		MinSimilarity:    minSimilarity,
		UseLLMRerank:     useRerank,
		MaxResultsForLLM: 10,
	}
	reranker := NewReranker(rerankerCfg, s.indexer.ollama)

	reranked, stats := reranker.Rerank(ctx, query, results)

	// Limit to requested top_k after reranking
	if len(reranked) > topK {
		reranked = reranked[:topK]
	}

	formatted := FormatRerankedResults(reranked, stats)
	return mcp.NewToolResultText(formatted), nil
}

func (s *Server) handleIndexStats(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	stats := s.indexer.Stats()
	output, _ := json.MarshalIndent(stats, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}

func (s *Server) handleCheckHealth(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	err := s.indexer.CheckHealth(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("health check failed: %v", err)), nil
	}

	return mcp.NewToolResultText("Ollama is healthy and embedding model is available"), nil
}

func (s *Server) handleReloadIndex(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	err := s.indexer.LoadIndex()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to reload index: %v", err)), nil
	}

	stats := s.indexer.Stats()
	result := map[string]interface{}{
		"success": true,
		"message": "Index reloaded successfully",
		"stats":   stats,
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}
