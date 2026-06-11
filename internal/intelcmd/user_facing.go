package intelcmd

import (
	"strings"

	"github.com/gate/gate-cli/internal/output"
)

// UserFacingCLICommand returns a user-facing gate-cli command reference without MCP wire names.
// shortcutPath is like "info/+token-risk"; mcpToolName is like "info_coin_get_coin_info" (leaf only).
func UserFacingCLICommand(backend, shortcutPath, mcpToolName string) string {
	if p := strings.TrimSpace(shortcutPath); p != "" {
		return p
	}
	backend = strings.TrimSpace(backend)
	leaf := strings.TrimSpace(toolNameToCLIPath(backend, mcpToolName))
	if backend == "" {
		return leaf
	}
	if leaf == "" {
		return backend
	}
	return backend + " " + leaf
}

// SanitizeUserFacingGateError clears MCP tool_name from stderr JSON and uses CLI command paths instead.
func SanitizeUserFacingGateError(ge *output.GateError, backend, shortcutPath, mcpToolName string) {
	if ge == nil {
		return
	}
	ge.ToolName = ""
	cmd := UserFacingCLICommand(backend, shortcutPath, mcpToolName)
	if cmd == "" {
		return
	}
	if ge.Request == nil {
		ge.Request = &output.RequestInfo{Method: "POST"}
	}
	ge.Request.URL = cmd
}
