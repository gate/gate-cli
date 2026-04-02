package unified

import "github.com/spf13/cobra"

// Cmd is the root command for the unified module.
var Cmd = &cobra.Command{
	Use:   "unified",
	Short: "Unified account commands",
}
