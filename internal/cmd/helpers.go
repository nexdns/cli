package cmd

import (
	"bufio"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"
	"gopkg.in/yaml.v3"

	"github.com/nexdns/cli/internal/api"
	"github.com/nexdns/cli/internal/config"
	"github.com/nexdns/cli/internal/output"
)

func newAPIClient(cmd *cobra.Command) (*api.Client, error) {
	cfg, err := config.Resolve(cmd)
	if err != nil {
		return nil, err
	}
	if cfg.Token == "" {
		return nil, fmt.Errorf("not authenticated. Run `nexdns auth token <TOKEN>` to authenticate.\nCreate an API key at https://nexdns.tech/settings/api-keys")
	}

	verbose, _ := cmd.Flags().GetBool("verbose")
	timeout := time.Duration(cfg.Timeout) * time.Second

	return api.NewClient(cfg.APIUrl, cfg.Token, timeout, verbose), nil
}

// writeFormatted prints a plain value in the requested machine-readable format.
// The Formatter interface is typed per resource; this covers the few commands
// whose payload is not one of those resources (DS records, for instance).
func writeFormatted(cmd *cobra.Command, format string, value map[string]interface{}) error {
	switch format {
	case "json":
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(value)
	case "yaml":
		enc := yaml.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent(2)
		defer enc.Close()
		return enc.Encode(value)
	case "csv":
		w := csv.NewWriter(cmd.OutOrStdout())
		defer w.Flush()
		keys := make([]string, 0, len(value))
		for k := range value {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			switch v := value[k].(type) {
			case []string:
				for _, item := range v {
					if err := w.Write([]string{k, item}); err != nil {
						return err
					}
				}
			default:
				if err := w.Write([]string{k, fmt.Sprint(v)}); err != nil {
					return err
				}
			}
		}
		return w.Error()
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}

func newFormatter(cmd *cobra.Command) (output.Formatter, error) {
	cfg, err := config.Resolve(cmd)
	if err != nil {
		return nil, err
	}
	quiet, _ := cmd.Flags().GetBool("quiet")
	return output.New(cfg.Output, os.Stdout, quiet, cfg.Color)
}

func resolveZone(ctx context.Context, client *api.Client, domain string) (string, error) {
	zone, err := client.GetZoneByName(ctx, domain)
	if err != nil {
		if api.IsNotFound(err) {
			return "", fmt.Errorf("zone %q not found", domain)
		}
		return "", fmt.Errorf("looking up zone %q: %w", domain, err)
	}
	return zone.ID, nil
}

func confirmPrompt(msg string) bool {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return false
	}
	fmt.Fprintf(os.Stderr, "%s [y/N]: ", msg)
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))
	return input == "y" || input == "yes"
}

func intPtr(v int) *int {
	return &v
}

func stringPtr(v string) *string {
	return &v
}

func isDryRun(cmd *cobra.Command) bool {
	v, _ := cmd.Flags().GetBool("dry-run")
	return v
}

// zoneDisplayName is the name to echo back to a human: the native-script form of
// an internationalized zone when the API returned one, otherwise the stored ACE
// name. Mirrors what the table formatter shows, so a zone does not change its
// spelling between the command that created it and the one that lists it.
func zoneDisplayName(zone *api.ZoneListItem) string {
	if zone.UnicodeName != "" {
		return zone.UnicodeName
	}
	return zone.Name
}

// ttlPtr converts a TTL read from a zone file or a nexdns.yaml operation into the
// request's optional field. A zero there means "the file did not say", which is
// exactly the case where the API should keep the rrset's own TTL, so it maps to
// no field rather than to a literal 0.
func ttlPtr(ttl int) *int {
	if ttl == 0 {
		return nil
	}
	return &ttl
}
