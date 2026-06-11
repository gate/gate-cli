package intelcmd

import (
	"github.com/gate/gate-cli/internal/toolargs"
)

// PrepareToolArguments normalizes aliases and applies agent defaults, then validates before MCP call.
func PrepareToolArguments(toolName string, args map[string]interface{}) (map[string]interface{}, error) {
	args = toolargs.NormalizeForTool(toolName, args)
	if err := toolargs.ValidateForTool(toolName, args); err != nil {
		return nil, err
	}
	return args, nil
}
