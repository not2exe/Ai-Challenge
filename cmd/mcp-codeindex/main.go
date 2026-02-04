// Command mcp-codeindex provides an MCP server for code indexing and search.
//
// This server provides tools for indexing code repositories using local Ollama
// embeddings and searching through the indexed code semantically.
//
// Usage:
//
//	./mcp-codeindex          # Start MCP server (stdio)
//	./mcp-codeindex --help   # Show help
//
// Environment:
//
//	OLLAMA_URL         Ollama API URL (default: http://localhost:11434)
//	OLLAMA_MODEL       Embedding model name (default: nomic-embed-text)
//
// Index storage:
//
//	Each project stores its index in PROJECT_ROOT/.codeindex/index.json
//	The server automatically finds the nearest .codeindex/ when searching.
//
// Before using:
//
//	1. Install Ollama: https://ollama.ai
//	2. Pull embedding model: ollama pull nomic-embed-text
//	3. Ensure Ollama is running: ollama serve
package main

import (
	"fmt"
	"os"

	"github.com/mark3labs/mcp-go/server"
	"github.com/notexe/cli-chat/internal/codeindex"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--help", "-h":
			printHelp()
			return
		}
	}

	ollamaURL := os.Getenv("OLLAMA_URL")
	if ollamaURL == "" {
		ollamaURL = "http://localhost:11434"
	}

	ollamaModel := os.Getenv("OLLAMA_MODEL")
	if ollamaModel == "" {
		ollamaModel = "nomic-embed-text"
	}

	// Create indexer
	indexer, err := codeindex.NewIndexer(codeindex.IndexerConfig{
		OllamaURL:   ollamaURL,
		ModelName:   ollamaModel,
		ChunkConfig: codeindex.DefaultChunkConfig(),
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create indexer: %v\n", err)
		os.Exit(1)
	}

	// Create MCP server
	s := codeindex.NewServer(indexer)

	// Serve via stdio
	if err := server.ServeStdio(s.MCPServer()); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Println(`MCP Code Index Server - Semantic code search via MCP protocol

DESCRIPTION:
    Index code repositories using local Ollama embeddings and search through
    them semantically. The indexer splits code into chunks, generates embeddings,
    and stores them in a JSON index for fast similarity search.

    Each project stores its index in PROJECT_ROOT/.codeindex/index.json
    This allows multiple projects to have independent indices.

USAGE:
    mcp-codeindex          Start MCP server (communicates via stdio)
    mcp-codeindex --help   Show this help

ENVIRONMENT:
    OLLAMA_URL       Ollama API endpoint
                     Default: http://localhost:11434

    OLLAMA_MODEL     Embedding model to use
                     Default: nomic-embed-text
                     Other options: all-minilm, mxbai-embed-large

INDEX STORAGE:
    Index is stored in .codeindex/index.json inside the indexed directory.
    When searching, the server looks for .codeindex/ starting from current
    directory and going up (similar to how git finds .git/).

    Example: If you index /projects/myapp, the index is saved to
             /projects/myapp/.codeindex/index.json

PREREQUISITES:
    1. Install Ollama:
       Visit https://ollama.ai and follow installation instructions

    2. Pull an embedding model:
       ollama pull nomic-embed-text

    3. Start Ollama (if not running):
       ollama serve

TOOLS:
    index_directory  Index all code files in a directory recursively.
                     Creates .codeindex/ in the target directory.
                     Parameters: path (required)

    search_code      Search indexed code by semantic similarity.
                     Automatically finds .codeindex/ from current directory.
                     Parameters: query (required), top_k (optional, default: 5)

    index_stats      Get statistics about the current index
                     (number of chunks, files, model used, index path)

    check_health     Verify Ollama connectivity and model availability

    reload_index     Reload the index from disk

SUPPORTED FILE TYPES:
    .go, .js, .ts, .jsx, .tsx, .py, .java, .c, .cpp, .h, .hpp, .rs,
    .rb, .php, .cs, .swift, .kt, .scala, .sh, .bash, .sql, .proto,
    .thrift, .graphql, .yaml, .yml, .json, .xml, .md

CONFIGURATION:
    Add to ~/.cli-chat/mcp.json:

    {
      "mcpServers": {
        "codeindex": {
          "command": "/path/to/mcp-codeindex",
          "args": [],
          "env": {
            "OLLAMA_MODEL": "nomic-embed-text"
          }
        }
      }
    }

EXAMPLE USAGE:
    1. Index current project:
       Use tool: index_directory with path="."

    2. Search for authentication code:
       Use tool: search_code with query="user authentication and login"

    3. Find error handling:
       Use tool: search_code with query="error handling and retries"

GITIGNORE:
    Add .codeindex/ to your .gitignore to avoid committing the index:
    echo ".codeindex/" >> .gitignore

TIPS:
    - Use specific queries for better results: "JWT token validation"
      vs "authentication"
    - Increase top_k to see more results
    - Re-index periodically as code changes
    - Keep Ollama running for best performance

MORE INFO:
    - Ollama: https://ollama.ai
    - MCP Protocol: https://modelcontextprotocol.io
    - Project: https://github.com/notexe/cli-chat`)
}
