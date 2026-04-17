package info

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/intelcmd"
	"github.com/gate/gate-cli/internal/intelfacade"
	"github.com/gate/gate-cli/internal/toolschema"
)

func addFallbackArgFlags(cmd *cobra.Command) {
	cmd.Flags().String("params", "", "Fallback JSON object arguments (use flat flags by default)")
	cmd.Flags().String("args-json", "", "Alias of --params (fallback JSON object)")
	cmd.Flags().String("args-file", "", "Path to JSON file containing fallback arguments object")
}

func makeInfoAliasCommand(use, toolName string) *cobra.Command {
	parts := strings.Split(toolName, "_")
	group := "info"
	if len(parts) >= 2 {
		group = parts[1]
	}
	cmd := &cobra.Command{
		Use:   use,
		Short: "Shortcut for " + toolName,
		Long:  "Calls " + toolName + ". Flat flags come from a static baseline plus any extra fields from the intel backend; use --params/--args-json/--args-file as JSON fallback.",
		Example: "  gate-cli info " + group + " " + use + " --format json\n" +
			"  gate-cli info " + group + " " + use + " --params '{\"key\":\"value\"}'",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInfoCallByName(cmd, toolName, map[string]struct{}{
				"params":    {},
				"args-json": {},
				"args-file": {},
			})
		},
	}
	addFallbackArgFlags(cmd)
	return cmd
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
	out := map[string]toolschema.ToolSummary{}
	defer mergeInfoBaselineInto(out)
	if cached, _, err := toolschema.LoadCache("info"); err == nil {
		for _, t := range cached {
			out[t.Name] = t
		}
	}
	return out
}

func mergeInfoBaselineInto(out map[string]toolschema.ToolSummary) {
	for _, name := range intelfacade.InfoToolBaseline {
		baseline := intelfacade.InfoBaselineInputSchema(name)
		if baseline == nil {
			continue
		}
		existing, ok := out[name]
		if !ok {
			out[name] = toolschema.ToolSummary{
				Name:           name,
				HasInputSchema: true,
				InputSchema:    baseline,
			}
			continue
		}
		if !existing.HasInputSchema || toolschema.IsEmptyInputSchema(existing.InputSchema) {
			existing.HasInputSchema = true
			existing.InputSchema = baseline
			out[name] = existing
		}
	}
}

var infoBusinessAliases = map[string][]string{
	"info_coin_get_coin_info":                 {"coin-info"},
	"info_marketsnapshot_get_market_overview": {"overview", "market-overview"},
	"info_markettrend_get_technical_analysis": {"ta", "trend-analysis"},
	"info_compliance_check_token_security":    {"token-risk"},
	"info_compliance_check_address_risk":      {"address-risk"},
}
