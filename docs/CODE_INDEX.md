# Code Indexing and Semantic Search

This document describes the code indexing and semantic search functionality built into the CLI chat application.

## Overview

The code indexing system allows you to:
- **Index** entire codebases by splitting code into chunks
- **Generate embeddings** using local Ollama models
- **Search semantically** through your code using natural language queries
- **Find relevant code** without knowing exact keywords

## Architecture

```
┌─────────────────┐
│   Chat Bot      │
│   (DeepSeek)    │
└────────┬────────┘
         │
         │ MCP Protocol
         │
┌────────▼────────┐
│  mcp-codeindex  │  ◄── MCP Server (stdio)
│     Server      │
└────────┬────────┘
         │
┌────────▼─────────────────────────────────┐
│           internal/codeindex             │
├──────────────────────────────────────────┤
│  Chunker    │  Indexer    │  Search     │
│  ├─ Split   │  ├─ Scan    │  ├─ Embed  │
│  ├─ Clean   │  ├─ Chunk   │  ├─ Similarity│
│  └─ Filter  │  └─ Store   │  └─ Rank    │
└──────────────┬───────────────────────────┘
               │
        ┌──────▼──────┐
        │   Ollama    │  ◄── Local embedding model
        │  (nomic-    │
        │   embed)    │
        └─────────────┘
```

## Components

### 1. Chunker (`chunker.go`)

Splits code files into manageable chunks:
- **Smart splitting**: Respects code structure (functions, classes)
- **Overlap**: Maintains context between chunks (default: 200 chars)
- **Max size**: Configurable (default: 1000 chars ≈ 200-250 tokens)
- **File filtering**: Only indexes code files (configurable extensions)

### 2. Ollama Client (`ollama.go`)

Communicates with local Ollama instance:
- **Embedding generation**: Converts text to vectors
- **Batch processing**: Handles multiple chunks efficiently
- **Health checks**: Verifies Ollama availability
- **Default model**: `nomic-embed-text` (768 dimensions)

### 3. Index (`index.go`)

Manages the searchable code index:
- **Storage**: JSON format (easy to inspect and version control)
- **Search**: Cosine similarity for semantic matching
- **Metadata**: File paths, line numbers, chunk indices
- **Statistics**: Track indexed files and chunks

### 4. Indexer (`indexer.go`)

Orchestrates the indexing process:
- **Directory scanning**: Recursive file traversal
- **Progress tracking**: Real-time feedback during indexing
- **Error handling**: Graceful handling of unreadable files
- **Incremental updates**: Can re-index individual files

### 5. MCP Server (`server.go`, `cmd/mcp-codeindex/main.go`)

Exposes functionality via MCP protocol:
- **Tools**: 5 tools for indexing and searching
- **Stdio communication**: Works with MCP managers
- **Configuration**: Environment variables for flexibility

## Installation

### Prerequisites

1. **Install Ollama**:
   ```bash
   # macOS
   brew install ollama

   # Or download from https://ollama.ai
   ```

2. **Pull embedding model**:
   ```bash
   ollama pull nomic-embed-text
   ```

3. **Start Ollama**:
   ```bash
   ollama serve
   ```

### Build the MCP Server

```bash
cd /Users/notexe/challengeAiFirst
go build -o mcp-codeindex ./cmd/mcp-codeindex
```

### Configure MCP

Add to `~/.cli-chat/config.yaml`:

```yaml
mcp:
  enabled: true
  servers:
    - name: codeindex
      command: /path/to/mcp-codeindex
      args: []
      env:
        - OLLAMA_URL=http://localhost:11434
        - OLLAMA_MODEL=nomic-embed-text
        - CODE_INDEX_PATH=/path/to/.cli-chat/code_index.json
```

## Usage

### 1. Index a Directory

```
> Use the index_directory tool to index the current project
AI: Indexing /path/to/project...
    Found 150 files, created 1,243 chunks
    Index saved to ~/.cli-chat/code_index.json
```

### 2. Search for Code

```
> Find code that handles user authentication
AI: [uses search_code tool]

    Found 5 result(s):

    Result 1 (similarity: 0.892):
    File: internal/auth/handler.go (lines 45-72)
    ```
    func (h *Handler) Login(ctx context.Context, req LoginRequest) (*User, error) {
        // Validate credentials
        user, err := h.store.FindByEmail(req.Email)
        if err != nil {
            return nil, fmt.Errorf("user not found: %w", err)
        }

        // Check password
        if !h.hasher.Compare(req.Password, user.PasswordHash) {
            return nil, ErrInvalidCredentials
        }

        // Generate session token
        token, err := h.tokenGen.Generate(user.ID)
        ...
    }
    ```
```

### 3. Check Index Stats

```
> How many files are indexed?
AI: [uses index_stats tool]

    {
      "total_chunks": 1243,
      "total_files": 150,
      "model": "nomic-embed-text"
    }
```

### 4. Verify Ollama Health

```
> Is Ollama working?
AI: [uses check_health tool]
    Ollama is healthy and embedding model is available
```

## Available Tools

### `index_directory`

Index all code files in a directory recursively.

**Parameters:**
- `path` (required): Path to directory to index

**Example:**
```json
{
  "path": "/Users/notexe/challengeAiFirst"
}
```

### `search_code`

Search indexed code by semantic similarity.

**Parameters:**
- `query` (required): Natural language query
- `top_k` (optional): Number of results (default: 5)

**Example:**
```json
{
  "query": "error handling with retries",
  "top_k": 10
}
```

### `index_stats`

Get statistics about the code index.

**Parameters:** None

**Returns:**
```json
{
  "total_chunks": 1243,
  "total_files": 150,
  "model": "nomic-embed-text",
  "index_path": "~/.cli-chat/code_index.json"
}
```

### `check_health`

Check if Ollama is running and model is available.

**Parameters:** None

**Returns:** Health status message

### `reload_index`

Reload the index from disk.

**Parameters:** None

**Use case:** After manual edits or external index updates

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `CODE_INDEX_PATH` | `~/.cli-chat/code_index.json` | Index file location |
| `OLLAMA_URL` | `http://localhost:11434` | Ollama API endpoint |
| `OLLAMA_MODEL` | `nomic-embed-text` | Embedding model name |

### Chunk Configuration

Default chunk settings:
- **Max chunk size**: 1000 characters (~200-250 tokens)
- **Overlap**: 200 characters (20%)

To customize, modify `DefaultChunkConfig()` in `chunker.go`.

### Supported File Types

Code files:
- Go: `.go`
- JavaScript/TypeScript: `.js`, `.ts`, `.jsx`, `.tsx`
- Python: `.py`
- Java: `.java`
- C/C++: `.c`, `.cpp`, `.h`, `.hpp`
- Rust: `.rs`
- Ruby: `.rb`
- PHP: `.php`
- C#: `.cs`
- Swift: `.swift`
- Kotlin: `.kt`
- Scala: `.scala`
- Shell: `.sh`, `.bash`
- SQL: `.sql`
- Proto/Thrift: `.proto`, `.thrift`
- GraphQL: `.graphql`
- Config: `.yaml`, `.yml`, `.json`, `.xml`
- Docs: `.md`

To add more, edit `ShouldIndexFile()` in `chunker.go`.

## Performance

### Indexing Speed

- **Chunking**: ~1000 files/second
- **Embedding**: ~10-50 chunks/second (depends on Ollama)
- **Total**: ~1-5 minutes for medium projects (100-500 files)

### Search Speed

- **Query embedding**: ~100ms
- **Similarity calculation**: ~1ms for 10,000 chunks
- **Total**: < 200ms for typical queries

### Index Size

- **Per file**: ~10-20KB (depends on file size)
- **100 files**: ~1-2MB
- **1000 files**: ~10-20MB

## Best Practices

### Indexing

1. **Index periodically**: Re-index after major changes
2. **Exclude directories**: The indexer already skips `.git`, `node_modules`, `vendor`
3. **Use specific paths**: Index only relevant directories for faster results

### Searching

1. **Be specific**: "JWT token validation" > "authentication"
2. **Use domain terms**: "database migration" > "change schema"
3. **Increase top_k**: For broad searches, use `top_k: 10-20`
4. **Iterate queries**: Refine based on initial results

### Performance

1. **Keep Ollama running**: Avoids startup overhead
2. **Use fast models**: `nomic-embed-text` is a good balance
3. **Index incrementally**: For large projects, index directories separately
4. **Monitor memory**: Large indices (10,000+ chunks) use ~100-500MB RAM

## Troubleshooting

### Ollama Not Running

```
Error: health check failed: connection refused
```

**Solution:**
```bash
ollama serve
```

### Model Not Found

```
Error: model 'nomic-embed-text' not found
```

**Solution:**
```bash
ollama pull nomic-embed-text
```

### Index File Locked

```
Error: failed to save index: file is locked
```

**Solution:** Close other processes using the index file

### Poor Search Results

**Solutions:**
1. Use more specific queries
2. Increase `top_k` parameter
3. Re-index if code has changed significantly
4. Try different phrasing

### Out of Memory

```
Error: cannot allocate memory
```

**Solutions:**
1. Index directories separately
2. Increase chunk size to reduce chunk count
3. Use a machine with more RAM

## Advanced Usage

### Custom Embedding Models

```yaml
mcp:
  servers:
    - name: codeindex
      env:
        - OLLAMA_MODEL=mxbai-embed-large  # Higher quality, slower
        # Or: all-minilm (faster, lower quality)
```

### Multiple Indices

Create separate indices for different projects:

```yaml
mcp:
  servers:
    - name: codeindex-project1
      command: /path/to/mcp-codeindex
      env:
        - CODE_INDEX_PATH=~/.cli-chat/project1_index.json

    - name: codeindex-project2
      command: /path/to/mcp-codeindex
      env:
        - CODE_INDEX_PATH=~/.cli-chat/project2_index.json
```

### Programmatic Access

```go
import "github.com/notexe/cli-chat/internal/codeindex"

// Create indexer
indexer, _ := codeindex.NewIndexer(codeindex.IndexerConfig{
    OllamaURL:   "http://localhost:11434",
    ModelName:   "nomic-embed-text",
    IndexPath:   "my_index.json",
    ChunkConfig: codeindex.DefaultChunkConfig(),
})

// Index directory
indexer.IndexDirectory(ctx, "/path/to/code", nil)

// Search
results, _ := indexer.Search(ctx, "authentication", 5)
```

## Future Enhancements

Potential improvements:
- **FAISS integration**: Faster search for large codebases
- **Incremental indexing**: Only re-index changed files
- **Metadata filtering**: Search by file type, date, author
- **Hybrid search**: Combine semantic + keyword search
- **Multi-model support**: Use different models for different languages
- **Streaming results**: Display results as they're found
- **Index compression**: Reduce storage size

## References

- [Ollama Documentation](https://ollama.ai)
- [MCP Protocol](https://modelcontextprotocol.io)
- [Nomic Embed Text Model](https://huggingface.co/nomic-ai/nomic-embed-text-v1)
- [Cosine Similarity](https://en.wikipedia.org/wiki/Cosine_similarity)
