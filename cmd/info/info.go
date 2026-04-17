package info

import "github.com/spf13/cobra"

// Cmd is the user-facing Info command group.
var Cmd = &cobra.Command{
	Use:   "info",
	Short: "Market and intelligence info commands",
}

func init() {
	// Schema refresh is env-only (no --refresh-schema flag): set GATE_INTEL_REFRESH_SCHEMA=1.
	// See repository README (Intel migration notes) and specs/cli/cli-first-mcp-technical-implementation-plan.md.
}
