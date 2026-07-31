package cmd

import (
	"context"

	"github.com/spf13/cobra"
)

var accountCmd = &cobra.Command{
	Use:   "account",
	Short: "Account information",
}

var accountInfoCmd = &cobra.Command{
	Use:   "info",
	Short: "Show account details",
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

		account, err := client.GetAccount(context.Background())
		if err != nil {
			return err
		}

		formatter.FormatAccount(account)
		return nil
	},
}

var accountAPIKeysCmd = &cobra.Command{
	Use:   "api-keys",
	Short: "List API keys",
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

		keys, err := client.ListAPIKeys(context.Background())
		if err != nil {
			return err
		}

		formatter.FormatAPIKeys(keys)
		return nil
	},
}

func init() {
	accountCmd.AddCommand(accountInfoCmd)
	accountCmd.AddCommand(accountAPIKeysCmd)
	rootCmd.AddCommand(accountCmd)
}
