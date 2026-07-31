// Package version holds the binary's build metadata.
//
// Version, Commit, and Date are stamped at build time with -ldflags (see the
// Makefile) and default to development placeholders for plain `go build` or
// `go run`. Full formats them into the line shown by `nexdns version`, and
// Version also feeds the API client's User-Agent and the root command's
// --version output.
package version
