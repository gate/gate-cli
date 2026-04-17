package news

import "github.com/spf13/cobra"

// Cmd is the user-facing News command group.
var Cmd = &cobra.Command{
	Use:   "news",
	Short: "News and market intelligence commands",
}

func init() {
	// Schema refresh is env-only (no --refresh-schema flag): set GATE_INTEL_REFRESH_SCHEMA=1.
	// See README.md (Intel), specs/intel-config-and-security.md, specs/open-items-and-dependencies.md,
	// and specs/cli/cli-first-mcp-technical-implementation-plan.md.
}
