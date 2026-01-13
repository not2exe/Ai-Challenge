# CLI Chat - DeepSeek Integration

A lightweight, interactive command-line chat interface for DeepSeek AI models built in Go.

## Overview

This CLI utility provides a REPL (Read-Eval-Print Loop) interface for conversing with DeepSeek's AI models directly from your terminal. It maintains conversation context, supports system prompts, and offers a clean, colored terminal UI.

## Quick Start

```bash
# Set your DeepSeek API key
export DEEPSEEK_API_KEY="your-api-key-here"

# Build and run
go build -o chat ./cmd/chat
./chat
```

## Features

- **Interactive REPL**: Continuous conversation with context preservation
- **Multi-turn Conversations**: Automatic history management with configurable limits
- **System Prompts**: Customize AI behavior with custom instructions
- **Terminal UI**: Colored output with lipgloss styling
- **Configuration**: YAML config files, environment variables, and CLI flags
- **Status Messages**: Real-time feedback during API requests
- **Token Tracking**: Display input/output token usage
- **Graceful Shutdown**: Signal handling with history preservation

## Architecture

```
cmd/chat/main.go          → Entry point, initialization, signal handling
internal/api/             → DeepSeek API client wrapper
internal/chat/            → Session and conversation history management
internal/config/          → Configuration loading and validation
internal/repl/            → REPL loop, command handling, I/O
internal/ui/              → Terminal formatting and styling
```

## How It Works

### Execution Flow

1. **Initialization** (main.go)
   - Parse CLI flags
   - Load configuration (defaults → file → env vars → flags)
   - Validate API key
   - Create API client, session, and REPL instance
   - Setup signal handlers (Ctrl+C)

2. **REPL Loop** (repl.go)
   - Display welcome message
   - Read user input via readline
   - Parse commands (starting with `/`) or treat as message
   - For messages: add to history → show status → call API → display response
   - For commands: execute special actions (/help, /clear, /system, etc.)

3. **API Request** (api/client.go)
   - Convert session history to DeepSeek format
   - System prompt becomes first message with role="system"
   - Send chat completion request
   - Extract response content and token usage
   - Return structured response

4. **Session Management** (chat/session.go)
   - Maintain conversation history with auto-truncation
   - Build API requests with current context
   - Support system prompt updates
   - Save/load conversation history

### Key Components

**API Client** (`internal/api/client.go`)
- Wraps go-deepseek/deepseek SDK
- Handles message format conversion
- Temperature type conversion (float64 → *float32)
- Error wrapping with context

**Configuration** (`internal/config/`)
- Three-tier precedence: CLI flags > Env vars > Config file
- DEEPSEEK_API_KEY environment variable support
- Expandable paths (~ → home directory)
- Validation for required fields

**Session** (`internal/chat/session.go`)
- Message history with configurable max size
- System prompt management
- API request building from current state
- JSON serialization for persistence

**REPL** (`internal/repl/repl.go`)
- Readline integration for input handling
- Command parser (/ prefix detection)
- Status display during API calls
- Response formatting and display

**UI Formatter** (`internal/ui/formatter.go`)
- Lipgloss-based terminal styling
- Colored output (can be disabled)
- Consistent formatting for user/assistant/error/system messages
- Token usage display

## Configuration

### Environment Variables

```bash
export DEEPSEEK_API_KEY="your-key"
```

### Config File (`~/.cli-chat/config.yaml`)

```yaml
api:
  key: ""
  base_url: "https://api.deepseek.com"
  timeout: 120

model:
  name: "deepseek-chat"
  max_tokens: 2048
  temperature: 1.0
  system_prompt: "You are a helpful AI assistant."

session:
  max_history: 50
  save_history: false
  history_file: "~/.cli-chat/history.json"

ui:
  show_token_count: true
  colored_output: true
  show_timestamps: false
```

### CLI Flags

```bash
./chat --model deepseek-reasoner --system-prompt "You are a Go expert" --no-color
```

## Available Commands

- `/help` or `/h` - Show help message
- `/clear` or `/c` - Clear conversation history
- `/system <prompt>` or `/s <prompt>` - Update system prompt
- `/show` - Display current system prompt
- `/count` - Show message count in conversation
- `/quit` or `/exit` or `/q` - Exit the chat

## Models

- `deepseek-chat` - Fast, general-purpose (default)
- `deepseek-reasoner` - Reasoning mode for complex problems

## DeepSeek API Integration

### Request Format

```go
&request.ChatCompletionsRequest{
    Model:       "deepseek-chat",
    Messages:    []*request.Message{...},
    MaxTokens:   2048,
    Temperature: &temp,
    Stream:      false,
}
```

### System Prompt Handling

Unlike Claude (separate System field), DeepSeek includes system prompts in the messages array:

```go
messages := []*request.Message{
    {Role: "system", Content: "You are a helpful assistant."},
    {Role: "user", Content: "Hello!"},
    {Role: "assistant", Content: "Hi! How can I help you?"},
}
```

### Response Structure

```go
resp.Choices[0].Message.Content        // AI response text
resp.Choices[0].FinishReason          // Why generation stopped
resp.Usage.PromptTokens               // Input tokens used
resp.Usage.CompletionTokens           // Output tokens generated
```

## Dependencies

- `github.com/go-deepseek/deepseek` - DeepSeek API client
- `github.com/knadh/koanf/v2` - Configuration management
- `github.com/charmbracelet/lipgloss` - Terminal styling
- `github.com/chzyer/readline` - Interactive input handling
- `golang.org/x/term` - Terminal utilities

## Development

### Build

```bash
go build -o chat ./cmd/chat
```

### Run

```bash
./chat
```

### Test Configuration Loading

```bash
./chat --config ./config.example.yaml
```

## Signal Handling

The application handles interrupts gracefully:

- **Ctrl+C** or **SIGTERM**: Save conversation history (if enabled) and exit cleanly
- **Ctrl+D** (EOF): Exit normally

```go
sigChan := make(chan os.Signal, 1)
signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

go func() {
    <-sigChan
    fmt.Println("\nInterrupted. Saving session...")
    cancel()
    replInstance.SaveHistory()
    os.Exit(0)
}()
```

## Error Handling

All errors are wrapped with context using Go's `%w` verb:

```go
if err != nil {
    return fmt.Errorf("API request failed: %w", err)
}
```

This allows error chain inspection and provides clear error messages to users.

## Conversation History

Messages are stored in a circular buffer:

```go
type History struct {
    messages []api.Message
    maxSize  int
}

func (h *History) Add(msg api.Message) {
    h.messages = append(h.messages, msg)
    if len(h.messages) > h.maxSize {
        h.messages = h.messages[len(h.messages)-h.maxSize:]
    }
}
```

When history exceeds `maxSize`, oldest messages are automatically removed to maintain context window.

## Token Usage

DeepSeek returns token counts in each response:

```
(tokens: input=245, output=128)
```

- **Input tokens**: Prompt + conversation history + system prompt
- **Output tokens**: Generated response length
- **Total tokens**: Sum of both (affects API costs)

## Future Enhancements

- **Streaming**: Display tokens as they arrive (SDK supports `StreamChatCompletionsChat()`)
- **Reasoning Display**: Show thinking process for deepseek-reasoner model
- **Cost Tracking**: Estimate API costs based on token usage
- **Multi-session**: Support multiple named conversation sessions
- **Export**: Save conversations in markdown or other formats

## API Documentation

- DeepSeek API Docs: https://api-docs.deepseek.com/
- DeepSeek Platform: https://platform.deepseek.com/
- Go SDK: https://github.com/go-deepseek/deepseek

## License

See LICENSE file for details.
