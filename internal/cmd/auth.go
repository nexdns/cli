package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/nexdns/cli/internal/api"
	"github.com/nexdns/cli/internal/config"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage authentication",
	Long:  "Manage API token authentication. Create an API key at https://nexdns.tech/settings/api-keys",
}

var authTokenCmd = &cobra.Command{
	Use:   "token <TOKEN>",
	Short: "Save API token to config file",
	Long:  "Validate token against the API and save to config file.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		token := args[0]

		if !strings.HasPrefix(token, "nxd_") {
			return fmt.Errorf("invalid token format: must start with 'nxd_' prefix.\nCreate an API key at https://nexdns.tech/settings/api-keys")
		}
		if len(token) < 10 {
			return fmt.Errorf("invalid token: too short")
		}

		// Resolve API URL for validation
		cfg, err := config.Resolve(cmd)
		if err != nil {
			return err
		}

		// Validate token against the API
		client := api.NewClient(cfg.APIUrl, token, time.Duration(cfg.Timeout)*time.Second, false)
		account, err := client.GetAccount(context.Background())
		if err != nil {
			return fmt.Errorf("token validation failed: %w", err)
		}

		path := config.ConfigPath(cmd)
		if err := config.SetToken(path, token); err != nil {
			return fmt.Errorf("saving token: %w", err)
		}

		// Persist the endpoint the token was verified against. Without this every
		// later command fell back to the default API URL, so a token issued on
		// another instance was silently sent to the wrong one.
		if cmd.Flags().Changed("api-url") {
			if err := config.SetAPIUrl(path, cfg.APIUrl); err != nil {
				return fmt.Errorf("saving api-url: %w", err)
			}
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Token verified. Authenticated as %s\n", account.Email)
		fmt.Fprintf(cmd.OutOrStdout(), "Token saved to %s\n", path)
		if cmd.Flags().Changed("api-url") {
			fmt.Fprintf(cmd.OutOrStdout(), "API URL saved: %s\n", cfg.APIUrl)
		}
		return nil
	},
}

var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Remove saved token from config file",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		path := config.ConfigPath(cmd)
		if err := config.RemoveToken(path); err != nil {
			return fmt.Errorf("removing token: %w", err)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Token removed from %s\n", path)
		return nil
	},
}

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current authentication state",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Resolve(cmd)
		if err != nil {
			return err
		}

		if cfg.Token == "" {
			return fmt.Errorf("not authenticated. Run `nexdns auth token <TOKEN>` to authenticate")
		}

		// Determine token source
		tokenSource := "config file"
		if cmd.Flags().Changed("token") {
			tokenSource = "--token flag"
		} else if v := cfg.Token; v != "" {
			if envToken := lookupEnvToken(); envToken != "" && envToken == v {
				tokenSource = "NEXDNS_TOKEN env"
			}
		}

		// Mask token for display
		tokenDisplay := cfg.Token
		if len(tokenDisplay) > 8 {
			tokenDisplay = tokenDisplay[:8] + "..."
		}

		client := api.NewClient(cfg.APIUrl, cfg.Token, time.Duration(cfg.Timeout)*time.Second, false)
		account, err := client.GetAccount(context.Background())
		if err != nil {
			return fmt.Errorf("authentication check failed: %w", err)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Authenticated as: %s\n", account.Email)
		fmt.Fprintf(cmd.OutOrStdout(), "API URL: %s\n", cfg.APIUrl)
		fmt.Fprintf(cmd.OutOrStdout(), "Token: %s  (from %s)\n", tokenDisplay, tokenSource)
		if account.Subscription != nil {
			fmt.Fprintf(cmd.OutOrStdout(), "Plan: %s (%s)\n", account.Subscription.Plan, account.Subscription.BillingCycle)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Status: %s\n", account.Status)
		return nil
	},
}

func lookupEnvToken() string {
	return os.Getenv("NEXDNS_TOKEN")
}

func init() {
	authCmd.AddCommand(authTokenCmd)
	authCmd.AddCommand(authLogoutCmd)
	authCmd.AddCommand(authStatusCmd)
	rootCmd.AddCommand(authCmd)
}
