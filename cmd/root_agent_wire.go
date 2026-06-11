//go:build agent

package cmd

import "github.com/gate/gate-cli/internal/agentcmd"

func init() {
	agentcmd.Register(rootCmd)
}
