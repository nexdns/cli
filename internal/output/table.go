package output

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/fatih/color"

	"github.com/nexdns/cli/internal/api"
)

// TableFormatter outputs data as human-readable tables.
type TableFormatter struct {
	w     io.Writer
	quiet bool
	bold  func(a ...interface{}) string
	green func(a ...interface{}) string
	red   func(a ...interface{}) string
}

// NewTableFormatter creates a new table formatter.
func NewTableFormatter(w io.Writer, quiet bool, colorMode string) *TableFormatter {
	switch colorMode {
	case "always":
		color.NoColor = false
	case "never":
		color.NoColor = true
		// "auto" uses fatih/color's default TTY detection
	}

	return &TableFormatter{
		w:     w,
		quiet: quiet,
		bold:  color.New(color.Bold).SprintFunc(),
		green: color.New(color.FgGreen).SprintFunc(),
		red:   color.New(color.FgRed).SprintFunc(),
	}
}

func (f *TableFormatter) newWriter() *tabwriter.Writer {
	return tabwriter.NewWriter(f.w, 0, 0, 2, ' ', 0)
}

func (f *TableFormatter) statusColor(status string) string {
	switch status {
	case "active":
		return f.green(status)
	case "suspended", "expired", "cancelled":
		return f.red(status)
	default:
		return status
	}
}

func truncate(s string, max int) string {
	if len(s) > max {
		return s[:max-3] + "..."
	}
	return s
}

func (f *TableFormatter) FormatZones(zones []api.ZoneListItem, meta *api.PaginationMeta) {
	tw := f.newWriter()
	fmt.Fprintln(tw, f.bold("NAME")+"\t"+f.bold("TYPE")+"\t"+f.bold("STATUS")+"\t"+f.bold("NS GROUP")+"\t"+f.bold("RECORDS")+"\t"+f.bold("CREATED"))

	for _, z := range zones {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%d\t%s\n",
			displayName(z.Name, z.UnicodeName), z.Type, f.statusColor(z.Status), z.NSGroup,
			z.RecordsCount, formatDate(z.CreatedAt))
	}
	tw.Flush()

	if !f.quiet {
		total := len(zones)
		if meta != nil {
			total = meta.Total
		}
		fmt.Fprintf(f.w, "\n%d zone(s)\n", total)
	}
}

func (f *TableFormatter) FormatZone(zone *api.Zone) {
	f.FormatZoneWithDNSSEC(zone, nil)
}

func (f *TableFormatter) FormatZoneWithDNSSEC(zone *api.Zone, dnssec *api.DNSSECStatus) {
	tw := f.newWriter()
	fmt.Fprintf(tw, "Name:\t%s\n", displayName(zone.Name, zone.UnicodeName))
	fmt.Fprintf(tw, "Type:\t%s\n", zone.Type)
	fmt.Fprintf(tw, "Status:\t%s\n", f.statusColor(zone.Status))
	if zone.NSGroup != nil {
		fmt.Fprintf(tw, "NS Group:\t%s (%s)\n", zone.NSGroup.Name, zone.NSGroup.Slug)
	}
	fmt.Fprintf(tw, "Records:\t%d\n", zone.RecordsCount)
	if dnssec != nil {
		if dnssec.Enabled {
			fmt.Fprintf(tw, "DNSSEC:\t%s\n", f.green("enabled"))
		} else {
			fmt.Fprintf(tw, "DNSSEC:\t%s\n", f.red("disabled"))
		}
	}
	if zone.Nameservers != nil {
		fmt.Fprintf(tw, "Nameservers:\t%s\n", strings.Join(zone.Nameservers, ", "))
	}
	if zone.SOA != nil {
		fmt.Fprintf(tw, "SOA Primary:\t%s\n", zone.SOA.PrimaryNS)
		fmt.Fprintf(tw, "SOA Email:\t%s\n", zone.SOA.AdminEmail)
		fmt.Fprintf(tw, "SOA Serial:\t%d\n", zone.SOA.Serial)
	}
	fmt.Fprintf(tw, "Created:\t%s\n", zone.CreatedAt)
	fmt.Fprintf(tw, "Updated:\t%s\n", zone.UpdatedAt)
	tw.Flush()
}

func (f *TableFormatter) FormatRecords(records []api.Record) {
	tw := f.newWriter()

	hasPriority := false
	for _, r := range records {
		if r.Priority != nil {
			hasPriority = true
			break
		}
	}

	header := f.bold("ID") + "\t" + f.bold("TYPE") + "\t" + f.bold("NAME") + "\t" + f.bold("CONTENT") + "\t" + f.bold("TTL")
	if hasPriority {
		header += "\t" + f.bold("PRIORITY")
	}
	fmt.Fprintln(tw, header)

	for _, r := range records {
		line := fmt.Sprintf("%s\t%s\t%s\t%s\t%d",
			r.ID, r.Type, displayName(r.Name, r.UnicodeName), truncate(r.Content, 50), r.TTL)
		if hasPriority {
			if r.Priority != nil {
				line += fmt.Sprintf("\t%d", *r.Priority)
			} else {
				line += "\t"
			}
		}
		fmt.Fprintln(tw, line)
	}
	tw.Flush()

	if !f.quiet {
		fmt.Fprintf(f.w, "\n%d record(s)\n", len(records))
	}
}

func (f *TableFormatter) FormatRecord(record *api.Record) {
	tw := f.newWriter()
	fmt.Fprintf(tw, "ID:\t%s\n", record.ID)
	fmt.Fprintf(tw, "Type:\t%s\n", record.Type)
	fmt.Fprintf(tw, "Name:\t%s\n", record.Name)
	fmt.Fprintf(tw, "Content:\t%s\n", record.Content)
	fmt.Fprintf(tw, "TTL:\t%d\n", record.TTL)
	if record.Priority != nil {
		fmt.Fprintf(tw, "Priority:\t%d\n", *record.Priority)
	}
	tw.Flush()
}

func (f *TableFormatter) FormatDNSSEC(status *api.DNSSECStatus) {
	tw := f.newWriter()
	if status.Enabled {
		fmt.Fprintf(tw, "DNSSEC:\t%s\n", f.green("enabled"))
	} else {
		fmt.Fprintf(tw, "DNSSEC:\t%s\n", f.red("disabled"))
	}
	tw.Flush()

	for _, key := range status.Keys {
		fmt.Fprintf(f.w, "\n%s:\n", strings.ToUpper(key.Type))
		tw = f.newWriter()
		fmt.Fprintf(tw, "  Algorithm:\t%s (%d bits)\n", key.Algorithm, key.Bits)
		fmt.Fprintf(tw, "  Active:\t%v\n", key.Active)
		fmt.Fprintf(tw, "  DNSKEY:\t%s\n", truncate(key.DNSKEY, 60))
		tw.Flush()
	}

	if len(status.DSRecords) > 0 {
		fmt.Fprintf(f.w, "\nDS Records (add these at your registrar):\n")
		for _, ds := range status.DSRecords {
			fmt.Fprintf(f.w, "  %s\n", ds)
		}
	}
}

func (f *TableFormatter) FormatAccount(account *api.Account) {
	tw := f.newWriter()
	fmt.Fprintf(tw, "Email:\t%s\n", account.Email)
	fmt.Fprintf(tw, "Name:\t%s\n", account.Name)
	fmt.Fprintf(tw, "Status:\t%s\n", f.statusColor(account.Status))
	if account.Subscription != nil {
		fmt.Fprintf(tw, "Plan:\t%s (%s)\n", account.Subscription.Plan, account.Subscription.BillingCycle)
		fmt.Fprintf(tw, "Period:\t%s – %s\n", account.Subscription.CurrentPeriodStart, account.Subscription.CurrentPeriodEnd)
	}
	tw.Flush()
}

func (f *TableFormatter) FormatAPIKeys(keys []api.APIKey) {
	tw := f.newWriter()
	fmt.Fprintln(tw, f.bold("ID")+"\t"+f.bold("NAME")+"\t"+f.bold("PREFIX")+"\t"+f.bold("PERMISSIONS")+"\t"+f.bold("LAST USED")+"\t"+f.bold("EXPIRES"))

	for _, k := range keys {
		perms := "full access"
		if len(k.Permissions) > 0 {
			perms = strings.Join(k.Permissions, ", ")
		}
		lastUsed := "never"
		if k.LastUsedAt != nil {
			lastUsed = formatDate(*k.LastUsedAt)
		}
		expires := "never"
		if k.ExpiresAt != nil {
			expires = formatDate(*k.ExpiresAt)
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\n",
			k.ID, k.Name, k.KeyPrefix, truncate(perms, 30), lastUsed, expires)
	}
	tw.Flush()

	if !f.quiet {
		fmt.Fprintf(f.w, "\n%d API key(s)\n", len(keys))
	}
}

func (f *TableFormatter) FormatWebhooks(webhooks []api.Webhook) {
	tw := f.newWriter()
	fmt.Fprintln(tw, f.bold("ID")+"\t"+f.bold("URL")+"\t"+f.bold("EVENTS")+"\t"+f.bold("ACTIVE")+"\t"+f.bold("FAILURES")+"\t"+f.bold("LAST TRIGGERED"))

	for _, wh := range webhooks {
		lastTriggered := "never"
		if wh.LastTriggeredAt != "" {
			lastTriggered = formatDate(wh.LastTriggeredAt)
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\n",
			wh.ID,
			truncate(wh.URL, 40),
			truncate(strings.Join(wh.Events, ", "), 34),
			f.activeFlag(wh.IsActive),
			f.failureCount(wh.FailureCount),
			lastTriggered,
		)
	}
	tw.Flush()

	if !f.quiet {
		fmt.Fprintf(f.w, "\n%d webhook(s)\n", len(webhooks))
	}
}

func (f *TableFormatter) FormatWebhook(webhook *api.Webhook) {
	tw := f.newWriter()
	fmt.Fprintf(tw, "ID:\t%s\n", webhook.ID)
	fmt.Fprintf(tw, "URL:\t%s\n", webhook.URL)
	if webhook.Description != "" {
		fmt.Fprintf(tw, "Description:\t%s\n", webhook.Description)
	}
	fmt.Fprintf(tw, "Active:\t%s\n", f.activeFlag(webhook.IsActive))
	fmt.Fprintf(tw, "Events:\t%s\n", strings.Join(webhook.Events, ", "))
	fmt.Fprintf(tw, "Failures:\t%s\n", f.failureCount(webhook.FailureCount))
	if webhook.LastTriggeredAt != "" {
		fmt.Fprintf(tw, "Last triggered:\t%s\n", webhook.LastTriggeredAt)
	}
	fmt.Fprintf(tw, "Created:\t%s\n", webhook.CreatedAt)
	tw.Flush()

	if len(webhook.RecentDeliveries) == 0 {
		return
	}

	fmt.Fprintf(f.w, "\n%s\n", f.bold("Recent deliveries"))
	dw := f.newWriter()
	fmt.Fprintln(dw, f.bold("EVENT")+"\t"+f.bold("TYPE")+"\t"+f.bold("STATUS")+"\t"+f.bold("ATTEMPTS")+"\t"+f.bold("CODE")+"\t"+f.bold("WHEN"))
	for _, d := range webhook.RecentDeliveries {
		code := "–"
		if d.LastStatusCode != nil {
			code = strconv.Itoa(*d.LastStatusCode)
		}
		when := d.CreatedAt
		if d.DeliveredAt != "" {
			when = d.DeliveredAt
		}
		fmt.Fprintf(dw, "%s\t%s\t%s\t%d\t%s\t%s\n",
			d.EventID, d.EventType, f.deliveryStatus(d.Status), d.Attempts, code, when)
	}
	dw.Flush()

	// The error is the reason someone is reading this table at all, so it is
	// printed in full under the row rather than truncated into a column.
	for _, d := range webhook.RecentDeliveries {
		if d.LastError != "" {
			fmt.Fprintf(f.w, "\n%s: %s\n", d.EventID, f.red(d.LastError))
		}
	}
}

func (f *TableFormatter) activeFlag(active bool) string {
	if active {
		return f.green("yes")
	}
	return f.red("no")
}

// failureCount colours a non-zero run of failures: the platform deactivates an
// endpoint that keeps failing, so the number is a warning, not a statistic.
func (f *TableFormatter) failureCount(count int) string {
	if count == 0 {
		return "0"
	}
	return f.red(strconv.Itoa(count))
}

func (f *TableFormatter) deliveryStatus(status string) string {
	switch status {
	case "delivered":
		return f.green(status)
	case "failed", "cancelled":
		return f.red(status)
	default:
		return status
	}
}

func (f *TableFormatter) FormatMessage(msg string) {
	fmt.Fprintln(f.w, msg)
}

func formatDate(s string) string {
	if len(s) >= 10 {
		return s[:10]
	}
	return s
}

// displayName prefers the native-script form of an internationalized zone name
// or record label for human-readable output. The machine-readable formats
// (json/yaml/csv) keep both fields, so scripts still see the canonical ACE name.
func displayName(name, unicodeName string) string {
	if unicodeName != "" && unicodeName != name {
		return unicodeName
	}
	return name
}
