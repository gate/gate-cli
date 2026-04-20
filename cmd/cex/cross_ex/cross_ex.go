package crossex

import "github.com/spf13/cobra"

// Cmd is the root command for the cross-exchange module.
var Cmd = &cobra.Command{
	Use:   "cross-ex",
	Short: "Cross-exchange trading commands",
}
