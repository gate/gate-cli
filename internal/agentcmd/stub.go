//go:build !agent

package agentcmd

import "github.com/spf13/cobra"

// Register is a no-op when the binary is built without -tags agent.
func Register(*cobra.Command) {}
