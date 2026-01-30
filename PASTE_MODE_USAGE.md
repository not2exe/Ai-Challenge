# How to Paste Multi-Line Prompts

## Quick Guide

### Step 1: Start Chat

```bash
./chat
```

### Step 2: Enter Paste Mode

```
Type here: /paste
```

You'll see:
```
=== PASTE MODE ===
Paste your content, then type 'END' on a new line and press Enter
```

### Step 3: Paste Your Content

Just paste your multi-line prompt. For example:

```
Boot iPhone 15 simulator,
build app from ~/Projects/MyApp/MyApp.xcodeproj (scheme: MyApp),
install and launch it,
take screenshot,
find and tap element with label containing "challenge",
wait 2 seconds,
take another screenshot,
then send both screenshots to Telegram with summary
```

### Step 4: Type END and Press Enter

```
END
```

You'll see:
```
=== END PASTE MODE ===
```

And your prompt will be sent to the AI!

## Complete Example

```bash
$ ./chat

Welcome to CLI Chat with DeepSeek!
Model: deepseek-chat
Type /help for available commands or start chatting.

Type here: /paste
=== PASTE MODE ===
Paste your content, then type 'END' on a new line and press Enter

Boot iPhone 15 simulator,
build app from ~/Projects/ChallengeApp/ChallengeApp.xcodeproj (scheme: ChallengeApp),
install and launch it,
take screenshot,
find and tap element with label containing "challenge",
wait 2 seconds,
take another screenshot,
then send both screenshots to Telegram with captions "📱 Initial App State"
and "🎯 Challenge Detail View" plus a summary message showing build status and bundle ID.
END
=== END PASTE MODE ===

[AI processes your request...]
```

## Tips

✅ **Copy prompt from file**: `cat prompts/ios-automation-ready.txt` then paste
✅ **Type /paste first**: This enters paste mode
✅ **Paste everything at once**: Don't paste line by line
✅ **Type END when done**: This submits your prompt
✅ **Use Ctrl+C to cancel**: If you make a mistake

## Ready-to-Use Prompt

Copy this template and replace PROJECT_PATH and SCHEME_NAME:

```
Boot iPhone 15 simulator,
build app from PROJECT_PATH (scheme: SCHEME_NAME),
install and launch it,
take screenshot,
find and tap element with label containing "challenge",
wait 2 seconds,
take another screenshot,
then send both screenshots to Telegram with captions "📱 Initial App State" and "🎯 Challenge Detail View" plus a summary message showing build status and bundle ID.
```

Then:
1. Type `/paste` in chat
2. Paste the above
3. Type `END`
4. Press Enter

Done! 🎉
