//go:build agent

package cmdhint

import (
	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/cmdindex"
)

// ShortcutPathMismatch describes a shipped shortcut missing from the cobra tree.
type ShortcutPathMismatch struct {
	Expected   string `json:"expected_path"`
	Suggestion string `json:"suggestion,omitempty"`
}

// AgentValidateReport is the combined CI report for MCP catalog + shortcuts.
type AgentValidateReport struct {
	OK         bool                   `json:"ok"`
	MCP        MCPValidateSection     `json:"mcp_catalog"`
	Shortcuts  ShortcutValidateSection `json:"shortcuts"`
	TotalCount int                    `json:"count"`
}

// MCPValidateSection holds baseline MCP path validation results.
type MCPValidateSection struct {
	OK         bool                  `json:"ok"`
	Count      int                   `json:"count"`
	Mismatches []CatalogPathMismatch `json:"mismatches"`
}

// ShortcutValidateSection holds shortcut path validation results.
type ShortcutValidateSection struct {
	OK         bool                   `json:"ok"`
	Count      int                    `json:"count"`
	Mismatches []ShortcutPathMismatch `json:"mismatches"`
}

// ValidateAgentDiscovery checks MCP catalog (46) and info/news shortcuts (10) against root.
func ValidateAgentDiscovery(root *cobra.Command) AgentValidateReport {
	mcp := ValidateMCPCatalogAgainstTree(root)
	shortcuts := ValidateShortcutsAgainstTree(root)
	total := len(mcp) + len(shortcuts)
	return AgentValidateReport{
		OK: total == 0,
		MCP: MCPValidateSection{
			OK:         len(mcp) == 0,
			Count:      len(mcp),
			Mismatches: mcp,
		},
		Shortcuts: ShortcutValidateSection{
			OK:         len(shortcuts) == 0,
			Count:      len(shortcuts),
			Mismatches: shortcuts,
		},
		TotalCount: total,
	}
}

// ValidateShortcutsAgainstTree checks shipped + shortcuts exist as runnable leaves.
func ValidateShortcutsAgainstTree(root *cobra.Command) []ShortcutPathMismatch {
	if root == nil {
		return nil
	}
	leafSet := make(map[string]struct{})
	for _, e := range cmdindex.FilterInfoNewsOnly(cmdindex.CollectLeaves(root)) {
		leafSet[e.Path] = struct{}{}
	}
	var out []ShortcutPathMismatch
	for _, path := range InfoNewsShortcutPaths {
		if _, ok := leafSet[path]; ok {
			continue
		}
		mm := ShortcutPathMismatch{Expected: path}
		if hint := cmdindex.ClosestPaths(root, path, 1); len(hint) > 0 {
			mm.Suggestion = hint[0]
		}
		out = append(out, mm)
	}
	return out
}
