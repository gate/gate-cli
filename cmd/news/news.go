package news

import "github.com/spf13/cobra"

// Cmd is the user-facing News command group.
var Cmd = &cobra.Command{
	Use:   "news",
	Short: "News and market intelligence commands",
	Long: `News and market intelligence commands.

Agent shortcuts (+ prefix): +brief, +event-explain, +community-scan.

Discovery: gate-cli news list --format table (CLI command paths, not MCP wire names).
Describe: gate-cli news describe --name "news feed search-news" (CLI path or MCP wire name).`,
}

func init() {
	// Schema refresh is env-only (no --refresh-schema flag): set GATE_INTEL_REFRESH_SCHEMA=1.
	// See README.md (Intel), specs/intel-config-and-security.md, specs/open-items-and-dependencies.md,
	// and specs/cli/cli-first-mcp-technical-implementation-plan.md.
}
