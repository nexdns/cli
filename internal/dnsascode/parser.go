package dnsascode

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

var envVarPattern = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)\}`)

// ParseConfigFile reads and parses a nexdns.yaml config file.
func ParseConfigFile(path string) (*ConfigFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	// Substitute environment variables
	content, unresolved := substituteEnvVars(string(data))
	if len(unresolved) > 0 {
		return nil, fmt.Errorf("unset environment variable(s) referenced by %s: %s", path, strings.Join(unresolved, ", "))
	}

	var cfg ConfigFile
	if err := yaml.Unmarshal([]byte(content), &cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	// Validate
	for zoneName, zone := range cfg.Zones {
		for i, r := range zone.Records {
			if r.Type == "" {
				return nil, fmt.Errorf("zone %s: record #%d: type is required", zoneName, i+1)
			}
			if r.Name == "" {
				return nil, fmt.Errorf("zone %s: record #%d: name is required", zoneName, i+1)
			}
			if r.Content == "" {
				return nil, fmt.Errorf("zone %s: record #%d: content is required", zoneName, i+1)
			}
			// Normalize type to uppercase
			cfg.Zones[zoneName].Records[i].Type = strings.ToUpper(r.Type)
		}
	}

	return &cfg, nil
}

// substituteEnvVars expands every ${VAR} reference and reports the names that
// had no value, so the caller can refuse the file.
//
// Leaving an unresolved reference in place - the previous behaviour - meant a
// CI run with a missing variable planned to overwrite live data with the literal
// "${VAR}". For an address type the API refuses it, which turns into a confusing
// mid-apply validation error; for TXT the string is legal, so the literal would
// have been published as the record's value.
func substituteEnvVars(content string) (string, []string) {
	var unresolved []string
	seen := map[string]bool{}

	expanded := envVarPattern.ReplaceAllStringFunc(content, func(match string) string {
		varName := envVarPattern.FindStringSubmatch(match)[1]
		if val, ok := os.LookupEnv(varName); ok {
			return val
		}
		if !seen[varName] {
			seen[varName] = true
			unresolved = append(unresolved, varName)
		}
		return match
	})

	return expanded, unresolved
}
