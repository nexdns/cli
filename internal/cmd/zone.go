package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/nexdns/cli/internal/api"
	"github.com/nexdns/cli/internal/dns"
)

var zoneCmd = &cobra.Command{
	Use:   "zone",
	Short: "Manage DNS zones",
}

var zoneListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all zones",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newAPIClient(cmd)
		if err != nil {
			return err
		}
		formatter, err := newFormatter(cmd)
		if err != nil {
			return err
		}

		ctx := context.Background()
		all, _ := cmd.Flags().GetBool("all")
		search, _ := cmd.Flags().GetString("search")

		if all {
			zones, err := client.ListAllZones(ctx, api.ListZonesOptions{Search: search})
			if err != nil {
				return err
			}
			formatter.FormatZones(zones, &api.PaginationMeta{Total: len(zones)})
			return nil
		}

		page, _ := cmd.Flags().GetInt("page")
		perPage, _ := cmd.Flags().GetInt("per-page")
		resp, err := client.ListZones(ctx, api.ListZonesOptions{
			Page:    page,
			PerPage: perPage,
			Search:  search,
		})
		if err != nil {
			return err
		}
		formatter.FormatZones(resp.Data, resp.Meta)
		return nil
	},
}

var zoneAddCmd = &cobra.Command{
	Use:   "add <domain>",
	Short: "Create a new zone",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		domain := args[0]

		if isDryRun(cmd) {
			fmt.Fprintf(cmd.OutOrStdout(), "[dry-run] Would create zone %s\n", domain)
			return nil
		}

		client, err := newAPIClient(cmd)
		if err != nil {
			return err
		}

		ctx := context.Background()
		req := api.CreateZoneRequest{Name: domain}
		if zoneType, _ := cmd.Flags().GetString("type"); zoneType != "" {
			req.Type = zoneType
		}
		if masterIP, _ := cmd.Flags().GetString("master-ip"); masterIP != "" {
			req.MasterIP = masterIP
		}

		nsGroup, err := resolveNSGroupFlag(ctx, client, cmd)
		if err != nil {
			return err
		}
		if nsGroup != "" {
			req.NSGroup = nsGroup
		}

		zone, err := client.CreateZone(ctx, req)
		if err != nil {
			return err
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Zone %s created (%s, NS group: %s)\n", zoneDisplayName(zone), zone.Type, zone.NSGroup)
		return nil
	},
}

var zoneDeleteCmd = &cobra.Command{
	Use:   "delete <domain>",
	Short: "Delete a zone",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		domain := args[0]
		client, err := newAPIClient(cmd)
		if err != nil {
			return err
		}

		ctx := context.Background()
		zoneID, err := resolveZone(ctx, client, domain)
		if err != nil {
			return err
		}

		if isDryRun(cmd) {
			fmt.Fprintf(cmd.OutOrStdout(), "[dry-run] Would delete zone %s\n", domain)
			return nil
		}

		force, _ := cmd.Flags().GetBool("force")
		if !force {
			if !confirmPrompt(fmt.Sprintf("Delete zone %s and all its records?", domain)) {
				fmt.Fprintln(cmd.OutOrStdout(), "Cancelled")
				return nil
			}
		}

		if err := client.DeleteZone(ctx, zoneID); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Zone %s deleted\n", domain)
		return nil
	},
}

var zoneInfoCmd = &cobra.Command{
	Use:   "info <domain>",
	Short: "Show zone details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		domain := args[0]
		client, err := newAPIClient(cmd)
		if err != nil {
			return err
		}
		formatter, err := newFormatter(cmd)
		if err != nil {
			return err
		}

		ctx := context.Background()
		zoneID, err := resolveZone(ctx, client, domain)
		if err != nil {
			return err
		}

		zone, err := client.GetZone(ctx, zoneID)
		if err != nil {
			return err
		}

		// Fetch DNSSEC status for zone info display
		dnssecStatus, _ := client.GetDNSSEC(ctx, zoneID)

		formatter.FormatZoneWithDNSSEC(zone, dnssecStatus)
		return nil
	},
}

var zoneMoveCmd = &cobra.Command{
	Use:   "move <domain> <ns-group>",
	Short: "Move a zone into another nameserver group",
	Long: `Move a zone into another nameserver group, named by its slug.

The zone keeps serving throughout: the new group's nameservers are provisioned
before the command returns, and removal from the old group is deferred so
resolvers holding the old delegation are still answered while they refresh.

Update the delegation at your registrar to the new nameservers – "nexdns zone
info" prints them – and note that the platform limits how often one zone may be
moved.

Examples:
  nexdns zone move example.com eu
  nexdns zone move example.com ru --dry-run`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		domain := args[0]
		slug := args[1]

		client, err := newAPIClient(cmd)
		if err != nil {
			return err
		}

		ctx := context.Background()

		// Validated against the account's own catalog, the same check `zone add
		// --ns-group` makes, so a typo is named here instead of coming back as a
		// generic validation error.
		if err := assertNSGroupAvailable(ctx, client, slug); err != nil {
			return err
		}

		zoneID, err := resolveZone(ctx, client, domain)
		if err != nil {
			return err
		}

		if isDryRun(cmd) {
			fmt.Fprintf(cmd.OutOrStdout(), "[dry-run] Would move zone %s to NS group %s\n", domain, slug)
			return nil
		}

		formatter, err := newFormatter(cmd)
		if err != nil {
			return err
		}

		zone, err := client.MoveZone(ctx, zoneID, slug)
		if err != nil {
			return err
		}

		formatter.FormatZone(zone)
		return nil
	},
}

var zoneExportCmd = &cobra.Command{
	Use:   "export <domain>",
	Short: "Export zone (BIND or JSON format)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		domain := args[0]
		client, err := newAPIClient(cmd)
		if err != nil {
			return err
		}

		ctx := context.Background()
		zoneID, err := resolveZone(ctx, client, domain)
		if err != nil {
			return err
		}

		format, _ := cmd.Flags().GetString("format")
		result, err := client.ExportZone(ctx, zoneID, format)
		if err != nil {
			return err
		}

		fmt.Fprint(cmd.OutOrStdout(), result)
		return nil
	},
}

var zoneEnsureCmd = &cobra.Command{
	Use:   "ensure <domain>",
	Short: "Create zone if it doesn't exist (idempotent)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		domain := args[0]
		client, err := newAPIClient(cmd)
		if err != nil {
			return err
		}

		ctx := context.Background()
		_, lookupErr := client.GetZoneByName(ctx, domain)
		if lookupErr == nil {
			fmt.Fprintf(cmd.OutOrStdout(), "Zone %s already exists (skipped)\n", domain)
			return nil
		}

		if !api.IsNotFound(lookupErr) {
			return lookupErr
		}

		if isDryRun(cmd) {
			fmt.Fprintf(cmd.OutOrStdout(), "[dry-run] Would create zone %s\n", domain)
			return nil
		}

		req := api.CreateZoneRequest{Name: domain}
		if zoneType, _ := cmd.Flags().GetString("type"); zoneType != "" {
			req.Type = zoneType
		}

		nsGroup, err := resolveNSGroupFlag(ctx, client, cmd)
		if err != nil {
			return err
		}
		if nsGroup != "" {
			req.NSGroup = nsGroup
		}

		zone, err := client.CreateZone(ctx, req)
		if err != nil {
			return err
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Zone %s created (%s, NS group: %s)\n", zoneDisplayName(zone), zone.Type, zone.NSGroup)
		return nil
	},
}

// resolveNSGroupFlag resolves the --ns-group slug flag, validating it against the
// available groups. The API references NS groups by their public slug.
func resolveNSGroupFlag(ctx context.Context, client *api.Client, cmd *cobra.Command) (string, error) {
	if !cmd.Flags().Changed("ns-group") {
		return "", nil
	}

	slug, _ := cmd.Flags().GetString("ns-group")
	if slug == "" {
		return "", nil
	}

	if err := assertNSGroupAvailable(ctx, client, slug); err != nil {
		return "", err
	}

	return slug, nil
}

// assertNSGroupAvailable checks a slug against the groups this account may use,
// naming the available ones when it does not match. The catalog is fetched rather
// than hardcoded: which groups an account gets is a server-side decision, and the
// set differs per instance.
func assertNSGroupAvailable(ctx context.Context, client *api.Client, slug string) error {
	groups, err := client.ListNSGroups(ctx)
	if err != nil {
		return fmt.Errorf("fetching NS groups: %w", err)
	}

	available := make([]string, 0, len(groups))
	for _, g := range groups {
		if g.Slug == slug {
			return nil
		}
		available = append(available, g.Slug)
	}

	return fmt.Errorf("NS group %q not found. Available: %s", slug, strings.Join(available, ", "))
}

var zoneImportCmd = &cobra.Command{
	Use:   "import <domain> <file>",
	Short: "Import records from a BIND zone file",
	Long: `Import records from a BIND zone file. Creates the zone if it doesn't exist.

Examples:
  nexdns zone import example.com zone.txt
  nexdns zone import example.com zone.txt --dry-run
  nexdns zone import example.com zone.txt --replace`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		domain := args[0]
		filePath := args[1]

		f, err := os.Open(filePath)
		if err != nil {
			return fmt.Errorf("opening zone file: %w", err)
		}
		defer f.Close()

		parsed, err := dns.ParseBINDZone(f, domain)
		if err != nil {
			return fmt.Errorf("parsing zone file: %w", err)
		}

		if len(parsed) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "No records found in zone file")
			return nil
		}

		client, err := newAPIClient(cmd)
		if err != nil {
			return err
		}

		ctx := context.Background()

		// Ensure zone exists
		zone, lookupErr := client.GetZoneByName(ctx, domain)
		var zoneID string
		if lookupErr != nil {
			if !api.IsNotFound(lookupErr) {
				return lookupErr
			}
			if isDryRun(cmd) {
				fmt.Fprintf(cmd.OutOrStdout(), "Zone: %s (will be created)\n", domain)
			} else {
				newZone, err := client.CreateZone(ctx, api.CreateZoneRequest{Name: domain})
				if err != nil {
					return fmt.Errorf("creating zone: %w", err)
				}
				zoneID = newZone.ID
				fmt.Fprintf(cmd.OutOrStdout(), "Zone: %s (created)\n", domain)
			}
		} else {
			zoneID = zone.ID
			fmt.Fprintf(cmd.OutOrStdout(), "Zone: %s (exists)\n", domain)
		}

		replace, _ := cmd.Flags().GetBool("replace")

		// Existing records are needed even without --replace: re-importing a file
		// must skip what is already there instead of failing on every duplicate.
		var existing []api.Record
		if zoneID != "" {
			existing, err = client.ListRecords(ctx, zoneID, api.ListRecordsOptions{})
			if err != nil {
				return fmt.Errorf("fetching existing records: %w", err)
			}
		}

		all := dns.ToCreateRequests(parsed)

		// Apex NS is owned by the platform (the API rejects it), so importing a
		// foreign nameserver set is reported as skipped rather than as an error.
		var reqs []api.CreateRecordRequest
		skippedNS := 0
		for _, req := range all {
			if req.Type == "NS" && (req.Name == "@" || req.Name == "") {
				skippedNS++
				continue
			}
			reqs = append(reqs, req)
		}

		existingKeys := make(map[string]bool, len(existing))
		for _, r := range existing {
			existingKeys[storedRecordKey(r)] = true
		}

		var toAdd []api.CreateRecordRequest
		unchanged := 0
		for _, req := range reqs {
			if existingKeys[importRecordKey(req)] {
				unchanged++
				continue
			}
			toAdd = append(toAdd, req)
		}

		var toDelete []api.Record
		if replace {
			toDelete = findRecordsToDelete(existing, reqs)
		}

		reportSkippedNS := func() {
			if skippedNS > 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "  ~ %d apex NS record(s) skipped – nameservers are managed by NexDNS\n", skippedNS)
			}
		}

		if isDryRun(cmd) {
			for _, req := range toAdd {
				fmt.Fprintf(cmd.OutOrStdout(), "  + %s %s %s (TTL %d)\n", req.Type, req.Name, req.Content, req.TTL)
			}
			for _, r := range toDelete {
				fmt.Fprintf(cmd.OutOrStdout(), "  - %s %s %s\n", r.Type, r.Name, r.Content)
			}
			reportSkippedNS()
			fmt.Fprintf(cmd.OutOrStdout(), "%d to add, %d unchanged", len(toAdd), unchanged)
			if replace {
				fmt.Fprintf(cmd.OutOrStdout(), ", %d to remove", len(toDelete))
			}
			fmt.Fprintln(cmd.OutOrStdout(), ". Run without --dry-run to apply.")
			return nil
		}

		// Create records
		added := 0
		failed := 0
		for _, req := range toAdd {
			_, err := client.CreateRecord(ctx, zoneID, req)
			if err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "  ✗ %s %s %s: %v\n", req.Type, req.Name, req.Content, err)
				failed++
				continue
			}
			fmt.Fprintf(cmd.OutOrStdout(), "  + %s %s %s (TTL %d)\n", req.Type, req.Name, req.Content, req.TTL)
			added++
		}

		// If --replace, delete records not in the file (skip NS and SOA at apex)
		deleted := 0
		for _, r := range toDelete {
			if err := client.DeleteRecord(ctx, zoneID, r.ID); err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "  ✗ delete %s %s %s: %v\n", r.Type, r.Name, r.Content, err)
				failed++
				continue
			}
			fmt.Fprintf(cmd.OutOrStdout(), "  - %s %s %s\n", r.Type, r.Name, r.Content)
			deleted++
		}

		reportSkippedNS()

		fmt.Fprintf(cmd.OutOrStdout(), "%d records added", added)
		if unchanged > 0 {
			fmt.Fprintf(cmd.OutOrStdout(), ", %d unchanged", unchanged)
		}
		if deleted > 0 {
			fmt.Fprintf(cmd.OutOrStdout(), ", %d removed", deleted)
		}
		fmt.Fprintln(cmd.OutOrStdout())

		if failed > 0 {
			return fmt.Errorf("%d record operation(s) failed", failed)
		}
		return nil
	},
}

func findRecordsToDelete(existing []api.Record, imports []api.CreateRecordRequest) []api.Record {
	importSet := make(map[string]bool)
	for _, req := range imports {
		importSet[importRecordKey(req)] = true
	}

	var toDelete []api.Record
	for _, r := range existing {
		// Never touch SOA or NS at zone apex
		if r.Name == "@" && (r.Type == "SOA" || r.Type == "NS") {
			continue
		}
		if !importSet[storedRecordKey(r)] {
			toDelete = append(toDelete, r)
		}
	}
	return toDelete
}

// storedRecordKey and importRecordKey render a record the API already holds and
// a record parsed from a zone file into the same comparable key.
//
// The API stores composed rdata ("10 mail.example.com.", `"v=spf1 -all"`,
// `0 issue "letsencrypt.org"`) while the zone file carries the parts separately,
// so comparing raw content marked every MX, TXT, SRV and CAA record as missing:
// --replace then deleted records the file actually defined and re-added nothing.
func storedRecordKey(r api.Record) string {
	return recordKey(r.Type, r.Name, r.Content)
}

func importRecordKey(req api.CreateRecordRequest) string {
	content := req.Content

	switch strings.ToUpper(req.Type) {
	case "MX":
		if req.Priority != nil {
			content = fmt.Sprintf("%d %s", *req.Priority, content)
		}
	case "SRV":
		if req.Priority != nil && req.Weight != nil && req.Port != nil {
			content = fmt.Sprintf("%d %d %d %s", *req.Priority, *req.Weight, *req.Port, content)
		}
	case "CAA":
		flags := 0
		if req.Flags != nil {
			flags = *req.Flags
		}
		content = fmt.Sprintf("%d %s %s", flags, req.Tag, content)
	}

	return recordKey(req.Type, req.Name, content)
}

func recordKey(recordType, name, content string) string {
	fields := strings.Fields(strings.ReplaceAll(content, `"`, ""))
	for i, f := range fields {
		fields[i] = strings.TrimSuffix(f, ".")
	}

	recordName := name
	if recordName == "" {
		recordName = "@"
	}

	return strings.ToUpper(recordType) + "|" + strings.ToLower(recordName) + "|" +
		strings.ToLower(strings.Join(fields, " "))
}

var zoneCheckCmd = &cobra.Command{
	Use:   "check <domain>",
	Short: "Check DNS propagation",
	Long:  "Check if DNS records are visible from public resolvers.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		domain := args[0]
		client, err := newAPIClient(cmd)
		if err != nil {
			return err
		}

		ctx := context.Background()
		zoneID, err := resolveZone(ctx, client, domain)
		if err != nil {
			return err
		}

		zone, err := client.GetZone(ctx, zoneID)
		if err != nil {
			return err
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Checking %s against public resolvers...\n\n", domain)

		resolvers := []string{"8.8.8.8", "1.1.1.1", "9.9.9.9", "208.67.222.222"}
		total := 0
		passed := 0

		// Check NS records
		fmt.Fprintln(cmd.OutOrStdout(), "NS records:")
		for _, resolver := range resolvers {
			total++
			results, err := dns.LookupNS(domain, resolver)
			if err != nil {
				fmt.Fprintf(cmd.OutOrStdout(), "  %-16s ✗  %v\n", resolver+":", err)
				continue
			}
			passed++
			fmt.Fprintf(cmd.OutOrStdout(), "  %-16s OK  %s\n", resolver+":", joinStrings(results))
		}

		// Check A records
		records, _ := client.ListRecords(ctx, zoneID, api.ListRecordsOptions{Type: "A", Name: "@"})
		if len(records) > 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "\nA record (@):\n")
			for _, resolver := range resolvers {
				total++
				results, err := dns.LookupA(domain, resolver)
				if err != nil {
					fmt.Fprintf(cmd.OutOrStdout(), "  %-16s ✗  %v\n", resolver+":", err)
					continue
				}
				passed++
				fmt.Fprintf(cmd.OutOrStdout(), "  %-16s OK  %s\n", resolver+":", joinStrings(results))
			}
		}

		// Check MX records
		mxRecords, _ := client.ListRecords(ctx, zoneID, api.ListRecordsOptions{Type: "MX", Name: "@"})
		if len(mxRecords) > 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "\nMX record (@):\n")
			for _, resolver := range resolvers {
				total++
				results, err := dns.LookupMX(domain, resolver)
				if err != nil {
					fmt.Fprintf(cmd.OutOrStdout(), "  %-16s ✗  %v\n", resolver+":", err)
					continue
				}
				passed++
				fmt.Fprintf(cmd.OutOrStdout(), "  %-16s OK  %s\n", resolver+":", joinStrings(results))
			}
		}

		fmt.Fprintf(cmd.OutOrStdout(), "\n")
		_ = zone // zone used for context
		if passed == total {
			fmt.Fprintf(cmd.OutOrStdout(), "All checks passed (%d/%d)\n", passed, total)
			return nil
		}

		// A failing propagation check must fail the command: this is the step a
		// migration pipeline gates on, and exiting 0 made every gate pass.
		fmt.Fprintf(cmd.OutOrStdout(), "Checks: %d/%d passed\n", passed, total)
		return &silentError{fmt.Errorf("%d of %d propagation check(s) failed", total-passed, total)}
	},
}

// silentError carries a non-zero exit status for a command that already printed
// its own report, so cobra does not repeat the message as "Error: ...".
type silentError struct{ error }

func (e *silentError) Unwrap() error { return e.error }

func joinStrings(s []string) string {
	result := ""
	for i, v := range s {
		if i > 0 {
			result += ", "
		}
		result += v
	}
	return result
}

func init() {
	zoneListCmd.Flags().String("search", "", "Filter by domain name")
	zoneListCmd.Flags().Int("page", 0, "Page number")
	zoneListCmd.Flags().Int("per-page", 0, "Items per page (max 100)")
	zoneListCmd.Flags().Bool("all", false, "Fetch all pages")

	zoneAddCmd.Flags().String("type", "master", "Zone type: master or slave")
	zoneAddCmd.Flags().String("ns-group", "", "NS group slug (see the groups available on your account)")
	zoneAddCmd.Flags().String("master-ip", "", "Master IP (required for slave zones)")

	zoneDeleteCmd.Flags().Bool("force", false, "Skip confirmation prompt")

	zoneExportCmd.Flags().String("format", "bind", "Export format: bind or json")

	zoneEnsureCmd.Flags().String("type", "master", "Zone type: master or slave")
	zoneEnsureCmd.Flags().String("ns-group", "", "NS group slug (see the groups available on your account)")

	zoneImportCmd.Flags().Bool("replace", false, "Delete records not in the file")

	zoneCmd.AddCommand(zoneListCmd)
	zoneCmd.AddCommand(zoneAddCmd)
	zoneCmd.AddCommand(zoneDeleteCmd)
	zoneCmd.AddCommand(zoneInfoCmd)
	zoneCmd.AddCommand(zoneMoveCmd)
	zoneCmd.AddCommand(zoneExportCmd)
	zoneCmd.AddCommand(zoneEnsureCmd)
	zoneCmd.AddCommand(zoneImportCmd)
	zoneCmd.AddCommand(zoneCheckCmd)
	rootCmd.AddCommand(zoneCmd)
}
