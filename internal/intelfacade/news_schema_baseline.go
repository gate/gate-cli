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
		"query":            newsStr("query; non-empty -> similarity mode (tickers not sent downstream)"),
		"coin":             newsStr("coin; comma-separated tickers when query empty"),
		"platform":         newsStr("platform; preferred source filter (over platform_type)"),
		"platform_type":    newsStr("platform_type; legacy mapping; ignored when platform set"),
		"lang":             newsStr("lang; MCP local filter only"),
		"time_range":       newsTimeRange13730Default24h("time_range; overrides start_time/end_time when set"),
		"start_time":       newsStr("start_time; ISO8601 or Unix sec/ms"),
		"end_time":         newsStr("end_time; date-only end treated as end-of-day"),
		"sort_by":          newsStrDefault("sort_by", "time"),
		"top_total_score":  newsNum("top_total_score; query non-empty -> 0; query empty default 1 unless explicitly 0"),
		"limit":            newsIntDefaultMax("limit", 10, 100),
		"page":             newsIntDefault("page", 1),
		"similarity_score": newsStr("similarity_score; default ~0.6 when query non-empty"),
	}),
	"news_feed_search_ugc": newsObj(map[string]interface{}{
		"query":        newsStr("query; non-empty -> vector API; query+coin may combine"),
		"coin":         newsStr("coin; required when query empty (OpenSearch branch); CLI: query or coin"),
		"platform":     newsStrEnum("platform", "all", "reddit", "discord", "telegram", "youtube", "all"),
		"domain":       newsStrEnum("domain", "all", "crypto", "defi", "finance", "macro", "ai_agent", "web3_dev", "all"),
		"channel":      newsStr("channel"),
		"quality_tier": newsStrEnum("quality_tier", "A", "A", "B", "all"),
		"time_range":   newsStrEnum("time_range", "7d", "1h", "24h", "7d", "30d", "all"),
		"sort_by":      newsStrEnum("sort_by", "relevance", "relevance", "upvotes", "recent"),
		"limit":        newsIntDefaultMax("limit", 10, 50),
	}),
	"news_feed_search_x": newsObj(map[string]interface{}{
		"query":                      newsStr("query; empty on xAI path returns empty result"),
		"days":                       newsIntDefaultMin("days; xAI lookback when time_range omitted (runtime default 1)", 1, 1),
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
		"query":      newsStr("query; required"),
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
		"event_type": newsStr("event_type; all or empty -> no type filter"),
		"coin":       newsStr("coin; comma-separated; expanded for related_coins/symbols"),
		"time_range": newsStrEnum("time_range; mutually exclusive with start_time/end_time", "", "1h", "24h", "7d"),
		"start_time": newsStr("start_time; mutually exclusive with time_range"),
		"end_time":   newsStr("end_time; mutually exclusive with time_range"),
		"cursor":     newsStr("cursor; reserved; not used for OpenSearch pagination today"),
		"limit":      newsIntDefaultMax("limit; omitted or <=0 -> 20 upstream; >100 invalid_size", 20, 100),
	}),
	"news_events_get_event_detail": newsObj(map[string]interface{}{
		"event_id": newsEventID("event_id; max 512; pattern A-Za-z0-9:_-"),
	}, "event_id"),
	"news_events_explain_market_move": newsObj(map[string]interface{}{
		"query":      newsStr("query; required; e.g. why an asset moved"),
		"coin":       newsStr("coin; required; normalized upstream"),
		"time_range": newsStrEnum("time_range; invalid/empty/7d -> 2h", "2h", "30m", "1h", "2h", "4h", "24h"),
		"mode":       newsStrEnum("mode", "auto", "auto", "price_move", "event_impact"),
		"lang":       newsStrEnum("lang", "zh", "zh", "en"),
	}, "query", "coin"),
	"news_prediction_get_volume_delta_ranking":   newsPredictionRankingProps(),
	"news_prediction_get_fastest_rising_ranking": newsPredictionRankingProps(),
	"news_prediction_get_market_orderbook": newsObj(map[string]interface{}{
		"venue":       newsStrEnum("venue; polymarket or predict_fun (trimmed)", "", "polymarket", "predict_fun"),
		"market_id":   newsStr("market_id; polymarket: venue_market_id (needs predictionMarketIndex); predict_fun: official numeric id"),
		"depth":       newsIntDefaultMinMax("depth; top-N per side in yes_bids/yes_asks (live book only)", 20, 1, 20),
		"mode":        newsStrEnum("mode; history rejected; empty -> current", "current", "current", ""),
		"granularity": newsStr("granularity; unsupported; non-empty -> invalid_param"),
		"start_time":  newsStr("start_time; unsupported; non-empty -> invalid_param"),
		"end_time":    newsStr("end_time; unsupported; non-empty -> invalid_param"),
		"page_token":  newsStr("page_token; unsupported; non-empty -> invalid_param"),
	}, "venue", "market_id"),
	"news_prediction_search_events": newsObj(map[string]interface{}{
		"query":        newsStr("query; wildcard venue_event_title; pure-digit also term venue_event_id; CLI: query or coin or category"),
		"coin":         newsStr("coin; NormalizeCoin; terms related_coins/symbols; coin-only (no query/category) -> status all"),
		"category":     newsPredictionSearchEventsCategory(),
		"status":       newsStrEnum("status; omit --status for MCP defaults (coin-only no query/category -> all); pass --status active|closed|resolved|all to override", "", "active", "closed", "resolved", "all"),
		"venue":        newsArrVenuePolymarketOpinionPredictFun("venue; venue.keyword filter"),
		"sort_by":      newsStrEnum("sort_by; signal index; default recently_listed; ES 400 fallback chain", "recently_listed", "attention", "volume", "liquidity", "recently_listed", "probability_change", "volume_delta_today"),
		"limit":        newsIntDefaultMinMax("limit; size=limit+1 for next_page_token", 20, 1, 100),
		"page_token":   newsStr("page_token; base64 {sort_by, search_after}; sort_by mismatch -> invalid_param"),
		"with_markets": newsBoolDefault("with_markets; attach dws_prediction_market_hf summaries", false),
	}),
	"news_prediction_get_event_signal": newsObj(map[string]interface{}{
		"event_ref":                 newsEventRef("event_ref; venue:venue_event_id (first ':' splits; id may contain more colons)"),
		"window":                    newsStrEnum("window; part_hour >= now-window; 1h may use signal_window_fallback", "24h", "1h", "24h", "7d"),
		"venue":                     newsArrVenuePolymarketOpinionPredictFun("venue; optional; must match event_ref venue"),
		"include_markets":           newsBoolDefault("include_markets; external markets or legacy JSON; predictionMarketIndex fallback (default true)", true),
		"include_orderbook_summary": newsBoolDefault("include_orderbook_summary; deprecated/unread; live depth -> get_market_orderbook", false),
	}, "event_ref"),
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
		"enum":        []interface{}{"zh", "en", "auto"},
		"default":     "zh",
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

func newsIntDefaultMinMax(desc string, def, min, max int) map[string]interface{} {
	return map[string]interface{}{
		"type":        "integer",
		"description": desc,
		"default":     float64(def),
		"minimum":     float64(min),
		"maximum":     float64(max),
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

func newsBoolDefault(desc string, def bool) map[string]interface{} {
	return map[string]interface{}{
		"type":        "boolean",
		"description": desc,
		"default":     def,
	}
}

func newsEventRef(desc string) map[string]interface{} {
	return map[string]interface{}{
		"type":        "string",
		"description": desc,
		"pattern":     "^[^:]+:[^:]+$",
	}
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
			"enum": []interface{}{"polymarket", "predict_fun"},
		},
	}
}

func newsPredictionRankingProps() map[string]interface{} {
	return newsObj(map[string]interface{}{
		"date_utc": newsDateUTCOptional("date_utc; UTC YYYY-MM-DD rank_date; omit -> today UTC"),
		"limit":    newsIntDefaultMinMax("limit; OpenSearch size (<=0 normalized to 20)", 20, 1, 100),
		"venue":    newsArrVenuePolymarketOpinionPredictFun("venue; empty -> all venues; terms filter"),
		"category": newsStr("category; empty omit; non-all exact term on rank index (no server enum)"),
		"status":   newsStrEnum("status; empty -> active; all -> no filter", "active", "active", "closed", "resolved", "all"),
	})
}

func newsPredictionSearchEventsCategory() map[string]interface{} {
	return newsStrEnum("category", "",
		"crypto_event", "crypto_price", "culture", "earnings", "elections",
		"finance", "geopolitics", "macro_economy", "mentions", "other",
		"politics", "sports", "tech_ai", "weather_climate", "world",
	)
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
