package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/nexdns/cli/internal/config"
	"github.com/nexdns/cli/internal/output"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage persisted CLI configuration",
	Long:  "Read and write settings in the config file (default: ~/.nexdns/config.yaml).",
}

// configKeys is what `config set` / `config get` accept: every field the config
// file itself holds. It used to be only api-url and token, while `config view`
// listed output, color and timeout as well - so the three settings a user would
// most want to make permanent could be changed only by hand-editing the YAML.
const configKeys = "api-url, token, output, color, timeout"

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Persist a setting to the config file",
	Long:  "Persist a setting to the config file. Valid keys: " + configKeys + ".",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]
		value := args[1]
		path := config.ConfigPath(cmd)

		switch key {
		case "api-url":
			if strings.TrimSpace(value) == "" {
				return fmt.Errorf("api-url must not be empty")
			}
			if !strings.HasPrefix(value, "http://") && !strings.HasPrefix(value, "https://") {
				return fmt.Errorf("invalid api-url: must be an http(s) URL")
			}
			if err := config.SetAPIUrl(path, value); err != nil {
				return fmt.Errorf("saving api-url: %w", err)
			}
		case "token":
			if strings.TrimSpace(value) == "" {
				return fmt.Errorf("token must not be empty")
			}
			if err := config.SetToken(path, value); err != nil {
				return fmt.Errorf("saving token: %w", err)
			}
		case "output":
			if err := output.ValidateFormat(value); err != nil {
				return err
			}
			if err := config.SetOutput(path, value); err != nil {
				return fmt.Errorf("saving output: %w", err)
			}
		case "color":
			if err := output.ValidateColorMode(value); err != nil {
				return err
			}
			if err := config.SetColor(path, value); err != nil {
				return fmt.Errorf("saving color: %w", err)
			}
		case "timeout":
			seconds, err := strconv.Atoi(value)
			if err != nil || seconds <= 0 {
				return fmt.Errorf("invalid timeout: %q (expected a positive number of seconds)", value)
			}
			if err := config.SetTimeout(path, seconds); err != nil {
				return fmt.Errorf("saving timeout: %w", err)
			}
		default:
			return fmt.Errorf("unknown key %q. Valid keys: %s", key, configKeys)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Saved %s to %s\n", key, path)
		return nil
	},
}

var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Print a stored setting",
	Long:  "Print the stored value for a key (empty if unset). Valid keys: api-url, token.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]
		path := config.ConfigPath(cmd)
		cfg, err := config.Load(path)
		if err != nil {
			return err
		}

		switch key {
		case "api-url":
			fmt.Fprintln(cmd.OutOrStdout(), cfg.APIUrl)
		case "token":
			fmt.Fprintln(cmd.OutOrStdout(), cfg.Token)
		case "output":
			fmt.Fprintln(cmd.OutOrStdout(), cfg.Output)
		case "color":
			fmt.Fprintln(cmd.OutOrStdout(), cfg.Color)
		case "timeout":
			if cfg.Timeout > 0 {
				fmt.Fprintln(cmd.OutOrStdout(), cfg.Timeout)
			} else {
				fmt.Fprintln(cmd.OutOrStdout())
			}
		default:
			return fmt.Errorf("unknown key %q. Valid keys: %s", key, configKeys)
		}
		return nil
	},
}

var configViewCmd = &cobra.Command{
	Use:     "view",
	Aliases: []string{"list"},
	Short:   "Show the resolved configuration",
	Args:    cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Resolve(cmd)
		if err != nil {
			return err
		}

		tokenDisplay := cfg.Token
		if len(tokenDisplay) > 8 {
			tokenDisplay = tokenDisplay[:8] + "..."
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Config file: %s\n", config.ConfigPath(cmd))
		fmt.Fprintf(cmd.OutOrStdout(), "api-url: %s\n", cfg.APIUrl)
		fmt.Fprintf(cmd.OutOrStdout(), "token: %s\n", tokenDisplay)
		fmt.Fprintf(cmd.OutOrStdout(), "output: %s\n", cfg.Output)
		fmt.Fprintf(cmd.OutOrStdout(), "color: %s\n", cfg.Color)
		fmt.Fprintf(cmd.OutOrStdout(), "timeout: %d\n", cfg.Timeout)
		return nil
	},
}

func init() {
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configViewCmd)
	rootCmd.AddCommand(configCmd)
}
