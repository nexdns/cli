package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var dnssecCmd = &cobra.Command{
	Use:   "dnssec",
	Short: "Manage DNSSEC",
}

var dnssecStatusCmd = &cobra.Command{
	Use:   "status <domain>",
	Short: "Show DNSSEC status",
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

		status, err := client.GetDNSSEC(ctx, zoneID)
		if err != nil {
			return err
		}

		formatter.FormatDNSSEC(status)
		return nil
	},
}

var dnssecEnableCmd = &cobra.Command{
	Use:   "enable <domain>",
	Short: "Enable DNSSEC for a zone",
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
			fmt.Fprintf(cmd.OutOrStdout(), "[dry-run] Would enable DNSSEC for %s\n", domain)
			return nil
		}

		status, err := client.EnableDNSSEC(ctx, zoneID)
		if err != nil {
			return err
		}

		fmt.Fprintf(cmd.OutOrStdout(), "DNSSEC enabled for %s\n", domain)
		if len(status.DSRecords) > 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "\nDS Records (add these at your registrar):")
			for _, ds := range status.DSRecords {
				fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", ds)
			}
		}
		return nil
	},
}

var dnssecDisableCmd = &cobra.Command{
	Use:   "disable <domain>",
	Short: "Disable DNSSEC for a zone",
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
			fmt.Fprintf(cmd.OutOrStdout(), "[dry-run] Would disable DNSSEC for %s\n", domain)
			return nil
		}

		force, _ := cmd.Flags().GetBool("force")
		if !force {
			if !confirmPrompt(fmt.Sprintf("Disable DNSSEC for %s? This will remove all keys.", domain)) {
				fmt.Fprintln(cmd.OutOrStdout(), "Cancelled")
				return nil
			}
		}

		if err := client.DisableDNSSEC(ctx, zoneID); err != nil {
			return err
		}

		fmt.Fprintf(cmd.OutOrStdout(), "DNSSEC disabled for %s\n", domain)
		return nil
	},
}

var dnssecDSRecordsCmd = &cobra.Command{
	Use:   "ds-records <domain>",
	Short: "Show DS records for registrar",
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

		status, err := client.GetDNSSEC(ctx, zoneID)
		if err != nil {
			return err
		}

		if !status.Enabled {
			return fmt.Errorf("DNSSEC is not enabled for %s", domain)
		}

		// Honour --output like every other command: registrar automation reads
		// these values, and plain text only was not machine-consumable.
		format, _ := cmd.Flags().GetString("output")
		if format != "" && format != "table" {
			return writeFormatted(cmd, format, map[string]interface{}{
				"domain":     domain,
				"ds_records": status.DSRecords,
			})
		}

		for _, ds := range status.DSRecords {
			fmt.Fprintln(cmd.OutOrStdout(), ds)
		}
		return nil
	},
}

func init() {
	dnssecDisableCmd.Flags().Bool("force", false, "Skip confirmation prompt")

	dnssecCmd.AddCommand(dnssecStatusCmd)
	dnssecCmd.AddCommand(dnssecEnableCmd)
	dnssecCmd.AddCommand(dnssecDisableCmd)
	dnssecCmd.AddCommand(dnssecDSRecordsCmd)
	rootCmd.AddCommand(dnssecCmd)
}
