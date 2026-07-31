package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadNonexistentFile(t *testing.T) {
	cfg, err := Load("/nonexistent/path/config.yaml")
	require.NoError(t, err)
	assert.Equal(t, &Config{}, cfg)
}

func TestLoadValidConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := []byte("api_url: https://example.com/api/v1\ntoken: nxd_test123\noutput: json\ncolor: never\ntimeout: 60\n")
	require.NoError(t, os.WriteFile(path, content, 0600))

	cfg, err := Load(path)
	require.NoError(t, err)
	assert.Equal(t, "https://example.com/api/v1", cfg.APIUrl)
	assert.Equal(t, "nxd_test123", cfg.Token)
	assert.Equal(t, "json", cfg.Output)
	assert.Equal(t, "never", cfg.Color)
	assert.Equal(t, 60, cfg.Timeout)
}

func TestLoadEmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte(""), 0600))

	cfg, err := Load(path)
	require.NoError(t, err)
	assert.Equal(t, &Config{}, cfg)
}

func TestSaveCreatesDirectoryAndFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "subdir", "config.yaml")

	cfg := &Config{
		APIUrl: "https://example.com/api/v1",
		Token:  "nxd_test123",
	}
	require.NoError(t, Save(path, cfg))

	info, err := os.Stat(filepath.Dir(path))
	require.NoError(t, err)
	assert.True(t, info.IsDir())
	assert.Equal(t, os.FileMode(0700), info.Mode().Perm())

	info, err = os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm())
}

func TestSetTokenPreservesOtherFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	cfg := &Config{
		APIUrl: "https://example.com/api/v1",
		Output: "json",
	}
	require.NoError(t, Save(path, cfg))

	require.NoError(t, SetToken(path, "nxd_newtoken"))

	loaded, err := Load(path)
	require.NoError(t, err)
	assert.Equal(t, "nxd_newtoken", loaded.Token)
	assert.Equal(t, "https://example.com/api/v1", loaded.APIUrl)
	assert.Equal(t, "json", loaded.Output)
}

func TestRemoveTokenClearsOnlyToken(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	cfg := &Config{
		APIUrl: "https://example.com/api/v1",
		Token:  "nxd_test123",
		Output: "json",
	}
	require.NoError(t, Save(path, cfg))

	require.NoError(t, RemoveToken(path))

	loaded, err := Load(path)
	require.NoError(t, err)
	assert.Equal(t, "", loaded.Token)
	assert.Equal(t, "https://example.com/api/v1", loaded.APIUrl)
	assert.Equal(t, "json", loaded.Output)
}

func TestSetTokenCreatesNewFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	require.NoError(t, SetToken(path, "nxd_test123"))

	loaded, err := Load(path)
	require.NoError(t, err)
	assert.Equal(t, "nxd_test123", loaded.Token)
}
