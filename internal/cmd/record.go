package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/nexdns/cli/internal/api"
)

var recordCmd = &cobra.Command{
	Use:   "record",
	Short: "Manage DNS records",
}

var recordListCmd = &cobra.Command{
	Use:   "list <domain>",
	Short: "List records for a zone",
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

		recordType, _ := cmd.Flags().GetString("type")
		name, _ := cmd.Flags().GetString("name")
		search, _ := cmd.Flags().GetString("search")

		records, err := client.ListRecords(ctx, zoneID, api.ListRecordsOptions{
			Type:   recordType,
			Name:   name,
			Search: search,
		})
		if err != nil {
			return err
		}

		formatter.FormatRecords(records)
		return nil
	},
}

var recordAddCmd = &cobra.Command{
	Use:   "add <domain> <type> <name> <content>",
	Short: "Create a new DNS record",
	Long: `Create a new DNS record.

Examples:
  nexdns record add example.com A @ 203.0.113.10
  nexdns record add example.com A www 203.0.113.10 --ttl 300
  nexdns record add example.com MX @ mail.example.com --priority 10
  nexdns record add example.com SRV _sip._tcp sip.example.com --priority 10 --weight 60 --port 5060
  nexdns record add example.com CAA @ letsencrypt.org --tag issue --flags 0`,
	Args: cobra.ExactArgs(4),
	RunE: func(cmd *cobra.Command, args []string) error {
		domain := args[0]
		recordType := strings.ToUpper(args[1])
		name := args[2]
		content := args[3]

		if isDryRun(cmd) {
			fmt.Fprintf(cmd.OutOrStdout(), "[dry-run] Would create %s record %s %s in %s\n", recordType, name, content, domain)
			return nil
		}

		client, err := newAPIClient(cmd)
		if err != nil {
			return err
		}

		ctx := context.Background()
		zoneID, err := resolveZone(ctx, client, domain)
		if err != nil {
			return err
		}

		req := buildCreateRecordRequest(cmd, recordType, name, content)
		record, err := client.CreateRecord(ctx, zoneID, req)
		if err != nil {
			return err
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Record %s %s %s created (ID: %s)\n", record.Type, record.Name, record.Content, record.ID)
		return nil
	},
}

var recordUpdateCmd = &cobra.Command{
	Use:   "update <domain> <record-id>",
	Short: "Update an existing DNS record",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		domain := args[0]
		recordID := args[1]

		client, err := newAPIClient(cmd)
		if err != nil {
			return err
		}

		ctx := context.Background()
		zoneID, err := resolveZone(ctx, client, domain)
		if err != nil {
			return err
		}

		req := api.UpdateRecordRequest{}
		if cmd.Flags().Changed("content") {
			v, _ := cmd.Flags().GetString("content")
			req.Content = stringPtr(v)
		}
		if cmd.Flags().Changed("record-name") {
			v, _ := cmd.Flags().GetString("record-name")
			req.Name = stringPtr(v)
		}
		if cmd.Flags().Changed("ttl") {
			v, _ := cmd.Flags().GetInt("ttl")
			req.TTL = intPtr(v)
		}
		if cmd.Flags().Changed("priority") {
			v, _ := cmd.Flags().GetInt("priority")
			req.Priority = intPtr(v)
		}

		if isDryRun(cmd) {
			fmt.Fprintf(cmd.OutOrStdout(), "[dry-run] Would update record %s in %s\n", recordID, domain)
			return nil
		}

		record, err := client.UpdateRecord(ctx, zoneID, recordID, req)
		if err != nil {
			return err
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Record %s %s %s updated (ID: %s)\n", record.Type, record.Name, record.Content, record.ID)
		return nil
	},
}

var recordDeleteCmd = &cobra.Command{
	Use:   "delete <domain> <record-id>",
	Short: "Delete a DNS record",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		domain := args[0]
		recordID := args[1]

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
			fmt.Fprintf(cmd.OutOrStdout(), "[dry-run] Would delete record %s from %s\n", recordID, domain)
			return nil
		}

		force, _ := cmd.Flags().GetBool("force")
		if !force {
			record, err := client.GetRecord(ctx, zoneID, recordID)
			if err != nil {
				return err
			}
			if !confirmPrompt(fmt.Sprintf("Delete %s record %q (%s)?", record.Type, record.Name, record.Content)) {
				fmt.Fprintln(cmd.OutOrStdout(), "Cancelled")
				return nil
			}
		}

		if err := client.DeleteRecord(ctx, zoneID, recordID); err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), "Record deleted")
		return nil
	},
}

var recordEnsureCmd = &cobra.Command{
	Use:   "ensure <domain> <type> <name> <content>",
	Short: "Create record if it doesn't exist (idempotent)",
	Long: `Idempotent record creation. Checks if an exact match (type + name + content) exists.
If found, skips. If not found, creates.

Safe for round-robin setups – will not touch existing records with different content.`,
	Args: cobra.ExactArgs(4),
	RunE: func(cmd *cobra.Command, args []string) error {
		domain := args[0]
		recordType := strings.ToUpper(args[1])
		name := args[2]
		content := args[3]

		client, err := newAPIClient(cmd)
		if err != nil {
			return err
		}

		ctx := context.Background()
		zoneID, err := resolveZone(ctx, client, domain)
		if err != nil {
			return err
		}

		records, err := client.ListRecords(ctx, zoneID, api.ListRecordsOptions{})
		if err != nil {
			return err
		}

		for _, r := range records {
			if r.Type == recordType && r.Name == name && strings.TrimSuffix(r.Content, ".") == strings.TrimSuffix(content, ".") {
				fmt.Fprintf(cmd.OutOrStdout(), "Record %s %s %s already exists (skipped)\n", recordType, name, content)
				return nil
			}
		}

		if isDryRun(cmd) {
			fmt.Fprintf(cmd.OutOrStdout(), "[dry-run] Would create %s record %s %s in %s\n", recordType, name, content, domain)
			return nil
		}

		req := buildCreateRecordRequest(cmd, recordType, name, content)
		record, err := client.CreateRecord(ctx, zoneID, req)
		if err != nil {
			return err
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Record %s %s %s created (ID: %s)\n", record.Type, record.Name, record.Content, record.ID)
		return nil
	},
}

func buildCreateRecordRequest(cmd *cobra.Command, recordType, name, content string) api.CreateRecordRequest {
	req := api.CreateRecordRequest{
		Name:    name,
		Type:    recordType,
		Content: content,
	}

	// Only an explicit --ttl is sent, and then verbatim. A TTL belongs to the
	// rrset, so submitting the flag's default alongside a new value silently
	// retimed every value already at that name; omitting it lets the API keep the
	// rrset's own TTL. Dropping an out-of-range value instead of sending it was
	// just as bad in the other direction - `--ttl -5` reported success and
	// created the record with the default, so the user was told the opposite of
	// what happened. The API owns the range check.
	if cmd.Flags().Changed("ttl") {
		ttl, _ := cmd.Flags().GetInt("ttl")
		req.TTL = &ttl
	}
	if cmd.Flags().Changed("priority") {
		v, _ := cmd.Flags().GetInt("priority")
		req.Priority = intPtr(v)
	}
	if cmd.Flags().Changed("weight") {
		v, _ := cmd.Flags().GetInt("weight")
		req.Weight = intPtr(v)
	}
	if cmd.Flags().Changed("port") {
		v, _ := cmd.Flags().GetInt("port")
		req.Port = intPtr(v)
	}
	if cmd.Flags().Changed("flags") {
		v, _ := cmd.Flags().GetInt("flags")
		req.Flags = intPtr(v)
	}
	if tag, _ := cmd.Flags().GetString("tag"); tag != "" {
		req.Tag = tag
	}
	if cmd.Flags().Changed("keytag") {
		v, _ := cmd.Flags().GetInt("keytag")
		req.KeyTag = intPtr(v)
	}
	if cmd.Flags().Changed("algorithm") {
		v, _ := cmd.Flags().GetInt("algorithm")
		req.Algorithm = intPtr(v)
	}
	if cmd.Flags().Changed("digest-type") {
		v, _ := cmd.Flags().GetInt("digest-type")
		req.DigestType = intPtr(v)
	}
	if cmd.Flags().Changed("usage") {
		v, _ := cmd.Flags().GetInt("usage")
		req.Usage = intPtr(v)
	}
	if cmd.Flags().Changed("selector") {
		v, _ := cmd.Flags().GetInt("selector")
		req.Selector = intPtr(v)
	}
	if cmd.Flags().Changed("matching-type") {
		v, _ := cmd.Flags().GetInt("matching-type")
		req.MatchingType = intPtr(v)
	}

	return req
}

func addRecordFlags(cmd *cobra.Command) {
	cmd.Flags().Int("ttl", 3600, "Record TTL in seconds")
	cmd.Flags().Int("priority", 10, "Record priority (MX, SRV)")
	cmd.Flags().Int("weight", 0, "Record weight (SRV)")
	cmd.Flags().Int("port", 0, "Record port (SRV)")
	cmd.Flags().Int("flags", 0, "CAA flags")
	cmd.Flags().String("tag", "", "CAA tag (issue, issuewild, iodef)")
	cmd.Flags().Int("keytag", 0, "DS key tag")
	cmd.Flags().Int("algorithm", 0, "DS algorithm")
	cmd.Flags().Int("digest-type", 0, "DS digest type")
	cmd.Flags().Int("usage", 0, "TLSA usage")
	cmd.Flags().Int("selector", 0, "TLSA selector")
	cmd.Flags().Int("matching-type", 0, "TLSA matching type")
}

func init() {
	recordListCmd.Flags().String("type", "", "Filter by record type")
	recordListCmd.Flags().String("name", "", "Filter by record name")
	recordListCmd.Flags().String("search", "", "Search in name and content")

	addRecordFlags(recordAddCmd)
	addRecordFlags(recordEnsureCmd)

	recordUpdateCmd.Flags().String("content", "", "New content value")
	recordUpdateCmd.Flags().String("record-name", "", "New record name")
	recordUpdateCmd.Flags().Int("ttl", 0, "New TTL")
	recordUpdateCmd.Flags().Int("priority", 0, "New priority (MX, SRV)")

	recordDeleteCmd.Flags().Bool("force", false, "Skip confirmation prompt")

	recordCmd.AddCommand(recordListCmd)
	recordCmd.AddCommand(recordAddCmd)
	recordCmd.AddCommand(recordUpdateCmd)
	recordCmd.AddCommand(recordDeleteCmd)
	recordCmd.AddCommand(recordEnsureCmd)
	rootCmd.AddCommand(recordCmd)
}
