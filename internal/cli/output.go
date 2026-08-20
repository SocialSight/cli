package cli

import (
	"encoding/json"

	"github.com/spf13/cobra"
)

// wantsJSON reports whether the global --json flag was set.
func wantsJSON(cmd *cobra.Command) bool {
	v, _ := cmd.Flags().GetBool("json")
	return v
}

// printJSON writes v to stdout as indented JSON.
func printJSON(cmd *cobra.Command, v interface{}) error {
	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
