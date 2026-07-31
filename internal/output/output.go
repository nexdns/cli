package output

import (
	"fmt"
	"io"

	"github.com/nexdns/cli/internal/api"
)

// Formatter defines the interface for output formatting.
type Formatter interface {
	FormatZones(zones []api.ZoneListItem, meta *api.PaginationMeta)
	FormatZone(zone *api.Zone)
	FormatZoneWithDNSSEC(zone *api.Zone, dnssec *api.DNSSECStatus)
	FormatRecords(records []api.Record)
	FormatRecord(record *api.Record)
	FormatDNSSEC(status *api.DNSSECStatus)
	FormatAccount(account *api.Account)
	FormatAPIKeys(keys []api.APIKey)
	FormatWebhooks(webhooks []api.Webhook)
	FormatWebhook(webhook *api.Webhook)
	FormatMessage(msg string)
}

// New creates a Formatter for the given format. An unknown colour mode is
// refused here as well: it used to be accepted silently and fall through to
// auto-detection, so `--color allways` looked like it had worked.
func New(format string, w io.Writer, quiet bool, colorMode string) (Formatter, error) {
	if err := ValidateFormat(format); err != nil {
		return nil, err
	}
	if err := ValidateColorMode(colorMode); err != nil {
		return nil, err
	}

	switch format {
	case "json":
		return NewJSONFormatter(w, quiet), nil
	case "yaml":
		return NewYAMLFormatter(w), nil
	case "csv":
		return NewCSVFormatter(w), nil
	default:
		return NewTableFormatter(w, quiet, colorMode), nil
	}
}

// ValidateFormat reports whether format names a supported output formatter.
func ValidateFormat(format string) error {
	switch format {
	case "table", "json", "yaml", "csv":
		return nil
	default:
		return fmt.Errorf("unknown output format: %q (supported: table, json, yaml, csv)", format)
	}
}

// ValidateColorMode reports whether mode names a supported colour mode.
func ValidateColorMode(mode string) error {
	switch mode {
	case "auto", "always", "never":
		return nil
	default:
		return fmt.Errorf("unknown color mode: %q (supported: auto, always, never)", mode)
	}
}
