package news

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/mcpclient"
	"github.com/gate/gate-cli/internal/output"
	"github.com/gate/gate-cli/internal/toolargs"
	"github.com/gate/gate-cli/internal/toolrender"
	"github.com/gate/gate-cli/internal/toolschema"
)

var invokeCmd = &cobra.Command{
	Use:     "invoke --name <tool-name> [flags]",
	Short:   "Run one News capability by tool name (flat flags when --name is on the command line)",
	Hidden:  true,
	Aliases: []string{"call"},
	RunE:    runNewsInvoke,
}

func init() {
	invokeCmd.Flags().String("name", "", "News tool name")
	invokeCmd.Flags().String("params", "", "JSON object arguments (fallback)")
	invokeCmd.Flags().String("args-json", "", "JSON object arguments (alias of --params)")
	invokeCmd.Flags().String("args-file", "", "Path to JSON file containing arguments object")
	_ = invokeCmd.MarkFlagRequired("name")
	toolschema.AttachInvokeFlagsFromArgv(invokeCmd, os.Args, "news", loadNewsToolSchemas())
	Cmd.AddCommand(invokeCmd)
	wrapInvokeHelpWithoutAliases(invokeCmd)
}

var cobraAliasesSectionRx = regexp.MustCompile(`(?m)^Aliases:\s*\r?\n(?:[ \t]+.*\r?\n)+`)

func wrapInvokeHelpWithoutAliases(cmd *cobra.Command) {
	inner := cmd.HelpFunc()
	cmd.SetHelpFunc(func(c *cobra.Command, args []string) {
		dest := c.OutOrStdout()
		var buf strings.Builder
		c.SetOut(&buf)
		inner(c, args)
		c.SetOut(dest)
		out := cobraAliasesSectionRx.ReplaceAllString(buf.String(), "")
		_, _ = fmt.Fprint(dest, out)
	})
}

func runNewsInvoke(cmd *cobra.Command, args []string) error {
	name, _ := cmd.Flags().GetString("name")
	return runNewsCallByName(cmd, name, map[string]struct{}{
		"name":      {},
		"params":    {},
		"args-json": {},
		"args-file": {},
	})
}

func runNewsCallByName(cmd *cobra.Command, name string, reserved map[string]struct{}) error {
	p := getPrinter(cmd)
	if p.IsTable() {
		p.PrintError(output.UnsupportedTableFormatError())
		return nil
	}
	maxOutputBytes, _ := cmd.Root().PersistentFlags().GetInt64("max-output-bytes")
	svc, err := newNewsService(cmd)
	if err != nil {
		p.PrintError(mcpclient.ParseError(err, nil, "POST", "news/invoke", name))
		return nil
	}

	arguments, err := toolargs.MergeFromCommand(cmd, toolargs.MergeOptions{ReservedFlags: reserved})
	if err != nil {
		p.PrintError(&output.GateError{
			Status:  400,
			Label:   "INVALID_ARGUMENTS",
			Message: err.Error(),
		})
		return nil
	}
	arguments = toolargs.NormalizeForTool(name, arguments)
	if tool, _, derr := svc.DescribeTool(cmd.Context(), name); derr == nil && tool != nil {
		if missing := toolschema.MissingRequiredArguments(arguments, tool.InputSchema); len(missing) > 0 {
			p.PrintError(&output.GateError{
				Status:  400,
				Label:   "INVALID_ARGUMENTS",
				Message: "missing required fields: " + strings.Join(missing, ", "),
			})
			return nil
		}
	}

	result, httpResp, err := svc.CallTool(cmd.Context(), name, arguments)
	if err != nil {
		p.PrintError(mcpclient.ParseError(err, httpResp, "POST", "news/invoke", name))
		return nil
	}
	if result.IsError {
		p.PrintError(&output.GateError{
			Status:   502,
			Label:    "INTEL_RESULT_ERROR",
			Message:  "tool returned isError=true",
			ToolName: name,
		})
		return nil
	}
	if err := toolrender.RenderCallResult(p, name, result, maxOutputBytes); err != nil {
		return err
	}
	return nil
}
