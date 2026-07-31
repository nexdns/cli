// Package config loads and persists CLI configuration.
//
// Config is the on-disk shape of ~/.nexdns/config.yaml. Resolve produces the
// effective configuration for a command by layering, in increasing priority:
// built-in defaults, the config file, environment variables (NEXDNS_TOKEN,
// NEXDNS_API_URL, NEXDNS_TIMEOUT), and explicitly-set command flags, with
// NO_COLOR forcing color off regardless. The mutating helpers (SetToken,
// SetAPIUrl, RemoveToken) read-modify-write the file, which is created 0600
// under a 0700 directory because it holds the API token.
package config
