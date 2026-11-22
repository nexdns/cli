package version

import (
	"fmt"
	"runtime/debug"
)

// Version, Commit, and Date are the build metadata, overridden at link time
// with -ldflags. They default to development placeholders.
var (
	Version = "dev"
	Commit  = "unknown"
	Date    = "unknown"
)

func init() {
	fillFromBuildInfo()
}

// fillFromBuildInfo fills in placeholder values from the build info Go
// records in every binary, for builds without -ldflags such as `go install`.
func fillFromBuildInfo() {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}

	// "(devel)" is a build from a working tree; keep the placeholder.
	if Version == "dev" && info.Main.Version != "" && info.Main.Version != "(devel)" {
		Version = info.Main.Version
	}

	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			if Commit == "unknown" && setting.Value != "" {
				Commit = setting.Value
			}
		case "vcs.time":
			if Date == "unknown" && setting.Value != "" {
				Date = setting.Value
			}
		}
	}
}

// Full returns the human-readable version line shown by `nexdns version`.
func Full() string {
	return fmt.Sprintf("nexdns version %s (commit: %s, built: %s)", Version, Commit, Date)
}
