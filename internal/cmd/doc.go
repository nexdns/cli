// Package cmd wires the nexdns command-line interface.
//
// It defines the Cobra command tree rooted at rootCmd and exposes Execute,
// which cmd/nexdns/main.go calls. Each file registers one command group through
// an init function: auth (token storage and validation), zone, record, dnssec,
// account, config (persisted settings), apply/diff/pull (the DNS-as-Code
// engine), acme (certbot DNS-01 hook), version, and completion.
//
// Commands own no business logic. A typical RunE resolves configuration and
// builds an *api.Client (newAPIClient) plus an output.Formatter (newFormatter),
// translates positional args and flags into an api request struct, calls one or
// a few client methods, and renders the result. Domain names are resolved to
// opaque zone IDs through resolveZone before any record call, so the user never
// handles IDs. Errors are returned rather than printed, so main can map them to
// an exit code: ExitCode is 2 for Cobra usage errors and 1 otherwise.
//
// The user-facing command, flag, and output contract is documented at
// https://nexdns.tech/docs/cli.
package cmd
