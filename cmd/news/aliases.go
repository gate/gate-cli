package news

import (
	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/intelcmd"
	"github.com/gate/gate-cli/internal/intelfacade"
	"github.com/gate/gate-cli/internal/toolschema"
)

func makeNewsAliasCommand(use, toolName string) *cobra.Command {
	return intelcmd.NewLeafAliasCommand(intelcmd.LeafAliasConfig{
		BackendCLI: "news",
		Use:        use,
		ToolName:   toolName,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runNewsCallByName(cmd, toolName, intelcmd.ReservedMCPJSONFallbackFlags())
		},
	})
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
	return intelcmd.LoadToolSchemasFromCache("news", func(out map[string]toolschema.ToolSummary) {
		intelcmd.MergeToolBaselineInto(out, intelfacade.NewsToolBaseline, intelfacade.NewsBaselineInputSchema)
	})
}

var newsBusinessAliases = map[string][]string{
	"news_feed_search_news":                {"search"},
	"news_events_get_latest_events":        {"latest-events"},
	"news_feed_get_social_sentiment":       {"sentiment"},
	"news_feed_get_exchange_announcements": {"announcements"},
	"news_events_get_event_detail":         {"event-detail"},
}
