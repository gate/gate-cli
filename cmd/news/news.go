package news

import "github.com/spf13/cobra"

// Cmd is the user-facing News command group.
var Cmd = &cobra.Command{
	Use:   "news",
	Short: "News and market intelligence commands",
}

func init() {
	Cmd.PersistentFlags().Bool("refresh-schema", false, "Force refresh tool schema from intel backend before building help flags")
}
