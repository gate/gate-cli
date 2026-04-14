package info

import "github.com/spf13/cobra"

// Cmd is the user-facing Info command group.
var Cmd = &cobra.Command{
	Use:   "info",
	Short: "Market and intelligence info commands",
}

func init() {
	Cmd.PersistentFlags().Bool("refresh-schema", false, "Force refresh tool schema from intel backend before building help flags")
}
