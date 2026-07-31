package dnsascode

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/nexdns/cli/internal/api"
)

func intPtr(v int) *int { return &v }

func TestDiffNoChanges(t *testing.T) {
	local := []RecordConfig{
		{Type: "A", Name: "@", Content: "1.2.3.4", TTL: 3600},
	}
	remote := []api.Record{
		{ID: "r1", Type: "A", Name: "@", Content: "1.2.3.4", TTL: 3600},
	}
	ops := ComputeDiff("example.com", local, remote, false)
	assert.Len(t, ops, 0)
}

func TestDiffAddRecord(t *testing.T) {
	local := []RecordConfig{
		{Type: "A", Name: "@", Content: "1.2.3.4", TTL: 3600},
		{Type: "CNAME", Name: "www", Content: "example.com", TTL: 3600},
	}
	remote := []api.Record{
		{ID: "r1", Type: "A", Name: "@", Content: "1.2.3.4", TTL: 3600},
	}
	ops := ComputeDiff("example.com", local, remote, false)
	assert.Len(t, ops, 1)
	assert.Equal(t, "add", ops[0].Action)
	assert.Equal(t, "CNAME", ops[0].Type)
	assert.Equal(t, "www", ops[0].Name)
}

func TestDiffUpdateContent(t *testing.T) {
	local := []RecordConfig{
		{Type: "A", Name: "@", Content: "5.6.7.8", TTL: 3600},
	}
	remote := []api.Record{
		{ID: "r1", Type: "A", Name: "@", Content: "1.2.3.4", TTL: 3600},
	}
	ops := ComputeDiff("example.com", local, remote, false)
	assert.Len(t, ops, 1)
	assert.Equal(t, "update", ops[0].Action)
	assert.Equal(t, "5.6.7.8", ops[0].Content)
	assert.Equal(t, "1.2.3.4", ops[0].OldContent)
	assert.Equal(t, "r1", ops[0].RecordID)
}

func TestDiffUpdateTTL(t *testing.T) {
	local := []RecordConfig{
		{Type: "A", Name: "@", Content: "1.2.3.4", TTL: 300},
	}
	remote := []api.Record{
		{ID: "r1", Type: "A", Name: "@", Content: "1.2.3.4", TTL: 3600},
	}
	ops := ComputeDiff("example.com", local, remote, false)
	assert.Len(t, ops, 1)
	assert.Equal(t, "update", ops[0].Action)
	assert.Equal(t, 300, ops[0].TTL)
}

func TestDiffDeleteRemote(t *testing.T) {
	local := []RecordConfig{
		{Type: "A", Name: "@", Content: "1.2.3.4", TTL: 3600},
	}
	remote := []api.Record{
		{ID: "r1", Type: "A", Name: "@", Content: "1.2.3.4", TTL: 3600},
		{ID: "r2", Type: "AAAA", Name: "@", Content: "2001:db8::1", TTL: 3600},
	}

	// Without delete flag
	ops := ComputeDiff("example.com", local, remote, false)
	assert.Len(t, ops, 0)

	// With delete flag
	ops = ComputeDiff("example.com", local, remote, true)
	assert.Len(t, ops, 1)
	assert.Equal(t, "delete", ops[0].Action)
	assert.Equal(t, "AAAA", ops[0].Type)
	assert.Equal(t, "r2", ops[0].RecordID)
}

func TestDiffNeverDeleteSOAorNS(t *testing.T) {
	local := []RecordConfig{}
	remote := []api.Record{
		{ID: "r1", Type: "SOA", Name: "@", Content: "ns1.example.com. admin.example.com. 2025010100 3600 900 604800 86400"},
		{ID: "r2", Type: "NS", Name: "@", Content: "ns1.example.com"},
		{ID: "r3", Type: "A", Name: "@", Content: "1.2.3.4", TTL: 3600},
	}

	ops := ComputeDiff("example.com", local, remote, true)
	assert.Len(t, ops, 1)
	assert.Equal(t, "delete", ops[0].Action)
	assert.Equal(t, "A", ops[0].Type)
}

func TestDiffMixed(t *testing.T) {
	local := []RecordConfig{
		{Type: "A", Name: "@", Content: "5.6.7.8", TTL: 300},
		{Type: "CNAME", Name: "www", Content: "example.com"},
		{Type: "MX", Name: "@", Content: "mail.example.com", Priority: intPtr(10)},
	}
	remote := []api.Record{
		{ID: "r1", Type: "A", Name: "@", Content: "1.2.3.4", TTL: 3600},
		{ID: "r2", Type: "TXT", Name: "@", Content: "v=spf1", TTL: 3600},
	}

	ops := ComputeDiff("example.com", local, remote, true)
	adds := 0
	updates := 0
	deletes := 0
	for _, op := range ops {
		switch op.Action {
		case "add":
			adds++
		case "update":
			updates++
		case "delete":
			deletes++
		}
	}
	assert.Equal(t, 2, adds)    // CNAME www, MX @
	assert.Equal(t, 1, updates) // A @ content change
	assert.Equal(t, 1, deletes) // TXT @
}

func TestDiffDefaultTTL(t *testing.T) {
	local := []RecordConfig{
		{Type: "A", Name: "@", Content: "1.2.3.4"}, // TTL=0 -> defaults to 3600
	}
	remote := []api.Record{
		{ID: "r1", Type: "A", Name: "@", Content: "1.2.3.4", TTL: 3600},
	}
	ops := ComputeDiff("example.com", local, remote, false)
	assert.Len(t, ops, 0) // No change because default TTL is 3600
}
