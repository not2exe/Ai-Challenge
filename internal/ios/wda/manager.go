package wda

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Manager handles WDA lifecycle - finding, starting, and managing WebDriverAgent.
type Manager struct {
	mu          sync.Mutex
	client      *Client
	wdaProcess  *exec.Cmd
	wdaPath     string
	port        int
	deviceID    string
	isRunning   bool
	startupWait time.Duration
}

// NewManager creates a new WDA manager.
func NewManager(port int) *Manager {
	if port == 0 {
		port = 8100
	}
	return &Manager{
		port:        port,
		startupWait: 30 * time.Second,
	}
}

// SetDeviceID sets the target simulator device ID.
func (m *Manager) SetDeviceID(deviceID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.deviceID = deviceID
}

// GetClient returns a WDA client, starting WDA if necessary.
func (m *Manager) GetClient(ctx context.Context) (*Client, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if we already have a working client
	if m.client != nil && m.isRunning {
		// Verify it's still responding
		if _, err := m.client.Status(ctx); err == nil {
			return m.client, nil
		}
		// Not responding, need to restart
		m.isRunning = false
	}

	// Try to connect to existing WDA
	m.client = NewClient("localhost", m.port)
	if _, err := m.client.Status(ctx); err == nil {
		m.isRunning = true
		return m.client, nil
	}

	// WDA not running, try to start it
	if err := m.startWDA(ctx); err != nil {
		return nil, fmt.Errorf("failed to start WDA: %w", err)
	}

	return m.client, nil
}

// startWDA finds and starts WebDriverAgent.
func (m *Manager) startWDA(ctx context.Context) error {
	// Find WDA project
	wdaPath, err := m.findWDAProject()
	if err != nil {
		return err
	}
	m.wdaPath = wdaPath

	// Get device ID if not set
	deviceID := m.deviceID
	if deviceID == "" {
		// Try to find a booted simulator
		deviceID, err = m.findBootedSimulator(ctx)
		if err != nil {
			return fmt.Errorf("no device ID set and no booted simulator found: %w", err)
		}
		m.deviceID = deviceID
	}

	// Build WDA first (in case it needs compilation)
	buildCmd := exec.CommandContext(ctx, "xcodebuild",
		"-project", wdaPath,
		"-scheme", "WebDriverAgentRunner",
		"-destination", fmt.Sprintf("platform=iOS Simulator,id=%s", deviceID),
		"build-for-testing",
	)
	buildCmd.Stdout = nil
	buildCmd.Stderr = nil
	if err := buildCmd.Run(); err != nil {
		return fmt.Errorf("failed to build WDA: %w", err)
	}

	// Start WDA test runner
	m.wdaProcess = exec.Command("xcodebuild",
		"-project", wdaPath,
		"-scheme", "WebDriverAgentRunner",
		"-destination", fmt.Sprintf("platform=iOS Simulator,id=%s", deviceID),
		"test-without-building",
	)
	m.wdaProcess.Stdout = nil
	m.wdaProcess.Stderr = nil

	if err := m.wdaProcess.Start(); err != nil {
		return fmt.Errorf("failed to start WDA: %w", err)
	}

	// Wait for WDA to be ready
	return m.waitForWDA(ctx)
}

// waitForWDA waits for WDA to become available.
func (m *Manager) waitForWDA(ctx context.Context) error {
	deadline := time.Now().Add(m.startupWait)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if time.Now().After(deadline) {
				return fmt.Errorf("WDA startup timeout after %v", m.startupWait)
			}

			if _, err := m.client.Status(ctx); err == nil {
				m.isRunning = true
				return nil
			}
		}
	}
}

// findWDAProject searches for WebDriverAgent.xcodeproj in common locations.
func (m *Manager) findWDAProject() (string, error) {
	// If already set, use it
	if m.wdaPath != "" {
		if _, err := os.Stat(m.wdaPath); err == nil {
			return m.wdaPath, nil
		}
	}

	homeDir, _ := os.UserHomeDir()

	// Common WDA locations
	searchPaths := []string{
		// Appium xcuitest driver location
		filepath.Join(homeDir, ".appium", "node_modules", "appium-xcuitest-driver", "node_modules", "appium-webdriveragent", "WebDriverAgent.xcodeproj"),
		// Global npm location
		"/usr/local/lib/node_modules/appium/node_modules/appium-webdriveragent/WebDriverAgent.xcodeproj",
		// Homebrew location
		"/opt/homebrew/lib/node_modules/appium/node_modules/appium-webdriveragent/WebDriverAgent.xcodeproj",
		// Manual installation
		filepath.Join(homeDir, "WebDriverAgent", "WebDriverAgent.xcodeproj"),
		// Current directory
		"WebDriverAgent/WebDriverAgent.xcodeproj",
	}

	for _, path := range searchPaths {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	// Try to find via locate or find command
	if path, err := m.findWDAViaGlob(homeDir); err == nil {
		return path, nil
	}

	return "", fmt.Errorf("WebDriverAgent.xcodeproj not found. Install it via: npm install -g appium && appium driver install xcuitest")
}

// findWDAViaGlob tries to find WDA using glob patterns.
func (m *Manager) findWDAViaGlob(homeDir string) (string, error) {
	patterns := []string{
		filepath.Join(homeDir, ".appium", "**", "WebDriverAgent.xcodeproj"),
		filepath.Join(homeDir, "*", "WebDriverAgent.xcodeproj"),
	}

	for _, pattern := range patterns {
		matches, err := filepath.Glob(pattern)
		if err == nil && len(matches) > 0 {
			return matches[0], nil
		}
	}

	return "", fmt.Errorf("not found via glob")
}

// findBootedSimulator finds a booted iOS simulator.
func (m *Manager) findBootedSimulator(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "xcrun", "simctl", "list", "devices", "-j")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	// Parse JSON to find booted device
	var result struct {
		Devices map[string][]struct {
			UDID  string `json:"udid"`
			State string `json:"state"`
			Name  string `json:"name"`
		} `json:"devices"`
	}

	if err := jsonUnmarshal(output, &result); err != nil {
		return "", err
	}

	for _, devices := range result.Devices {
		for _, device := range devices {
			if strings.EqualFold(device.State, "booted") {
				return device.UDID, nil
			}
		}
	}

	return "", fmt.Errorf("no booted simulator found")
}

// Stop stops the WDA process if running.
func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.wdaProcess != nil && m.wdaProcess.Process != nil {
		if err := m.wdaProcess.Process.Kill(); err != nil {
			return err
		}
		m.wdaProcess = nil
	}

	m.isRunning = false
	return nil
}

// IsRunning returns whether WDA is currently running.
func (m *Manager) IsRunning() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.isRunning
}

// GetWDAPath returns the path to WebDriverAgent.xcodeproj.
func (m *Manager) GetWDAPath() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.wdaPath
}

// SetWDAPath sets a custom WDA path.
func (m *Manager) SetWDAPath(path string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.wdaPath = path
}

// Helper for JSON unmarshaling
func jsonUnmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}
