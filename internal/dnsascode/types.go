package dnsascode

// ConfigFile represents the top-level nexdns.yaml structure.
type ConfigFile struct {
	Zones map[string]ZoneConfig `yaml:"zones"`
}

// ZoneConfig represents a zone in nexdns.yaml.
type ZoneConfig struct {
	NSGroup string         `yaml:"ns_group,omitempty"`
	DNSSEC  bool           `yaml:"dnssec,omitempty"`
	Records []RecordConfig `yaml:"records"`
}

// RecordConfig represents a record in nexdns.yaml.
type RecordConfig struct {
	Type     string `yaml:"type"`
	Name     string `yaml:"name"`
	Content  string `yaml:"content"`
	TTL      int    `yaml:"ttl,omitempty"`
	Priority *int   `yaml:"priority,omitempty"`
	Flags    *int   `yaml:"flags,omitempty"`
	Tag      string `yaml:"tag,omitempty"`
	Weight   *int   `yaml:"weight,omitempty"`
	Port     *int   `yaml:"port,omitempty"`
}

// Operation represents a change to be applied.
type Operation struct {
	Action   string // "add", "update", "delete"
	Zone     string
	Type     string
	Name     string
	Content  string
	TTL      int
	Priority *int
	Weight   *int
	Port     *int
	Flags    *int
	Tag      string
	// For updates: the old values
	OldContent string
	OldTTL     int
	// For deletes: the record ID
	RecordID string
}
