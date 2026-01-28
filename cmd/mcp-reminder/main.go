// Command mcp-reminder provides an MCP server for reminder management.
//
// This server provides tools for creating, listing, completing, and managing
// reminders stored in a SQLite database.
//
// Usage:
//
//	./mcp-reminder          # Start MCP server (stdio)
//	./mcp-reminder --help   # Show help
//
// Environment:
//
//	REMINDER_DB_PATH  Path to SQLite database (default: ~/.cli-chat/reminders.db)
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mark3labs/mcp-go/server"
	"github.com/notexe/cli-chat/internal/reminder"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--help", "-h":
			printHelp()
			return
		}
	}

	dbPath := os.Getenv("REMINDER_DB_PATH")
	if dbPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to get home directory: %v\n", err)
			os.Exit(1)
		}
		dir := filepath.Join(home, ".cli-chat")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create config directory: %v\n", err)
			os.Exit(1)
		}
		dbPath = filepath.Join(dir, "reminders.db")
	}

	store, err := reminder.NewStore(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open database: %v\n", err)
		os.Exit(1)
	}
	defer store.Close()

	s := reminder.NewServer(store)

	if err := server.ServeStdio(s.MCPServer()); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Println(`MCP Reminder Server - Reminder management via MCP protocol

USAGE:
    mcp-reminder          Start MCP server (communicates via stdio)
    mcp-reminder --help   Show this help

ENVIRONMENT:
    REMINDER_DB_PATH  Path to SQLite database file
                      Default: ~/.cli-chat/reminders.db

TOOLS:
    add_reminder       Add a new reminder (title, due_date, description, priority)
    list_reminders     List all reminders (optional status filter)
    get_due_reminders  Get pending reminders that are due or overdue
    complete_reminder  Mark a reminder as completed
    delete_reminder    Delete a reminder permanently
    update_reminder    Update reminder fields (title, description, due_date, priority)

CONFIGURATION:
    Add to ~/.cli-chat/mcp.json:
    {
      "mcpServers": {
        "reminder": {
          "command": "/path/to/mcp-reminder",
          "args": []
        }
      }
    }`)
}
