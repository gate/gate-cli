package margin

import "github.com/spf13/cobra"

// Cmd is the root command for the margin module.
var Cmd = &cobra.Command{
	Use:   "margin",
	Short: "Margin trading commands",
}
