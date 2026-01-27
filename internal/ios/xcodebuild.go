package ios

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// XcodeBuild provides methods to interact with xcodebuild commands.
type XcodeBuild struct{}

// NewXcodeBuild creates a new XcodeBuild instance.
func NewXcodeBuild() *XcodeBuild {
	return &XcodeBuild{}
}

// BuildOptions contains options for building an Xcode project.
type BuildOptions struct {
	ProjectPath    string // Path to .xcodeproj or directory containing it
	WorkspacePath  string // Path to .xcworkspace (takes precedence over ProjectPath)
	Scheme         string // Build scheme name
	Configuration  string // Build configuration (Debug, Release), default: Debug
	SimulatorName  string // Simulator name, default: iPhone 15
	DerivedDataPath string // Custom derived data path
}

// Build builds an Xcode project for iOS simulator.
// Returns the path to the built .app bundle.
func (x *XcodeBuild) Build(ctx context.Context, opts BuildOptions) (*BuildResult, error) {
	args := []string{}

	// Add workspace or project
	if opts.WorkspacePath != "" {
		args = append(args, "-workspace", opts.WorkspacePath)
	} else if opts.ProjectPath != "" {
		// Check if it's a directory or .xcodeproj file
		if !strings.HasSuffix(opts.ProjectPath, ".xcodeproj") {
			// Try to find .xcodeproj in directory
			matches, err := filepath.Glob(filepath.Join(opts.ProjectPath, "*.xcodeproj"))
			if err != nil || len(matches) == 0 {
				return nil, fmt.Errorf("no .xcodeproj found in %s", opts.ProjectPath)
			}
			opts.ProjectPath = matches[0]
		}
		args = append(args, "-project", opts.ProjectPath)
	} else {
		return nil, fmt.Errorf("either ProjectPath or WorkspacePath must be specified")
	}

	// Add scheme
	if opts.Scheme == "" {
		return nil, fmt.Errorf("scheme is required")
	}
	args = append(args, "-scheme", opts.Scheme)

	// Add configuration
	config := opts.Configuration
	if config == "" {
		config = "Debug"
	}
	args = append(args, "-configuration", config)

	// Add destination
	simulator := opts.SimulatorName
	if simulator == "" {
		simulator = "iPhone 15"
	}
	destination := fmt.Sprintf("platform=iOS Simulator,name=%s", simulator)
	args = append(args, "-destination", destination)

	// Add derived data path if specified
	derivedDataPath := opts.DerivedDataPath
	if derivedDataPath == "" {
		derivedDataPath = filepath.Join(filepath.Dir(opts.ProjectPath), "build")
	}
	args = append(args, "-derivedDataPath", derivedDataPath)

	// Build only (no run)
	args = append(args, "build")

	// Run xcodebuild
	cmd := exec.CommandContext(ctx, "xcodebuild", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("xcodebuild failed: %s\n%s", err.Error(), stderr.String())
	}

	// Find the built .app path from build output
	appPath, err := x.findAppPath(stdout.String(), derivedDataPath, opts.Scheme, config)
	if err != nil {
		return nil, err
	}

	// Extract bundle ID from Info.plist
	bundleID, _ := x.getBundleID(appPath)

	return &BuildResult{
		AppPath:  appPath,
		BundleID: bundleID,
		Scheme:   opts.Scheme,
		BuildDir: derivedDataPath,
	}, nil
}

// findAppPath finds the .app bundle path from build output or derived data.
func (x *XcodeBuild) findAppPath(output, derivedDataPath, scheme, config string) (string, error) {
	// Try to find from build settings output
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, ".app") && strings.Contains(line, "Build/Products") {
			// Extract path
			if idx := strings.Index(line, "/"); idx >= 0 {
				path := strings.TrimSpace(line[idx:])
				if strings.HasSuffix(path, ".app") {
					return path, nil
				}
			}
		}
	}

	// Fall back to standard location
	pattern := filepath.Join(
		derivedDataPath,
		"Build/Products",
		fmt.Sprintf("%s-iphonesimulator", config),
		"*.app",
	)
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return "", fmt.Errorf("failed to find app bundle: %w", err)
	}
	if len(matches) == 0 {
		return "", fmt.Errorf("no .app bundle found in %s", pattern)
	}

	return matches[0], nil
}

// getBundleID extracts the bundle ID from an app's Info.plist.
func (x *XcodeBuild) getBundleID(appPath string) (string, error) {
	plistPath := filepath.Join(appPath, "Info.plist")
	cmd := exec.Command("defaults", "read", plistPath, "CFBundleIdentifier")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to read bundle ID: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// ListSchemes lists available schemes in a project/workspace.
func (x *XcodeBuild) ListSchemes(ctx context.Context, projectPath string) ([]string, error) {
	args := []string{"-list", "-json"}

	if strings.HasSuffix(projectPath, ".xcworkspace") {
		args = append(args, "-workspace", projectPath)
	} else {
		if !strings.HasSuffix(projectPath, ".xcodeproj") {
			matches, err := filepath.Glob(filepath.Join(projectPath, "*.xcodeproj"))
			if err != nil || len(matches) == 0 {
				// Try workspace
				matches, err = filepath.Glob(filepath.Join(projectPath, "*.xcworkspace"))
				if err != nil || len(matches) == 0 {
					return nil, fmt.Errorf("no Xcode project found in %s", projectPath)
				}
				args = append(args, "-workspace", matches[0])
			} else {
				args = append(args, "-project", matches[0])
			}
		} else {
			args = append(args, "-project", projectPath)
		}
	}

	cmd := exec.CommandContext(ctx, "xcodebuild", args...)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("xcodebuild -list failed: %w", err)
	}

	// Parse JSON output (simplified - just extract scheme names)
	var schemes []string
	lines := strings.Split(string(out), "\n")
	inSchemes := false
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, `"schemes"`) {
			inSchemes = true
			continue
		}
		if inSchemes {
			if line == "]" || line == "]," {
				break
			}
			scheme := strings.Trim(line, `",`)
			if scheme != "" && scheme != "[" {
				schemes = append(schemes, scheme)
			}
		}
	}

	return schemes, nil
}

// Clean cleans the build artifacts.
func (x *XcodeBuild) Clean(ctx context.Context, opts BuildOptions) error {
	args := []string{}

	if opts.WorkspacePath != "" {
		args = append(args, "-workspace", opts.WorkspacePath)
	} else if opts.ProjectPath != "" {
		args = append(args, "-project", opts.ProjectPath)
	}

	if opts.Scheme != "" {
		args = append(args, "-scheme", opts.Scheme)
	}

	args = append(args, "clean")

	cmd := exec.CommandContext(ctx, "xcodebuild", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("xcodebuild clean failed: %s", stderr.String())
	}
	return nil
}
