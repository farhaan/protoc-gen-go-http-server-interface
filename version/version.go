package version

import (
	"runtime/debug"
	"strings"
)

// These variables are set at build time via -ldflags
var (
	// Version is the version of the plugin, set via -ldflags "-X version.Version=..."
	Version = "dev"

	// GitCommit is the git commit hash, set via -ldflags "-X version.GitCommit=..."
	GitCommit = ""

	// BuildTime is the build timestamp, set via -ldflags "-X version.BuildTime=..."
	BuildTime = ""
)

// Info represents version information
type Info struct {
	Version   string
	GitCommit string
	BuildTime string
	GoVersion string
}

// Get returns the version information
func Get() Info {
	return Info{
		Version:   getVersion(),
		GitCommit: GitCommit,
		BuildTime: BuildTime,
		GoVersion: getGoVersion(),
	}
}

// GetVersion returns just the version string
func GetVersion() string {
	return getVersion()
}

// getVersion returns the version, attempting to get it from build info if not set via ldflags
func getVersion() string {
	if Version != "dev" && Version != "" {
		return Version
	}

	// Try to get version from build info (when built with go install)
	if buildInfo, ok := debug.ReadBuildInfo(); ok {
		// Check if this is a tagged version
		for _, setting := range buildInfo.Settings {
			if setting.Key == "vcs.revision" && len(setting.Value) >= 7 {
				// Use short commit hash as version if no tag available
				if Version == "dev" {
					return "dev-" + setting.Value[:7]
				}
			}
		}

		// If we have a version from VCS tags, it would be in buildInfo.Main.Version
		if buildInfo.Main.Version != "" && buildInfo.Main.Version != "(devel)" {
			return buildInfo.Main.Version
		}
	}

	return "dev"
}

// getGoVersion returns the Go version used to build the binary
func getGoVersion() string {
	if buildInfo, ok := debug.ReadBuildInfo(); ok {
		return buildInfo.GoVersion
	}
	return "unknown"
}

// String returns a formatted version string
func (i Info) String() string {
	var parts []string

	if i.Version != "" {
		parts = append(parts, i.Version)
	}

	if i.GitCommit != "" {
		if len(i.GitCommit) >= 7 {
			parts = append(parts, "commit:"+i.GitCommit[:7])
		} else {
			parts = append(parts, "commit:"+i.GitCommit)
		}
	}

	if i.BuildTime != "" {
		parts = append(parts, "built:"+i.BuildTime)
	}

	if len(parts) == 0 {
		return "dev"
	}

	return strings.Join(parts, " ")
}
