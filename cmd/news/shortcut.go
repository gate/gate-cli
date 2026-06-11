package news

import (
	"context"
	"strings"

	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/cmdutil"
	"github.com/gate/gate-cli/internal/intelcmd"
	"github.com/gate/gate-cli/internal/output"
	"github.com/gate/gate-cli/internal/toolrender"
)

func init() {
	buildNewsShortcuts()
}

func buildNewsShortcuts() {
	Cmd.AddCommand(newNewsBriefCmd(), newNewsEventExplainCmd(), newNewsCommunityScanCmd())
}

func newNewsBriefCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "+brief",
		Aliases: []string{"brief"},
		RunE: func(cmd *cobra.Command, args []string) error {
			coin, _ := cmd.Flags().GetString("coin")
			query, _ := cmd.Flags().GetString("query")
			timeRange, _ := cmd.Flags().GetString("time-range")
			return runNewsShortcut(cmd, "news/+brief", func(ctx context.Context, svc newsService) (map[string]interface{}, error) {
				if strings.TrimSpace(coin) == "" && strings.TrimSpace(query) == "" {
					return nil, intelcmd.ShortcutArgsError("coin or query is required")
				}
				events, errEvents := callNewsShortcutTool(ctx, svc, "news_events_get_latest_events", map[string]interface{}{"coin": coin, "time_range": timeRange, "limit": 10})
				news, errNews := callNewsShortcutTool(ctx, svc, "news_feed_search_news", map[string]interface{}{"coin": coin, "query": query, "limit": 10, "sort_by": "importance"})
				if errEvents != nil && errNews != nil {
					return nil, errEvents
				}
				missing := []string{}
				out := map[string]interface{}{
					"headline_summary":  map[string]interface{}{},
					"top_events":        events,
					"news_digest":       news,
					"sentiment_summary": map[string]interface{}{},
					"watch_items":       []interface{}{},
					"partial":           false,
					"missing_sections":  []string{},
				}
				if errEvents != nil {
					missing = append(missing, "top_events")
				}
				if errNews != nil {
					missing = append(missing, "news_digest")
				}
				if sentiment, err := callNewsShortcutTool(ctx, svc, "news_feed_get_social_sentiment", map[string]interface{}{"coin": coin, "time_range": timeRange}); err == nil {
					out["sentiment_summary"] = sentiment
				} else {
					missing = append(missing, "sentiment_summary")
				}
				if len(missing) > 0 {
					out["partial"] = true
					out["missing_sections"] = missing
				}
				return out, nil
			})
		},
	}
	cmd.Flags().String("coin", "", "Coin symbol")
	cmd.Flags().String("query", "", "Keyword query")
	cmd.Flags().String("time-range", "24h", "Time range")
	return cmd
}

func newNewsEventExplainCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "+event-explain",
		Aliases: []string{"event-explain"},
		RunE: func(cmd *cobra.Command, args []string) error {
			eventID, _ := cmd.Flags().GetString("event-id")
			coin, _ := cmd.Flags().GetString("coin")
			query, _ := cmd.Flags().GetString("query")
			timeRange, _ := cmd.Flags().GetString("time-range")
			return runNewsShortcut(cmd, "news/+event-explain", func(ctx context.Context, svc newsService) (map[string]interface{}, error) {
				if strings.TrimSpace(eventID) == "" && strings.TrimSpace(coin) == "" && strings.TrimSpace(query) == "" {
					return nil, intelcmd.ShortcutArgsError("event-id or coin/query is required")
				}
				if strings.TrimSpace(eventID) == "" {
					latest, err := callNewsShortcutTool(ctx, svc, "news_events_get_latest_events", map[string]interface{}{"coin": coin, "time_range": timeRange, "limit": 10})
					if err != nil {
						return nil, err
					}
					if id := readFirstStringField(latest, "event_id"); id != "" {
						eventID = id
					}
				}
				if strings.TrimSpace(eventID) == "" {
					return nil, intelcmd.ShortcutArgsError("failed to resolve event-id")
				}
				detail, err := callNewsShortcutTool(ctx, svc, "news_events_get_event_detail", map[string]interface{}{"event_id": eventID})
				if err != nil {
					return nil, err
				}
				searchTerm := firstNonEmpty(query, coin, readFirstStringField(detail, "title"))
				if strings.TrimSpace(searchTerm) == "" {
					searchTerm = eventID
				}
				newsCov, err := callNewsShortcutTool(ctx, svc, "news_feed_search_news", map[string]interface{}{"query": searchTerm, "limit": 10})
				if err != nil {
					return nil, err
				}
				out := map[string]interface{}{
					"event_summary":         detail,
					"timeline":              detail,
					"source_coverage":       newsCov,
					"market_interpretation": map[string]interface{}{},
					"open_questions":        []interface{}{},
				}
				if xCov, err := callNewsShortcutTool(ctx, svc, "news_feed_search_x", map[string]interface{}{"query": searchTerm, "limit": 10, "time_range": timeRange}); err == nil {
					out["market_interpretation"] = xCov
				} else {
					out["partial"] = true
					out["missing_sections"] = []string{"market_interpretation"}
				}
				return out, nil
			})
		},
	}
	cmd.Flags().String("event-id", "", "Event ID")
	cmd.Flags().String("coin", "", "Coin symbol")
	cmd.Flags().String("query", "", "Keyword query")
	cmd.Flags().String("time-range", "24h", "Time range")
	return cmd
}

func newNewsCommunityScanCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "+community-scan",
		Aliases: []string{"community-scan"},
		RunE: func(cmd *cobra.Command, args []string) error {
			coin, _ := cmd.Flags().GetString("coin")
			query, _ := cmd.Flags().GetString("query")
			timeRange, _ := cmd.Flags().GetString("time-range")
			return runNewsShortcut(cmd, "news/+community-scan", func(ctx context.Context, svc newsService) (map[string]interface{}, error) {
				if strings.TrimSpace(coin) == "" && strings.TrimSpace(query) == "" {
					return nil, intelcmd.ShortcutArgsError("coin or query is required")
				}
				ugc, errUGC := callNewsShortcutTool(ctx, svc, "news_feed_search_ugc", map[string]interface{}{"coin": coin, "query": query, "platform": "all", "time_range": timeRange, "sort_by": "relevance"})
				xres, errX := callNewsShortcutTool(ctx, svc, "news_feed_search_x", map[string]interface{}{"coin": coin, "query": query, "time_range": timeRange, "limit": 10})
				if errUGC != nil && errX != nil {
					return nil, errUGC
				}
				missing := []string{}
				out := map[string]interface{}{
					"community_summary": map[string]interface{}{"coin": coin, "query": query},
					"top_ugc_posts":     ugc,
					"top_x_threads":     xres,
					"sentiment_summary": map[string]interface{}{},
					"narratives":        []interface{}{},
					"partial":           false,
					"missing_sections":  []string{},
				}
				if errUGC != nil {
					missing = append(missing, "top_ugc_posts")
				}
				if errX != nil {
					missing = append(missing, "top_x_threads")
				}
				if sentiment, err := callNewsShortcutTool(ctx, svc, "news_feed_get_social_sentiment", map[string]interface{}{"coin": coin, "time_range": timeRange}); err == nil {
					out["sentiment_summary"] = sentiment
				} else {
					missing = append(missing, "sentiment_summary")
				}
				if len(missing) > 0 {
					out["partial"] = true
					out["missing_sections"] = missing
				}
				return out, nil
			})
		},
	}
	cmd.Flags().String("coin", "", "Coin symbol")
	cmd.Flags().String("query", "", "Keyword query")
	cmd.Flags().String("time-range", "24h", "Time range")
	return cmd
}

func runNewsShortcut(cmd *cobra.Command, path string, runner func(ctx context.Context, svc newsService) (map[string]interface{}, error)) error {
	p := getPrinter(cmd)
	if p.IsTable() {
		return intelcmd.FailLeafUnsupportedTable(p, "news")
	}
	svc, err := newNewsService(cmd)
	if err != nil {
		return intelcmd.FailIntelClientInit(p, err, "news", "shortcut", "")
	}
	out, err := runner(cmd.Context(), svc)
	if err != nil {
		ge := intelcmd.GateErrorFromShortcutErr(err, path)
		output.FillAgentErrorConvergence(ge)
		return intelcmd.FailAfterPrintError(p, ge)
	}
	return toolrender.RenderIntelPayload(p, path, out, cmdutil.GetMaxOutputBytes(cmd))
}

func callNewsShortcutTool(ctx context.Context, svc newsService, name string, args map[string]interface{}) (map[string]interface{}, error) {
	return intelcmd.CallShortcutTool(ctx, svc, name, args)
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if t := strings.TrimSpace(v); t != "" {
			return t
		}
	}
	return ""
}

func readFirstStringField(m map[string]interface{}, key string) string {
	if m == nil {
		return ""
	}
	if v, ok := m[key].(string); ok && strings.TrimSpace(v) != "" {
		return v
	}
	for _, v := range m {
		arr, ok := v.([]interface{})
		if !ok || len(arr) == 0 {
			continue
		}
		first, ok := arr[0].(map[string]interface{})
		if !ok {
			continue
		}
		if s, ok := first[key].(string); ok && strings.TrimSpace(s) != "" {
			return s
		}
	}
	return ""
}
