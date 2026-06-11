package intelcmd

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/output"
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
// errors to stderr before propagating the error. When installed on the root command,
// this covers the full tree (cex/config included) so SilenceErrors does not hide flag
// diagnostics from agents.
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
// user sees the diagnostic. When --format json (or pretty), emits the same {"error":…}
// envelope as PrintError for agent parsers. Returning the error preserves the non-zero
// exit code while SilenceErrors=true prevents cobra from echoing the same message.
func printFlagErrorToStderr(cmd *cobra.Command, err error) error {
	if cmd == nil || err == nil {
		return err
	}
	w := cmd.ErrOrStderr()
	format := resolveOutputFormat(cmd)
	if format == output.FormatJSON || format == output.FormatPretty {
		p := output.NewWithStderr(io.Discard, w, format)
		ge := output.InvalidArgsError(normalizeFlagErrorMessage(err.Error()))
		p.PrintError(ge)
		return err
	}
	_, _ = fmt.Fprintf(w, "Error: %s\n", err.Error())
	return err
}

func resolveOutputFormat(cmd *cobra.Command) output.Format {
	if cmd == nil {
		return output.FormatPretty
	}
	root := cmd.Root()
	if root == nil {
		return output.FormatPretty
	}
	f := root.PersistentFlags().Lookup("format")
	if f == nil {
		return output.FormatPretty
	}
	raw := strings.TrimSpace(f.Value.String())
	if raw == "" {
		raw = strings.TrimSpace(f.DefValue)
	}
	if raw == "" {
		return output.FormatPretty
	}
	return output.ParseFormat(raw)
}

func normalizeFlagErrorMessage(msg string) string {
	msg = strings.TrimSpace(msg)
	if msg == "" {
		return "invalid flag arguments"
	}
	if strings.HasPrefix(msg, "Error: ") {
		return strings.TrimPrefix(msg, "Error: ")
	}
	return msg
}
