# Telegram MCP Server

A Model Context Protocol (MCP) server for Telegram Bot API integration.

## Features

✅ **Send Messages** - Text messages with HTML/Markdown formatting
✅ **Send Photos** - Upload local files or use HTTP URLs
✅ **Send with Keyboards** - Interactive inline keyboard buttons
✅ **Message Management** - Edit and delete messages
✅ **Chat Info** - Get chat and bot information
✅ **Local File Upload** - Automatically uploads local images

## Quick Start

### 1. Build

```bash
go build -o mcp-telegram ./cmd/mcp-telegram
```

### 2. Configure

Set environment variables or update config:

```bash
export TELEGRAM_BOT_TOKEN="your-bot-token"
export TELEGRAM_CHAT_ID="your-chat-id"
```

Or in `mcp.json`:

```json
{
  "mcpServers": {
    "telegram": {
      "command": "./mcp-telegram",
      "args": [],
      "env": {
        "TELEGRAM_BOT_TOKEN": "your-token",
        "TELEGRAM_CHAT_ID": "your-chat-id"
      }
    }
  }
}
```

### 3. Test

```bash
./mcp-telegram --check
```

## Tools

### send_message

Send text messages with optional formatting:

```javascript
{
  "text": "Hello from MCP!",
  "parse_mode": "HTML",  // Optional: HTML, Markdown, MarkdownV2
  "disable_notification": false  // Optional
}
```

### send_photo

**NEW**: Now supports local file uploads!

```javascript
{
  "photo_url": "/Users/you/image.png",  // Local file
  // OR
  "photo_url": "https://example.com/image.jpg",  // URL
  // OR
  "photo_url": "AgACAgIAAxkBAAIC...",  // file_id

  "caption": "📱 Screenshot",  // Optional
  "parse_mode": "HTML"  // Optional
}
```

Supported formats:
- Local paths: `/path/to/file.png`, `~/image.jpg`, `file:///path/to/file.png`
- HTTP URLs: `https://example.com/image.jpg`
- File IDs: Previously uploaded photo IDs

### send_message_with_keyboard

Send messages with interactive buttons:

```javascript
{
  "text": "Choose an option:",
  "buttons": "[[{\"text\":\"Option 1\",\"callback_data\":\"opt1\"}]]",
  "parse_mode": "HTML"
}
```

### edit_message

Edit previously sent messages:

```javascript
{
  "message_id": 123,
  "text": "Updated text",
  "parse_mode": "HTML"
}
```

### delete_message

Delete messages:

```javascript
{
  "message_id": 123
}
```

### get_chat

Get information about the configured chat:

```javascript
{}
```

### get_me

Get bot information:

```javascript
{}
```

## Usage Examples

### iOS Screenshot to Telegram

```bash
./chat

Type here: /paste
Take a screenshot of the iOS app initial state,
then send it to Telegram with caption "📱 Initial App State"
END
```

The AI will:
1. Use iOS MCP to take screenshot → `/tmp/screenshot.png`
2. Use Telegram MCP to upload → `send_photo` with local path
3. Photo appears in your Telegram chat

### Automation Workflow

```bash
Type here: /paste
Execute iOS automation:
1. Boot iPhone 15 simulator
2. Build and launch app
3. Take 2 screenshots (before/after tap)
4. Send both to Telegram with captions
END
```

Both screenshots will be automatically uploaded and sent.

### Send Web Image

```bash
Type here: Send this image to Telegram: https://example.com/logo.png
```

### Format Message

```bash
Type here: Send a message to Telegram with bold "Success" and the app bundle ID
```

AI generates:
```html
<b>Success</b>
App Bundle ID: com.example.app
```

## File Upload Details

When you provide a local file path to `send_photo`:

1. **Auto-detection**: Checks if file exists locally
2. **Multipart upload**: Uses proper Telegram API format
3. **Format support**: JPG, PNG, GIF, WEBP (max 10 MB)
4. **Error handling**: Clear messages if file missing/invalid

See [docs/TELEGRAM_FILE_UPLOAD.md](docs/TELEGRAM_FILE_UPLOAD.md) for details.

## Getting Credentials

### Bot Token

1. Open Telegram
2. Message [@BotFather](https://t.me/BotFather)
3. Send `/newbot`
4. Follow instructions
5. Copy the token (looks like `123456789:ABCdef...`)

### Chat ID

1. Message [@userinfobot](https://t.me/userinfobot)
2. Copy your ID (looks like `123456789`)

Or for group chats:
1. Add bot to group
2. Send a message in the group
3. Visit: `https://api.telegram.org/bot<YOUR_TOKEN>/getUpdates`
4. Find chat ID in the response

## Configuration

### Environment Variables

```bash
TELEGRAM_BOT_TOKEN=123456:ABC-DEF...
TELEGRAM_CHAT_ID=987654321
```

### MCP Config (mcp.json)

```json
{
  "mcpServers": {
    "telegram": {
      "command": "./mcp-telegram",
      "args": [],
      "env": {
        "TELEGRAM_BOT_TOKEN": "your-bot-token",
        "TELEGRAM_CHAT_ID": "your-chat-id"
      }
    }
  }
}
```

### Chat Config (~/.cli-chat/config.yaml)

```yaml
mcp:
  enabled: true
  servers:
    - name: telegram
      command: "./mcp-telegram"
      env:
        TELEGRAM_BOT_TOKEN: "your-token"
        TELEGRAM_CHAT_ID: "your-chat-id"
```

## Integration with iOS Automation

Perfect for automated iOS testing workflows:

```
1. Build iOS app (iOS MCP)
2. Take screenshots (iOS MCP)
3. Send results to Telegram (Telegram MCP)
```

The AI orchestrates all tools automatically based on your prompt.

## Troubleshooting

### "TELEGRAM_BOT_TOKEN not set"
```bash
export TELEGRAM_BOT_TOKEN="your-token"
./mcp-telegram --check
```

### "File not found"
Use absolute paths:
```javascript
// ❌ Relative path might fail
{"photo_url": "image.png"}

// ✅ Absolute path works
{"photo_url": "/Users/you/Downloads/image.png"}
```

### "API error 400: Unsupported URL protocol"
This was the old behavior with `file://` URLs. Now fixed! The server automatically uploads local files.

### "API error 401: Unauthorized"
Invalid bot token. Get a new one from [@BotFather](https://t.me/BotFather).

### "API error 400: chat not found"
Wrong chat ID. Check with [@userinfobot](https://t.me/userinfobot).

## Documentation

- [Multi-Line Input Guide](docs/MULTI_LINE_INPUT.md)
- [Telegram File Upload](docs/TELEGRAM_FILE_UPLOAD.md)
- [iOS Automation Guide](docs/IOS_AUTOMATION_GUIDE.md)
- [Paste Mode Usage](PASTE_MODE_USAGE.md)

## API Reference

Telegram Bot API: https://core.telegram.org/bots/api

## Changelog

### v1.1.0 - Local File Upload Support
- ✅ Added automatic local file upload for `send_photo`
- ✅ Supports `file://`, absolute, and relative paths
- ✅ Multipart/form-data upload implementation
- ✅ Better error messages for invalid paths

### v1.0.0 - Initial Release
- ✅ 7 Telegram Bot API tools
- ✅ MCP server implementation
- ✅ Environment-based configuration

## License

See LICENSE file for details.
