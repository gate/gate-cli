package intelfacade

// NewsBaselineInputSchemas are the stable, static JSON-Schema-shaped contract for CLI flat flags (primary).
// MCP tools/list may register additional non-colliding flags afterward; --params / --args-json are JSON fallback.
// Server-side tools/call validation remains authoritative at runtime.
//
// Keys and types follow specs/cli/mcp-wire-appendix.md and QC examples; extend when upstream adds fields.
var NewsBaselineInputSchemas = map[string]map[string]interface{}{
	"news_feed_search_news": newsObj(map[string]interface{}{
		"query":   newsStr("Search query"),
		"coin":    newsStr("Optional coin filter"),
		"limit":   newsInt("Result limit"),
		"sort_by": newsStr("Sort: time | importance | sentiment"),
	}, "query"),
	"news_feed_search_ugc": newsObj(map[string]interface{}{
		"query":    newsStr("Search query"),
		"coin":     newsStr("Coin filter"),
		"platform": newsStr("Platform filter"),
		"limit":    newsInt("Result limit"),
	}, "query"),
	"news_feed_search_x": newsObj(map[string]interface{}{
		"query": newsStr("Search query"),
		"days":  newsInt("Lookback days"),
		"limit": newsInt("Result limit"),
	}, "query"),
	"news_feed_web_search": newsObj(map[string]interface{}{
		"query": newsStr("Search query"),
		"coin":  newsStr("Coin context"),
		"limit": newsInt("Result limit"),
	}, "query"),
	"news_feed_get_social_sentiment": newsObj(map[string]interface{}{
		"post_id":    newsStr("Post identifier (upstream may require this)"),
		"coin":       newsStr("Coin symbol"),
		"time_range": newsStr("Time window e.g. 24h"),
		"limit":      newsInt("Result limit"),
	}),
	"news_feed_get_exchange_announcements": newsObj(map[string]interface{}{
		"exchange": newsStr("Exchange id e.g. gate"),
		"limit":    newsInt("Result limit"),
	}, "exchange"),
	"news_events_get_latest_events": newsObj(map[string]interface{}{
		"coin":       newsStr("Coin symbol"),
		"time_range": newsStr("Time window e.g. 24h"),
		"limit":      newsInt("Result limit"),
	}, "coin"),
	"news_events_get_event_detail": newsObj(map[string]interface{}{
		"event_id": newsStr("Opaque event id from get_latest_events"),
	}, "event_id"),
}

func newsStr(desc string) map[string]interface{} {
	return map[string]interface{}{"type": "string", "description": desc}
}

func newsInt(desc string) map[string]interface{} {
	return map[string]interface{}{"type": "integer", "description": desc}
}

func newsObj(props map[string]interface{}, required ...string) map[string]interface{} {
	out := map[string]interface{}{
		"type":       "object",
		"properties": props,
	}
	if len(required) > 0 {
		req := make([]interface{}, len(required))
		for i, r := range required {
			req[i] = r
		}
		out["required"] = req
	}
	return out
}

// NewsBaselineInputSchema returns a shallow copy of the baseline schema for toolName, or nil.
func NewsBaselineInputSchema(toolName string) map[string]interface{} {
	raw, ok := NewsBaselineInputSchemas[toolName]
	if !ok || len(raw) == 0 {
		return nil
	}
	// shallow copy so callers cannot mutate package globals
	out := make(map[string]interface{}, len(raw))
	for k, v := range raw {
		out[k] = v
	}
	if props, ok := raw["properties"].(map[string]interface{}); ok {
		pc := make(map[string]interface{}, len(props))
		for pk, pv := range props {
			pc[pk] = pv
		}
		out["properties"] = pc
	}
	return out
}
