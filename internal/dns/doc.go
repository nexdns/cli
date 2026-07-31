// Package dns provides local DNS utilities that do not go through the API.
//
// bind.go parses BIND-format zone files (ParseBINDZone) into ParsedRecord
// values, handling $TTL and $ORIGIN directives, comments, and multi-line
// parenthesised records, and converts them to api.CreateRecordRequest values
// (ToCreateRequests) for `nexdns zone import`. propagation.go (LookupNS,
// LookupA, LookupMX) queries individual public resolvers directly over UDP for
// `nexdns zone check`, so propagation is observed from outside the platform
// rather than self-reported by the API.
package dns
