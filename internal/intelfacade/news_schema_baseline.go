package intelfacade

import "sync"

// NewsBaselineInputSchemas are the stable, static JSON-Schema-shaped contract for CLI flat flags (primary).
// MCP tools/list may register additional non-colliding flags afterward; --params / --args-json are JSON fallback.
// Server-side tools/call validation remains authoritative at runtime.
//
// Keys, types, enum/default, and JSON Schema bounds (minimum/maximum/maxLength/maxItems, etc.)
// mirror specs/mcp/news-tools-args-and-logic.json for flat-flag help text (incl. LLM-facing ranges); extend when upstream adds fields.
var NewsBaselineInputSchemas = map[string]map[string]interface{}{
	"news_feed_search_news": newsObj(map[string]interface{}{
		"query":            newsStr("query"),
		"coin":             newsStr("coin"),
		"platform":         newsStr("platform"),
		"platform_type":    newsStr("platform_type"),
		"lang":             newsStr("lang"),
		"time_range":       newsTimeRange13730Default24h("time_range"),
		"start_time":       newsStr("start_time"),
		"end_time":         newsStr("end_time"),
		"sort_by":          newsStrDefault("sort_by", "time"),
		"top_total_score":  newsNum("top_total_score"),
		"limit":            newsIntDefaultMax("limit", 10, 100),
		"page":             newsIntDefault("page", 1),
		"similarity_score": newsStr("similarity_score; upstream default ~0.6 when query is non-empty"),
	}),
	"news_feed_search_ugc": newsObj(map[string]interface{}{
		"query":        newsStr("query"),
		"coin":         newsStr("coin"),
		"platform":     newsStrEnum("platform", "all", "reddit", "discord", "telegram", "youtube", "all"),
		"domain":       newsStrEnum("domain", "all", "crypto", "defi", "finance", "macro", "ai_agent", "web3_dev", "all"),
		"channel":      newsStr("channel"),
		"quality_tier": newsStrEnum("quality_tier", "A", "A", "B", "all"),
		"time_range":   newsStrEnum("time_range", "7d", "1h", "24h", "7d", "30d", "all"),
		"sort_by":      newsStrEnum("sort_by", "relevance", "relevance", "upvotes", "recent"),
		"limit":        newsIntDefaultMax("limit", 10, 50),
	}),
	"news_feed_search_x": newsObj(map[string]interface{}{
		"query":                      newsStr("query"),
		"days":                       newsIntDefaultMin("days", 7, 1),
		"allowed_handles":            newsArrStrMaxItems("allowed_handles", 10),
		"excluded_handles":           newsArrStrMaxItems("excluded_handles", 10),
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
		"limit":                      newsIntDefault("limit", 10),
		"page":                       newsIntDefault("page", 1),
		"similarity_score":           newsStr("similarity_score"),
	}),
	"news_feed_web_search": newsObj(map[string]interface{}{
		"query":      newsStr("query"),
		"coin":       newsStr("coin"),
		"mode":       newsStrEnum("mode", "analysis", "analysis", "brief"),
		"time_range": newsTimeRange13730Default24h("time_range"),
		"lang":       newsWebSearchLang("lang"),
		"limit":      newsIntDefaultMax("limit", 5, 10),
	}, "query"),
	"news_feed_get_exchange_announcements": newsObj(map[string]interface{}{
		"exchange":          newsStr("exchange"),
		"platform":          newsStr("platform"),
		"query":             newsStr("query"),
		"coin":              newsStr("coin"),
		"announcement_type": newsStrEnum("announcement_type", "", "listing", "delisting", "maintenance", "all"),
		"limit":             newsIntMax("limit", 100),
		"from":              newsInt("from"),
		"to":                newsInt("to"),
	}),
	"news_feed_get_social_sentiment": newsObj(map[string]interface{}{
		"coin":       newsStrDefault("coin", "BTC"),
		"time_range": newsStrEnum("time_range", "24h", "1h", "24h", "7d"),
	}),
	"news_events_get_latest_events": newsObj(map[string]interface{}{
		"event_type": newsStr("event_type"),
		"coin":       newsStr("coin"),
		"time_range": newsStrEnum("time_range", "", "1h", "24h", "7d"),
		"start_time": newsStr("start_time"),
		"end_time":   newsStr("end_time"),
		"cursor":     newsStr("cursor"),
		"limit":      newsIntDefaultMax("limit", 20, 100),
	}),
	"news_events_get_event_detail": newsObj(map[string]interface{}{
		"event_id": newsEventID("event_id"),
	}, "event_id"),
	"news_prediction_get_volume_delta_ranking":   newsPredictionRankingProps(),
	"news_prediction_get_fastest_rising_ranking": newsPredictionRankingProps(),
}

func newsStr(desc string) map[string]interface{} {
	return map[string]interface{}{"type": "string", "description": desc}
}

// newsStrEnum builds a string field with enum (and optional default) for CLI flag usage; values follow specs/mcp/news-tools-args-and-logic.json.
func newsStrEnum(desc, defaultVal string, enum ...string) map[string]interface{} {
	ev := make([]interface{}, len(enum))
	for i, s := range enum {
		ev[i] = s
	}
	m := map[string]interface{}{
		"type":        "string",
		"description": desc,
		"enum":        ev,
	}
	if defaultVal != "" {
		m["default"] = defaultVal
	}
	return m
}

func newsTimeRange13730Default24h(desc string) map[string]interface{} {
	return map[string]interface{}{
		"type":        "string",
		"description": desc,
		"enum":        []interface{}{"1h", "24h", "7d", "30d"},
		"default":     "24h",
	}
}

func newsWebSearchLang(desc string) map[string]interface{} {
	return map[string]interface{}{
		"type":        "string",
		"description": desc,
		"enum":        []interface{}{"en", "zh", "auto"},
		"default":     "en",
	}
}

func newsStrDefault(desc, defaultVal string) map[string]interface{} {
	return map[string]interface{}{
		"type":        "string",
		"description": desc,
		"default":     defaultVal,
	}
}

func newsIntDefault(desc string, def int) map[string]interface{} {
	return map[string]interface{}{
		"type":        "integer",
		"description": desc,
		"default":     float64(def),
	}
}

func newsIntDefaultMax(desc string, def, max int) map[string]interface{} {
	return map[string]interface{}{
		"type":        "integer",
		"description": desc,
		"default":     float64(def),
		"maximum":     float64(max),
	}
}

func newsIntDefaultMin(desc string, def, min int) map[string]interface{} {
	return map[string]interface{}{
		"type":        "integer",
		"description": desc,
		"default":     float64(def),
		"minimum":     float64(min),
	}
}

func newsIntMax(desc string, max int) map[string]interface{} {
	return map[string]interface{}{
		"type":        "integer",
		"description": desc,
		"maximum":     float64(max),
	}
}

func newsArrStrMaxItems(desc string, maxItems int) map[string]interface{} {
	return map[string]interface{}{
		"type":        "array",
		"description": desc,
		"items":       map[string]interface{}{"type": "string"},
		"maxItems":    float64(maxItems),
	}
}

func newsEventID(desc string) map[string]interface{} {
	return map[string]interface{}{
		"type":        "string",
		"description": desc,
		"maxLength":   float64(512),
		"pattern":     "^[A-Za-z0-9:_-]+$",
	}
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

func newsDateUTCOptional(desc string) map[string]interface{} {
	return map[string]interface{}{
		"type":        "string",
		"description": desc,
		"pattern":     `^\d{4}-\d{2}-\d{2}$`,
	}
}

func newsArrVenuePolymarketOpinionPredictFun(desc string) map[string]interface{} {
	return map[string]interface{}{
		"type":        "array",
		"description": desc,
		"items": map[string]interface{}{
			"type": "string",
			"enum": []interface{}{"polymarket", "opinion", "predict_fun"},
		},
	}
}

func newsPredictionRankingProps() map[string]interface{} {
	return newsObj(map[string]interface{}{
		"date_utc": newsDateUTCOptional("date_utc; UTC YYYY-MM-DD; omit for today UTC"),
		"limit":    newsIntDefaultMax("limit", 20, 100),
		"venue":    newsArrVenuePolymarketOpinionPredictFun("venue"),
		"category": newsStrEnum("category", "", "crypto_price", "macro", "policy", "sports", "all"),
		"status":   newsStrEnum("status", "active", "active", "closed", "resolved", "all"),
	})
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
