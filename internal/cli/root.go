// Package cli wires up the socialsight command tree.
package cli

import (
	"github.com/spf13/cobra"
)

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "socialsight",
		Short:         "Generate images and videos with SocialSight models",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.AddCommand(newVersionCmd())
	root.AddCommand(newAuthCmd())
	root.AddCommand(newModelCmd())
	root.AddCommand(newGenerateCmd())
	root.AddCommand(newJobsCmd())

	return root
}

// Execute runs the root command.
func Execute() error {
	return newRootCmd().Execute()
}
