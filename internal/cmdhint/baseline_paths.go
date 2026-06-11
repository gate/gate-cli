//go:build agent

package cmdhint

import (
	"fmt"
	"strings"
)

// MCPToolToCLICommand maps a baseline MCP tool name to a runnable gate-cli leaf command.
func MCPToolToCLICommand(toolName string) string {
	toolName = strings.TrimSpace(toolName)
	if toolName == "" {
		return cliBinaryName
	}
	parts := strings.Split(toolName, "_")
	if len(parts) < 3 {
		return cliBinaryName + " " + strings.ReplaceAll(toolName, "_", " ")
	}
	backend := parts[0]
	group := parts[1]
	leaf := strings.Join(parts[2:], "-")
	return fmt.Sprintf("%s %s %s %s --format json", cliBinaryName, backend, group, leaf)
}
