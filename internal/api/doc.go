// Package api is the HTTP client for the NexDNS REST API (v1).
//
// Client wraps a single bearer-token-authenticated *http.Client and exposes one
// method per documented endpoint, grouped by file: zones (zones.go), records
// (records.go), DNSSEC (dnssec.go), account and API keys (account.go), and
// billing (billing.go). Request and response bodies are the plain structs in
// types.go; API errors are decoded into *APIError (errors.go), with the
// IsNotFound, IsConflict, and IsRateLimited helpers for status-code branching.
//
// Every call takes a context.Context as its first argument and returns either a
// typed value or an error. Responses are unwrapped from the standard
// {"status":"success","data":...} envelope before decoding. Only idempotent
// GETs are retried automatically (exponential backoff plus jitter, honouring
// Retry-After on 5xx); writes are never retried, because record IDs are
// content-derived and a replayed write would surface a spurious 409 or 404.
//
// This package sits at the bottom of the data flow: the cmd package builds a
// Client and calls these methods, then hands the returned structs to the output
// package. The endpoint surface mirrors the REST API reference at
// https://nexdns.tech/docs/api, which is authoritative for paths, scopes, rate
// limits, and payload semantics.
package api
