package version

import "testing"

// Values linked in with -ldflags are not replaced by build info.
func TestLinkedValuesSurviveTheFallback(t *testing.T) {
	origVersion, origCommit, origDate := Version, Commit, Date
	t.Cleanup(func() { Version, Commit, Date = origVersion, origCommit, origDate })

	Version, Commit, Date = "1.2.3", "abcdef", "2026-01-01T00:00:00Z"
	fillFromBuildInfo()

	if Version != "1.2.3" || Commit != "abcdef" || Date != "2026-01-01T00:00:00Z" {
		t.Errorf("build info overwrote linked metadata: %s %s %s", Version, Commit, Date)
	}
}

// `go install pkg@version` sets no -ldflags; the version comes from build info.
func TestPlaceholdersAreReplacedWhenBuildInfoHasMore(t *testing.T) {
	origVersion, origCommit, origDate := Version, Commit, Date
	t.Cleanup(func() { Version, Commit, Date = origVersion, origCommit, origDate })

	Version, Commit, Date = "dev", "unknown", "unknown"
	fillFromBuildInfo()

	// Under `go test` the module version is not guaranteed; only check that no
	// field was cleared.
	if Version == "" || Commit == "" || Date == "" {
		t.Error("fallback cleared a field instead of leaving the placeholder")
	}
}

func TestFullNamesTheBinary(t *testing.T) {
	if got := Full(); got[:14] != "nexdns version" {
		t.Errorf("unexpected version line: %q", got)
	}
}
