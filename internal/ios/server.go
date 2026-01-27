package ios

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/notexe/cli-chat/internal/ios/wda"
)

const (
	serverName    = "ios-simulator"
	serverVersion = "1.0.0"
)

// Server is the MCP server for iOS simulator automation.
type Server struct {
	mcpServer  *server.MCPServer
	simctl     *SimCtl
	xcodebuild *XcodeBuild
	wdaManager *wda.Manager
	wdaPort    int
}

// NewServer creates a new iOS MCP server.
func NewServer() *Server {
	s := &Server{
		simctl:     NewSimCtl(),
		xcodebuild: NewXcodeBuild(),
		wdaManager: wda.NewManager(8100),
		wdaPort:    8100,
	}

	s.mcpServer = server.NewMCPServer(
		serverName,
		serverVersion,
		server.WithToolCapabilities(false),
	)

	// Register all tools
	s.registerSimulatorTools()
	s.registerAppTools()
	s.registerUITools()

	return s
}

// MCPServer returns the underlying MCP server for serving.
func (s *Server) MCPServer() *server.MCPServer {
	return s.mcpServer
}

// registerSimulatorTools registers simulator management tools.
func (s *Server) registerSimulatorTools() {
	// list_simulators
	s.mcpServer.AddTool(
		mcp.NewTool("list_simulators",
			mcp.WithDescription("List all available iOS simulators with their UDID, name, state, and runtime"),
		),
		s.handleListSimulators,
	)

	// boot_simulator
	s.mcpServer.AddTool(
		mcp.NewTool("boot_simulator",
			mcp.WithDescription("Boot an iOS simulator by UDID or name"),
			mcp.WithString("device_id", mcp.Required(), mcp.Description("Simulator UDID or name")),
		),
		s.handleBootSimulator,
	)

	// shutdown_simulator
	s.mcpServer.AddTool(
		mcp.NewTool("shutdown_simulator",
			mcp.WithDescription("Shutdown an iOS simulator"),
			mcp.WithString("device_id", mcp.Required(), mcp.Description("Simulator UDID or name")),
		),
		s.handleShutdownSimulator,
	)

	// screenshot
	s.mcpServer.AddTool(
		mcp.NewTool("screenshot",
			mcp.WithDescription("Take a screenshot of the iOS simulator"),
			mcp.WithString("device_id", mcp.Description("Simulator UDID (uses booted device if not specified)")),
			mcp.WithString("output_path", mcp.Description("Output file path (uses temp file if not specified)")),
		),
		s.handleScreenshot,
	)

	// record_video_start
	s.mcpServer.AddTool(
		mcp.NewTool("record_video_start",
			mcp.WithDescription("Start video recording on the iOS simulator"),
			mcp.WithString("device_id", mcp.Description("Simulator UDID (uses booted device if not specified)")),
			mcp.WithString("output_path", mcp.Description("Output file path (uses temp file if not specified)")),
		),
		s.handleRecordVideoStart,
	)

	// record_video_stop
	s.mcpServer.AddTool(
		mcp.NewTool("record_video_stop",
			mcp.WithDescription("Stop video recording and return the video file path"),
		),
		s.handleRecordVideoStop,
	)

	// open_url
	s.mcpServer.AddTool(
		mcp.NewTool("open_url",
			mcp.WithDescription("Open a URL in the simulator's default browser"),
			mcp.WithString("device_id", mcp.Description("Simulator UDID (uses booted device if not specified)")),
			mcp.WithString("url", mcp.Required(), mcp.Description("URL to open")),
		),
		s.handleOpenURL,
	)
}

// registerAppTools registers app management tools.
func (s *Server) registerAppTools() {
	// build_app
	s.mcpServer.AddTool(
		mcp.NewTool("build_app",
			mcp.WithDescription("Build an Xcode project for iOS simulator"),
			mcp.WithString("project_path", mcp.Required(), mcp.Description("Path to .xcodeproj or directory containing it")),
			mcp.WithString("scheme", mcp.Required(), mcp.Description("Build scheme name")),
			mcp.WithString("simulator", mcp.Description("Simulator name (default: iPhone 15)")),
			mcp.WithString("configuration", mcp.Description("Build configuration (default: Debug)")),
		),
		s.handleBuildApp,
	)

	// install_app
	s.mcpServer.AddTool(
		mcp.NewTool("install_app",
			mcp.WithDescription("Install an app bundle on the iOS simulator"),
			mcp.WithString("device_id", mcp.Description("Simulator UDID (uses booted device if not specified)")),
			mcp.WithString("app_path", mcp.Required(), mcp.Description("Path to the .app bundle")),
		),
		s.handleInstallApp,
	)

	// launch_app
	s.mcpServer.AddTool(
		mcp.NewTool("launch_app",
			mcp.WithDescription("Launch an app on the iOS simulator"),
			mcp.WithString("device_id", mcp.Description("Simulator UDID (uses booted device if not specified)")),
			mcp.WithString("bundle_id", mcp.Required(), mcp.Description("App bundle identifier")),
		),
		s.handleLaunchApp,
	)

	// terminate_app
	s.mcpServer.AddTool(
		mcp.NewTool("terminate_app",
			mcp.WithDescription("Terminate a running app on the iOS simulator"),
			mcp.WithString("device_id", mcp.Description("Simulator UDID (uses booted device if not specified)")),
			mcp.WithString("bundle_id", mcp.Required(), mcp.Description("App bundle identifier")),
		),
		s.handleTerminateApp,
	)

	// uninstall_app
	s.mcpServer.AddTool(
		mcp.NewTool("uninstall_app",
			mcp.WithDescription("Uninstall an app from the iOS simulator"),
			mcp.WithString("device_id", mcp.Description("Simulator UDID (uses booted device if not specified)")),
			mcp.WithString("bundle_id", mcp.Required(), mcp.Description("App bundle identifier")),
		),
		s.handleUninstallApp,
	)

	// list_schemes
	s.mcpServer.AddTool(
		mcp.NewTool("list_schemes",
			mcp.WithDescription("List available schemes in an Xcode project"),
			mcp.WithString("project_path", mcp.Required(), mcp.Description("Path to Xcode project or workspace")),
		),
		s.handleListSchemes,
	)
}

// registerUITools registers UI interaction tools (WebDriverAgent).
func (s *Server) registerUITools() {
	// wda_set_device - set target device for WDA
	s.mcpServer.AddTool(
		mcp.NewTool("wda_set_device",
			mcp.WithDescription("Set the target simulator device for WDA. Call this before using UI tools if you have multiple simulators."),
			mcp.WithString("device_id", mcp.Required(), mcp.Description("Simulator UDID from list_simulators")),
		),
		s.handleWDASetDevice,
	)

	// wda_status
	s.mcpServer.AddTool(
		mcp.NewTool("wda_status",
			mcp.WithDescription("Check WebDriverAgent status. WDA will be auto-started if not running."),
		),
		s.handleWDAStatus,
	)

	// wda_create_session
	s.mcpServer.AddTool(
		mcp.NewTool("wda_create_session",
			mcp.WithDescription("Create a new WebDriverAgent session. WDA will be auto-started if not running."),
		),
		s.handleWDACreateSession,
	)

	// get_ui_tree
	s.mcpServer.AddTool(
		mcp.NewTool("get_ui_tree",
			mcp.WithDescription("Get the UI hierarchy (accessibility tree) of the current screen. WDA will be auto-started if not running."),
			mcp.WithString("format", mcp.Description("Output format: 'xml' or 'json' (default: xml)")),
		),
		s.handleGetUITree,
	)

	// find_element
	s.mcpServer.AddTool(
		mcp.NewTool("find_element",
			mcp.WithDescription("Find a UI element by accessibility ID, name, class, or XPath. WDA will be auto-started if not running."),
			mcp.WithString("using", mcp.Required(), mcp.Description("Search strategy: 'accessibility id', 'name', 'class name', 'xpath', 'predicate string'")),
			mcp.WithString("value", mcp.Required(), mcp.Description("Value to search for")),
		),
		s.handleFindElement,
	)

	// tap
	s.mcpServer.AddTool(
		mcp.NewTool("tap",
			mcp.WithDescription("Tap at coordinates or on an element. WDA will be auto-started if not running."),
			mcp.WithNumber("x", mcp.Description("X coordinate (required if element_id not specified)")),
			mcp.WithNumber("y", mcp.Description("Y coordinate (required if element_id not specified)")),
			mcp.WithString("element_id", mcp.Description("Element ID from find_element (alternative to coordinates)")),
		),
		s.handleTap,
	)

	// long_press
	s.mcpServer.AddTool(
		mcp.NewTool("long_press",
			mcp.WithDescription("Long press at coordinates"),
			mcp.WithNumber("x", mcp.Required(), mcp.Description("X coordinate")),
			mcp.WithNumber("y", mcp.Required(), mcp.Description("Y coordinate")),
			mcp.WithNumber("duration", mcp.Description("Duration in seconds (default: 1.0)")),
		),
		s.handleLongPress,
	)

	// swipe
	s.mcpServer.AddTool(
		mcp.NewTool("swipe",
			mcp.WithDescription("Perform a swipe gesture"),
			mcp.WithString("direction", mcp.Description("Swipe direction: 'up', 'down', 'left', 'right'")),
			mcp.WithNumber("start_x", mcp.Description("Start X coordinate (required if direction not specified)")),
			mcp.WithNumber("start_y", mcp.Description("Start Y coordinate (required if direction not specified)")),
			mcp.WithNumber("end_x", mcp.Description("End X coordinate (required if direction not specified)")),
			mcp.WithNumber("end_y", mcp.Description("End Y coordinate (required if direction not specified)")),
			mcp.WithNumber("duration", mcp.Description("Swipe duration in seconds (default: 0.3)")),
		),
		s.handleSwipe,
	)

	// input_text
	s.mcpServer.AddTool(
		mcp.NewTool("input_text",
			mcp.WithDescription("Type text into the currently focused input field"),
			mcp.WithString("text", mcp.Required(), mcp.Description("Text to type")),
		),
		s.handleInputText,
	)

	// press_button
	s.mcpServer.AddTool(
		mcp.NewTool("press_button",
			mcp.WithDescription("Press a hardware button"),
			mcp.WithString("button", mcp.Required(), mcp.Description("Button name: 'home', 'volumeUp', 'volumeDown'")),
		),
		s.handlePressButton,
	)

	// get_elements_with_coords - parse UI tree and show tappable coordinates
	s.mcpServer.AddTool(
		mcp.NewTool("get_elements_with_coords",
			mcp.WithDescription("Get all visible UI elements with their tap coordinates (center point). Useful when accessibility labels are missing."),
			mcp.WithBoolean("visible_only", mcp.Description("Only show visible elements (default: true)")),
		),
		s.handleGetElementsWithCoords,
	)
}

// Tool handlers

func (s *Server) handleListSimulators(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	devices, err := s.simctl.ListDevices(ctx)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Format output as JSON
	output, err := json.MarshalIndent(devices, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to format output: %v", err)), nil
	}

	return mcp.NewToolResultText(string(output)), nil
}

func (s *Server) handleBootSimulator(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	deviceID := req.GetString("device_id", "")
	if deviceID == "" {
		return mcp.NewToolResultError("device_id is required"), nil
	}

	if err := s.simctl.Boot(ctx, deviceID); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Simulator %s booted successfully", deviceID)), nil
}

func (s *Server) handleShutdownSimulator(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	deviceID := req.GetString("device_id", "")
	if deviceID == "" {
		return mcp.NewToolResultError("device_id is required"), nil
	}

	if err := s.simctl.Shutdown(ctx, deviceID); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Simulator %s shut down successfully", deviceID)), nil
}

func (s *Server) handleScreenshot(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	deviceID := req.GetString("device_id", "")
	outputPath := req.GetString("output_path", "")

	// Get booted device if not specified
	if deviceID == "" {
		booted, err := s.simctl.GetBooted(ctx)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		if booted == "" {
			return mcp.NewToolResultError("no booted simulator found, specify device_id or boot a simulator first"), nil
		}
		deviceID = booted
	}

	path, err := s.simctl.Screenshot(ctx, deviceID, outputPath)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Screenshot saved to: %s", path)), nil
}

func (s *Server) handleRecordVideoStart(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	deviceID := req.GetString("device_id", "")
	outputPath := req.GetString("output_path", "")

	if deviceID == "" {
		booted, err := s.simctl.GetBooted(ctx)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		if booted == "" {
			return mcp.NewToolResultError("no booted simulator found"), nil
		}
		deviceID = booted
	}

	if err := s.simctl.StartRecording(ctx, deviceID, outputPath); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText("Video recording started"), nil
}

func (s *Server) handleRecordVideoStop(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path, err := s.simctl.StopRecording()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Recording saved to: %s", path)), nil
}

func (s *Server) handleOpenURL(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	deviceID := req.GetString("device_id", "")
	url := req.GetString("url", "")

	if url == "" {
		return mcp.NewToolResultError("url is required"), nil
	}

	if deviceID == "" {
		booted, err := s.simctl.GetBooted(ctx)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		if booted == "" {
			return mcp.NewToolResultError("no booted simulator found"), nil
		}
		deviceID = booted
	}

	if err := s.simctl.OpenURL(ctx, deviceID, url); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Opened URL: %s", url)), nil
}

func (s *Server) handleBuildApp(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectPath := req.GetString("project_path", "")
	scheme := req.GetString("scheme", "")
	simulator := req.GetString("simulator", "")
	configuration := req.GetString("configuration", "")

	if projectPath == "" || scheme == "" {
		return mcp.NewToolResultError("project_path and scheme are required"), nil
	}

	opts := BuildOptions{
		ProjectPath:   projectPath,
		Scheme:        scheme,
		SimulatorName: simulator,
		Configuration: configuration,
	}

	result, err := s.xcodebuild.Build(ctx, opts)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}

func (s *Server) handleInstallApp(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	deviceID := req.GetString("device_id", "")
	appPath := req.GetString("app_path", "")

	if appPath == "" {
		return mcp.NewToolResultError("app_path is required"), nil
	}

	if deviceID == "" {
		booted, err := s.simctl.GetBooted(ctx)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		if booted == "" {
			return mcp.NewToolResultError("no booted simulator found"), nil
		}
		deviceID = booted
	}

	if err := s.simctl.Install(ctx, deviceID, appPath); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("App installed successfully from: %s", appPath)), nil
}

func (s *Server) handleLaunchApp(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	deviceID := req.GetString("device_id", "")
	bundleID := req.GetString("bundle_id", "")

	if bundleID == "" {
		return mcp.NewToolResultError("bundle_id is required"), nil
	}

	if deviceID == "" {
		booted, err := s.simctl.GetBooted(ctx)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		if booted == "" {
			return mcp.NewToolResultError("no booted simulator found"), nil
		}
		deviceID = booted
	}

	if err := s.simctl.Launch(ctx, deviceID, bundleID); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("App %s launched successfully", bundleID)), nil
}

func (s *Server) handleTerminateApp(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	deviceID := req.GetString("device_id", "")
	bundleID := req.GetString("bundle_id", "")

	if bundleID == "" {
		return mcp.NewToolResultError("bundle_id is required"), nil
	}

	if deviceID == "" {
		booted, err := s.simctl.GetBooted(ctx)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		if booted == "" {
			return mcp.NewToolResultError("no booted simulator found"), nil
		}
		deviceID = booted
	}

	if err := s.simctl.Terminate(ctx, deviceID, bundleID); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("App %s terminated successfully", bundleID)), nil
}

func (s *Server) handleUninstallApp(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	deviceID := req.GetString("device_id", "")
	bundleID := req.GetString("bundle_id", "")

	if bundleID == "" {
		return mcp.NewToolResultError("bundle_id is required"), nil
	}

	if deviceID == "" {
		booted, err := s.simctl.GetBooted(ctx)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		if booted == "" {
			return mcp.NewToolResultError("no booted simulator found"), nil
		}
		deviceID = booted
	}

	if err := s.simctl.Uninstall(ctx, deviceID, bundleID); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("App %s uninstalled successfully", bundleID)), nil
}

func (s *Server) handleListSchemes(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectPath := req.GetString("project_path", "")

	if projectPath == "" {
		return mcp.NewToolResultError("project_path is required"), nil
	}

	schemes, err := s.xcodebuild.ListSchemes(ctx, projectPath)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	output, _ := json.MarshalIndent(schemes, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}

// UI Tool Handlers (WebDriverAgent)

// getWDAClient returns a WDA client, auto-starting WDA if necessary.
func (s *Server) getWDAClient(ctx context.Context) (*wda.Client, error) {
	return s.wdaManager.GetClient(ctx)
}

func (s *Server) handleWDASetDevice(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	deviceID := req.GetString("device_id", "")
	if deviceID == "" {
		return mcp.NewToolResultError("device_id is required"), nil
	}

	s.wdaManager.SetDeviceID(deviceID)
	return mcp.NewToolResultText(fmt.Sprintf("WDA target device set to: %s", deviceID)), nil
}

func (s *Server) handleWDAStatus(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := s.getWDAClient(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("WDA not available: %v", err)), nil
	}

	status, err := client.Status(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("WDA not available: %v. Make sure WebDriverAgent is running.", err)), nil
	}

	output, _ := json.MarshalIndent(status, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}

func (s *Server) handleWDACreateSession(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := s.getWDAClient(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get WDA client: %v", err)), nil
	}

	session, err := client.CreateSession(ctx)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	output, _ := json.MarshalIndent(session, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}

func (s *Server) handleGetUITree(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	format := req.GetString("format", "xml")

	client, err := s.getWDAClient(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to start WDA: %v", err)), nil
	}

	if client.GetSessionID() == "" {
		if _, err := client.CreateSession(ctx); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to create WDA session: %v", err)), nil
		}
	}

	var source string
	if format == "json" {
		source, err = client.SourceAccessible(ctx)
	} else {
		source, err = client.Source(ctx)
	}

	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(source), nil
}

func (s *Server) handleFindElement(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	using := req.GetString("using", "")
	value := req.GetString("value", "")

	if using == "" || value == "" {
		return mcp.NewToolResultError("using and value are required"), nil
	}

	client, err := s.getWDAClient(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to start WDA: %v", err)), nil
	}

	if client.GetSessionID() == "" {
		if _, err := client.CreateSession(ctx); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to create WDA session: %v", err)), nil
		}
	}

	element, err := client.FindElement(ctx, using, value)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Get element details
	rect, _ := client.GetElementRect(ctx, element.ElementID)

	result := map[string]any{
		"element_id": element.ElementID,
	}
	if rect != nil {
		result["rect"] = rect
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}

func (s *Server) handleTap(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	x := req.GetFloat("x", -1)
	y := req.GetFloat("y", -1)
	elementID := req.GetString("element_id", "")

	client, err := s.getWDAClient(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to start WDA: %v", err)), nil
	}

	if client.GetSessionID() == "" {
		if _, sessErr := client.CreateSession(ctx); sessErr != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to create WDA session: %v", sessErr)), nil
		}
	}

	if elementID != "" {
		err = client.Click(ctx, elementID)
	} else if x >= 0 && y >= 0 {
		err = client.Tap(ctx, int(x), int(y))
	} else {
		return mcp.NewToolResultError("either element_id or both x and y coordinates are required"), nil
	}

	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText("Tap successful"), nil
}

func (s *Server) handleLongPress(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	x := req.GetFloat("x", 0)
	y := req.GetFloat("y", 0)
	duration := req.GetFloat("duration", 1.0)

	client, err := s.getWDAClient(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to start WDA: %v", err)), nil
	}

	if client.GetSessionID() == "" {
		if _, err := client.CreateSession(ctx); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to create WDA session: %v", err)), nil
		}
	}

	if err := client.LongPress(ctx, int(x), int(y), duration); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText("Long press successful"), nil
}

func (s *Server) handleSwipe(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	direction := req.GetString("direction", "")
	startX := req.GetFloat("start_x", 0)
	startY := req.GetFloat("start_y", 0)
	endX := req.GetFloat("end_x", 0)
	endY := req.GetFloat("end_y", 0)
	duration := req.GetFloat("duration", 0.3)

	client, err := s.getWDAClient(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to start WDA: %v", err)), nil
	}

	if client.GetSessionID() == "" {
		if _, err := client.CreateSession(ctx); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to create WDA session: %v", err)), nil
		}
	}

	// Get window size for direction-based swipes
	if direction != "" {
		size, err := client.WindowSize(ctx)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to get window size: %v", err)), nil
		}

		centerX := size.Width / 2
		centerY := size.Height / 2
		offset := size.Height / 4

		switch direction {
		case "up":
			startX, startY = float64(centerX), float64(centerY+offset)
			endX, endY = float64(centerX), float64(centerY-offset)
		case "down":
			startX, startY = float64(centerX), float64(centerY-offset)
			endX, endY = float64(centerX), float64(centerY+offset)
		case "left":
			startX, startY = float64(centerX+size.Width/4), float64(centerY)
			endX, endY = float64(centerX-size.Width/4), float64(centerY)
		case "right":
			startX, startY = float64(centerX-size.Width/4), float64(centerY)
			endX, endY = float64(centerX+size.Width/4), float64(centerY)
		default:
			return mcp.NewToolResultError("invalid direction, use: up, down, left, right"), nil
		}
	}

	if err := client.Swipe(ctx, int(startX), int(startY), int(endX), int(endY), duration); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText("Swipe successful"), nil
}

func (s *Server) handleInputText(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	text := req.GetString("text", "")

	if text == "" {
		return mcp.NewToolResultError("text is required"), nil
	}

	client, err := s.getWDAClient(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to start WDA: %v", err)), nil
	}

	if client.GetSessionID() == "" {
		if _, err := client.CreateSession(ctx); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to create WDA session: %v", err)), nil
		}
	}

	if err := client.SendKeys(ctx, text); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Typed: %s", text)), nil
}

func (s *Server) handlePressButton(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	button := req.GetString("button", "")

	if button == "" {
		return mcp.NewToolResultError("button is required"), nil
	}

	client, err := s.getWDAClient(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to start WDA: %v", err)), nil
	}

	if client.GetSessionID() == "" {
		if _, err := client.CreateSession(ctx); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to create WDA session: %v", err)), nil
		}
	}

	if err := client.PressButton(ctx, button); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Pressed button: %s", button)), nil
}

// UIElement represents a parsed UI element with coordinates
type UIElement struct {
	Type    string `json:"type"`
	Name    string `json:"name,omitempty"`
	Label   string `json:"label,omitempty"`
	Value   string `json:"value,omitempty"`
	Visible bool   `json:"visible"`
	X       int    `json:"x"`
	Y       int    `json:"y"`
	Width   int    `json:"width"`
	Height  int    `json:"height"`
	TapX    int    `json:"tap_x"` // Center X coordinate for tapping
	TapY    int    `json:"tap_y"` // Center Y coordinate for tapping
}

func (s *Server) handleGetElementsWithCoords(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	visibleOnly := req.GetBool("visible_only", true)

	client, err := s.getWDAClient(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to start WDA: %v", err)), nil
	}

	if client.GetSessionID() == "" {
		if _, err := client.CreateSession(ctx); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to create WDA session: %v", err)), nil
		}
	}

	// Get XML source
	source, err := client.Source(ctx)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Parse XML using decoder for flexible element names
	var elements []UIElement
	decoder := xml.NewDecoder(strings.NewReader(source))
	parseXMLElements(decoder, &elements, visibleOnly, 0)

	// Format output
	var output strings.Builder
	fmt.Fprintf(&output, "Found %d elements with coordinates:\n\n", len(elements))

	for i, el := range elements {
		name := el.Name
		if name == "" {
			name = el.Label
		}

		// Shorten type name for readability
		shortType := strings.TrimPrefix(el.Type, "XCUIElementType")
		if shortType == "" {
			shortType = el.Type
		}

		fmt.Fprintf(&output, "%d. [%s]", i+1, shortType)
		if name != "" {
			fmt.Fprintf(&output, " \"%s\"", name)
		}
		fmt.Fprintf(&output, "\n   Tap: (%d, %d)  Rect: %dx%d at (%d,%d)\n\n",
			el.TapX, el.TapY, el.Width, el.Height, el.X, el.Y)
	}

	return mcp.NewToolResultText(output.String()), nil
}

// parseXMLElements recursively parses WDA XML using a streaming decoder
func parseXMLElements(decoder *xml.Decoder, elements *[]UIElement, visibleOnly bool, depth int) {
	for {
		token, err := decoder.Token()
		if err != nil {
			return
		}

		switch t := token.(type) {
		case xml.StartElement:
			// Extract attributes
			attrs := make(map[string]string)
			for _, attr := range t.Attr {
				attrs[attr.Name.Local] = attr.Value
			}

			// Check visibility
			visible := attrs["visible"] == "true"
			if visibleOnly && !visible && depth > 0 {
				// Skip this element but still need to consume its content
				decoder.Skip()
				continue
			}

			// Parse coordinates
			x, _ := strconv.Atoi(attrs["x"])
			y, _ := strconv.Atoi(attrs["y"])
			w, _ := strconv.Atoi(attrs["width"])
			h, _ := strconv.Atoi(attrs["height"])

			elementType := t.Name.Local // Element tag name IS the type
			name := attrs["name"]
			label := attrs["label"]

			// Add element if it has size
			if w > 0 && h > 0 {
				// Filter to interesting elements
				isInteresting := name != "" || label != "" ||
					strings.Contains(elementType, "Button") ||
					strings.Contains(elementType, "TextField") ||
					strings.Contains(elementType, "Text") ||
					strings.Contains(elementType, "Image") ||
					strings.Contains(elementType, "Cell") ||
					strings.Contains(elementType, "Switch") ||
					strings.Contains(elementType, "Slider") ||
					strings.Contains(elementType, "ScrollView") ||
					strings.Contains(elementType, "Table") ||
					depth <= 2

				if isInteresting {
					*elements = append(*elements, UIElement{
						Type:    elementType,
						Name:    name,
						Label:   label,
						Value:   attrs["value"],
						Visible: visible,
						X:       x,
						Y:       y,
						Width:   w,
						Height:  h,
						TapX:    x + w/2,
						TapY:    y + h/2,
					})
				}
			}

			// Recursively parse children
			parseXMLElements(decoder, elements, visibleOnly, depth+1)

		case xml.EndElement:
			return
		}
	}
}
