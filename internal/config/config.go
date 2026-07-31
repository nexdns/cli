package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// Default values used by Resolve when a setting is not supplied by the config
// file, an environment variable, or a flag.
const (
	DefaultAPIURL  = "https://api.nexdns.tech/v1"
	DefaultOutput  = "table"
	DefaultColor   = "auto"
	DefaultTimeout = 30

	configDirName  = ".nexdns"
	configFileName = "config.yaml"
)

// Config is the persisted CLI configuration, stored as YAML in
// ~/.nexdns/config.yaml. All fields are optional; empty fields fall back to
// defaults during Resolve.
type Config struct {
	APIUrl  string `yaml:"api_url,omitempty"`
	Token   string `yaml:"token,omitempty"`
	Output  string `yaml:"output,omitempty"`
	Color   string `yaml:"color,omitempty"`
	Timeout int    `yaml:"timeout,omitempty"`
}

// DefaultConfigPath returns the default config file path, ~/.nexdns/config.yaml.
func DefaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join("~", configDirName, configFileName)
	}
	return filepath.Join(home, configDirName, configFileName)
}

// Load reads and parses the config file at path. A missing file is not an
// error: it returns an empty *Config so first-run commands work without setup.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	return &cfg, nil
}

// Save writes cfg to path as YAML, creating the parent directory (0700) and the
// file (0600) since it stores the API token.
func Save(path string, cfg *Config) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}
	return nil
}

// SetToken loads the config at path, sets the API token, and saves it.
func SetToken(path, token string) error {
	cfg, err := Load(path)
	if err != nil {
		return err
	}
	cfg.Token = token
	return Save(path, cfg)
}

// SetAPIUrl loads the config at path, sets the API base URL, and saves it.
func SetAPIUrl(path, url string) error {
	cfg, err := Load(path)
	if err != nil {
		return err
	}
	cfg.APIUrl = url
	return Save(path, cfg)
}

// SetOutput persists the default output format in the config at path.
func SetOutput(path, format string) error {
	cfg, err := Load(path)
	if err != nil {
		return err
	}
	cfg.Output = format
	return Save(path, cfg)
}

// SetColor persists the default colour mode in the config at path.
func SetColor(path, mode string) error {
	cfg, err := Load(path)
	if err != nil {
		return err
	}
	cfg.Color = mode
	return Save(path, cfg)
}

// SetTimeout persists the default HTTP timeout (seconds) in the config at path.
func SetTimeout(path string, seconds int) error {
	cfg, err := Load(path)
	if err != nil {
		return err
	}
	cfg.Timeout = seconds
	return Save(path, cfg)
}

// RemoveToken clears the saved API token in the config at path (used by logout).
func RemoveToken(path string) error {
	cfg, err := Load(path)
	if err != nil {
		return err
	}
	cfg.Token = ""
	return Save(path, cfg)
}

func resolveConfigPath(cmd *cobra.Command) string {
	if cmd.Flags().Changed("config") {
		v, _ := cmd.Flags().GetString("config")
		return v
	}
	if v := os.Getenv("NEXDNS_CONFIG"); v != "" {
		return v
	}
	return DefaultConfigPath()
}

// Resolve returns the effective configuration for cmd, layering sources in
// increasing priority: built-in defaults, the config file, environment
// variables (NEXDNS_TOKEN, NEXDNS_API_URL, NEXDNS_TIMEOUT), then explicitly-set
// flags. NO_COLOR, if present, forces the color mode to "never".
func Resolve(cmd *cobra.Command) (*Config, error) {
	path := resolveConfigPath(cmd)
	cfg, err := Load(path)
	if err != nil {
		return nil, err
	}

	resolved := &Config{
		APIUrl:  DefaultAPIURL,
		Output:  DefaultOutput,
		Color:   DefaultColor,
		Timeout: DefaultTimeout,
	}

	// Layer 1: config file values
	if cfg.APIUrl != "" {
		resolved.APIUrl = cfg.APIUrl
	}
	if cfg.Token != "" {
		resolved.Token = cfg.Token
	}
	if cfg.Output != "" {
		resolved.Output = cfg.Output
	}
	if cfg.Color != "" {
		resolved.Color = cfg.Color
	}
	if cfg.Timeout != 0 {
		resolved.Timeout = cfg.Timeout
	}

	// Layer 2: environment variables
	if v := os.Getenv("NEXDNS_TOKEN"); v != "" {
		resolved.Token = v
	}
	if v := os.Getenv("NEXDNS_API_URL"); v != "" {
		resolved.APIUrl = v
	}
	if v := os.Getenv("NEXDNS_TIMEOUT"); v != "" {
		if t, err := strconv.Atoi(v); err == nil {
			resolved.Timeout = t
		}
	}

	// Layer 3: flags (highest priority)
	if cmd.Flags().Changed("token") {
		resolved.Token, _ = cmd.Flags().GetString("token")
	}
	if cmd.Flags().Changed("api-url") {
		resolved.APIUrl, _ = cmd.Flags().GetString("api-url")
	}
	if cmd.Flags().Changed("output") {
		resolved.Output, _ = cmd.Flags().GetString("output")
	}
	if cmd.Flags().Changed("color") {
		resolved.Color, _ = cmd.Flags().GetString("color")
	}
	if cmd.Flags().Changed("timeout") {
		resolved.Timeout, _ = cmd.Flags().GetInt("timeout")
	}
	if cmd.Flags().Changed("quiet") {
		// quiet is handled per-command, just ensure it's accessible
	}
	if cmd.Flags().Changed("verbose") {
		// verbose is handled per-command, just ensure it's accessible
	}

	// NO_COLOR override
	if _, ok := os.LookupEnv("NO_COLOR"); ok {
		resolved.Color = "never"
	}

	return resolved, nil
}

// ConfigPath returns the config file path cmd will use, honouring the --config
// flag and the NEXDNS_CONFIG environment variable before the default path.
func ConfigPath(cmd *cobra.Command) string {
	return resolveConfigPath(cmd)
}
