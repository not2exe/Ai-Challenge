// Package mcp provides MCP (Model Context Protocol) client functionality.
// MCP allows AI applications to connect to external tools and data sources.
package mcp

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

// Tool represents an MCP tool with its metadata
type Tool struct {
	Name        string
	Description string
	InputSchema mcp.ToolInputSchema
}

// Client wraps MCP client functionality
type Client struct {
	mcpClient *client.Client
	connected bool
}

// NewClient creates a new MCP client that connects to a server via stdio.
// command is the path to the MCP server executable.
// args are optional arguments passed to the server.
func NewClient(command string, args ...string) (*Client, error) {
	mcpClient, err := client.NewStdioMCPClient(command, nil, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to create MCP client: %w", err)
	}

	return &Client{
		mcpClient: mcpClient,
		connected: false,
	}, nil
}

// Connect initializes the connection to the MCP server.
// This performs the protocol handshake and capability negotiation.
func (c *Client) Connect(ctx context.Context) error {
	// Initialize the connection with client info
	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "cli-chat",
		Version: "1.0.0",
	}
	initRequest.Params.Capabilities = mcp.ClientCapabilities{}

	_, err := c.mcpClient.Initialize(ctx, initRequest)
	if err != nil {
		return fmt.Errorf("MCP initialization failed: %w", err)
	}

	c.connected = true
	return nil
}

// ListTools returns all available tools from the MCP server.
// This is the main function requested in the task - it gets the list of tools.
func (c *Client) ListTools(ctx context.Context) ([]Tool, error) {
	if !c.connected {
		return nil, fmt.Errorf("not connected to MCP server, call Connect() first")
	}

	// Request tools list from server
	toolsResult, err := c.mcpClient.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		return nil, fmt.Errorf("failed to list tools: %w", err)
	}

	// Convert to our Tool type
	tools := make([]Tool, 0, len(toolsResult.Tools))
	for _, t := range toolsResult.Tools {
		tool := Tool{
			Name:        t.Name,
			Description: t.Description,
			InputSchema: t.InputSchema,
		}
		tools = append(tools, tool)
	}

	return tools, nil
}

// CallTool executes a tool on the MCP server with the given arguments.
func (c *Client) CallTool(ctx context.Context, name string, args map[string]interface{}) (string, error) {
	if !c.connected {
		return "", fmt.Errorf("not connected to MCP server")
	}

	request := mcp.CallToolRequest{}
	request.Params.Name = name
	request.Params.Arguments = args

	result, err := c.mcpClient.CallTool(ctx, request)
	if err != nil {
		return "", fmt.Errorf("tool call failed: %w", err)
	}

	// Extract text content from result
	var output string
	for _, content := range result.Content {
		if textContent, ok := content.(mcp.TextContent); ok {
			output += textContent.Text
		}
	}

	return output, nil
}

// Close closes the connection to the MCP server.
func (c *Client) Close() error {
	if c.mcpClient != nil {
		return c.mcpClient.Close()
	}
	return nil
}
