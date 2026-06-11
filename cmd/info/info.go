package info

import "github.com/spf13/cobra"

// Cmd is the user-facing Info command group.
var Cmd = &cobra.Command{
	Use:   "info",
	Short: "Market and intelligence info commands",
	Long: `Market and intelligence info commands.

Agent shortcuts (+ prefix): +coin-overview, +market-overview, +coin-compare, +trend-analysis, +token-risk, +address-tracker, +token-onchain.
+address-risk remains deferred until info_compliance_check_address_risk ships.

Discovery: gate-cli info list --format table (CLI command paths, not MCP wire names).
Describe: gate-cli info describe --name "info coin get-coin-info" (CLI path or MCP wire name).`,
}

func init() {
	// Schema refresh is env-only (no --refresh-schema flag): set GATE_INTEL_REFRESH_SCHEMA=1.
	// See README.md (Intel), specs/intel-config-and-security.md, specs/open-items-and-dependencies.md,
	// and specs/cli/cli-first-mcp-technical-implementation-plan.md.
}
