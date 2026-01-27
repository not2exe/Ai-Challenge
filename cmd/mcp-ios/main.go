// Command mcp-ios provides an MCP server for iOS simulator automation.
//
// This server provides tools for:
// - Simulator management (list, boot, shutdown, screenshot, video recording)
// - App management (build, install, launch, terminate, uninstall)
// - UI automation via WebDriverAgent (find elements, tap, swipe, input text)
//
// Usage:
//
//	./mcp-ios          # Start MCP server (stdio)
//	./mcp-ios --check  # Check prerequisites
//	./mcp-ios --help   # Show help
//
// The server communicates via stdio using the MCP protocol.
// Add it to your MCP client configuration in ~/.cli-chat/mcp.json
//
// For UI automation tools, WebDriverAgent is auto-started if Appium is installed.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/mark3labs/mcp-go/server"
	"github.com/notexe/cli-chat/internal/ios"
)

func main() {
	// Handle flags
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--check", "-c":
			checkPrerequisites()
			return
		case "--help", "-h":
			printHelp()
			return
		}
	}

	// Start MCP server
	s := ios.NewServer()

	if err := server.ServeStdio(s.MCPServer()); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Println(`MCP iOS Server - iOS Simulator automation via MCP protocol

USAGE:
    mcp-ios              Start MCP server (communicates via stdio)
    mcp-ios --check      Check if prerequisites are installed
    mcp-ios --help       Show this help

PREREQUISITES:
    1. Xcode & Command Line Tools
       xcode-select --install

    2. iOS Simulator (boot one before using UI tools)
       xcrun simctl boot "iPhone 16 Pro"
       open -a Simulator

    3. Appium + XCUITest driver (for UI automation - tap, swipe, etc.)
       npm install -g appium
       appium driver install xcuitest

CONFIGURATION:
    Add to ~/.cli-chat/mcp.json:
    {
      "mcpServers": {
        "ios": {
          "command": "/path/to/mcp-ios",
          "args": []
        }
      }
    }

TOOLS:
    Simulator: list_simulators, boot_simulator, screenshot, record_video_*
    Apps:      build_app, install_app, launch_app, terminate_app
    UI:        get_ui_tree, get_elements_with_coords, tap, swipe, input_text

For more info see: cmd/mcp-ios/README.md`)
}

func checkPrerequisites() {
	fmt.Println("Checking MCP iOS Server prerequisites...\n")

	allGood := true

	// Check Xcode
	fmt.Print("✓ Xcode Command Line Tools: ")
	if _, err := exec.LookPath("xcodebuild"); err != nil {
		fmt.Println("NOT FOUND")
		fmt.Println("  → Install: xcode-select --install")
		allGood = false
	} else {
		out, _ := exec.Command("xcodebuild", "-version").Output()
		version := strings.Split(string(out), "\n")[0]
		fmt.Println(version)
	}

	// Check simctl
	fmt.Print("✓ Simulator (simctl): ")
	if _, err := exec.LookPath("xcrun"); err != nil {
		fmt.Println("NOT FOUND")
		allGood = false
	} else {
		fmt.Println("OK")
	}

	// Check for booted simulator
	fmt.Print("✓ Booted Simulator: ")
	out, _ := exec.Command("xcrun", "simctl", "list", "devices", "-j").Output()
	if strings.Contains(string(out), `"state" : "Booted"`) {
		fmt.Println("YES")
	} else {
		fmt.Println("NONE")
		fmt.Println("  → Boot one: xcrun simctl boot \"iPhone 16 Pro\" && open -a Simulator")
		allGood = false
	}

	// Check Appium
	fmt.Print("✓ Appium: ")
	if _, err := exec.LookPath("appium"); err != nil {
		fmt.Println("NOT FOUND")
		fmt.Println("  → Install: npm install -g appium")
		allGood = false
	} else {
		out, _ := exec.Command("appium", "--version").Output()
		fmt.Println(strings.TrimSpace(string(out)))
	}

	// Check WDA
	fmt.Print("✓ WebDriverAgent: ")
	homeDir, _ := os.UserHomeDir()
	wdaPath := filepath.Join(homeDir, ".appium", "node_modules", "appium-xcuitest-driver",
		"node_modules", "appium-webdriveragent", "WebDriverAgent.xcodeproj")
	if _, err := os.Stat(wdaPath); err != nil {
		fmt.Println("NOT FOUND")
		fmt.Println("  → Install: appium driver install xcuitest")
		allGood = false
	} else {
		fmt.Println("OK")
		fmt.Printf("  Path: %s\n", filepath.Dir(wdaPath))
	}

	// Summary
	fmt.Println()
	if allGood {
		fmt.Println("✅ All prerequisites met! MCP iOS Server is ready to use.")
	} else {
		fmt.Println("❌ Some prerequisites are missing. Install them and run --check again.")
		os.Exit(1)
	}
}
