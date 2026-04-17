package news

import "github.com/spf13/cobra"

// Cmd is the user-facing News command group.
var Cmd = &cobra.Command{
	Use:   "news",
	Short: "News and market intelligence commands",
}

func init() {
	// Deprecated: schema refresh is controlled via GATE_INTEL_REFRESH_SCHEMA.
}
