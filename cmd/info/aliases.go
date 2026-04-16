package info

import (
	"strings"

	"github.com/spf13/cobra"

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
	groups := map[string]*cobra.Command{}
	orderedGroupNames := []string{}
	for _, tool := range intelfacade.InfoToolBaseline {
		parts := strings.Split(tool, "_")
		if len(parts) < 3 || parts[0] != "info" {
			continue
		}
		groupName := parts[1]
		toolUse := strings.Join(parts[2:], "-")
		group, ok := groups[groupName]
		if !ok {
			group = &cobra.Command{Use: groupName, Short: "Info " + groupName + " shortcuts"}
			groups[groupName] = group
			orderedGroupNames = append(orderedGroupNames, groupName)
		}
		alias := makeInfoAliasCommand(toolUse, tool)
		if aliases, ok := infoBusinessAliases[tool]; ok {
			alias.Aliases = aliases
		}
		if schema, ok := schemas[tool]; ok && !toolschema.IsEmptyInputSchema(schema.InputSchema) {
			toolschema.ApplyInputSchemaFlags(alias, schema.InputSchema)
		}
		if b := intelfacade.InfoBaselineInputSchema(tool); b != nil {
			toolschema.ApplyInputSchemaFlags(alias, b)
		}
		if tool == "info_coin_get_coin_info" && alias.Flags().Lookup("symbol") == nil {
			alias.Flags().String("symbol", "", "Coin symbol alias to query")
		}
		group.AddCommand(alias)
	}
	for _, name := range orderedGroupNames {
		Cmd.AddCommand(groups[name])
	}
}

func loadInfoToolSchemas() map[string]toolschema.ToolSummary {
	out := map[string]toolschema.ToolSummary{}
	defer mergeInfoBaselineInto(out)
	if !toolschema.IsBackendInvoked("info") {
		if cached, _, err := toolschema.LoadCache("info"); err == nil {
			for _, t := range cached {
				out[t.Name] = t
			}
		}
		return out
	}
	forceRefresh := toolschema.ForceRefreshEnabled()
	if !forceRefresh {
		if cached, fresh, err := toolschema.LoadCache("info"); err == nil {
			for _, t := range cached {
				out[t.Name] = t
			}
			if fresh {
				return out
			}
		}
	}
	tmp := &cobra.Command{}
	svc, err := newInfoService(tmp)
	if err != nil {
		return out
	}
	tools, _, err := svc.ListTools(tmp.Context())
	if err != nil || len(tools) == 0 {
		return out
	}
	payload := make([]toolschema.ToolSummary, 0, len(tools))
	for _, t := range tools {
		item := toolschema.ToolSummary{
			Name:           t.Name,
			Description:    t.Description,
			HasInputSchema: t.HasInputSchema,
			InputSchema:    t.InputSchema,
		}
		payload = append(payload, item)
		out[t.Name] = item
	}
	_ = toolschema.SaveCache("info", payload)
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
