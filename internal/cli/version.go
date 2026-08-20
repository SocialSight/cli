package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Set via -ldflags at build time (see .goreleaser.yaml).
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the socialsight CLI version",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "socialsight %s (commit %s, built %s)\n", version, commit, date)
			return err
		},
	}
}
