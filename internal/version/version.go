package version

import "fmt"

// Version, Commit, and Date are the build metadata, overridden at link time
// with -ldflags. They default to development placeholders.
var (
	Version = "dev"
	Commit  = "unknown"
	Date    = "unknown"
)

// Full returns the human-readable version line shown by `nexdns version`.
func Full() string {
	return fmt.Sprintf("nexdns version %s (commit: %s, built: %s)", Version, Commit, Date)
}
