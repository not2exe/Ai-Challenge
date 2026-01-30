package main

import (
	"fmt"
	"log"
	"os"

	"github.com/mark3labs/mcp-go/server"
	"github.com/notexe/cli-chat/internal/telegram"
)

func main() {
	// Check for help flag
	if len(os.Args) > 1 && (os.Args[1] == "--help" || os.Args[1] == "-h") {
		printHelp()
		os.Exit(0)
	}

	// Check for check flag
	if len(os.Args) > 1 && os.Args[1] == "--check" {
		checkEnvironment()
		os.Exit(0)
	}

	// Create the Telegram MCP server
	s := telegram.NewServer()

	// Serve via stdio
	if err := server.ServeStdio(s.MCPServer()); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func printHelp() {
	fmt.Println("Telegram MCP Server")
	fmt.Println()
	fmt.Println("A Model Context Protocol server for Telegram Bot API operations.")
	fmt.Println()
	fmt.Println("USAGE:")
	fmt.Println("  mcp-telegram [FLAGS]")
	fmt.Println()
	fmt.Println("FLAGS:")
	fmt.Println("  --help, -h    Show this help message")
	fmt.Println("  --check       Check environment variables and exit")
	fmt.Println()
	fmt.Println("ENVIRONMENT VARIABLES:")
	fmt.Println("  TELEGRAM_BOT_TOKEN  (required)  Telegram bot token from @BotFather")
	fmt.Println("  TELEGRAM_CHAT_ID    (required)  Chat ID to send messages to")
	fmt.Println()
	fmt.Println("TOOLS:")
	fmt.Println("  send_message              Send a text message")
	fmt.Println("  send_message_with_keyboard Send a message with inline keyboard buttons")
	fmt.Println("  send_photo                Send a photo")
	fmt.Println("  get_chat                  Get chat information")
	fmt.Println("  edit_message              Edit a previously sent message")
	fmt.Println("  delete_message            Delete a message")
	fmt.Println("  get_me                    Get bot information")
	fmt.Println()
	fmt.Println("EXAMPLES:")
	fmt.Println("  # Set environment variables")
	fmt.Println("  export TELEGRAM_BOT_TOKEN=\"123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11\"")
	fmt.Println("  export TELEGRAM_CHAT_ID=\"987654321\"")
	fmt.Println()
	fmt.Println("  # Check configuration")
	fmt.Println("  ./mcp-telegram --check")
	fmt.Println()
	fmt.Println("  # Run the server (normally called by MCP manager)")
	fmt.Println("  ./mcp-telegram")
	fmt.Println()
	fmt.Println("MCP CONFIGURATION:")
	fmt.Println("  Add to your mcp.json:")
	fmt.Println(`  {`)
	fmt.Println(`    "mcpServers": {`)
	fmt.Println(`      "telegram": {`)
	fmt.Println(`        "command": "./mcp-telegram",`)
	fmt.Println(`        "args": [],`)
	fmt.Println(`        "env": {`)
	fmt.Println(`          "TELEGRAM_BOT_TOKEN": "your-bot-token",`)
	fmt.Println(`          "TELEGRAM_CHAT_ID": "your-chat-id"`)
	fmt.Println(`        }`)
	fmt.Println(`      }`)
	fmt.Println(`    }`)
	fmt.Println(`  }`)
	fmt.Println()
	fmt.Println("DOCUMENTATION:")
	fmt.Println("  Telegram Bot API: https://core.telegram.org/bots/api")
	fmt.Println("  Get bot token:    https://t.me/BotFather")
	fmt.Println("  Find chat ID:     https://t.me/userinfobot")
}

func checkEnvironment() {
	fmt.Println("Checking Telegram MCP Server Configuration...")
	fmt.Println()

	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	chatID := os.Getenv("TELEGRAM_CHAT_ID")

	// Check bot token
	if botToken == "" {
		fmt.Println("❌ TELEGRAM_BOT_TOKEN: NOT SET")
		fmt.Println("   Set with: export TELEGRAM_BOT_TOKEN=\"your-token\"")
	} else {
		// Mask token for security
		masked := botToken
		if len(botToken) > 10 {
			masked = botToken[:10] + "..." + botToken[len(botToken)-4:]
		}
		fmt.Printf("✓ TELEGRAM_BOT_TOKEN: %s\n", masked)
	}

	// Check chat ID
	if chatID == "" {
		fmt.Println("❌ TELEGRAM_CHAT_ID: NOT SET")
		fmt.Println("   Set with: export TELEGRAM_CHAT_ID=\"your-chat-id\"")
	} else {
		fmt.Printf("✓ TELEGRAM_CHAT_ID: %s\n", chatID)
	}

	fmt.Println()

	if botToken == "" || chatID == "" {
		fmt.Println("Configuration incomplete. Please set the required environment variables.")
		os.Exit(1)
	}

	fmt.Println("Configuration complete! Server is ready to run.")
}
