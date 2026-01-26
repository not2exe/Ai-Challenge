// Command mcp-tools demonstrates MCP client functionality.
// It connects to an MCP server and lists all available tools.
//
// Usage:
//
//	./mcp-tools <server-command> [args...]
//
// Example with GitHub MCP:
//
//	GITHUB_TOKEN=ghp_xxx ./mcp-tools npx -y @modelcontextprotocol/server-github
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/notexe/cli-chat/internal/mcp"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]
	args := os.Args[2:]

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	fmt.Printf("Connecting to MCP server: %s %v\n", command, args)
	fmt.Println()

	// Create MCP client
	client, err := mcp.NewClient(command, args...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating MCP client: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	// Connect and initialize
	if err := client.Connect(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting to MCP server: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Connected! Fetching tools list...")
	fmt.Println()

	// Get list of tools
	tools, err := client.ListTools(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listing tools: %v\n", err)
		os.Exit(1)
	}

	// Display tools
	fmt.Printf("Found %d tool(s):\n", len(tools))
	fmt.Println(strings(50, '='))

	for i, tool := range tools {
		fmt.Printf("\n%d. %s\n", i+1, tool.Name)
		if tool.Description != "" {
			fmt.Printf("   Description: %s\n", tool.Description)
		}
		if len(tool.InputSchema.Properties) > 0 {
			fmt.Printf("   Parameters: %d\n", len(tool.InputSchema.Properties))
			for name := range tool.InputSchema.Properties {
				fmt.Printf("     - %s\n", name)
			}
		}
	}

	fmt.Println()
	fmt.Println("Done!")
}

func printUsage() {
	fmt.Println("MCP Tools Lister")
	fmt.Println("================")
	fmt.Println()
	fmt.Println("Lists all tools available from an MCP server.")
	fmt.Println()
	fmt.Println("Usage: mcp-tools <server-command> [args...]")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println()
	fmt.Println("  # GitHub MCP Server (requires GITHUB_TOKEN)")
	fmt.Println("  GITHUB_TOKEN=ghp_xxx ./mcp-tools npx -y @modelcontextprotocol/server-github")
	fmt.Println()
	fmt.Println("  # Filesystem MCP Server")
	fmt.Println("  ./mcp-tools npx -y @modelcontextprotocol/server-filesystem /tmp")
	fmt.Println()
	fmt.Println("  # Brave Search MCP Server")
	fmt.Println("  BRAVE_API_KEY=xxx ./mcp-tools npx -y @modelcontextprotocol/server-brave-search")
	fmt.Println()
	fmt.Println("Available MCP servers: https://github.com/modelcontextprotocol/servers")
}

func strings(n int, r rune) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte(r)
	}
	return string(b)
}
