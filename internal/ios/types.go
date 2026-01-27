// Package ios provides MCP server functionality for iOS simulator automation.
package ios

// Device represents an iOS simulator device.
type Device struct {
	UDID             string `json:"udid"`
	Name             string `json:"name"`
	State            string `json:"state"`
	IsAvailable      bool   `json:"isAvailable"`
	DeviceTypeID     string `json:"deviceTypeIdentifier"`
	RuntimeID        string `json:"runtimeIdentifier,omitempty"`
	RuntimeName      string `json:"runtimeName,omitempty"`
	LastBootedAt     string `json:"lastBootedAt,omitempty"`
	DataPath         string `json:"dataPath,omitempty"`
	LogPath          string `json:"logPath,omitempty"`
	DataPathSize     int64  `json:"dataPathSize,omitempty"`
	AvailabilityError string `json:"availabilityError,omitempty"`
}

// DeviceList represents the JSON output from simctl list devices.
type DeviceList struct {
	Devices map[string][]Device `json:"devices"`
}

// Runtime represents an iOS simulator runtime.
type Runtime struct {
	BuildVersion       string `json:"buildversion"`
	BundlePath         string `json:"bundlePath"`
	Identifier         string `json:"identifier"`
	IsAvailable        bool   `json:"isAvailable"`
	IsInternal         bool   `json:"isInternal"`
	Name               string `json:"name"`
	Platform           string `json:"platform"`
	SupportedDeviceTypes []struct {
		BundlePath string `json:"bundlePath"`
		Name       string `json:"name"`
		Identifier string `json:"identifier"`
	} `json:"supportedDeviceTypes"`
	Version string `json:"version"`
}

// RuntimeList represents the JSON output from simctl list runtimes.
type RuntimeList struct {
	Runtimes []Runtime `json:"runtimes"`
}

// BuildResult contains information about a successful Xcode build.
type BuildResult struct {
	AppPath   string `json:"appPath"`
	BundleID  string `json:"bundleId"`
	Scheme    string `json:"scheme"`
	BuildDir  string `json:"buildDir"`
}

// RecordingState tracks video recording state.
type RecordingState struct {
	IsRecording bool
	OutputPath  string
	ProcessID   int
}
