package info

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/mcpclient"
	"github.com/gate/gate-cli/internal/output"
	"github.com/gate/gate-cli/internal/toolargs"
	"github.com/gate/gate-cli/internal/toolrender"
	"github.com/gate/gate-cli/internal/toolschema"
)

var callCmd = &cobra.Command{
	Use:   "call --name <tool-name> [flags]",
	Short: "Call one Info capability",
	RunE:  runInfoCall,
}

func init() {
	callCmd.Flags().String("name", "", "Info tool name")
	callCmd.Flags().String("params", "", "JSON object arguments (fallback)")
	callCmd.Flags().String("args-json", "", "JSON object arguments (alias of --params)")
	callCmd.Flags().String("args-file", "", "Path to JSON file containing arguments object")
	_ = callCmd.MarkFlagRequired("name")
	Cmd.AddCommand(callCmd)
}

func runInfoCall(cmd *cobra.Command, args []string) error {
	name, _ := cmd.Flags().GetString("name")
	return runInfoCallByName(cmd, name, map[string]struct{}{
		"name":      {},
		"params":    {},
		"args-json": {},
		"args-file": {},
	})
}

func runInfoCallByName(cmd *cobra.Command, name string, reserved map[string]struct{}) error {
	p := getPrinter(cmd)
	maxOutputBytes, _ := cmd.Root().PersistentFlags().GetInt64("max-output-bytes")
	svc, err := newInfoService(cmd)
	if err != nil {
		p.PrintError(mcpclient.ParseError(err, nil, "POST", "info/call", name))
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
		p.PrintError(mcpclient.ParseError(err, httpResp, "POST", "info/call", name))
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
