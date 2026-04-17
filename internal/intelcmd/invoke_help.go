package intelcmd

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
)

var cobraInvokeAliasesSectionRx = regexp.MustCompile(`(?m)^Aliases:\s*\r?\n(?:[ \t]+.*\r?\n)+`)

// WrapInvokeHelpStripAliasesSection replaces the default help writer so the generated
// "Aliases:" block (from cobra's `Aliases` field) is omitted—invoke keeps `call` as a
// compatibility alias without advertising it in -h output.
func WrapInvokeHelpStripAliasesSection(cmd *cobra.Command) {
	inner := cmd.HelpFunc()
	cmd.SetHelpFunc(func(c *cobra.Command, args []string) {
		dest := c.OutOrStdout()
		var buf strings.Builder
		c.SetOut(&buf)
		inner(c, args)
		c.SetOut(dest)
		out := cobraInvokeAliasesSectionRx.ReplaceAllString(buf.String(), "")
		_, _ = fmt.Fprint(dest, out)
	})
}
