package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/nexdns/cli/internal/api"
)

var acmeCmd = &cobra.Command{
	Use:   "acme",
	Short: "ACME DNS-01 challenge helpers",
}

var acmeHookCmd = &cobra.Command{
	Use:   "hook",
	Short: "Create or delete ACME challenge TXT records",
	Long: `Manage ACME DNS-01 challenge TXT records for wildcard certificate issuance.

Use with certbot manual hooks:
  certbot certonly --manual \
    --preferred-challenges dns \
    --manual-auth-hook "nexdns acme hook --action create" \
    --manual-cleanup-hook "nexdns acme hook --action delete" \
    -d '*.example.com'

Environment variables (set by certbot):
  CERTBOT_DOMAIN      – the domain being validated
  CERTBOT_VALIDATION  – the challenge token value`,
	RunE: func(cmd *cobra.Command, args []string) error {
		action, _ := cmd.Flags().GetString("action")
		domain, _ := cmd.Flags().GetString("domain")
		token, _ := cmd.Flags().GetString("token")

		// Fall back to certbot environment variables
		if domain == "" {
			domain = getEnv("CERTBOT_DOMAIN")
		}
		if token == "" {
			token = getEnv("CERTBOT_VALIDATION")
		}

		if domain == "" {
			return fmt.Errorf("domain is required (--domain or CERTBOT_DOMAIN env)")
		}
		if token == "" {
			return fmt.Errorf("token is required (--token or CERTBOT_VALIDATION env)")
		}

		if action != "create" && action != "delete" {
			return fmt.Errorf("--action must be 'create' or 'delete'")
		}

		client, err := newAPIClient(cmd)
		if err != nil {
			return err
		}

		ctx := context.Background()
		challengeName := "_acme-challenge"

		// Find zone by iterating subdomains
		zoneID, subDomain, err := findZoneForDomain(ctx, client, domain)
		if err != nil {
			return err
		}

		if subDomain != "" {
			challengeName = "_acme-challenge." + subDomain
		}

		switch action {
		case "create":
			_, err := client.CreateRecord(ctx, zoneID, api.CreateRecordRequest{
				Name:    challengeName,
				Type:    "TXT",
				Content: token,
				TTL:     intPtr(120),
			})
			if err != nil {
				return fmt.Errorf("failed to create TXT record: %w", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Created TXT record %s = %s\n", challengeName, token)

		case "delete":
			records, err := client.ListRecords(ctx, zoneID, api.ListRecordsOptions{
				Type: "TXT",
				Name: challengeName,
			})
			if err != nil {
				return fmt.Errorf("failed to list TXT records: %w", err)
			}

			for _, r := range records {
				value := extractTXTValue(r)
				if value == token {
					if err := client.DeleteRecord(ctx, zoneID, r.ID); err != nil {
						return fmt.Errorf("failed to delete TXT record: %w", err)
					}
					fmt.Fprintf(cmd.OutOrStdout(), "Deleted TXT record %s (ID: %s)\n", challengeName, r.ID)
					return nil
				}
			}
			fmt.Fprintln(cmd.OutOrStdout(), "TXT record not found (may have been cleaned up already)")
		}

		return nil
	},
}

// findZoneForDomain resolves the zone ID and subdomain part for a domain.
// Tries each parent domain: sub.example.com → example.com
func findZoneForDomain(ctx context.Context, client *api.Client, domain string) (zoneID, subDomain string, err error) {
	parts := strings.Split(domain, ".")
	for i := range parts {
		candidate := strings.Join(parts[i:], ".")
		if !strings.Contains(candidate, ".") {
			break
		}

		zone, lookupErr := client.GetZoneByName(ctx, candidate)
		if lookupErr == nil {
			sub := ""
			if i > 0 {
				sub = strings.Join(parts[:i], ".")
			}
			return zone.ID, sub, nil
		}
	}
	return "", "", fmt.Errorf("zone not found for domain %s. Make sure the domain is configured in your NexDNS account", domain)
}

// extractTXTValue gets the clean TXT value from a record.
func extractTXTValue(r api.Record) string {
	if r.Fields != nil {
		if v, ok := r.Fields["value"]; ok {
			if s, ok := v.(string); ok {
				return s
			}
		}
	}
	content := r.Content
	if len(content) >= 2 && content[0] == '"' && content[len(content)-1] == '"' {
		return content[1 : len(content)-1]
	}
	return content
}

func getEnv(key string) string {
	return os.Getenv(key)
}

func init() {
	acmeHookCmd.Flags().String("action", "", "Action: create or delete (required)")
	acmeHookCmd.Flags().String("domain", "", "Domain name (or CERTBOT_DOMAIN env)")
	acmeHookCmd.Flags().String("token", "", "Challenge token value (or CERTBOT_VALIDATION env)")

	acmeCmd.AddCommand(acmeHookCmd)
	rootCmd.AddCommand(acmeCmd)
}
