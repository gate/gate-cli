//go:build agent

package agentcmd

import "github.com/spf13/cobra"

// Register adds GateAI agent discovery commands to the gate-cli root (wired from cmd/root_agent_wire.go).
func Register(root *cobra.Command) {
	if root == nil {
		return
	}
	root.AddCommand(
		NewLeavesCmd(),
		NewResolveCmd(),
		NewSearchCmd(),
		NewIndexCmd(),
		NewValidateCmd(),
	)
}
