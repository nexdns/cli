package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/nexdns/cli/internal/version"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(version.Full())
	},
}
