package intelcmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// SilenceCommandTree sets SilenceErrors and SilenceUsage on root and all descendants
// so cobra does not print a second line after PrintError + exitcode returns. This is
// the historical behavior (since 17985cf) covering the entire command tree including
// non-Intel subtrees (cex, config). Call after all subcommands (including dynamically
// registered leaves) are on root.
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

// InstallFlagErrorHook recursively installs a FlagErrorFunc that prints pflag parse
// errors to stderr before propagating the error. It is intentionally scoped to the
// Intel command subtrees (info/news/preflight/doctor/migrate) so the SilenceCommandTree
// behavior on non-Intel subtrees (cex/config) stays unchanged — per the Intel guardrail
// rule, info/news changes must not alter other domains.
//
// SilenceErrors=true on these subtrees already prevents cobra from echoing the same
// message twice; this hook only makes sure the user actually sees the diagnostic.
func InstallFlagErrorHook(c *cobra.Command) {
	if c == nil {
		return
	}
	c.SetFlagErrorFunc(printFlagErrorToStderr)
	for _, sub := range c.Commands() {
		InstallFlagErrorHook(sub)
	}
}

// printFlagErrorToStderr writes the pflag parse error to the command's stderr so the
// user sees the diagnostic. Returning the error preserves the non-zero exit code while
// SilenceErrors=true prevents cobra from echoing the same message.
func printFlagErrorToStderr(cmd *cobra.Command, err error) error {
	if cmd == nil || err == nil {
		return err
	}
	_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Error: %s\n", err.Error())
	return err
}
