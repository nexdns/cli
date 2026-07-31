package dnsascode

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseConfigFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nexdns.yaml")

	content := `zones:
  example.com:
    ns_group: ru
    dnssec: true
    records:
      - type: A
        name: "@"
        content: "1.2.3.4"
        ttl: 300
      - type: CNAME
        name: www
        content: example.com
      - type: MX
        name: "@"
        content: mail.example.com
        priority: 10
`
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))

	cfg, err := ParseConfigFile(path)
	require.NoError(t, err)
	assert.Len(t, cfg.Zones, 1)

	zone := cfg.Zones["example.com"]
	assert.Equal(t, "ru", zone.NSGroup)
	assert.True(t, zone.DNSSEC)
	assert.Len(t, zone.Records, 3)
	assert.Equal(t, "A", zone.Records[0].Type)
	assert.Equal(t, 300, zone.Records[0].TTL)
	assert.Equal(t, "CNAME", zone.Records[1].Type)
	assert.Equal(t, 10, *zone.Records[2].Priority)
}

func TestParseConfigFileEnvSubstitution(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nexdns.yaml")

	content := `zones:
  ${TEST_DOMAIN}:
    records:
      - type: A
        name: "@"
        content: "${TEST_IP}"
`
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))

	t.Setenv("TEST_DOMAIN", "example.com")
	t.Setenv("TEST_IP", "1.2.3.4")

	cfg, err := ParseConfigFile(path)
	require.NoError(t, err)

	zone, ok := cfg.Zones["example.com"]
	require.True(t, ok)
	assert.Equal(t, "1.2.3.4", zone.Records[0].Content)
}

func TestParseConfigFileValidation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nexdns.yaml")

	content := `zones:
  example.com:
    records:
      - type: A
        name: "@"
`
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))

	_, err := ParseConfigFile(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "content is required")
}

func TestParseConfigFileMultipleZones(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nexdns.yaml")

	content := `zones:
  example.com:
    records:
      - type: A
        name: "@"
        content: "1.2.3.4"
  api.example.com:
    records:
      - type: A
        name: "@"
        content: "5.6.7.8"
`
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))

	cfg, err := ParseConfigFile(path)
	require.NoError(t, err)
	assert.Len(t, cfg.Zones, 2)
	assert.Contains(t, cfg.Zones, "example.com")
	assert.Contains(t, cfg.Zones, "api.example.com")
}
