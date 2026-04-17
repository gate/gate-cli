package intelcmd

import "github.com/spf13/cobra"

// SilenceCommandTree sets SilenceErrors and SilenceUsage on root and all descendants
// so cobra does not print a second line after PrintError + exitcode returns.
// Call after all subcommands (including dynamically registered leaves) are on root.
func SilenceCommandTree(root *cobra.Command) {
	if root == nil {
		return
	}
	for _, c := range root.Commands() {
		silenceRecursive(c)
	}
}

func silenceRecursive(c *cobra.Command) {
	c.SilenceErrors = true
	c.SilenceUsage = true
	for _, sub := range c.Commands() {
		silenceRecursive(sub)
	}
}
