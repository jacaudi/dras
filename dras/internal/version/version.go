package version

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"strings"
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

// Get returns the version information, applying a runtime/debug fallback
// for VCS metadata when ldflags weren't set (e.g. `go install`, local
// `go build` from a git workdir).
func Get() Info {
	bi, _ := debug.ReadBuildInfo()
	return resolveInfo(Version, BuildTime, GitCommit, GitBranch, GoVersion, bi)
}

// resolveInfo picks the most-specific value for each field. Priority:
//
//	ldflag-set value > runtime/debug VCS info > the original "development"/"unknown" sentinels.
//
// Split out from Get() so it's unit-testable without re-binding package vars
// or shelling out to `go build`.
func resolveInfo(version, buildTime, commit, branch, goVersion string, bi *debug.BuildInfo) Info {
	info := Info{
		Version:   version,
		BuildTime: buildTime,
		GitCommit: commit,
		GitBranch: branch,
		GoVersion: goVersion,
	}
	if bi == nil {
		return info
	}
	// Module-path version, set when installed via `go install pkg@vX.Y.Z`.
	// "(devel)" is what `go build` writes for a non-tagged checkout.
	if info.Version == "development" && bi.Main.Version != "" && bi.Main.Version != "(devel)" {
		info.Version = strings.TrimPrefix(bi.Main.Version, "v")
	}
	// VCS settings auto-embedded by `go build` since Go 1.18 when building in
	// a git workdir. Only fill in when the ldflag-set value is still default.
	for _, s := range bi.Settings {
		switch s.Key {
		case "vcs.revision":
			if info.GitCommit == "unknown" && s.Value != "" {
				info.GitCommit = s.Value
			}
		case "vcs.time":
			if info.BuildTime == "unknown" && s.Value != "" {
				info.BuildTime = s.Value
			}
		}
	}
	return info
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
