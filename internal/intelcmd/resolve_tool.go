package intelcmd

import "github.com/gate/gate-cli/internal/toolschema"

// ResolveMCPToolName preserves the command-layer API. Schema registration and
// execution share one resolver so a documented CLI path binds the same flags
// and calls the same remote tool as its wire-name spelling.
func ResolveMCPToolName(backend, nameOrPath string) string {
	return toolschema.ResolveMCPToolName(backend, nameOrPath)
}
