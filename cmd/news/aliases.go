package news

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
		Args:  cobra.NoArgs,
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
	groups := map[string]*cobra.Command{}
	orderedGroupNames := []string{}
	for _, tool := range intelfacade.NewsToolBaseline {
		parts := strings.Split(tool, "_")
		if len(parts) < 3 || parts[0] != "news" {
			continue
		}
		groupName := parts[1]
		toolUse := strings.Join(parts[2:], "-")
		group, ok := groups[groupName]
		if !ok {
			group = &cobra.Command{Use: groupName, Short: "News " + groupName + " shortcuts"}
			groups[groupName] = group
			orderedGroupNames = append(orderedGroupNames, groupName)
		}
		alias := makeNewsAliasCommand(toolUse, tool)
		if aliases, ok := newsBusinessAliases[tool]; ok {
			alias.Aliases = aliases
		}
		if b := intelfacade.NewsBaselineInputSchema(tool); b != nil {
			toolschema.ApplyInputSchemaFlags(alias, b)
		}
		if schema, ok := schemas[tool]; ok && !toolschema.IsEmptyInputSchema(schema.InputSchema) {
			toolschema.ApplyInputSchemaFlags(alias, schema.InputSchema)
		}
		group.AddCommand(alias)
	}
	for _, name := range orderedGroupNames {
		Cmd.AddCommand(groups[name])
	}
}

func loadNewsToolSchemas() map[string]toolschema.ToolSummary {
	out := map[string]toolschema.ToolSummary{}
	defer mergeNewsBaselineInto(out)

	if !toolschema.IsBackendInvoked("news") {
		if cached, _, err := toolschema.LoadCache("news"); err == nil {
			for _, t := range cached {
				out[t.Name] = t
			}
		}
		return out
	}
	forceRefresh := toolschema.ForceRefreshEnabled()
	if !forceRefresh {
		if cached, fresh, err := toolschema.LoadCache("news"); err == nil {
			for _, t := range cached {
				out[t.Name] = t
			}
			if fresh {
				return out
			}
		}
	}
	tmp := &cobra.Command{}
	svc, err := newNewsService(tmp)
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
	_ = toolschema.SaveCache("news", payload)
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
