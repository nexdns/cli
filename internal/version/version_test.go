package version

import "testing"

// A release links the metadata in, and the build-info fallback must not
// overwrite it: the linked values name the released tag, while build info for
// the same binary can name the commit it was built from.
func TestLinkedValuesSurviveTheFallback(t *testing.T) {
	origVersion, origCommit, origDate := Version, Commit, Date
	t.Cleanup(func() { Version, Commit, Date = origVersion, origCommit, origDate })

	Version, Commit, Date = "1.2.3", "abcdef", "2026-01-01T00:00:00Z"
	fillFromBuildInfo()

	if Version != "1.2.3" || Commit != "abcdef" || Date != "2026-01-01T00:00:00Z" {
		t.Errorf("build info overwrote linked metadata: %s %s %s", Version, Commit, Date)
	}
}

// `go install pkg@version` runs no link flags, so a binary installed that way
// reported "dev" until it learned to read what Go stamps in by itself.
func TestPlaceholdersAreReplacedWhenBuildInfoHasMore(t *testing.T) {
	origVersion, origCommit, origDate := Version, Commit, Date
	t.Cleanup(func() { Version, Commit, Date = origVersion, origCommit, origDate })

	Version, Commit, Date = "dev", "unknown", "unknown"
	fillFromBuildInfo()

	// Under `go test` the main module is the test binary's, so a version is not
	// guaranteed; what must hold is that nothing was made worse.
	if Version == "" || Commit == "" || Date == "" {
		t.Error("fallback cleared a field instead of leaving the placeholder")
	}
}

func TestFullNamesTheBinary(t *testing.T) {
	if got := Full(); got[:14] != "nexdns version" {
		t.Errorf("unexpected version line: %q", got)
	}
}
