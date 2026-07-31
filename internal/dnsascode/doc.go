// Package dnsascode implements the declarative DNS-as-Code engine behind the
// `nexdns apply`, `diff`, and `pull` commands.
//
// ParseConfigFile reads a nexdns.yaml file (ConfigFile, ZoneConfig,
// RecordConfig), expanding ${ENV_VAR} references, and validates that every
// record carries a type, name, and content. ComputeDiff compares the desired
// records against the zone's current records (fetched via package api) and
// returns the ordered list of Operations (add, update, delete) that the apply
// command renders and, with --confirm, executes. Records are matched by
// type+name with trailing dots normalised; SOA and apex NS records are never
// deleted.
//
// The file format and the apply/diff/pull semantics are documented at
// https://nexdns.tech/docs/cli.
package dnsascode
