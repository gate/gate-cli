package news

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

func makeNewsAliasCommand(use, toolName string) *cobra.Command {
	parts := strings.Split(toolName, "_")
	group := "news"
	if len(parts) >= 2 {
		group = parts[1]
	}
	cmd := &cobra.Command{
		Use:   use,
		Short: "Shortcut for " + toolName,
		Long:  "Calls " + toolName + ". Flat flags come from a static baseline plus any extra fields from the intel backend; use --params/--args-json/--args-file as JSON fallback.",
		Example: "  gate-cli news " + group + " " + use + " --format json\n" +
			"  gate-cli news " + group + " " + use + " --params '{\"key\":\"value\"}'",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runNewsCallByName(cmd, toolName, map[string]struct{}{
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
	buildNewsAliases()
}

var newsSchemaLoader = loadNewsToolSchemas

func buildNewsAliases() {
	schemas := newsSchemaLoader()
	groups := intelcmd.BuildGroupedAliases(intelcmd.AliasBuildOptions{
		BackendPrefix:   "news",
		BackendTitle:    "News",
		ToolBaseline:    intelfacade.NewsToolBaseline,
		SchemaSummaries: schemas,
		BusinessAliases: newsBusinessAliases,
		MakeAlias:       makeNewsAliasCommand,
		ApplyBaseline: func(toolName string, cmd *cobra.Command) {
			if b := intelfacade.NewsBaselineInputSchema(toolName); b != nil {
				toolschema.ApplyInputSchemaFlags(cmd, b)
			}
		},
	})
	for _, group := range groups {
		Cmd.AddCommand(group)
	}
}

func loadNewsToolSchemas() map[string]toolschema.ToolSummary {
	out := map[string]toolschema.ToolSummary{}
	defer mergeNewsBaselineInto(out)
	if cached, _, err := toolschema.LoadCache("news"); err == nil {
		for _, t := range cached {
			out[t.Name] = t
		}
	}
	return out
}

func mergeNewsBaselineInto(out map[string]toolschema.ToolSummary) {
	for _, name := range intelfacade.NewsToolBaseline {
		baseline := intelfacade.NewsBaselineInputSchema(name)
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

var newsBusinessAliases = map[string][]string{
	"news_feed_search_news":                {"search"},
	"news_events_get_latest_events":        {"latest-events"},
	"news_feed_get_social_sentiment":       {"sentiment"},
	"news_feed_get_exchange_announcements": {"announcements"},
	"news_events_get_event_detail":         {"event-detail"},
}
