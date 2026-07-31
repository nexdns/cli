// Command nexdns is the NexDNS command-line client for managing DNS zones and
// records through the REST API. It is a thin entry point: the command tree and
// all behaviour live in package internal/cmd.
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
