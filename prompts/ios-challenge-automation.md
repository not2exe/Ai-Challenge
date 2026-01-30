# iOS Challenge App Automation Prompt

Use this prompt with the agentic system to automate building, testing, and reporting on an iOS app with challenge cards.

## Prompt Template

```
I need you to automate an iOS app workflow using the available MCP tools (mcp-ios and mcp-telegram).

**Target App Details:**
- Project Path: /path/to/YourApp.xcodeproj
- Scheme: YourAppScheme
- Simulator: iPhone 15
- Challenge Card Identifier: accessibility ID "challenge_card" (or provide coordinates)

**Workflow Steps:**

1. **Boot Simulator**
   - Use `boot_simulator` to launch iPhone 15 simulator
   - Wait for boot to complete

2. **Build and Install App**
   - Use `build_app` with:
     - project_path: "/path/to/YourApp.xcodeproj"
     - scheme: "YourAppScheme"
     - simulator: "iPhone 15"
   - Capture the returned bundle_id and app_path
   - Use `install_app` with the app_path
   - Use `launch_app` with the bundle_id

3. **First Screenshot**
   - Wait 3 seconds for app to load
   - Use `screenshot` to capture initial state
   - Save the screenshot path as screenshot1

4. **Interact with Challenge Card**
   - Use `get_elements_with_coords` to find all tappable elements
   - Identify the challenge card element by name/label containing "challenge"
   - Use `tap` with the card's x,y coordinates
   - Wait 2 seconds for navigation/animation

5. **Second Screenshot**
   - Use `screenshot` to capture the challenge detail view
   - Save the screenshot path as screenshot2

6. **Send to Telegram**
   - Use `send_photo` to send screenshot1 with caption "Initial App State"
   - Use `send_photo` to send screenshot2 with caption "Challenge Detail View"
   - Use `send_message` to send a work summary:
     "✅ iOS App Automation Complete

     📱 App: [App Name]
     🎯 Workflow: Build → Launch → Tap Challenge → Capture

     Summary:
     - Successfully built and installed app
     - Captured initial app state
     - Tapped on challenge card
     - Captured challenge detail view

     All screenshots sent above."

**Error Handling:**
- If build fails, report the error to Telegram
- If element not found, send UI tree to Telegram for debugging
- If simulator not available, list available simulators

Please execute this workflow step by step and report any issues.
```

## Usage Examples

### Example 1: Specific Coordinates
If you know the exact tap coordinates:

```
Execute iOS automation workflow:
- Project: ~/Projects/ChallengeApp/ChallengeApp.xcodeproj
- Scheme: ChallengeApp
- Tap coordinates: x=187, y=400 (challenge card location)
- Send results to Telegram
```

### Example 2: Dynamic Element Discovery
If you want to find the element dynamically:

```
Automate iOS app testing:
1. Build ~/Projects/ChallengeApp from scheme "ChallengeApp"
2. Launch on iPhone 15 simulator
3. Screenshot initial state
4. Find element with accessibility ID "challengeCard" or label containing "Challenge"
5. Tap that element
6. Screenshot the result
7. Send both screenshots to Telegram with summary
```

### Example 3: With Video Recording
For comprehensive testing with video:

```
iOS automation with video recording:
1. Boot iPhone 15 simulator
2. Start video recording
3. Build and launch app from ~/Projects/ChallengeApp
4. Screenshot initial state
5. Tap on challenge card (find by label "Daily Challenge")
6. Screenshot detail view
7. Stop video recording
8. Send screenshots and video to Telegram with work summary
```

## Customization Variables

Replace these placeholders with your actual values:

| Variable | Example | Description |
|----------|---------|-------------|
| `PROJECT_PATH` | `/Users/you/Projects/App/App.xcodeproj` | Path to Xcode project |
| `SCHEME` | `MyApp` | Build scheme name |
| `SIMULATOR` | `iPhone 15` or `iPhone 15 Pro Max` | Target simulator |
| `CHALLENGE_ID` | `challenge_card` or `challengeButton` | Accessibility identifier |
| `BUNDLE_ID` | `com.company.app` | App bundle ID (auto-detected) |

## Finding Element Identifiers

If you don't know the accessibility ID, use this helper prompt first:

```
Please help me find the challenge card element:
1. Build and launch my app at ~/Projects/ChallengeApp
2. Use get_ui_tree to dump the entire UI hierarchy
3. Search for elements with labels/names containing "challenge" or "card"
4. Show me the accessibility IDs and coordinates of matching elements
```

## Tips

- **First time WDA setup**: The first UI automation call may take 30-60 seconds as WebDriverAgent builds
- **Element not found**: Increase wait times or use visual coordinates from `get_elements_with_coords`
- **Screenshots**: Are automatically saved and returned as base64 PNG data
- **Telegram photos**: Must be sent via `send_photo` tool, not `send_message`
- **UI Tree**: JSON format is easier to parse than XML for finding elements

## Integration with Scheduler

To run this automatically on a schedule, add to your scheduler config:

```yaml
scheduler:
  enabled: true
  interval: "1h"  # Run every hour
  system_prompt: "You are an iOS automation assistant."
  prompt_template: |
    [Paste the automation prompt from above]
```

Then the scheduler will execute this workflow periodically and send results to Telegram.
