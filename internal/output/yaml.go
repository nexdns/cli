package output

import (
	"fmt"
	"io"

	"gopkg.in/yaml.v3"

	"github.com/nexdns/cli/internal/api"
)

// YAMLFormatter outputs data as YAML.
type YAMLFormatter struct {
	w io.Writer
}

// NewYAMLFormatter creates a new YAML formatter.
func NewYAMLFormatter(w io.Writer) *YAMLFormatter {
	return &YAMLFormatter{w: w}
}

func (f *YAMLFormatter) marshal(v interface{}) {
	data, err := yaml.Marshal(v)
	if err != nil {
		fmt.Fprintf(f.w, "error: %s\n", err)
		return
	}
	fmt.Fprint(f.w, string(data))
}

func (f *YAMLFormatter) FormatZones(zones []api.ZoneListItem, _ *api.PaginationMeta) {
	f.marshal(zones)
}

func (f *YAMLFormatter) FormatZone(zone *api.Zone) {
	f.marshal(zone)
}

func (f *YAMLFormatter) FormatZoneWithDNSSEC(zone *api.Zone, dnssec *api.DNSSECStatus) {
	data := map[string]interface{}{"zone": zone}
	if dnssec != nil {
		data["dnssec"] = dnssec
	}
	f.marshal(data)
}

func (f *YAMLFormatter) FormatRecords(records []api.Record) {
	f.marshal(records)
}

func (f *YAMLFormatter) FormatRecord(record *api.Record) {
	f.marshal(record)
}

func (f *YAMLFormatter) FormatDNSSEC(status *api.DNSSECStatus) {
	f.marshal(status)
}

func (f *YAMLFormatter) FormatAccount(account *api.Account) {
	f.marshal(account)
}

func (f *YAMLFormatter) FormatAPIKeys(keys []api.APIKey) {
	f.marshal(keys)
}

func (f *YAMLFormatter) FormatWebhooks(webhooks []api.Webhook) {
	f.marshal(webhooks)
}

func (f *YAMLFormatter) FormatWebhook(webhook *api.Webhook) {
	f.marshal(webhook)
}

func (f *YAMLFormatter) FormatMessage(msg string) {
	fmt.Fprintln(f.w, msg)
}
