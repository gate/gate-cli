package info

import (
	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/intelcmd"
	"github.com/gate/gate-cli/internal/intelfacade"
	"github.com/gate/gate-cli/internal/toolschema"
)

func makeInfoAliasCommand(use, toolName string) *cobra.Command {
	return intelcmd.NewLeafAliasCommand(intelcmd.LeafAliasConfig{
		BackendCLI: "info",
		Use:        use,
		ToolName:   toolName,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInfoCallByName(cmd, toolName, intelcmd.ReservedMCPJSONFallbackFlags())
		},
	})
}

func init() {
	buildInfoAliases()
}

var infoSchemaLoader = loadInfoToolSchemas

func buildInfoAliases() {
	schemas := infoSchemaLoader()
	groups := intelcmd.BuildGroupedAliases(intelcmd.AliasBuildOptions{
		BackendPrefix:   "info",
		BackendTitle:    "Info",
		ToolBaseline:    intelfacade.InfoToolBaseline,
		SchemaSummaries: schemas,
		BusinessAliases: infoBusinessAliases,
		MakeAlias:       makeInfoAliasCommand,
		ApplyBaseline: func(toolName string, cmd *cobra.Command) {
			// Baseline first so committed JSON-schema shapes win for flag wiring.
			if b := intelfacade.InfoBaselineInputSchema(toolName); b != nil {
				toolschema.ApplyInputSchemaFlags(cmd, b)
			}
		},
		AfterAliasBuilt: func(toolName string, cmd *cobra.Command) {
			if toolName == "info_coin_get_coin_info" && cmd.Flags().Lookup("symbol") == nil {
				cmd.Flags().String("symbol", "", "Coin symbol alias to query")
			}
		},
	})
	for _, group := range groups {
		Cmd.AddCommand(group)
	}
}

func loadInfoToolSchemas() map[string]toolschema.ToolSummary {
	return intelcmd.LoadToolSchemasFromCache("info", func(out map[string]toolschema.ToolSummary) {
		intelcmd.MergeToolBaselineInto(out, intelfacade.InfoToolBaseline, intelfacade.InfoBaselineInputSchema)
	})
}

var infoBusinessAliases = map[string][]string{
	"info_coin_get_coin_info":                 {"coin-info"},
	"info_marketsnapshot_get_market_overview": {"overview", "market-overview"},
	"info_markettrend_get_technical_analysis": {"ta", "trend-analysis"},
	"info_compliance_check_token_security":    {"token-risk"},
	"info_compliance_check_address_risk":      {"address-risk"},
}
