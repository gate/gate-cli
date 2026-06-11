//go:build agent

package cmdhint

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/cmdindex"
)

// CatalogPathMismatch describes a baseline MCP tool with no matching runnable cobra leaf.
type CatalogPathMismatch struct {
	ToolName   string `json:"tool_name"`
	Expected   string `json:"expected_path"`
	Command    string `json:"command"`
	Suggestion string `json:"suggestion,omitempty"`
}

// ValidateMCPCatalogAgainstTree checks each baseline tool maps to an existing leaf path in root.
func ValidateMCPCatalogAgainstTree(root *cobra.Command) []CatalogPathMismatch {
	if root == nil {
		return nil
	}
	leafSet := make(map[string]struct{})
	for _, e := range cmdindex.FilterInfoNewsOnly(cmdindex.CollectLeaves(root)) {
		leafSet[e.Path] = struct{}{}
	}
	var out []CatalogPathMismatch
	for _, name := range baselineToolNames() {
		leaf := BaselineToolLeaf(name)
		path := catalogCLIPath(leaf.Command)
		if _, ok := leafSet[path]; ok {
			continue
		}
		mm := CatalogPathMismatch{
			ToolName: name,
			Expected: path,
			Command:  leaf.Command,
		}
		if hint := cmdindex.ClosestPaths(root, path, 1); len(hint) > 0 {
			mm.Suggestion = hint[0]
		}
		out = append(out, mm)
	}
	return out
}

func catalogCLIPath(command string) string {
	command = strings.TrimSpace(command)
	if strings.HasPrefix(command, cliBinaryName+" ") {
		command = strings.TrimPrefix(command, cliBinaryName+" ")
	}
	if idx := strings.Index(command, " --"); idx >= 0 {
		command = command[:idx]
	}
	return strings.TrimSpace(command)
}
