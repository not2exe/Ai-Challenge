# CLI Chat - Interactive DeepSeek CLI Tool
A Go-based command-line interface for chatting with DeepSeek AI models. Features an interactive REPL with multi-turn conversation support, configurable system prompts, and beautiful terminal output.

[DEMO FIRST DAY](https://drive.google.com/file/d/1n2vILKWM8VqzTluYd3TqC7oW8xTh4BEb/view?usp=sharing)

[DEMO SECOND DAY](https://drive.google.com/file/d/1c2NqfliFL_v3-xcokEe6dUyx85BlXA6N/view?usp=sharing)

[DEMO THIRD DAY](https://drive.google.com/file/d/1sx90AYNCWIWlzOz2raiCf8j6NNOvMxzg/view?usp=sharing)

[DEMO SIXTH DAY](https://drive.google.com/file/d/1uyrCISu8hLML0qZ-_yNdrBgQPmvhHeuh/view?usp=drive_link)

[DEMO SEVENTH DAY](https://drive.google.com/file/d/1afYm9NUjT5XHEOYx-AiCsEknv-EMkRZt/view?usp=sharing)

## Features

- **Interactive REPL**: Continuous chat sessions with DeepSeek AI
- **Multi-turn Conversations**: DeepSeek remembers context throughout your session
- **System Prompts**: Customize DeepSeek's behavior and personality
- **Status Messages**: Visual feedback during API requests
- **Configuration Management**: YAML config file with environment variable support
- **Colored Output**: Beautiful terminal formatting with customizable colors
- **Token Usage Tracking**: Monitor input/output token consumption
- **Special Commands**: Built-in commands for session management

## Requirements

- Go 1.22 or higher
- DeepSeek API key ([get one here](https://platform.deepseek.com/api_keys))

## Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/notexe/cli-chat.git
cd cli-chat

# Build the binary
go build -o chat ./cmd/chat

# Optionally, install to $GOPATH/bin
go install ./cmd/chat
```

### Quick Start

```bash
# Set your API key
export DEEPSEEK_API_KEY="your-api-key-here"

# Run the chat
./chat
```

## Configuration

### Environment Variable (Simplest)

```bash
export DEEPSEEK_API_KEY="your-api-key-here"
./chat
```

### Config File (Recommended)

1. Create the config directory:
```bash
mkdir -p ~/.cli-chat
```

2. Copy the example config:
```bash
cp config.example.yaml ~/.cli-chat/config.yaml
```

3. Edit the config file and add your API key:
```bash
vim ~/.cli-chat/config.yaml
```

Example configuration:
```yaml
api:
  key: "your-deepseek-api-key"
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

### Command-Line Flags

Override config settings with command-line flags:

```bash
# Use a specific config file
./chat --config /path/to/config.yaml

# Override model
./chat --model deepseek-reasoner

# Set system prompt
./chat --system-prompt "You are a Go programming expert."

# Disable colored output
./chat --no-color
```

### Configuration Precedence

Settings are loaded in this order (later overrides earlier):
1. Default values
2. Config file (`~/.cli-chat/config.yaml`)
3. Environment variables (`DEEPSEEK_API_KEY`)
4. Command-line flags

## Usage

### Starting a Chat

```bash
$ ./chat

Welcome to CLI Chat with DeepSeek!
Model: deepseek-chat
Type /help for available commands or start chatting.

You:
```

### Interactive Commands

| Command | Description |
|---------|-------------|
| `/help` or `/h` | Show available commands |
| `/clear` or `/c` | Clear conversation history |
| `/system <prompt>` or `/s <prompt>` | Update system prompt |
| `/show` | Display current system prompt |
| `/count` | Show message count in current session |
| `/quit` or `/exit` or `/q` | Exit the chat |

### Example Session

```
You: What is a goroutine in Go?

Waiting for response...

DeepSeek: A goroutine is a lightweight thread managed by the Go runtime. It's one of Go's
core features for concurrent programming. Goroutines are functions or methods that run
concurrently with other goroutines in the same address space.

(tokens: input=8, output=67)

You: Can you show me an example?

DeepSeek: Here's a simple example of using goroutines:

```go
package main

import (
    "fmt"
    "time"
)

func printNumbers() {
    for i := 1; i <= 5; i++ {
        fmt.Printf("%d ", i)
        time.Sleep(100 * time.Millisecond)
    }
}

func main() {
    // Launch goroutine
    go printNumbers()

    // Do other work
    fmt.Println("Main function")

    // Wait for goroutine to finish
    time.Sleep(600 * time.Millisecond)
}
```

You: /system You are a Go expert. Be concise and focus on best practices.

System prompt updated.

You: What's the best way to handle errors in Go?

DeepSeek: Best practices for error handling in Go:

1. Always check errors: `if err != nil { return err }`
2. Add context: Use `fmt.Errorf("operation failed: %w", err)`
3. Handle early: Check errors immediately, don't defer
4. Return errors up: Let callers decide how to handle
5. Use custom error types for specific cases
6. Avoid panic except for unrecoverable situations

Key pattern:
```go
if err != nil {
    return fmt.Errorf("context: %w", err)
}
```

You: /quit

Goodbye!
```

## Project Structure

```
cli-chat/
├── cmd/chat/
│   └── main.go              # Entry point
├── internal/
│   ├── api/
│   │   ├── client.go        # DeepSeek API client
│   │   └── types.go         # API types
│   ├── chat/
│   │   ├── session.go       # Session management
│   │   ├── history.go       # Message history
│   │   └── prompt.go        # System prompts
│   ├── config/
│   │   ├── config.go        # Configuration
│   │   └── defaults.go      # Default values
│   ├── repl/
│   │   ├── repl.go          # REPL loop
│   │   ├── input.go         # Input handling
│   │   └── output.go        # Output formatting
│   └── ui/
│       ├── formatter.go     # Terminal formatting
│       └── status.go        # Status messages
├── config.example.yaml      # Example configuration
├── go.mod
├── go.sum
└── README.md
```

## How It Works

### Multi-turn Conversations

The chat maintains conversation context by keeping a history of all messages. Each API request includes the full conversation history (up to the configured `max_history` limit), allowing DeepSeek to understand context from earlier in the conversation.

When the history exceeds the maximum, older messages are automatically truncated, keeping only the most recent messages.

### System Prompts

System prompts define DeepSeek's behavior and personality. You can:
- Set a default in the config file
- Override via CLI flag at startup
- Change mid-conversation with `/system` command

Example use cases:
- "You are a Python expert. Provide code examples."
- "You are a creative writer. Be poetic and expressive."
- "You are a technical interviewer. Ask challenging questions."

### Status Messages

While waiting for DeepSeek's response, the CLI displays status messages:
- "Waiting for response..." - API request in progress
- Token usage after each response (if enabled)
- Error messages with helpful context

## Development

### Building

```bash
# Build for current platform
go build -o chat ./cmd/chat

# Build for specific platforms
GOOS=linux GOARCH=amd64 go build -o chat-linux ./cmd/chat
GOOS=darwin GOARCH=arm64 go build -o chat-macos ./cmd/chat
GOOS=windows GOARCH=amd64 go build -o chat.exe ./cmd/chat
```

### Testing

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package tests
go test ./internal/chat/...
```

### Dependencies

Key dependencies:
- [go-deepseek/deepseek](https://github.com/go-deepseek/deepseek) - DeepSeek Go client
- [koanf](https://github.com/knadh/koanf) - Configuration management
- [lipgloss](https://github.com/charmbracelet/lipgloss) - Terminal styling
- [readline](https://github.com/chzyer/readline) - REPL input handling

## Troubleshooting

### API Key Issues

**Error:** "API key is required"
- Set `DEEPSEEK_API_KEY` environment variable
- Or add `api.key` to `~/.cli-chat/config.yaml`

### Configuration Not Loading

- Check file exists: `ls ~/.cli-chat/config.yaml`
- Verify YAML syntax: Invalid YAML will cause errors
- Use `--config` flag to specify custom location

### Network Timeouts

- Increase timeout in config: `api.timeout: 180`
- Check internet connection
- Verify API endpoint is reachable

### Colors Not Displaying

- Use `--no-color` flag to disable colors
- Or set `ui.colored_output: false` in config
- Some terminals don't support ANSI colors

## Models

DeepSeek offers two main models:

- **deepseek-chat**: Fast, general-purpose model (DeepSeek-V3.2 non-thinking mode)
- **deepseek-reasoner**: Reasoning mode, better for complex problems and detailed analysis

Switch models using the `--model` flag or in config:
```bash
./chat --model deepseek-reasoner
```

## Future Enhancements

Planned features:
- [ ] Streaming responses (display tokens as they arrive)
- [ ] Session save/load (name and resume conversations)
- [ ] Markdown rendering (better code block formatting)
- [ ] Multiline input support
- [ ] Command autocomplete
- [ ] Export conversations (to Markdown/JSON)
- [ ] Cost tracking (estimate API costs)
- [ ] Model switching mid-conversation

## Contributing

Contributions are welcome! Please:
1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## License

MIT License - see LICENSE file for details

## Acknowledgments

- Built with [DeepSeek AI](https://www.deepseek.com/)
- Terminal UI powered by [Charm](https://charm.sh/) libraries

## Support

- Report issues: [GitHub Issues](https://github.com/notexe/cli-chat/issues)
- API docs: [DeepSeek API Documentation](https://api-docs.deepseek.com/)
- Get API key: [DeepSeek Platform](https://platform.deepseek.com/api_keys)
