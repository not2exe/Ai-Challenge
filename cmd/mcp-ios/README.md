# MCP iOS Server

MCP (Model Context Protocol) server for iOS Simulator automation. Provides tools for building, installing, launching apps, and UI automation via WebDriverAgent.

## Prerequisites

### 1. Xcode & Command Line Tools

```bash
# Install Xcode from App Store, then:
xcode-select --install
```

### 2. iOS Simulator

Boot a simulator before using UI tools:

```bash
# List available simulators
xcrun simctl list devices

# Boot a simulator
xcrun simctl boot "iPhone 16 Pro"

# Open Simulator app
open -a Simulator
```

### 3. WebDriverAgent (for UI automation)

WDA is required for UI tools (tap, swipe, get_ui_tree, etc.). The server will **auto-start WDA** if you have Appium installed.

**Install Appium + XCUITest driver:**

```bash
# Install Appium
npm install -g appium

# Install XCUITest driver (includes WDA)
appium driver install xcuitest
```

**Verify installation:**

```bash
# Check Appium
appium --version

# Check driver
appium driver list --installed
# Should show: xcuitest
```

WDA will be auto-detected from: `~/.appium/node_modules/appium-xcuitest-driver/node_modules/appium-webdriveragent/`

## Build & Run

```bash
# Build the MCP server
go build -o mcp-ios ./cmd/mcp-ios

# Test it manually (Ctrl+C to exit)
./mcp-ios
```

## Configuration

Add to your MCP config (`~/.cli-chat/mcp.json`):

```json
{
  "mcpServers": {
    "ios": {
      "command": "/path/to/mcp-ios",
      "args": [],
      "env": {}
    }
  }
}
```

Or use the example config:

```bash
cp mcp.example.json ~/.cli-chat/mcp.json
```

## Available Tools

### Simulator Management

| Tool | Description |
|------|-------------|
| `list_simulators` | List all iOS simulators with UDID, state |
| `boot_simulator` | Boot a simulator by UDID or name |
| `shutdown_simulator` | Shutdown a simulator |
| `screenshot` | Take a screenshot (PNG) |
| `record_video_start` | Start video recording |
| `record_video_stop` | Stop recording, get video file |
| `open_url` | Open URL in simulator browser |

### App Management

| Tool | Description |
|------|-------------|
| `list_schemes` | List Xcode project schemes |
| `build_app` | Build app for simulator |
| `install_app` | Install .app bundle |
| `launch_app` | Launch app by bundle ID |
| `terminate_app` | Terminate running app |
| `uninstall_app` | Uninstall app |

### UI Automation (requires WDA)

| Tool | Description |
|------|-------------|
| `wda_status` | Check if WDA is running |
| `wda_set_device` | Set target simulator for WDA |
| `wda_create_session` | Create WDA session |
| `get_ui_tree` | Get UI hierarchy (XML/JSON) |
| `get_elements_with_coords` | Get elements with tap coordinates |
| `find_element` | Find element by accessibility ID, name, xpath |
| `tap` | Tap at coordinates or element |
| `long_press` | Long press gesture |
| `swipe` | Swipe gesture (direction or coordinates) |
| `input_text` | Type text into focused field |
| `press_button` | Press hardware button (home, volume) |

## WDA Auto-Start

The server automatically manages WDA:

1. When you call any UI tool (`get_ui_tree`, `tap`, etc.)
2. Server checks if WDA is running on port 8100
3. If not, it automatically:
   - Finds WDA project (from Appium installation)
   - Finds booted simulator
   - Builds and starts WDA
   - Waits for it to be ready (~30 seconds first time)
4. Subsequent calls reuse the running WDA

**First UI tool call may take 30-60 seconds** while WDA starts.

## Troubleshooting

### "WDA not found"

```bash
# Reinstall Appium XCUITest driver
appium driver uninstall xcuitest
appium driver install xcuitest
```

### "No booted simulator"

```bash
# Boot a simulator first
xcrun simctl boot "iPhone 16 Pro"
open -a Simulator
```

### "WDA build failed"

```bash
# Open WDA project in Xcode to fix signing
open ~/.appium/node_modules/appium-xcuitest-driver/node_modules/appium-webdriveragent/WebDriverAgent.xcodeproj

# Select WebDriverAgentRunner scheme
# Set signing team in Signing & Capabilities
# Build manually once: Cmd+B
```

### "Empty UI tree" or "No elements found"

Your app needs accessibility labels. In SwiftUI:

```swift
Button("Login") { }
    .accessibilityIdentifier("login_button")

Text("Hello")
    .accessibilityLabel("Greeting text")
```

### WDA stops unexpectedly

WDA may stop if:
- Simulator was restarted
- Xcodebuild process was killed
- System went to sleep

Just call any UI tool again - it will auto-restart.

## Example Usage

In CLI Chat:

```
> list_simulators
> boot_simulator device_id="iPhone 16 Pro"
> build_app project_path="MyApp" scheme="MyApp"
> install_app app_path="/path/to/MyApp.app"
> launch_app bundle_id="com.example.MyApp"
> screenshot output_path="screen.png"
> get_elements_with_coords
> tap x=200 y=400
```

## Architecture

```
cmd/mcp-ios/
  main.go              → Entry point, starts MCP server

internal/ios/
  server.go            → MCP server, tool handlers
  simctl.go            → xcrun simctl wrapper
  xcodebuild.go        → xcodebuild wrapper
  types.go             → Shared types
  wda/
    client.go          → WDA HTTP client
    manager.go         → WDA lifecycle (auto-start)
    types.go           → WDA response types
```
