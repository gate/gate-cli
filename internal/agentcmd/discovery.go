//go:build agent

package agentcmd

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/cmdhint"
	"github.com/gate/gate-cli/internal/cmdindex"
)

// DiscoveryEntries returns leaf catalog entries for agent discovery commands.
// When domain is empty and GATE_CLI_AGENT=1, scope is limited to info + news.
func DiscoveryEntries(root *cobra.Command, domain string) []cmdindex.Entry {
	entries := cmdindex.CollectLeaves(root)
	if d := strings.TrimSpace(domain); d != "" {
		return cmdindex.FilterByDomain(entries, d)
	}
	if cmdhint.AgentModeEnabled() {
		return cmdindex.FilterInfoNewsOnly(entries)
	}
	return entries
}
