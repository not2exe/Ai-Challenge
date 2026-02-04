# Code Indexing Quick Start Guide

Get started with semantic code search in 5 minutes.

## Prerequisites

1. **Ollama installed and running**
   ```bash
   # Check if Ollama is installed
   ollama --version

   # If not installed, visit: https://ollama.ai
   ```

2. **Embedding model downloaded**
   ```bash
   ollama pull nomic-embed-text
   ```

3. **Ollama server running**
   ```bash
   ollama serve
   # Or it may already be running as a service
   ```

## Step 1: Build the MCP Server

```bash
cd /Users/notexe/challengeAiFirst
go build -o mcp-codeindex ./cmd/mcp-codeindex
```

This creates the `mcp-codeindex` binary.

## Step 2: Configure Your Chat Bot

Add the code index server to your MCP configuration:

### Option A: Add to `~/.cli-chat/config.yaml`

```yaml
mcp:
  enabled: true
  servers:
    - name: codeindex
      command: /Users/notexe/challengeAiFirst/mcp-codeindex
      args: []
      env:
        - OLLAMA_URL=http://localhost:11434
        - OLLAMA_MODEL=nomic-embed-text
```

### Option B: Add to `~/.cli-chat/mcp.json`

```json
{
  "mcpServers": {
    "codeindex": {
      "command": "/Users/notexe/challengeAiFirst/mcp-codeindex",
      "args": [],
      "env": {
        "OLLAMA_URL": "http://localhost:11434",
        "OLLAMA_MODEL": "nomic-embed-text"
      }
    }
  }
}
```

## Step 3: Start Your Chat Bot

```bash
./chat
```

You should see:
```
Connecting to MCP server: codeindex...
Connected to codeindex (5 tools)
```

## Step 4: Index Your Code

In the chat, ask the bot to index your project:

```
> Index the current project directory
```

The bot will use the `index_directory` tool and show progress:
```
Indexing: internal/api/client.go
Indexing: internal/chat/session.go
...
Successfully indexed 150 files (1,243 chunks)
Index saved to ~/.cli-chat/code_index.json
```

## Step 5: Search Your Code

Now you can search semantically:

```
> Find code that handles API requests and retries

AI: I found several relevant code chunks:

Result 1 (similarity: 0.87):
File: internal/api/client.go (lines 45-72)
```go
func (c *Client) SendWithRetry(ctx context.Context, req *Request) (*Response, error) {
    var lastErr error
    for attempt := 0; attempt < c.maxRetries; attempt++ {
        resp, err := c.send(ctx, req)
        if err == nil {
            return resp, nil
        }
        lastErr = err
        // Exponential backoff
        time.Sleep(time.Duration(math.Pow(2, float64(attempt))) * time.Second)
    }
    return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}
```

Result 2 (similarity: 0.82):
...
```

## Common Use Cases

### 1. Understanding Authentication

```
> Show me how authentication works in this codebase
```

### 2. Finding Error Handling

```
> Where do we handle database errors?
```

### 3. Locating Configuration

```
> Find configuration loading code
```

### 4. Discovering APIs

```
> What HTTP endpoints do we expose?
```

### 5. Understanding Data Models

```
> Show me the user data model
```

## Troubleshooting

### "Connection refused" Error

**Problem**: Ollama is not running

**Solution**:
```bash
ollama serve
```

### "Model not found" Error

**Problem**: Embedding model not downloaded

**Solution**:
```bash
ollama pull nomic-embed-text
```

### No Results Found

**Problem**: Index is empty

**Solution**: Index your project first:
```
> Index the current directory
```

### MCP Server Not Connected

**Problem**: Configuration incorrect or binary not found

**Solution**:
1. Verify binary path in config
2. Check binary is executable: `chmod +x ./mcp-codeindex`
3. Restart chat bot

## Next Steps

- Read the [full documentation](CODE_INDEX.md)
- Try different [embedding models](CODE_INDEX.md#custom-embedding-models)
- Set up [multiple indices](CODE_INDEX.md#multiple-indices) for different projects
- Learn about [performance tuning](CODE_INDEX.md#performance)

## Tips for Better Results

1. **Be specific**: "JWT token validation" works better than "authentication"
2. **Use technical terms**: The model understands code terminology
3. **Ask for context**: "Show me how X connects to Y"
4. **Iterate**: Refine your query based on initial results
5. **Increase results**: Ask for "top 10 results" for broader searches

## Example Session

```
$ ./chat

CLI Chat v1.0.0
Connected to codeindex (5 tools)
Type /help for commands

> Index the current project

Indexing /Users/notexe/challengeAiFirst...
Indexed 150 files, 1,243 chunks
Done!

> Find code that parses configuration files

[AI searches and finds config.go, showing relevant chunks with YAML/JSON parsing]

> How do we handle MCP server connections?

[AI finds mcp/manager.go and shows connection handling logic]

> Thanks! Now find all error wrapping code

[AI finds error handling patterns across the codebase]
```

Happy coding! 🚀
