package dnsascode

import (
	"strconv"
	"strings"

	"github.com/nexdns/cli/internal/api"
)

// normalizeContent puts record content in a form that compares equal between
// nexdns.yaml and the API, which stores composed rdata such as
// "10 mail.example.com." for MX and `"v=spf1 -all"` for TXT.
func normalizeContent(content string) string {
	fields := strings.Fields(strings.ReplaceAll(content, `"`, ""))
	for i, f := range fields {
		fields[i] = strings.TrimSuffix(f, ".")
	}
	return strings.ToLower(strings.Join(fields, " "))
}

// localContent composes a config record's content the way the API stores it,
// so normalizeContent() compares like with like.
func localContent(r RecordConfig) string {
	content := r.Content

	switch strings.ToUpper(r.Type) {
	case "MX":
		if r.Priority != nil && !startsWithNumber(content) {
			content = strconv.Itoa(*r.Priority) + " " + content
		}
	case "SRV":
		if r.Priority != nil && r.Weight != nil && r.Port != nil && !startsWithNumber(content) {
			content = strconv.Itoa(*r.Priority) + " " + strconv.Itoa(*r.Weight) + " " +
				strconv.Itoa(*r.Port) + " " + content
		}
	case "CAA":
		if r.Tag != "" && !startsWithNumber(content) {
			flags := 0
			if r.Flags != nil {
				flags = *r.Flags
			}
			content = strconv.Itoa(flags) + " " + r.Tag + " " + content
		}
	}

	return content
}

// startsWithNumber reports whether content already carries its numeric prefix
// (as produced by `nexdns pull`), so it is not composed a second time.
func startsWithNumber(content string) bool {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return false
	}
	return trimmed[0] >= '0' && trimmed[0] <= '9'
}

// ComputeDiff compares local config records with remote records and returns operations.
// Matching is by type + name (for apply, which manages desired state).
// deleteRemote controls whether records in remote but not in local are deleted.
func ComputeDiff(zone string, local []RecordConfig, remote []api.Record, deleteRemote bool) []Operation {
	var ops []Operation

	// Index remote records by type+name for lookup
	type remoteKey struct {
		Type string
		Name string
	}
	remoteMap := make(map[remoteKey][]api.Record)
	for _, r := range remote {
		key := remoteKey{Type: r.Type, Name: r.Name}
		remoteMap[key] = append(remoteMap[key], r)
	}

	// Track which remote records are accounted for
	matched := make(map[string]bool)

	for _, lr := range local {
		key := remoteKey{Type: lr.Type, Name: lr.Name}
		remotes := remoteMap[key]

		if len(remotes) == 0 {
			// No remote match: add
			ops = append(ops, Operation{
				Action:   "add",
				Zone:     zone,
				Type:     lr.Type,
				Name:     lr.Name,
				Content:  lr.Content,
				TTL:      effectiveTTL(lr.TTL),
				Priority: lr.Priority,
				Weight:   lr.Weight,
				Port:     lr.Port,
				Flags:    lr.Flags,
				Tag:      lr.Tag,
			})
			continue
		}

		// Find exact content match or first match
		found := false
		for _, rr := range remotes {
			if normalizeContent(rr.Content) == normalizeContent(localContent(lr)) {
				matched[rr.ID] = true
				// Check if TTL or priority differ
				if needsUpdate(lr, rr) {
					ops = append(ops, Operation{
						Action:     "update",
						Zone:       zone,
						Type:       lr.Type,
						Name:       lr.Name,
						Content:    lr.Content,
						TTL:        effectiveTTL(lr.TTL),
						Priority:   lr.Priority,
						Weight:     lr.Weight,
						Port:       lr.Port,
						Flags:      lr.Flags,
						Tag:        lr.Tag,
						OldContent: rr.Content,
						OldTTL:     rr.TTL,
						RecordID:   rr.ID,
					})
				} else {
					// No change
					matched[rr.ID] = true
				}
				found = true
				break
			}
		}

		if !found {
			// Content differs: update the first unmatched remote record.
			for _, rr := range remotes {
				if !matched[rr.ID] {
					matched[rr.ID] = true
					ops = append(ops, Operation{
						Action:     "update",
						Zone:       zone,
						Type:       lr.Type,
						Name:       lr.Name,
						Content:    lr.Content,
						TTL:        effectiveTTL(lr.TTL),
						Priority:   lr.Priority,
						Weight:     lr.Weight,
						Port:       lr.Port,
						Flags:      lr.Flags,
						Tag:        lr.Tag,
						OldContent: rr.Content,
						OldTTL:     rr.TTL,
						RecordID:   rr.ID,
					})
					found = true
					break
				}
			}
			if !found {
				// All remote records of this type+name are matched, add a new one
				ops = append(ops, Operation{
					Action:   "add",
					Zone:     zone,
					Type:     lr.Type,
					Name:     lr.Name,
					Content:  lr.Content,
					TTL:      effectiveTTL(lr.TTL),
					Priority: lr.Priority,
					Weight:   lr.Weight,
					Port:     lr.Port,
					Flags:    lr.Flags,
					Tag:      lr.Tag,
				})
			}
		}
	}

	// Delete remote records not in local config
	if deleteRemote {
		for _, r := range remote {
			// Never delete SOA or NS at apex
			if r.Name == "@" && (r.Type == "SOA" || r.Type == "NS") {
				continue
			}
			if !matched[r.ID] {
				ops = append(ops, Operation{
					Action:   "delete",
					Zone:     zone,
					Type:     r.Type,
					Name:     r.Name,
					Content:  r.Content,
					RecordID: r.ID,
				})
			}
		}
	}

	return ops
}

func needsUpdate(local RecordConfig, remote api.Record) bool {
	localTTL := effectiveTTL(local.TTL)
	if localTTL != remote.TTL {
		return true
	}
	// Priority is part of the content compared above. The API returns it under
	// fields, so a nil remote priority is not a difference.
	if local.Priority != nil && remote.Priority != nil && *local.Priority != *remote.Priority {
		return true
	}
	return false
}

func effectiveTTL(ttl int) int {
	if ttl == 0 {
		return 3600
	}
	return ttl
}
