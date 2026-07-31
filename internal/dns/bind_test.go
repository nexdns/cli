package dns

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSimpleZone(t *testing.T) {
	zone := `
$TTL 3600
$ORIGIN example.com.
@       IN  A     192.168.1.1
www     IN  A     192.168.1.1
@       IN  AAAA  2001:db8::1
`
	records, err := ParseBINDZone(strings.NewReader(zone), "example.com")
	require.NoError(t, err)
	assert.Len(t, records, 3)

	assert.Equal(t, "@", records[0].Name)
	assert.Equal(t, "A", records[0].Type)
	assert.Equal(t, "192.168.1.1", records[0].Content)
	assert.Equal(t, 3600, records[0].TTL)

	assert.Equal(t, "www", records[1].Name)
	assert.Equal(t, "A", records[1].Type)

	assert.Equal(t, "AAAA", records[2].Type)
	assert.Equal(t, "2001:db8::1", records[2].Content)
}

func TestParseMXRecord(t *testing.T) {
	zone := `@ IN MX 10 mail.example.com.`
	records, err := ParseBINDZone(strings.NewReader(zone), "example.com")
	require.NoError(t, err)
	require.Len(t, records, 1)

	assert.Equal(t, "MX", records[0].Type)
	// RDATA stays fully qualified: shortening it to "mail" made the API store a
	// root-level "mail." target.
	assert.Equal(t, "mail.example.com", records[0].Content)
	assert.NotNil(t, records[0].Priority)
	assert.Equal(t, 10, *records[0].Priority)
}

func TestParseRelativeRDataIsExpanded(t *testing.T) {
	zone := `@ IN MX 10 mail
www IN CNAME host`
	records, err := ParseBINDZone(strings.NewReader(zone), "example.com")
	require.NoError(t, err)
	require.Len(t, records, 2)

	assert.Equal(t, "mail.example.com", records[0].Content)
	assert.Equal(t, "host.example.com", records[1].Content)
}

func TestParseOutOfZoneRDataIsKept(t *testing.T) {
	zone := `@ IN MX 20 mx-backup.example.net.`
	records, err := ParseBINDZone(strings.NewReader(zone), "example.com")
	require.NoError(t, err)
	require.Len(t, records, 1)

	assert.Equal(t, "mx-backup.example.net", records[0].Content)
}

func TestOwnerNameInheritance(t *testing.T) {
	// A blank owner repeats the previous one (RFC 1035 §5.1) – the shape every
	// round-robin block in a real zone file uses.
	zone := "www IN A 198.51.100.11\n\tIN A 198.51.100.12\n\tIN A 198.51.100.13\nmail IN A 198.51.100.20\n"
	records, err := ParseBINDZone(strings.NewReader(zone), "example.com")
	require.NoError(t, err)
	require.Len(t, records, 4)

	assert.Equal(t, "www", records[0].Name)
	assert.Equal(t, "www", records[1].Name)
	assert.Equal(t, "www", records[2].Name)
	assert.Equal(t, "mail", records[3].Name)
}

func TestParseCNAMERecord(t *testing.T) {
	zone := `www IN CNAME example.com.`
	records, err := ParseBINDZone(strings.NewReader(zone), "example.com")
	require.NoError(t, err)
	require.Len(t, records, 1)

	assert.Equal(t, "CNAME", records[0].Type)
	assert.Equal(t, "www", records[0].Name)
	assert.Equal(t, "example.com", records[0].Content)
}

func TestParseTXTRecord(t *testing.T) {
	zone := `@ IN TXT "v=spf1 include:_spf.google.com ~all"`
	records, err := ParseBINDZone(strings.NewReader(zone), "example.com")
	require.NoError(t, err)
	require.Len(t, records, 1)

	assert.Equal(t, "TXT", records[0].Type)
	assert.Equal(t, `"v=spf1 include:_spf.google.com ~all"`, records[0].Content)
}

func TestParseMultiStringTXTRecord(t *testing.T) {
	// Two character-strings must stay two: joining them changes the record.
	zone := `long IN TXT "first part" "second part"`
	records, err := ParseBINDZone(strings.NewReader(zone), "example.com")
	require.NoError(t, err)
	require.Len(t, records, 1)

	assert.Equal(t, `"first part" "second part"`, records[0].Content)
}

func TestParseSRVRecord(t *testing.T) {
	zone := `_sip._tcp IN SRV 10 60 5060 sip.example.com.`
	records, err := ParseBINDZone(strings.NewReader(zone), "example.com")
	require.NoError(t, err)
	require.Len(t, records, 1)

	r := records[0]
	assert.Equal(t, "SRV", r.Type)
	assert.Equal(t, "_sip._tcp", r.Name)
	assert.Equal(t, "sip.example.com", r.Content)
	assert.Equal(t, 10, *r.Priority)
	assert.Equal(t, 60, *r.Weight)
	assert.Equal(t, 5060, *r.Port)
}

func TestParseCAARecord(t *testing.T) {
	zone := `@ IN CAA 0 issue "letsencrypt.org"`
	records, err := ParseBINDZone(strings.NewReader(zone), "example.com")
	require.NoError(t, err)
	require.Len(t, records, 1)

	r := records[0]
	assert.Equal(t, "CAA", r.Type)
	assert.Equal(t, 0, *r.Flags)
	assert.Equal(t, "issue", r.Tag)
	assert.Equal(t, "letsencrypt.org", r.Content)
}

func TestSkipSOA(t *testing.T) {
	zone := `@ IN SOA ns1.example.com. admin.example.com. 2025010100 3600 900 604800 86400`
	records, err := ParseBINDZone(strings.NewReader(zone), "example.com")
	require.NoError(t, err)
	assert.Len(t, records, 0)
}

func TestSkipComments(t *testing.T) {
	zone := `
; This is a comment
@ IN A 1.2.3.4 ; inline comment
`
	records, err := ParseBINDZone(strings.NewReader(zone), "example.com")
	require.NoError(t, err)
	require.Len(t, records, 1)
	assert.Equal(t, "1.2.3.4", records[0].Content)
}

func TestMultiLineRecord(t *testing.T) {
	zone := `@ IN SOA ns1.example.com. admin.example.com. (
    2025010100 ; serial
    3600       ; refresh
    900        ; retry
    604800     ; expire
    86400      ; minimum
)
@ IN A 1.2.3.4`
	records, err := ParseBINDZone(strings.NewReader(zone), "example.com")
	require.NoError(t, err)
	// SOA is skipped, only A record
	require.Len(t, records, 1)
	assert.Equal(t, "A", records[0].Type)
}

func TestTTLDirective(t *testing.T) {
	zone := `
$TTL 300
@ IN A 1.2.3.4
`
	records, err := ParseBINDZone(strings.NewReader(zone), "example.com")
	require.NoError(t, err)
	require.Len(t, records, 1)
	assert.Equal(t, 300, records[0].TTL)
}

func TestPerRecordTTL(t *testing.T) {
	zone := `
$TTL 3600
@   600 IN A 1.2.3.4
www     IN A 5.6.7.8
`
	records, err := ParseBINDZone(strings.NewReader(zone), "example.com")
	require.NoError(t, err)
	require.Len(t, records, 2)
	assert.Equal(t, 600, records[0].TTL)
	assert.Equal(t, 3600, records[1].TTL)
}

func TestTimeSuffixTTL(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"3600", 3600},
		{"1h", 3600},
		{"30m", 1800},
		{"1d", 86400},
		{"1w", 604800},
		{"1h30m", 5400},
	}
	for _, tt := range tests {
		v, err := parseTTLValue(tt.input)
		require.NoError(t, err, "input: %s", tt.input)
		assert.Equal(t, tt.expected, v, "input: %s", tt.input)
	}
}

func TestFQDNNormalization(t *testing.T) {
	zone := `example.com. IN A 1.2.3.4
www.example.com. IN A 5.6.7.8
sub.example.com. IN CNAME example.com.`
	records, err := ParseBINDZone(strings.NewReader(zone), "example.com")
	require.NoError(t, err)
	require.Len(t, records, 3)
	assert.Equal(t, "@", records[0].Name)
	assert.Equal(t, "www", records[1].Name)
	assert.Equal(t, "sub", records[2].Name)
}

func TestToCreateRequests(t *testing.T) {
	priority := 10
	parsed := []ParsedRecord{
		{Name: "@", TTL: 3600, Type: "A", Content: "1.2.3.4"},
		{Name: "@", TTL: 3600, Type: "MX", Content: "mail.example.com", Priority: &priority},
	}
	reqs := ToCreateRequests(parsed)
	assert.Len(t, reqs, 2)
	assert.Equal(t, "A", reqs[0].Type)
	assert.Equal(t, "MX", reqs[1].Type)
	assert.Equal(t, 10, *reqs[1].Priority)
}

func TestOwnerNameMayLookLikeARecordType(t *testing.T) {
	// "ns", "alias" and friends are ordinary hostnames. Deciding by the token
	// alone turned `alias IN CNAME …` into an ALIAS record at the apex.
	zone := `alias IN CNAME host.example.com.
ns IN A 198.51.100.1
mx 300 IN A 198.51.100.2`
	records, err := ParseBINDZone(strings.NewReader(zone), "example.com")
	require.NoError(t, err)
	require.Len(t, records, 3)

	assert.Equal(t, "CNAME", records[0].Type)
	assert.Equal(t, "alias", records[0].Name)
	assert.Equal(t, "host.example.com", records[0].Content)

	assert.Equal(t, "A", records[1].Type)
	assert.Equal(t, "ns", records[1].Name)

	assert.Equal(t, "A", records[2].Type)
	assert.Equal(t, "mx", records[2].Name)
	assert.Equal(t, 300, records[2].TTL)
}
