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
	return Info{
		Version:   Version,
		BuildTime: BuildTime,
		GitCommit: GitCommit,
		GitBranch: GitBranch,
		GoVersion: GoVersion,
	}
}

// String returns a formatted version string
func (i Info) String() string {
	if i.Version == "development" {
		return fmt.Sprintf("DRAS %s (commit: %s, branch: %s, go: %s)",
			i.Version, i.GitCommit, i.GitBranch, i.GoVersion)
	}
	return fmt.Sprintf("DRAS v%s (built: %s, commit: %s, go: %s)",
		i.Version, i.BuildTime, i.GitCommit, i.GoVersion)
}

// Short returns a short version string
func (i Info) Short() string {
	if i.Version == "development" {
		return fmt.Sprintf("DRAS %s", i.Version)
	}
	return fmt.Sprintf("DRAS v%s", i.Version)
}