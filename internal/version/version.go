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

// fillFromBuildInfo recovers the metadata Go records itself, for builds that
// carry no -ldflags.
//
// `go install github.com/nexdns/cli/cmd/nexdns@latest` is a documented way to
// install this tool and it does not run our link flags, so the binary reported
// "nexdns version dev" - which reads as a broken build rather than as a
// perfectly good install. Go stamps the module version and the VCS state into
// the binary regardless, so the honest values are already there to be read.
//
// Only placeholders are replaced: a release built by GoReleaser keeps the
// values it was linked with.
func fillFromBuildInfo() {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}

	// "(devel)" is what Go reports for a build from a working tree rather than
	// from a published module version; it says less than our own placeholder.
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
