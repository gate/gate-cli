package news

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/intelcmd"
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
	intelcmd.WrapInvokeHelpStripAliasesSection(invokeCmd)
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
	maxOutputBytes, _ := cmd.Root().PersistentFlags().GetInt64("max-output-bytes")
	svc, err := newNewsService(cmd)
	if err != nil {
		return intelcmd.FailIntelClientInit(p, err, "news", "invoke", name)
	}
	return intelcmd.RunToolCall(cmd, p, svc, name, reserved, "news", maxOutputBytes)
}
