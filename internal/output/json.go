package output

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/nexdns/cli/internal/api"
)

// JSONFormatter outputs data as JSON.
type JSONFormatter struct {
	w     io.Writer
	quiet bool
}

// NewJSONFormatter creates a new JSON formatter.
func NewJSONFormatter(w io.Writer, quiet bool) *JSONFormatter {
	return &JSONFormatter{w: w, quiet: quiet}
}

func (f *JSONFormatter) marshal(v interface{}) {
	var data []byte
	var err error
	if f.quiet {
		data, err = json.Marshal(v)
	} else {
		data, err = json.MarshalIndent(v, "", "  ")
	}
	if err != nil {
		fmt.Fprintf(f.w, `{"error":"marshal failed: %s"}`, err)
		return
	}
	fmt.Fprintln(f.w, string(data))
}

func (f *JSONFormatter) FormatZones(zones []api.ZoneListItem, _ *api.PaginationMeta) {
	f.marshal(zones)
}

func (f *JSONFormatter) FormatZone(zone *api.Zone) {
	f.marshal(zone)
}

func (f *JSONFormatter) FormatZoneWithDNSSEC(zone *api.Zone, dnssec *api.DNSSECStatus) {
	data := map[string]interface{}{"zone": zone}
	if dnssec != nil {
		data["dnssec"] = dnssec
	}
	f.marshal(data)
}

func (f *JSONFormatter) FormatRecords(records []api.Record) {
	f.marshal(records)
}

func (f *JSONFormatter) FormatRecord(record *api.Record) {
	f.marshal(record)
}

func (f *JSONFormatter) FormatDNSSEC(status *api.DNSSECStatus) {
	f.marshal(status)
}

func (f *JSONFormatter) FormatAccount(account *api.Account) {
	f.marshal(account)
}

func (f *JSONFormatter) FormatAPIKeys(keys []api.APIKey) {
	f.marshal(keys)
}

func (f *JSONFormatter) FormatWebhooks(webhooks []api.Webhook) {
	f.marshal(webhooks)
}

func (f *JSONFormatter) FormatWebhook(webhook *api.Webhook) {
	f.marshal(webhook)
}

func (f *JSONFormatter) FormatMessage(msg string) {
	f.marshal(map[string]string{"message": msg})
}
