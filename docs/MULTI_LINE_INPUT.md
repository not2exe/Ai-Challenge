# Multi-Line Input Guide

The chat supports multi-line input with two methods: natural typing and paste mode.

## Method 1: Natural Typing

Start typing your message and press Enter for new lines. Type **END** on a new line to submit.

```
Type here: Boot iPhone 15 simulator,
(Press Enter twice to submit, or type 'END' on new line)
... build app from ~/Projects/MyApp/MyApp.xcodeproj (scheme: MyApp),
... install and launch it,
... take screenshot
... END
```

## Method 2: Paste Mode (Recommended for Copy-Paste)

Use `/paste` command to enter paste mode, then paste your content:

```bash
Type here: /paste
=== PASTE MODE ===
Paste your content, then type 'END' on a new line and press Enter

<paste your multi-line content here>
END
=== END PASTE MODE ===
```

### Example:

```
Type here: /paste
=== PASTE MODE ===
Paste your content, then type 'END' on a new line and press Enter

Boot iPhone 15 simulator,
build app from ~/Projects/MyApp/MyApp.xcodeproj (scheme: MyApp),
install and launch it,
take screenshot,
find and tap element with label containing "challenge",
wait 2 seconds,
take another screenshot,
then send both screenshots to Telegram with summary
END
=== END PASTE MODE ===
```

## Quick Reference

| Action | Command |
|--------|---------|
| Enter paste mode | `/paste` |
| Submit (paste mode) | Type `END` on new line |
| Submit (typing mode) | Type `END` or press Enter twice |
| Cancel | Press Ctrl+C |
| Single-line commands | `/help`, `/clear`, etc. |

## Use Cases

### 1. Pasting iOS Automation Prompt

```bash
./chat
Type here: /paste
[Paste from prompts/ios-automation-ready.txt]
END
```

### 2. Typing Multi-Line Naturally

```bash
./chat
Type here: I need you to:
... 1. Boot simulator
... 2. Build app
... 3. Take screenshots
... 4. Send to Telegram
... END
```

### 3. Pasting Code or Configuration

```bash
Type here: /paste
{
  "mcp": {
    "enabled": true,
    "servers": [...]
  }
}

Please analyze this config.
END
```

## Tips

✅ **Use /paste for copy-paste operations** - Most reliable
✅ **Type END to submit** - Clear and explicit
✅ **Press Ctrl+C to cancel** - Anytime during input
✅ **Commands (/) are single-line** - No multi-line needed
✅ **Paste mode preserves formatting** - All whitespace kept

## Troubleshooting

**Q: My paste didn't work**
A: Make sure you:
1. Type `/paste` first
2. Press Enter after `/paste`
3. Paste your content
4. Type `END` on a new line
5. Press Enter after `END`

**Q: How do I know I'm in paste mode?**
A: You'll see green `=== PASTE MODE ===` message

**Q: Can I paste without /paste command?**
A: You can try, but `/paste` is more reliable for large multi-line content

**Q: What if my content contains "END"?**
A: Use `<<<` instead as the terminator

**Q: Do I need to type "..." at the start of each line?**
A: No! The "..." is just the prompt. Just paste/type your content normally.
