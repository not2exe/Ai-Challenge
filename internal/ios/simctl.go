package ios

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
)

// SimCtl provides methods to interact with xcrun simctl commands.
type SimCtl struct {
	mu              sync.Mutex
	activeRecording *activeRecording
}

type activeRecording struct {
	deviceID   string
	outputPath string
	cmd        *exec.Cmd
}

// NewSimCtl creates a new SimCtl instance.
func NewSimCtl() *SimCtl {
	return &SimCtl{}
}

// ListDevices returns all available iOS simulators.
func (s *SimCtl) ListDevices(ctx context.Context) ([]Device, error) {
	cmd := exec.CommandContext(ctx, "xcrun", "simctl", "list", "devices", "-j")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("simctl list devices failed: %w", err)
	}

	var deviceList DeviceList
	if err := json.Unmarshal(out, &deviceList); err != nil {
		return nil, fmt.Errorf("failed to parse devices JSON: %w", err)
	}

	// Flatten devices from all runtimes
	var devices []Device
	for runtime, devs := range deviceList.Devices {
		for _, d := range devs {
			d.RuntimeID = runtime
			// Extract runtime name from identifier
			parts := strings.Split(runtime, ".")
			if len(parts) > 0 {
				d.RuntimeName = parts[len(parts)-1]
			}
			devices = append(devices, d)
		}
	}

	return devices, nil
}

// ListRuntimes returns all available iOS simulator runtimes.
func (s *SimCtl) ListRuntimes(ctx context.Context) ([]Runtime, error) {
	cmd := exec.CommandContext(ctx, "xcrun", "simctl", "list", "runtimes", "-j")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("simctl list runtimes failed: %w", err)
	}

	var runtimeList RuntimeList
	if err := json.Unmarshal(out, &runtimeList); err != nil {
		return nil, fmt.Errorf("failed to parse runtimes JSON: %w", err)
	}

	return runtimeList.Runtimes, nil
}

// Boot boots a simulator by UDID or name.
func (s *SimCtl) Boot(ctx context.Context, deviceID string) error {
	cmd := exec.CommandContext(ctx, "xcrun", "simctl", "boot", deviceID)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// Check if already booted
		if strings.Contains(stderr.String(), "current state: Booted") {
			return nil // Already booted, not an error
		}
		return fmt.Errorf("simctl boot failed: %s", stderr.String())
	}
	return nil
}

// Shutdown shuts down a simulator.
func (s *SimCtl) Shutdown(ctx context.Context, deviceID string) error {
	cmd := exec.CommandContext(ctx, "xcrun", "simctl", "shutdown", deviceID)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// Check if already shutdown
		if strings.Contains(stderr.String(), "current state: Shutdown") {
			return nil // Already shutdown, not an error
		}
		return fmt.Errorf("simctl shutdown failed: %s", stderr.String())
	}
	return nil
}

// Screenshot takes a screenshot of the simulator.
// If outputPath is empty, a temporary file is created.
// Returns the path to the screenshot file.
func (s *SimCtl) Screenshot(ctx context.Context, deviceID string, outputPath string) (string, error) {
	if outputPath == "" {
		// Create temp file with timestamp
		timestamp := time.Now().Format("20060102_150405")
		outputPath = filepath.Join(os.TempDir(), fmt.Sprintf("ios_screenshot_%s.png", timestamp))
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}

	cmd := exec.CommandContext(ctx, "xcrun", "simctl", "io", deviceID, "screenshot", outputPath)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("simctl screenshot failed: %s", stderr.String())
	}

	return outputPath, nil
}

// StartRecording starts video recording on the simulator.
func (s *SimCtl) StartRecording(ctx context.Context, deviceID string, outputPath string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.activeRecording != nil {
		return fmt.Errorf("recording already in progress for device %s", s.activeRecording.deviceID)
	}

	if outputPath == "" {
		timestamp := time.Now().Format("20060102_150405")
		outputPath = filepath.Join(os.TempDir(), fmt.Sprintf("ios_recording_%s.mov", timestamp))
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Start recording in background
	cmd := exec.Command("xcrun", "simctl", "io", deviceID, "recordVideo", outputPath)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start recording: %w", err)
	}

	s.activeRecording = &activeRecording{
		deviceID:   deviceID,
		outputPath: outputPath,
		cmd:        cmd,
	}

	return nil
}

// StopRecording stops the current video recording.
// Returns the path to the recorded video.
func (s *SimCtl) StopRecording() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.activeRecording == nil {
		return "", fmt.Errorf("no recording in progress")
	}

	// Send SIGINT to stop recording gracefully
	if s.activeRecording.cmd.Process != nil {
		if err := s.activeRecording.cmd.Process.Signal(syscall.SIGINT); err != nil {
			return "", fmt.Errorf("failed to stop recording: %w", err)
		}
	}

	// Wait for process to finish
	_ = s.activeRecording.cmd.Wait()

	outputPath := s.activeRecording.outputPath
	s.activeRecording = nil

	return outputPath, nil
}

// IsRecording returns whether a recording is in progress.
func (s *SimCtl) IsRecording() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.activeRecording != nil
}

// Install installs an app bundle on the simulator.
func (s *SimCtl) Install(ctx context.Context, deviceID string, appPath string) error {
	cmd := exec.CommandContext(ctx, "xcrun", "simctl", "install", deviceID, appPath)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("simctl install failed: %s", stderr.String())
	}
	return nil
}

// Launch launches an app on the simulator.
func (s *SimCtl) Launch(ctx context.Context, deviceID string, bundleID string) error {
	cmd := exec.CommandContext(ctx, "xcrun", "simctl", "launch", deviceID, bundleID)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("simctl launch failed: %s", stderr.String())
	}
	return nil
}

// Terminate terminates an app on the simulator.
func (s *SimCtl) Terminate(ctx context.Context, deviceID string, bundleID string) error {
	cmd := exec.CommandContext(ctx, "xcrun", "simctl", "terminate", deviceID, bundleID)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("simctl terminate failed: %s", stderr.String())
	}
	return nil
}

// Uninstall uninstalls an app from the simulator.
func (s *SimCtl) Uninstall(ctx context.Context, deviceID string, bundleID string) error {
	cmd := exec.CommandContext(ctx, "xcrun", "simctl", "uninstall", deviceID, bundleID)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("simctl uninstall failed: %s", stderr.String())
	}
	return nil
}

// GetBooted returns the UDID of the first booted simulator, or empty if none.
func (s *SimCtl) GetBooted(ctx context.Context) (string, error) {
	devices, err := s.ListDevices(ctx)
	if err != nil {
		return "", err
	}

	for _, d := range devices {
		if d.State == "Booted" && d.IsAvailable {
			return d.UDID, nil
		}
	}
	return "", nil
}

// OpenURL opens a URL in the simulator's default browser.
func (s *SimCtl) OpenURL(ctx context.Context, deviceID string, url string) error {
	cmd := exec.CommandContext(ctx, "xcrun", "simctl", "openurl", deviceID, url)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("simctl openurl failed: %s", stderr.String())
	}
	return nil
}

// StatusBarOverride overrides the status bar on the simulator.
func (s *SimCtl) StatusBarOverride(ctx context.Context, deviceID string, time string, battery string) error {
	args := []string{"simctl", "status_bar", deviceID, "override"}
	if time != "" {
		args = append(args, "--time", time)
	}
	if battery != "" {
		args = append(args, "--batteryLevel", battery)
	}

	cmd := exec.CommandContext(ctx, "xcrun", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("simctl status_bar failed: %s", stderr.String())
	}
	return nil
}
