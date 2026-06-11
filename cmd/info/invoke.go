package info

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/intelcmd"
	"github.com/gate/gate-cli/internal/toolschema"
)

var invokeCmd = &cobra.Command{
	Use:     "invoke --name <tool-name-or-command> [flags]",
	Short:   "Run one Info capability (MCP wire name or CLI command path)",
	Example: "  gate-cli info invoke --name \"info coin get-coin-info\" --query BTC --format json\n  gate-cli info invoke --name info_coin_get_coin_info --query BTC --format json",
	Hidden:  true,
	Aliases: []string{"call"},
	RunE:    runInfoInvoke,
}

func init() {
	invokeCmd.Flags().String("name", "", "Info MCP tool name or CLI command (e.g. info coin get-coin-info)")
	invokeCmd.Flags().String("params", "", "JSON object arguments (fallback)")
	invokeCmd.Flags().String("args-json", "", "JSON object arguments (alias of --params)")
	invokeCmd.Flags().String("args-file", "", "Path to JSON file containing arguments object")
	_ = invokeCmd.MarkFlagRequired("name")
	toolschema.AttachInvokeFlagsFromArgv(invokeCmd, os.Args, "info", loadInfoToolSchemas())
	Cmd.AddCommand(invokeCmd)
	intelcmd.WrapInvokeHelpStripAliasesSection(invokeCmd)
}

func runInfoInvoke(cmd *cobra.Command, args []string) error {
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
	name = intelcmd.ResolveMCPToolName("info", name)
	svc, err := newInfoService(cmd)
	if err != nil {
		return intelcmd.FailIntelClientInit(p, err, "info", "invoke", name)
	}
	return intelcmd.RunToolCall(cmd, p, svc, name, reserved, "info", maxOutputBytes)
}
