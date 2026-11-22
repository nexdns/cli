// Command nexdns is the NexDNS command-line client.
package main

import (
	"fmt"
	"os"

	"github.com/nexdns/cli/internal/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		if !cmd.IsSilent(err) {
			fmt.Fprintln(os.Stderr, "Error:", err)
		}
		os.Exit(cmd.ExitCode)
	}
}
