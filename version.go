package xpath

import (
	"fmt"
	"runtime"
	"strconv"
	"strings"
)

const (
	// Version represents the current version of the library.
	Version = "1.7.1"

	// APIVersion represents the API version for compatibility tracking.
	APIVersion = "v1"

	// MinSupportedGoVersion is the minimum Go version required by this module.
	MinSupportedGoVersion = "1.21"
)

var (
	// GitCommit is populated by release builds. It is empty for ordinary library
	// builds and `go test`.
	GitCommit string
	// BuildTime is populated by release builds as an RFC 3339 timestamp. It is
	// empty for ordinary library builds and `go test`.
	BuildTime string
)

// BuildInfo contains build and version information.
type BuildInfo struct {
	Version    string `json:"version"`
	APIVersion string `json:"api_version"`
	GoVersion  string `json:"go_version"`
	GitCommit  string `json:"git_commit,omitempty"`
	BuildTime  string `json:"build_time,omitempty"`
	Platform   string `json:"platform"`
	Compiler   string `json:"compiler"`
}

// GetBuildInfo returns build information.
func GetBuildInfo() BuildInfo {
	return BuildInfo{
		Version:    Version,
		APIVersion: APIVersion,
		GoVersion:  runtime.Version(),
		GitCommit:  GitCommit,
		BuildTime:  BuildTime,
		Platform:   fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
		Compiler:   runtime.Compiler,
	}
}

// IsCompatible checks if the given API version is compatible.
func IsCompatible(apiVersion string) bool {
	return apiVersion == APIVersion || apiVersion == "v1.0" || apiVersion == "1.0"
}

// CheckGoVersion verifies the Go runtime version meets the module's minimum
// requirement. Development toolchains are accepted because their version is
// not stable enough for a meaningful lower-bound comparison.
func CheckGoVersion() error {
	version := runtime.Version()
	if strings.HasPrefix(version, "devel ") {
		return nil
	}

	currentMajor, currentMinor, ok := parseGoVersion(version)
	minimumMajor, minimumMinor, minimumOK := parseGoVersion("go" + MinSupportedGoVersion)
	if !ok || !minimumOK || currentMajor < minimumMajor ||
		(currentMajor == minimumMajor && currentMinor < minimumMinor) {
		return fmt.Errorf("go version %s is not supported, minimum required: %s",
			version, MinSupportedGoVersion)
	}

	return nil
}

func parseGoVersion(version string) (major, minor int, ok bool) {
	if !strings.HasPrefix(version, "go") {
		return 0, 0, false
	}
	parts := strings.Split(strings.TrimPrefix(version, "go"), ".")
	if len(parts) < 2 {
		return 0, 0, false
	}
	major, ok = parseGoVersionComponent(parts[0])
	if !ok {
		return 0, 0, false
	}
	minor, ok = parseGoVersionComponent(parts[1])
	if !ok {
		return 0, 0, false
	}
	return major, minor, true
}

func parseGoVersionComponent(component string) (int, bool) {
	end := 0
	for end < len(component) && component[end] >= '0' && component[end] <= '9' {
		end++
	}
	if end == 0 {
		return 0, false
	}
	value, err := strconv.Atoi(component[:end])
	return value, err == nil
}
