package info

import "github.com/spf13/cobra"

// Cmd is the user-facing Info command group.
var Cmd = &cobra.Command{
	Use:   "info",
	Short: "Market and intelligence info commands",
}

func init() {
	// Deprecated: schema refresh is controlled via GATE_INTEL_REFRESH_SCHEMA.
}
