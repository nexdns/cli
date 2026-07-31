package cmd

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/nexdns/cli/internal/api"
	"github.com/nexdns/cli/internal/dnsascode"
)

var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply DNS configuration from nexdns.yaml",
	Long: `Compare nexdns.yaml with remote state and show diff.
By default, this is a dry-run. Use --confirm to actually apply changes.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runApply(cmd, false)
	},
}

var diffCmd = &cobra.Command{
	Use:   "diff",
	Short: "Show diff between nexdns.yaml and remote state",
	Long:  "Alias for 'nexdns apply' (dry-run mode).",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runApply(cmd, false)
	},
}

func runApply(cmd *cobra.Command, _ bool) error {
	filePath, _ := cmd.Flags().GetString("file")
	if filePath == "" {
		filePath = "nexdns.yaml"
	}

	confirm, _ := cmd.Flags().GetBool("confirm")
	deleteRemote, _ := cmd.Flags().GetBool("delete")
	zoneFilter, _ := cmd.Flags().GetString("zone")

	fmt.Fprintf(cmd.OutOrStdout(), "Reading %s...\n\n", filePath)

	cfg, err := dnsascode.ParseConfigFile(filePath)
	if err != nil {
		return err
	}

	client, err := newAPIClient(cmd)
	if err != nil {
		return err
	}

	ctx := context.Background()
	totalOps := 0
	failedOps := 0
	zonesWithChanges := 0

	// Sort zone names for deterministic output
	zoneNames := make([]string, 0, len(cfg.Zones))
	for name := range cfg.Zones {
		zoneNames = append(zoneNames, name)
	}
	sort.Strings(zoneNames)

	// A --zone that names nothing in the file used to filter every zone out and
	// report "No changes needed" with exit 0, so a typo in a pipeline looked like
	// a converged apply.
	if zoneFilter != "" {
		if _, ok := cfg.Zones[zoneFilter]; !ok {
			return fmt.Errorf("zone %q is not defined in %s (defined: %s)", zoneFilter, filePath, strings.Join(zoneNames, ", "))
		}
	}

	for _, zoneName := range zoneNames {
		zoneCfg := cfg.Zones[zoneName]

		if zoneFilter != "" && zoneFilter != zoneName {
			continue
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Zone: %s\n", zoneName)

		// Resolve or create zone
		zone, lookupErr := client.GetZoneByName(ctx, zoneName)
		var zoneID string
		if lookupErr != nil {
			if !api.IsNotFound(lookupErr) {
				return fmt.Errorf("looking up zone %s: %w", zoneName, lookupErr)
			}
			if confirm {
				req := api.CreateZoneRequest{Name: zoneName}
				newZone, err := client.CreateZone(ctx, req)
				if err != nil {
					return fmt.Errorf("creating zone %s: %w", zoneName, err)
				}
				zoneID = newZone.ID
				fmt.Fprintf(cmd.OutOrStdout(), "  (zone created)\n")
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "  (zone will be created)\n")
			}
		} else {
			zoneID = zone.ID
		}

		// Enable DNSSEC if configured
		if zoneCfg.DNSSEC && zoneID != "" {
			status, err := client.GetDNSSEC(ctx, zoneID)
			if err == nil && !status.Enabled {
				if confirm {
					_, err := client.EnableDNSSEC(ctx, zoneID)
					if err != nil {
						failedOps++
						fmt.Fprintf(cmd.ErrOrStderr(), "  ✗ enable DNSSEC: %v\n", err)
					} else {
						fmt.Fprintf(cmd.OutOrStdout(), "  DNSSEC enabled\n")
					}
				} else {
					fmt.Fprintf(cmd.OutOrStdout(), "  DNSSEC will be enabled\n")
				}
			}
		}

		// Get remote records for diff
		var remoteRecords []api.Record
		if zoneID != "" {
			remoteRecords, err = client.ListRecords(ctx, zoneID, api.ListRecordsOptions{})
			if err != nil {
				return fmt.Errorf("fetching records for %s: %w", zoneName, err)
			}
		}

		ops := dnsascode.ComputeDiff(zoneName, zoneCfg.Records, remoteRecords, deleteRemote)

		if len(ops) == 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "  (no changes)\n")
		} else {
			zonesWithChanges++
			for _, op := range ops {
				totalOps++
				switch op.Action {
				case "add":
					symbol := "+"
					detail := fmt.Sprintf("%s %s %s (TTL %d)", op.Type, op.Name, op.Content, op.TTL)
					if confirm {
						req := buildApplyCreateRequest(op)
						_, err := client.CreateRecord(ctx, zoneID, req)
						if err != nil {
							failedOps++
							fmt.Fprintf(cmd.OutOrStdout(), "  %s %s  ✗ %v\n", symbol, detail, err)
						} else {
							fmt.Fprintf(cmd.OutOrStdout(), "  %s %s  ✓\n", symbol, detail)
						}
					} else {
						fmt.Fprintf(cmd.OutOrStdout(), "  %s %s\n", symbol, detail)
					}

				case "update":
					symbol := "~"
					detail := fmt.Sprintf("%s %s – content: %s → %s", op.Type, op.Name, op.OldContent, op.Content)
					if op.OldTTL != op.TTL {
						detail += fmt.Sprintf(", TTL: %d → %d", op.OldTTL, op.TTL)
					}
					if confirm {
						req := api.UpdateRecordRequest{
							Content: stringPtr(op.Content),
							TTL:     intPtr(op.TTL),
						}
						if op.Priority != nil {
							req.Priority = op.Priority
						}
						_, err := client.UpdateRecord(ctx, zoneID, op.RecordID, req)
						if err != nil {
							failedOps++
							fmt.Fprintf(cmd.OutOrStdout(), "  %s %s  ✗ %v\n", symbol, detail, err)
						} else {
							fmt.Fprintf(cmd.OutOrStdout(), "  %s %s  ✓\n", symbol, detail)
						}
					} else {
						fmt.Fprintf(cmd.OutOrStdout(), "  %s %s\n", symbol, detail)
					}

				case "delete":
					symbol := "-"
					detail := fmt.Sprintf("%s %s %s", op.Type, op.Name, op.Content)
					if confirm {
						err := client.DeleteRecord(ctx, zoneID, op.RecordID)
						if err != nil {
							failedOps++
							fmt.Fprintf(cmd.OutOrStdout(), "  %s %s  ✗ %v\n", symbol, detail, err)
						} else {
							fmt.Fprintf(cmd.OutOrStdout(), "  %s %s  ✓\n", symbol, detail)
						}
					} else {
						fmt.Fprintf(cmd.OutOrStdout(), "  %s %s (will be removed)\n", symbol, detail)
					}
				}
			}
		}
		fmt.Fprintln(cmd.OutOrStdout())
	}

	if confirm {
		applied := totalOps - failedOps
		if failedOps > 0 {
			// Reporting success while operations failed let a broken apply pass a
			// CI gate unnoticed.
			fmt.Fprintf(cmd.OutOrStdout(), "%d of %d operation(s) applied, %d failed\n", applied, totalOps, failedOps)
			return fmt.Errorf("%d operation(s) failed", failedOps)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "All changes applied (%d operations)\n", totalOps)
	} else {
		if totalOps > 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "%d zone(s) with changes, %d total operations.\n", zonesWithChanges, totalOps)
			fmt.Fprintln(cmd.OutOrStdout(), "Run with --confirm to apply.")
		} else {
			fmt.Fprintln(cmd.OutOrStdout(), "No changes needed.")
		}
	}

	return nil
}

// buildApplyCreateRequest turns a planned operation into an API create request.
//
// The API takes the type-specific parts as separate fields, but nexdns.yaml may
// carry them either way: a hand-written file sets `priority` / `tag` / `flags`
// next to a bare content value, while a file produced by `nexdns pull` holds the
// composed rdata ("10 mail.example.com.", `0 issue "letsencrypt.org"`). Posting
// the composed form as `content` made the API reject it, so a CAA row in a
// hand-written config could never be applied. Both shapes are accepted here.
func buildApplyCreateRequest(op dnsascode.Operation) api.CreateRecordRequest {
	req := api.CreateRecordRequest{
		Name:     op.Name,
		Type:     op.Type,
		Content:  op.Content,
		TTL:      ttlPtr(op.TTL),
		Priority: op.Priority,
		Weight:   op.Weight,
		Port:     op.Port,
		Flags:    op.Flags,
		Tag:      op.Tag,
	}

	fields := strings.Fields(op.Content)

	switch strings.ToUpper(op.Type) {
	case "MX":
		if req.Priority == nil && len(fields) >= 2 {
			if priority, err := strconv.Atoi(fields[0]); err == nil {
				req.Priority = &priority
				req.Content = fields[1]
			}
		}
	case "SRV":
		if req.Port == nil && len(fields) >= 4 {
			priority, err1 := strconv.Atoi(fields[0])
			weight, err2 := strconv.Atoi(fields[1])
			port, err3 := strconv.Atoi(fields[2])
			if err1 == nil && err2 == nil && err3 == nil {
				req.Priority, req.Weight, req.Port = &priority, &weight, &port
				req.Content = fields[3]
			}
		}
	case "CAA":
		if req.Tag == "" && len(fields) >= 3 {
			if flags, err := strconv.Atoi(fields[0]); err == nil {
				req.Flags = &flags
				req.Tag = fields[1]
				req.Content = strings.Trim(strings.Join(fields[2:], " "), `"`)
			}
		}
	}

	return req
}

var pullCmd = &cobra.Command{
	Use:   "pull <domain>",
	Short: "Download remote zone state as YAML",
	Long: `Download remote zone records and output as nexdns.yaml format.

Examples:
  nexdns pull example.com
  nexdns pull example.com > nexdns.yaml
  nexdns pull example.com --file nexdns.yaml`,
	Args: cobra.ExactArgs(1),
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

		records, err := client.ListRecords(ctx, zoneID, api.ListRecordsOptions{})
		if err != nil {
			return err
		}

		// Convert to config format
		var cfgRecords []dnsascode.RecordConfig
		for _, r := range records {
			// Skip SOA and NS at apex
			if r.Name == "@" && (r.Type == "SOA" || r.Type == "NS") {
				continue
			}
			cr := dnsascode.RecordConfig{
				Type:    r.Type,
				Name:    r.Name,
				Content: r.Content,
				TTL:     r.TTL,
			}
			if r.Priority != nil {
				cr.Priority = r.Priority
			}
			cfgRecords = append(cfgRecords, cr)
		}

		cfg := dnsascode.ConfigFile{
			Zones: map[string]dnsascode.ZoneConfig{
				domain: {Records: cfgRecords},
			},
		}

		data, err := yaml.Marshal(cfg)
		if err != nil {
			return fmt.Errorf("marshaling YAML: %w", err)
		}

		filePath, _ := cmd.Flags().GetString("file")
		if filePath != "" {
			if err := writeFile(filePath, data, cmd); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Written to %s\n", filePath)
			return nil
		}

		fmt.Fprint(cmd.OutOrStdout(), string(data))
		return nil
	},
}

func writeFile(path string, data []byte, cmd *cobra.Command) error {
	appendMode, _ := cmd.Flags().GetBool("append")
	if appendMode {
		f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("opening file for append: %w", err)
		}
		defer f.Close()
		_, err = f.Write(data)
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func init() {
	applyCmd.Flags().Bool("confirm", false, "Actually apply changes (default: dry-run)")
	applyCmd.Flags().String("file", "nexdns.yaml", "Config file path")
	applyCmd.Flags().String("zone", "", "Apply only to specific zone")
	applyCmd.Flags().Bool("delete", false, "Delete remote records not in config")

	diffCmd.Flags().String("file", "nexdns.yaml", "Config file path")
	diffCmd.Flags().String("zone", "", "Diff only for specific zone")
	diffCmd.Flags().Bool("delete", false, "Show records that would be deleted")

	pullCmd.Flags().String("file", "", "Write directly to file instead of stdout")
	pullCmd.Flags().Bool("append", false, "Append to existing file")

	rootCmd.AddCommand(applyCmd)
	rootCmd.AddCommand(diffCmd)
	rootCmd.AddCommand(pullCmd)
}
