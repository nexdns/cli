package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/nexdns/cli/internal/version"
)

var rootCmd = &cobra.Command{
	Use:           "nexdns",
	Short:         "NexDNS CLI – manage DNS zones and records",
	Long:          "Command-line tool for managing DNS zones and records via the NexDNS API.",
	Version:       version.Version,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	pf := rootCmd.PersistentFlags()

	pf.String("token", "", "API token (env: NEXDNS_TOKEN)")
	pf.String("api-url", "https://api.nexdns.tech/v1", "API base URL (env: NEXDNS_API_URL)")
	pf.String("config", "", "Config file path (default: ~/.nexdns/config.yaml, env: NEXDNS_CONFIG)")
	pf.StringP("output", "o", "table", "Output format: table, json, yaml, csv")
	pf.String("color", "auto", "Color mode: auto, always, never (also respects NO_COLOR env)")
	pf.BoolP("quiet", "q", false, "Suppress non-essential output")
	pf.BoolP("verbose", "v", false, "Show HTTP requests/responses")
	pf.Bool("dry-run", false, "Preview changes without applying")
	pf.Int("timeout", 30, "HTTP request timeout in seconds (env: NEXDNS_TIMEOUT)")

	// Handle NO_COLOR env per https://no-color.org
	if _, ok := os.LookupEnv("NO_COLOR"); ok {
		_ = pf.Set("color", "never")
	}

	rootCmd.SetVersionTemplate("{{.Version}}\n")

	// Register -V as short flag for --version
	rootCmd.Flags().BoolP("version", "V", false, "Print version and exit")

	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(completionCmd)
}

// ExitCode holds the exit code to use. Set to 2 for usage errors.
var ExitCode = 1

// Execute runs the root command and returns its error (if any). It sets
// ExitCode to 2 for Cobra usage errors (unknown command/flag, wrong arg count)
// so the caller can distinguish them from runtime failures, which keep the
// default exit code 1.
func Execute() error {
	rejectUnknownSubcommands(rootCmd)

	err := rootCmd.Execute()
	if err != nil {
		// Cobra usage errors: unknown commands, wrong arg count, invalid flags
		if isUsageError(err) {
			ExitCode = 2
		}
	}
	return err
}

// rejectUnknownSubcommands makes a group command (one that only holds
// subcommands, such as `zone` or `record`) fail on a name it does not know.
// Cobra's default for such a command is to print its help and succeed, so
// `nexdns zone frobincate` exited 0 and a script could not tell a typo from a
// completed operation - while the same typo at the top level exited 2. Called
// from Execute, after every package init has registered its commands.
func rejectUnknownSubcommands(cmd *cobra.Command) {
	for _, sub := range cmd.Commands() {
		rejectUnknownSubcommands(sub)
	}

	if !cmd.HasSubCommands() || cmd.Run != nil || cmd.RunE != nil {
		return
	}

	cmd.RunE = func(c *cobra.Command, args []string) error {
		if len(args) > 0 {
			return fmt.Errorf("unknown command %q for %q", args[0], c.CommandPath())
		}
		return c.Help()
	}
}

// IsSilent reports whether an error from Execute has already been reported to
// the user by the command itself (a propagation check that printed its own
// per-resolver table, for example). The caller should exit non-zero without
// printing anything further.
func IsSilent(err error) bool {
	var silent *silentError
	return errors.As(err, &silent)
}

func isUsageError(err error) bool {
	msg := err.Error()
	for _, pattern := range []string{
		"unknown command",
		"unknown flag",
		"unknown shorthand flag",
		"accepts ",
		"requires ",
		"invalid argument",
	} {
		if len(msg) >= len(pattern) && containsStr(msg, pattern) {
			return true
		}
	}
	return false
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
