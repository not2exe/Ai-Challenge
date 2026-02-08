package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/go-deepseek/deepseek/request"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

// ServerConfig defines MCP server configuration.
type ServerConfig struct {
	Name    string
	Command string
	Args    []string
	Env     []string
}

// Manager manages multiple MCP server connections.
type Manager struct {
	servers map[string]*serverInstance
	tools   map[string]*toolInfo // tool name -> server that provides it
}

type serverInstance struct {
	name   string
	client *client.Client
	tools  []Tool
}

type toolInfo struct {
	serverName string
	tool       Tool
}

// NewManager creates a new MCP manager.
func NewManager() *Manager {
	return &Manager{
		servers: make(map[string]*serverInstance),
		tools:   make(map[string]*toolInfo),
	}
}

// AddServer connects to an MCP server and registers its tools.
func (m *Manager) AddServer(ctx context.Context, cfg ServerConfig) error {
	// Verify command exists before spawning to avoid mcp-go nil reader panic
	if _, err := exec.LookPath(cfg.Command); err != nil {
		return fmt.Errorf("MCP server command not found for %s: %w", cfg.Name, err)
	}

	// Build environment
	env := os.Environ()
	for _, e := range cfg.Env {
		env = append(env, e)
	}

	// Create client
	c, err := client.NewStdioMCPClient(cfg.Command, env, cfg.Args...)
	if err != nil {
		return fmt.Errorf("failed to create MCP client for %s: %w", cfg.Name, err)
	}

	// Initialize
	initReq := mcp.InitializeRequest{}
	initReq.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initReq.Params.ClientInfo = mcp.Implementation{
		Name:    "cli-chat",
		Version: "1.0.0",
	}

	_, err = c.Initialize(ctx, initReq)
	if err != nil {
		c.Close()
		return fmt.Errorf("failed to initialize MCP server %s: %w", cfg.Name, err)
	}

	// Get tools
	toolsResult, err := c.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		c.Close()
		return fmt.Errorf("failed to list tools from %s: %w", cfg.Name, err)
	}

	// Convert tools
	tools := make([]Tool, 0, len(toolsResult.Tools))
	for _, t := range toolsResult.Tools {
		tool := Tool{
			Name:        t.Name,
			Description: t.Description,
			InputSchema: t.InputSchema,
		}
		tools = append(tools, tool)

		// Register tool -> server mapping
		m.tools[t.Name] = &toolInfo{
			serverName: cfg.Name,
			tool:       tool,
		}
	}

	m.servers[cfg.Name] = &serverInstance{
		name:   cfg.Name,
		client: c,
		tools:  tools,
	}

	return nil
}

// GetAllTools returns all tools from all connected servers.
func (m *Manager) GetAllTools() []Tool {
	var all []Tool
	for _, srv := range m.servers {
		all = append(all, srv.tools...)
	}
	return all
}

// GetDeepSeekTools returns all tools in DeepSeek format.
func (m *Manager) GetDeepSeekTools() []request.Tool {
	return ToDeepSeekTools(m.GetAllTools())
}

// CallTool calls a tool by name with given arguments.
func (m *Manager) CallTool(ctx context.Context, name string, argsJSON string) (string, error) {
	info, ok := m.tools[name]
	if !ok {
		return "", fmt.Errorf("unknown tool: %s", name)
	}

	srv, ok := m.servers[info.serverName]
	if !ok {
		return "", fmt.Errorf("server not found for tool %s", name)
	}

	// Parse arguments
	var args map[string]interface{}
	if argsJSON != "" && argsJSON != "{}" {
		if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
			return "", fmt.Errorf("failed to parse tool arguments: %w", err)
		}
	}

	// Call tool
	req := mcp.CallToolRequest{}
	req.Params.Name = name
	req.Params.Arguments = args

	result, err := srv.client.CallTool(ctx, req)
	if err != nil {
		return "", fmt.Errorf("tool call failed: %w", err)
	}

	// Extract result
	var parts []string
	for _, content := range result.Content {
		if tc, ok := content.(mcp.TextContent); ok {
			parts = append(parts, tc.Text)
		}
	}

	return strings.Join(parts, "\n"), nil
}

// Close closes all server connections.
func (m *Manager) Close() error {
	var errs []string
	for name, srv := range m.servers {
		if err := srv.client.Close(); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", name, err))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("errors closing MCP servers: %s", strings.Join(errs, "; "))
	}
	return nil
}

// ListServers returns names of all connected servers.
func (m *Manager) ListServers() []string {
	names := make([]string, 0, len(m.servers))
	for name := range m.servers {
		names = append(names, name)
	}
	return names
}

// ServerToolCount returns number of tools per server.
func (m *Manager) ServerToolCount() map[string]int {
	counts := make(map[string]int)
	for name, srv := range m.servers {
		counts[name] = len(srv.tools)
	}
	return counts
}

// HasFilesystemTools checks if filesystem tools (read_text_file, directory_tree, etc.) are available.
func (m *Manager) HasFilesystemTools() bool {
	filesystemTools := []string{"read_text_file", "read_file", "directory_tree", "list_directory", "search_files"}
	for _, toolName := range filesystemTools {
		if _, ok := m.tools[toolName]; ok {
			return true
		}
	}
	return false
}

// HasCodeIndexTools checks if code index tools (semantic_search, index_directory, etc.) are available.
func (m *Manager) HasCodeIndexTools() bool {
	codeIndexTools := []string{"semantic_search", "index_directory", "index_stats"}
	for _, toolName := range codeIndexTools {
		if _, ok := m.tools[toolName]; ok {
			return true
		}
	}
	return false
}
