package version

import (
	"fmt"
	"runtime"
)

// Build information set via ldflags during compilation
var (
	Version   = "development"
	BuildTime = "unknown"
	GitCommit = "unknown"
	GitBranch = "unknown"
	GoVersion = runtime.Version()
)

// Info represents version information
type Info struct {
	Version   string `json:"version"`
	BuildTime string `json:"build_time"`
	GitCommit string `json:"git_commit"`
	GitBranch string `json:"git_branch"`
	GoVersion string `json:"go_version"`
}

// Get returns the version information
func Get() Info {
	return Info{Version, BuildTime, GitCommit, GitBranch, GoVersion}
}

// String returns a formatted version string
func (i Info) String() string {
	isDev := i.Version == "development"
	version := i.Version
	if !isDev {
		version = "v" + version
	}
	
	if isDev {
		return fmt.Sprintf("DRAS %s (commit: %s, branch: %s, go: %s)",
			version, i.GitCommit, i.GitBranch, i.GoVersion)
	}
	return fmt.Sprintf("DRAS %s (built: %s, commit: %s, go: %s)",
		version, i.BuildTime, i.GitCommit, i.GoVersion)
}

// Short returns a short version string
func (i Info) Short() string {
	version := i.Version
	if version != "development" {
		version = "v" + version
	}
	return fmt.Sprintf("DRAS %s", version)
}
