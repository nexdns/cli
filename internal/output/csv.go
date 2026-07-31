package output

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/nexdns/cli/internal/api"
)

// CSVFormatter outputs data as CSV.
type CSVFormatter struct {
	w io.Writer
}

// NewCSVFormatter creates a new CSV formatter.
func NewCSVFormatter(w io.Writer) *CSVFormatter {
	return &CSVFormatter{w: w}
}

func (f *CSVFormatter) writeCSV(header []string, rows [][]string) {
	cw := csv.NewWriter(f.w)
	cw.Write(header)
	for _, row := range rows {
		cw.Write(row)
	}
	cw.Flush()
}

func (f *CSVFormatter) FormatZones(zones []api.ZoneListItem, _ *api.PaginationMeta) {
	header := []string{"name", "type", "status", "ns_group", "records_count", "created_at"}
	var rows [][]string
	for _, z := range zones {
		rows = append(rows, []string{z.Name, z.Type, z.Status, z.NSGroup, strconv.Itoa(z.RecordsCount), z.CreatedAt})
	}
	f.writeCSV(header, rows)
}

func (f *CSVFormatter) FormatZoneWithDNSSEC(zone *api.Zone, dnssec *api.DNSSECStatus) {
	dnssecStr := ""
	if dnssec != nil {
		if dnssec.Enabled {
			dnssecStr = "enabled"
		} else {
			dnssecStr = "disabled"
		}
	}
	nsGroup := ""
	if zone.NSGroup != nil {
		nsGroup = zone.NSGroup.Name
	}
	header := []string{"name", "type", "status", "ns_group", "records_count", "dnssec", "created_at"}
	rows := [][]string{{zone.Name, zone.Type, zone.Status, nsGroup, strconv.Itoa(zone.RecordsCount), dnssecStr, zone.CreatedAt}}
	f.writeCSV(header, rows)
}

func (f *CSVFormatter) FormatZone(zone *api.Zone) {
	nsGroup := ""
	if zone.NSGroup != nil {
		nsGroup = zone.NSGroup.Name
	}
	header := []string{"name", "type", "status", "ns_group", "records_count", "created_at"}
	rows := [][]string{{zone.Name, zone.Type, zone.Status, nsGroup, strconv.Itoa(zone.RecordsCount), zone.CreatedAt}}
	f.writeCSV(header, rows)
}

func (f *CSVFormatter) FormatRecords(records []api.Record) {
	header := []string{"id", "type", "name", "content", "ttl", "priority"}
	var rows [][]string
	for _, r := range records {
		priority := ""
		if r.Priority != nil {
			priority = strconv.Itoa(*r.Priority)
		}
		rows = append(rows, []string{r.ID, r.Type, r.Name, r.Content, strconv.Itoa(r.TTL), priority})
	}
	f.writeCSV(header, rows)
}

func (f *CSVFormatter) FormatRecord(record *api.Record) {
	f.FormatRecords([]api.Record{*record})
}

func (f *CSVFormatter) FormatDNSSEC(status *api.DNSSECStatus) {
	header := []string{"enabled", "ds_records"}
	ds := ""
	if len(status.DSRecords) > 0 {
		ds = fmt.Sprintf("%v", status.DSRecords)
	}
	rows := [][]string{{strconv.FormatBool(status.Enabled), ds}}
	f.writeCSV(header, rows)
}

func (f *CSVFormatter) FormatAccount(account *api.Account) {
	header := []string{"email", "name", "status"}
	rows := [][]string{{account.Email, account.Name, account.Status}}
	f.writeCSV(header, rows)
}

func (f *CSVFormatter) FormatAPIKeys(keys []api.APIKey) {
	header := []string{"id", "name", "key_prefix", "permissions", "last_used_at", "expires_at"}
	var rows [][]string
	for _, k := range keys {
		perms := ""
		if len(k.Permissions) > 0 {
			perms = fmt.Sprintf("%v", k.Permissions)
		}
		lastUsed := ""
		if k.LastUsedAt != nil {
			lastUsed = *k.LastUsedAt
		}
		expires := ""
		if k.ExpiresAt != nil {
			expires = *k.ExpiresAt
		}
		rows = append(rows, []string{k.ID, k.Name, k.KeyPrefix, perms, lastUsed, expires})
	}
	f.writeCSV(header, rows)
}

func (f *CSVFormatter) FormatWebhooks(webhooks []api.Webhook) {
	header := []string{"id", "url", "events", "is_active", "failure_count", "last_triggered_at", "created_at"}
	var rows [][]string
	for _, wh := range webhooks {
		rows = append(rows, []string{
			wh.ID,
			wh.URL,
			strings.Join(wh.Events, " "),
			strconv.FormatBool(wh.IsActive),
			strconv.Itoa(wh.FailureCount),
			wh.LastTriggeredAt,
			wh.CreatedAt,
		})
	}
	f.writeCSV(header, rows)
}

// FormatWebhook emits the deliveries when the detail view has them: a single
// webhook as one CSV row carries less than the attempts that explain its state.
func (f *CSVFormatter) FormatWebhook(webhook *api.Webhook) {
	if len(webhook.RecentDeliveries) == 0 {
		f.FormatWebhooks([]api.Webhook{*webhook})
		return
	}

	header := []string{"event_id", "event_type", "status", "attempts", "last_status_code", "last_error", "created_at", "delivered_at"}
	var rows [][]string
	for _, d := range webhook.RecentDeliveries {
		code := ""
		if d.LastStatusCode != nil {
			code = strconv.Itoa(*d.LastStatusCode)
		}
		rows = append(rows, []string{
			d.EventID, d.EventType, d.Status, strconv.Itoa(d.Attempts),
			code, d.LastError, d.CreatedAt, d.DeliveredAt,
		})
	}
	f.writeCSV(header, rows)
}

func (f *CSVFormatter) FormatMessage(msg string) {
	fmt.Fprintln(f.w, msg)
}
