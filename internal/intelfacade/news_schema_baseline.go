package intelfacade

import "sync"

// NewsBaselineInputSchemas are the stable, static JSON-Schema-shaped contract for CLI flat flags (primary).
// MCP tools/list may register additional non-colliding flags afterward; --params / --args-json are JSON fallback.
// Server-side tools/call validation remains authoritative at runtime.
//
// Keys and types follow specs/mcp/news-tools-args-and-logic.json; extend when upstream adds fields.
var NewsBaselineInputSchemas = map[string]map[string]interface{}{
	"news_feed_search_news": newsObj(map[string]interface{}{
		"query":            newsStr("query"),
		"coin":             newsStr("coin"),
		"platform":         newsStr("platform"),
		"platform_type":    newsStr("platform_type"),
		"lang":             newsStr("lang"),
		"time_range":       newsStr("time_range"),
		"start_time":       newsStr("start_time"),
		"end_time":         newsStr("end_time"),
		"sort_by":          newsStr("sort_by"),
		"top_total_score":  newsNum("top_total_score"),
		"limit":            newsInt("limit"),
		"page":             newsInt("page"),
		"similarity_score": newsStr("similarity_score"),
	}),
	"news_feed_search_ugc": newsObj(map[string]interface{}{
		"query":        newsStr("query"),
		"coin":         newsStr("coin"),
		"platform":     newsStr("platform"),
		"domain":       newsStr("domain"),
		"channel":      newsStr("channel"),
		"quality_tier": newsStr("quality_tier"),
		"time_range":   newsStr("time_range"),
		"sort_by":      newsStr("sort_by"),
		"limit":        newsInt("limit"),
	}),
	"news_feed_search_x": newsObj(map[string]interface{}{
		"query":                      newsStr("query"),
		"days":                       newsInt("days"),
		"allowed_handles":            newsArrStr("allowed_handles"),
		"excluded_handles":           newsArrStr("excluded_handles"),
		"model":                      newsStr("model"),
		"enable_image_understanding": newsBool("enable_image_understanding"),
		"enable_video_understanding": newsBool("enable_video_understanding"),
		"coin":                       newsStr("coin"),
		"platform":                   newsStr("platform"),
		"platform_type":              newsStr("platform_type"),
		"lang":                       newsLangDefaultZh("lang"),
		"time_range":                 newsTimeRange24h("time_range"),
		"start_time":                 newsStr("start_time"),
		"end_time":                   newsStr("end_time"),
		"sort_by":                    newsStr("sort_by"),
		"top_total_score":            newsNum("top_total_score"),
		"limit":                      newsInt("limit"),
		"page":                       newsInt("page"),
		"similarity_score":           newsStr("similarity_score"),
	}),
	"news_feed_web_search": newsObj(map[string]interface{}{
		"query":      newsStr("query"),
		"coin":       newsStr("coin"),
		"mode":       newsStr("mode"),
		"time_range": newsStr("time_range"),
		"lang":       newsLangDefaultZh("lang"),
		"limit":      newsInt("limit"),
	}, "query"),
	"news_feed_get_exchange_announcements": newsObj(map[string]interface{}{
		"exchange":          newsStr("exchange"),
		"platform":          newsStr("platform"),
		"query":             newsStr("query"),
		"coin":              newsStr("coin"),
		"announcement_type": newsStr("announcement_type"),
		"limit":             newsInt("limit"),
		"from":              newsInt("from"),
		"to":                newsInt("to"),
	}),
	"news_feed_get_social_sentiment": newsObj(map[string]interface{}{
		"coin":       newsStr("coin"),
		"time_range": newsStr("time_range"),
	}),
	"news_events_get_latest_events": newsObj(map[string]interface{}{
		"event_type": newsStr("event_type"),
		"coin":       newsStr("coin"),
		"time_range": newsStr("time_range"),
		"start_time": newsStr("start_time"),
		"end_time":   newsStr("end_time"),
		"cursor":     newsStr("cursor"),
		"limit":      newsInt("limit"),
	}),
	"news_events_get_event_detail": newsObj(map[string]interface{}{
		"event_id": newsStr("event_id"),
	}, "event_id"),
}

func newsStr(desc string) map[string]interface{} {
	return map[string]interface{}{"type": "string", "description": desc}
}

func newsInt(desc string) map[string]interface{} {
	return map[string]interface{}{"type": "integer", "description": desc}
}

func newsNum(desc string) map[string]interface{} {
	return map[string]interface{}{"type": "number", "description": desc}
}

func newsBool(desc string) map[string]interface{} {
	return map[string]interface{}{"type": "boolean", "description": desc}
}

func newsTimeRange24h(desc string) map[string]interface{} {
	return map[string]interface{}{
		"type":        "string",
		"description": desc,
		"enum":        []interface{}{"1h", "24h", "7d"},
		"default":     "24h",
	}
}

func newsLangDefaultZh(desc string) map[string]interface{} {
	return map[string]interface{}{
		"type":        "string",
		"description": desc,
		"enum":        []interface{}{"zh", "en", "auto"},
		"default":     "zh",
	}
}

func newsArrStr(desc string) map[string]interface{} {
	return map[string]interface{}{
		"type":        "array",
		"description": desc,
		"items":       map[string]interface{}{"type": "string"},
	}
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

// NewsBaselineInputSchema returns a deep copy of the baseline schema for toolName, or nil.
func NewsBaselineInputSchema(toolName string) map[string]interface{} {
	newsBaselineOnce.Do(initNewsBaselineFrozen)
	raw, ok := newsBaselineFrozen[toolName]
	if !ok || len(raw) == 0 {
		return nil
	}
	return deepCloneSchemaMap(raw)
}

var (
	newsBaselineOnce   sync.Once
	newsBaselineFrozen map[string]map[string]interface{}
)

func initNewsBaselineFrozen() {
	newsBaselineFrozen = make(map[string]map[string]interface{}, len(NewsBaselineInputSchemas))
	freezeToolBaseline(newsBaselineFrozen, NewsBaselineInputSchemas)
}
