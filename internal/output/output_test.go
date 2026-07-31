package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nexdns/cli/internal/api"
)

func sampleZones() []api.ZoneListItem {
	return []api.ZoneListItem{
		{ID: "abc", Name: "example.com", Type: "master", Status: "active", NSGroup: "Russia", RecordsCount: 12, CreatedAt: "2025-01-15"},
		{ID: "def", Name: "test.org", Type: "master", Status: "active", NSGroup: "Europe", RecordsCount: 5, CreatedAt: "2025-02-01"},
	}
}

func sampleRecords() []api.Record {
	priority := 10
	return []api.Record{
		{ID: "r1", Name: "@", Type: "A", Content: "1.2.3.4", TTL: 3600},
		{ID: "r2", Name: "@", Type: "MX", Content: "mail.example.com", TTL: 3600, Priority: &priority},
	}
}

func TestTableFormatZones(t *testing.T) {
	var buf bytes.Buffer
	f := NewTableFormatter(&buf, false, "never")
	meta := &api.PaginationMeta{Total: 2}
	f.FormatZones(sampleZones(), meta)

	out := buf.String()
	assert.Contains(t, out, "example.com")
	assert.Contains(t, out, "test.org")
	assert.Contains(t, out, "2 zone(s)")
}

func TestTableFormatRecords(t *testing.T) {
	var buf bytes.Buffer
	f := NewTableFormatter(&buf, false, "never")
	f.FormatRecords(sampleRecords())

	out := buf.String()
	assert.Contains(t, out, "1.2.3.4")
	assert.Contains(t, out, "PRIORITY")
	assert.Contains(t, out, "10")
	assert.Contains(t, out, "2 record(s)")
}

func TestTableTruncatesLongContent(t *testing.T) {
	var buf bytes.Buffer
	f := NewTableFormatter(&buf, false, "never")
	records := []api.Record{
		{ID: "r1", Name: "@", Type: "TXT", Content: strings.Repeat("a", 100), TTL: 3600},
	}
	f.FormatRecords(records)

	out := buf.String()
	assert.Contains(t, out, "...")
	assert.NotContains(t, out, strings.Repeat("a", 100))
}

func TestJSONFormatZones(t *testing.T) {
	var buf bytes.Buffer
	f := NewJSONFormatter(&buf, false)
	f.FormatZones(sampleZones(), nil)

	var zones []api.ZoneListItem
	err := json.Unmarshal(buf.Bytes(), &zones)
	require.NoError(t, err)
	assert.Len(t, zones, 2)
	assert.Equal(t, "example.com", zones[0].Name)
}

func TestJSONCompactInQuietMode(t *testing.T) {
	var buf bytes.Buffer
	f := NewJSONFormatter(&buf, true)
	f.FormatZones(sampleZones(), nil)

	out := buf.String()
	assert.NotContains(t, out, "\n  ")
}

func TestYAMLFormatZones(t *testing.T) {
	var buf bytes.Buffer
	f := NewYAMLFormatter(&buf)
	f.FormatZones(sampleZones(), nil)

	out := buf.String()
	assert.Contains(t, out, "name: example.com")
	assert.Contains(t, out, "name: test.org")

	// Keys must be snake_case (from yaml struct tags), not lowercased Go field names.
	assert.Contains(t, out, "id:")
	assert.Contains(t, out, "ns_group:")
	assert.Contains(t, out, "records_count:")
	assert.Contains(t, out, "created_at:")
	assert.NotContains(t, out, "publicid:")
	assert.NotContains(t, out, "nsgroup:")
	assert.NotContains(t, out, "recordscount:")
	assert.NotContains(t, out, "createdat:")
}

func TestCSVFormatZones(t *testing.T) {
	var buf bytes.Buffer
	f := NewCSVFormatter(&buf)
	f.FormatZones(sampleZones(), nil)

	out := buf.String()
	lines := strings.Split(strings.TrimSpace(out), "\n")
	assert.Len(t, lines, 3) // header + 2 rows
	assert.Contains(t, lines[0], "name,type,status")
	assert.Contains(t, lines[1], "example.com")
}

func TestCSVFormatRecordsWithCommaInContent(t *testing.T) {
	var buf bytes.Buffer
	f := NewCSVFormatter(&buf)
	records := []api.Record{
		{ID: "r1", Name: "@", Type: "TXT", Content: "v=spf1 include:a,include:b ~all", TTL: 3600},
	}
	f.FormatRecords(records)

	out := buf.String()
	// Content with comma should be quoted
	assert.Contains(t, out, `"v=spf1 include:a,include:b ~all"`)
}

func TestNewFormatterUnknownFormat(t *testing.T) {
	_, err := New("xml", nil, false, "auto")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown output format")
}
