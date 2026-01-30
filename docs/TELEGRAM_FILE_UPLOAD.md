# Telegram File Upload Guide

The Telegram MCP server now supports uploading local files directly.

## Supported Photo Sources

The `send_photo` tool accepts three types of photo sources:

1. **Local file paths** - Upload files from your computer
2. **HTTP/HTTPS URLs** - Send photos from the web
3. **Telegram file_id** - Reuse previously uploaded files

## Local File Upload

### File Path Formats

All of these work:

```
/Users/username/image.png
~/Downloads/photo.jpg
./relative/path/screenshot.png
file:///Users/username/image.png
```

The system automatically:
- Removes `file://` prefix if present
- Checks if the file exists locally
- Uploads it using multipart/form-data
- Sends it to Telegram

### Example Usage

```javascript
// From iOS automation
{
  "tool": "send_photo",
  "arguments": {
    "photo_url": "/Users/notexe/Downloads/ChallengeIt/initial_screen_fresh.png",
    "caption": "📱 Initial Screen - ChallengeIt iOS App",
    "parse_mode": "HTML"
  }
}
```

### In Chat Prompt

```
Take a screenshot of the iOS app and send it to Telegram with caption "App Screenshot"
```

The AI will:
1. Use iOS MCP tool to take screenshot (returns local file path)
2. Use Telegram MCP tool to upload the file
3. Send it to your configured chat

## HTTP URL

For photos already on the web:

```javascript
{
  "tool": "send_photo",
  "arguments": {
    "photo_url": "https://example.com/image.jpg",
    "caption": "Photo from the web"
  }
}
```

## File ID (Reuse)

To reuse a previously uploaded photo:

```javascript
{
  "tool": "send_photo",
  "arguments": {
    "photo_url": "AgACAgIAAxkBAAIC...",  // file_id from previous upload
    "caption": "Same photo again"
  }
}
```

## Caption Support

Captions support HTML, Markdown, and MarkdownV2:

```javascript
{
  "photo_url": "/path/to/image.png",
  "caption": "<b>Bold text</b>\n<i>Italic text</i>",
  "parse_mode": "HTML"
}
```

## File Size Limits

Telegram Bot API limits:
- **Photos**: Up to 10 MB
- **File formats**: JPG, PNG, GIF, WEBP

## Error Handling

The tool will return clear errors:

```
❌ "Invalid photo path: /nonexistent.png (file not found and not a valid HTTP URL)"
❌ "Failed to upload photo: file too large"
❌ "API error (status 400): invalid file format"
```

## Complete iOS Automation Example

```
Take screenshot of iOS app initial state,
save it to a local file,
then send it to Telegram with caption "📱 Initial State"
```

The workflow:
1. iOS MCP: `screenshot` → returns `/tmp/screenshot_123.png`
2. Telegram MCP: `send_photo` with local path
3. File is uploaded and sent

## Comparison: Before vs After

### Before (Error)

```
Tool: send_photo
Args: {"photo_url": "file:///Users/.../image.png"}
Result: ❌ API error (status 400): Unsupported URL protocol
```

### After (Success)

```
Tool: send_photo
Args: {"photo_url": "file:///Users/.../image.png"}
Result: ✅ Photo uploaded: {"ok":true,"result":{...}}
```

## Technical Details

When you provide a local file path:

1. **Path normalization**: Removes `file://` prefix
2. **File existence check**: Verifies file exists
3. **Multipart upload**: Creates form-data with:
   - `chat_id`: Your configured chat
   - `photo`: File binary data
   - `caption`: Optional caption
   - `parse_mode`: Optional format
4. **POST request**: Sends to Telegram Bot API
5. **Response**: Returns result with file_id

## Tips

✅ **Use absolute paths** - More reliable than relative paths
✅ **Check file exists** - Tool will give clear error if not found
✅ **Support all formats** - JPG, PNG, GIF work great
✅ **Add captions** - Makes messages more informative
✅ **Use HTML formatting** - Bold, italic, links in captions

## Troubleshooting

**Q: "file not found" error**
A: Check the path is correct and file exists. Use absolute paths.

**Q: "invalid file format" error**
A: Telegram only accepts JPG, PNG, GIF, WEBP for photos.

**Q: "file too large" error**
A: Photos must be under 10 MB. Compress before sending.

**Q: Can I send videos?**
A: Not yet with send_photo. We can add send_video if needed.

**Q: Can I send documents?**
A: Not yet. We can add send_document for PDFs, etc.

## Future Enhancements

Possible additions:
- `send_video` - Upload video files
- `send_document` - Upload any file type
- `send_audio` - Upload audio files
- `send_media_group` - Send multiple photos at once
- Automatic compression for large files

Let me know if you need any of these!
