# iOS Automation Guide

Complete guide for automating iOS app testing with screenshot capture and Telegram reporting.

## Overview

This guide shows you how to use the MCP servers (mcp-ios + mcp-telegram) to:
1. Build and launch an iOS app in the simulator
2. Take screenshots
3. Tap on UI elements (like challenge cards)
4. Send results to Telegram

## Prerequisites

### 1. Build the MCP Servers

```bash
# Build the iOS MCP server
go build -o mcp-ios ./cmd/mcp-ios

# Build the Telegram MCP server
go build -o mcp-telegram ./cmd/mcp-telegram

# Build the test automation tool
go build -o test-ios-automation ./cmd/test-ios-automation
```

### 2. Install iOS Automation Tools

```bash
# Install Appium (required for UI automation)
npm install -g appium

# Install XCUITest driver
appium driver install xcuitest
```

### 3. Configure MCP Servers

Create or update `~/.cli-chat/mcp.json`:

```json
{
  "mcpServers": {
    "ios": {
      "command": "./mcp-ios",
      "args": [],
      "env": {}
    },
    "telegram": {
      "command": "./mcp-telegram",
      "args": [],
      "env": {
        "TELEGRAM_BOT_TOKEN": "YOUR_BOT_TOKEN_HERE",
        "TELEGRAM_CHAT_ID": "YOUR_CHAT_ID_HERE"
      }
    }
  }
}
```

### 4. Get Telegram Credentials

```bash
# Create a bot and get token
# 1. Message @BotFather on Telegram
# 2. Send /newbot
# 3. Follow instructions
# 4. Copy the token

# Get your chat ID
# 1. Message @userinfobot on Telegram
# 2. Copy the ID shown
```

### 5. Update Config File

Edit `~/.cli-chat/config.yaml`:

```yaml
provider: deepseek
api:
  key: "your-deepseek-api-key"

model:
  name: "deepseek-chat"
  max_tokens: 4096
  temperature: 1.0

mcp:
  enabled: true
  servers:
    - name: ios
      command: "./mcp-ios"
      args: []
      env: {}
    - name: telegram
      command: "./mcp-telegram"
      args: []
      env:
        TELEGRAM_BOT_TOKEN: "YOUR_BOT_TOKEN"
        TELEGRAM_CHAT_ID: "YOUR_CHAT_ID"
```

## Usage

### Method 1: Using the Test Automation Tool

The simplest way to run the automation:

```bash
./test-ios-automation \
  --project ~/Projects/MyApp/MyApp.xcodeproj \
  --scheme MyApp \
  --simulator "iPhone 15" \
  --challenge-id challenge_card
```

**Options:**
- `--project` (required): Path to .xcodeproj or .xcworkspace
- `--scheme` (required): Build scheme name
- `--simulator`: Target simulator (default: "iPhone 15")
- `--challenge-id`: Accessibility ID of challenge card (optional)
- `--config`: Config file path (default: ~/.cli-chat/config.yaml)

**Example without challenge ID** (auto-discover):
```bash
./test-ios-automation \
  --project ~/Projects/ChallengeApp/ChallengeApp.xcodeproj \
  --scheme ChallengeApp
```

### Method 2: Using the Chat REPL

Start the interactive chat:

```bash
./chat
```

Then paste the automation prompt:

```
Execute iOS app automation workflow with these steps:

1. Boot the iPhone 15 simulator

2. Build my iOS app:
   - Project: /Users/yourname/Projects/MyApp/MyApp.xcodeproj
   - Scheme: MyApp
   - Target: iPhone 15 simulator

3. Install and launch the built app

4. Take a screenshot of the initial app state

5. Find and tap on the challenge card:
   - Use get_elements_with_coords to find all elements
   - Look for element with label containing "challenge"
   - Tap that element
   - Wait 2 seconds

6. Take a second screenshot of the challenge detail view

7. Send both screenshots to Telegram with a summary message

Please execute this workflow and report any issues.
```

### Method 3: Using the Scheduler

For periodic automated testing, add to `~/.cli-chat/config.yaml`:

```yaml
scheduler:
  enabled: true
  interval: "2h"  # Run every 2 hours
  system_prompt: "You are an iOS automation assistant."
  prompt_template: |
    Execute iOS automation workflow:
    1. Build and launch app from ~/Projects/MyApp/MyApp.xcodeproj (scheme: MyApp)
    2. Screenshot initial state
    3. Tap challenge card
    4. Screenshot detail view
    5. Send results to Telegram with summary
```

Then start with scheduler enabled:

```bash
./chat  # Scheduler runs in background
```

## Workflow Steps Explained

### 1. Boot Simulator

```
Tool: boot_simulator
Parameters:
  - device_id: "iPhone 15" (or full UDID)
```

Lists available simulators and boots the specified one.

### 2. Build App

```
Tool: build_app
Parameters:
  - project_path: "/path/to/App.xcodeproj"
  - scheme: "MyApp"
  - simulator: "iPhone 15"
  - configuration: "Debug" (optional)

Returns:
  - app_path: "/path/to/Build/MyApp.app"
  - bundle_id: "com.company.myapp"
```

Builds the app using xcodebuild.

### 3. Install and Launch

```
Tool: install_app
Parameters:
  - app_path: "/path/to/MyApp.app"

Tool: launch_app
Parameters:
  - bundle_id: "com.company.myapp"
```

Installs the .app bundle and launches it.

### 4. Take Screenshots

```
Tool: screenshot

Returns:
  - Base64 encoded PNG image data
```

Captures the current simulator screen.

### 5. Find and Tap Elements

**Option A: By Accessibility ID**
```
Tool: find_element
Parameters:
  - using: "accessibility id"
  - value: "challenge_card"

Tool: tap
Parameters:
  - element_id: <element from find_element>
```

**Option B: By Coordinates**
```
Tool: get_elements_with_coords

Returns: JSON array of all tappable elements with:
  - label, name, type
  - x, y, width, height

Tool: tap
Parameters:
  - x: 187
  - y: 400
```

### 6. Send to Telegram

```
Tool: send_photo
Parameters:
  - photo_url: <base64 image data>
  - caption: "📱 Initial App State"
  - parse_mode: "HTML"

Tool: send_message
Parameters:
  - text: "Summary message with HTML formatting"
  - parse_mode: "HTML"
```

## Troubleshooting

### "WebDriverAgent not available"

First UI automation call takes 30-60 seconds to build WDA:

```bash
# Check WDA status
./mcp-ios --check

# Manually test
./test-ios-automation --project <path> --scheme <name>
```

### "Element not found"

Use the UI tree to find the correct identifier:

```
In chat REPL:
> Can you use get_ui_tree to show me all elements in the app, and find anything related to "challenge"?
```

Or use coordinates from `get_elements_with_coords`.

### "Build failed"

Check the scheme name:

```bash
# List available schemes
xcodebuild -list -project ~/Projects/MyApp/MyApp.xcodeproj
```

### "Simulator not available"

List available simulators:

```bash
# Using mcp-ios
./mcp-ios --check

# Using xcrun
xcrun simctl list devices available
```

### "Telegram send failed"

Verify credentials:

```bash
# Check configuration
./mcp-telegram --check

# Test manually
curl "https://api.telegram.org/bot<YOUR_TOKEN>/getMe"
```

## Advanced Usage

### Find Element by Multiple Criteria

```
Use find_element with predicate:
- name CONTAINS "challenge"
- label CONTAINS "Challenge"
- value CONTAINS "challenge"
```

### Wait for Element

```
Retry find_element up to 5 times with 1 second delay between attempts
```

### Record Video

```
1. Use record_video_start before launching app
2. Execute workflow
3. Use record_video_stop
4. Video saved to ~/Desktop/<timestamp>.mp4
5. Upload to Telegram (requires file upload support)
```

### Multiple Screenshots

```
Take screenshots at different stages:
- After app launch
- After tapping card
- After scrolling
- After navigation
```

### Batch Automation

Create a script to test multiple scenarios:

```bash
#!/bin/bash

scenarios=(
  "challenge_card_1:Challenge 1"
  "challenge_card_2:Challenge 2"
  "settings_button:Settings"
)

for scenario in "${scenarios[@]}"; do
  id="${scenario%%:*}"
  name="${scenario##*:}"

  echo "Testing: $name"
  ./test-ios-automation \
    --project ~/Projects/MyApp/MyApp.xcodeproj \
    --scheme MyApp \
    --challenge-id "$id"

  sleep 5
done
```

## Integration Examples

### CI/CD Pipeline

```yaml
# .github/workflows/ios-automation.yml
name: iOS Automation Tests

on:
  push:
    branches: [main]
  schedule:
    - cron: '0 */6 * * *'  # Every 6 hours

jobs:
  test:
    runs-on: macos-latest
    steps:
      - uses: actions/checkout@v3

      - name: Setup iOS automation
        run: |
          npm install -g appium
          appium driver install xcuitest

      - name: Build MCP servers
        run: |
          go build -o mcp-ios ./cmd/mcp-ios
          go build -o mcp-telegram ./cmd/mcp-telegram
          go build -o test-ios-automation ./cmd/test-ios-automation

      - name: Run automation
        env:
          TELEGRAM_BOT_TOKEN: ${{ secrets.TELEGRAM_BOT_TOKEN }}
          TELEGRAM_CHAT_ID: ${{ secrets.TELEGRAM_CHAT_ID }}
          DEEPSEEK_API_KEY: ${{ secrets.DEEPSEEK_API_KEY }}
        run: |
          ./test-ios-automation \
            --project ./MyApp.xcodeproj \
            --scheme MyApp
```

### Slack Integration

Modify telegram server to also send to Slack webhooks, or use a Slack MCP server.

### Dashboard

Send results to a web dashboard via webhook:

```
Add send_webhook tool to collect metrics:
- Build success rate
- Screenshot comparisons
- Performance metrics
```

## Tips

1. **Set Accessibility IDs** in your iOS app for reliable automation
2. **Use descriptive labels** for UI elements
3. **Wait for animations** before taking screenshots (2-3 seconds)
4. **Check simulator state** before running automation
5. **Close other apps** to free up simulator resources
6. **Use HTML formatting** in Telegram messages for better readability
7. **Compress screenshots** if Telegram upload is slow
8. **Cache WDA** to speed up subsequent runs
9. **Use simulator UDIDs** instead of names for precision
10. **Monitor token usage** - each automation uses ~500-2000 tokens

## Example Prompts

### Basic Workflow
```
Build my iOS app from ~/Projects/MyApp/MyApp.xcodeproj (scheme: MyApp),
take a screenshot, tap on the challenge card, take another screenshot,
and send both to Telegram with a summary.
```

### With Error Recovery
```
Automate iOS testing with error recovery:
1. Build and launch app
2. If build fails, send error to Telegram and stop
3. Take screenshot
4. Try to find challenge card, if not found, send UI tree to Telegram
5. If found, tap it and take another screenshot
6. Send results to Telegram
```

### Comparison Testing
```
Take 3 screenshots:
1. Initial app state
2. After tapping challenge card
3. After scrolling down
Send all 3 to Telegram in order with captions showing the workflow progression.
```

## API Reference

See the individual MCP server documentation:
- [iOS Tools](../internal/ios/README.md) (if exists)
- [Telegram Tools](../internal/telegram/README.md) (if exists)

Or check tool definitions in:
- `internal/ios/server.go`
- `internal/telegram/server.go`

## Support

For issues or questions:
1. Check logs: `~/.cli-chat/logs/`
2. Test MCP servers: `./mcp-ios --check`, `./mcp-telegram --check`
3. Verify configuration: `cat ~/.cli-chat/config.yaml`
4. Check simulator availability: `xcrun simctl list devices`
