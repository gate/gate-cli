package intelfacade

import "sync"

// InfoBaselineInputSchemas are static JSON-Schema-shaped objects: the stable source for CLI flat flags.
// MCP tools/list may add additional flags (non-colliding) on top; --params / --args-json remain JSON fallback.
// Enum/default/bounds mirror specs/mcp/info-mcp-tools-inputs-logic.json for flat-flag help (LLM-facing hints).
var InfoBaselineInputSchemas = map[string]map[string]interface{}{
	"info_coin_get_coin_info": infoObj(map[string]interface{}{
		"query":      infoStr("query"),
		"query_type": infoStrEnum("query_type", "auto", "auto", "address", "symbol", "name", "project", "gate_symbol", "source_id"),
		"chain":      infoStr("chain"),
		"scope":      infoStrEnum("scope", "basic", "basic", "detailed", "full", "with_project", "with_tokenomics"),
		"size":       infoIntDefaultMax("size", 10, 100),
		"fields":     infoArrStr("fields"),
		"symbol":     infoStr("Coin symbol alias to query"),
	}, "query"),
	"info_coin_search_coins": infoObj(map[string]interface{}{
		"category":       infoStr("category"),
		"chain":          infoStr("chain"),
		"market_cap_min": infoNum("market_cap_min"),
		"market_cap_max": infoNum("market_cap_max"),
		"asset_type":     infoStrEnum("asset_type", "crypto", "crypto", "tradefi", "all"),
		"sort_by":        infoStrEnum("sort_by", "market_cap", "market_cap", "fdv", "circulating_supply"),
		"limit":          infoIntDefaultMax("limit", 20, 100),
		"offset":         infoIntDefaultMax("offset", 0, 100000),
	}),
	"info_coin_get_coin_rankings": infoObj(map[string]interface{}{
		"ranking_type":    infoStrEnum("ranking_type", "", "popular", "top_gainers", "top_losers", "twitter_hot", "airdrop", "new_listing"),
		"time_range":      infoStrEnum("time_range", "", "1h", "24h", "7d"),
		"limit":           infoIntDefaultMax("limit", 50, 100),
		"listing_query":   infoStr("listing_query"),
		"listing_from":    infoInt("listing_from"),
		"listing_tickers": infoStr("listing_tickers"),
	}, "ranking_type"),
	"info_markettrend_get_kline": infoObj(map[string]interface{}{
		"symbol":          infoStr("symbol"),
		"timeframe":       infoStrEnum("timeframe", "", "1m", "5m", "15m", "1h", "4h", "1d"),
		"period":          infoStrEnum("period", "24h", "1h", "4h", "24h", "7d", "3d", "5d", "10d", "all"),
		"size":            infoIntDefaultMax("size", 100, 2000),
		"limit":           infoIntDefaultMax("limit", 100, 2000),
		"start_time":      infoStr("start_time"),
		"end_time":        infoStr("end_time"),
		"with_indicators": infoBool("with_indicators"),
	}, "symbol", "timeframe"),
	"info_markettrend_get_indicator_history": infoObj(map[string]interface{}{
		"symbol":     infoStr("symbol"),
		"indicators": infoArrStrIndicatorHints("indicators"),
		"timeframe":  infoStrEnum("timeframe", "", "15m", "1h", "4h", "1d"),
		"start_time": infoStr("start_time"),
		"end_time":   infoStr("end_time"),
		"limit":      infoIntDefaultMax("limit", 100, 500),
		"source":     infoStrEnum("source", "spot", "alpha", "spot", "future", "fx", "futures"),
		"quote":      infoStr("quote"),
	}, "symbol", "indicators", "timeframe"),
	"info_markettrend_get_technical_analysis": infoObj(map[string]interface{}{
		"symbol":     infoStr("symbol"),
		"period":     infoStrEnum("period", "3d", "1h", "4h", "24h", "7d", "3d", "5d", "10d", "all"),
		"start_time": infoStr("start_time"),
		"end_time":   infoStr("end_time"),
	}, "symbol"),
	"info_marketsnapshot_get_market_snapshot": infoObj(map[string]interface{}{
		"symbol":              infoStr("symbol"),
		"timeframe":           infoStrEnum("timeframe", "1h", "15m", "1h", "4h", "1d"),
		"indicator_timeframe": infoStrEnum("indicator_timeframe", "", "15m", "1h", "4h", "1d"),
		"source":              infoStrEnum("source", "spot", "alpha", "spot", "future", "fx", "futures"),
		"quote":               infoStr("quote"),
		"scope":               infoStrEnum("scope", "basic", "basic", "detailed", "full"),
	}, "symbol"),
	"info_marketsnapshot_batch_market_snapshot": infoObj(map[string]interface{}{
		"symbols":   infoArrStrMaxItems("symbols", 20),
		"timeframe": infoStrEnum("timeframe", "1h", "15m", "1h", "4h", "1d"),
		"source":    infoStrEnum("source", "spot", "alpha", "spot", "future", "fx", "futures"),
		"quote":     infoStr("quote"),
		"scope":     infoStrEnum("scope", "basic", "basic", "detailed", "full"),
	}, "symbols"),
	"info_marketsnapshot_get_market_overview": infoObj(map[string]interface{}{}),
	"info_onchain_get_address_info": infoObj(map[string]interface{}{
		"address":              infoStr("address"),
		"chain":                infoStr("chain"),
		"scope":                infoStrEnum("scope", "basic", "basic", "with_defi", "with_counterparties", "with_pnl", "full", "detailed"),
		"min_value_usd":        infoNum("min_value_usd"),
		"include_upstream_raw": infoBool("include_upstream_raw"),
		"upstream_raw_mode":    infoStrEnum("upstream_raw_mode", "off", "off", "lite", "full"),
	}, "address"),
	"info_onchain_get_address_transactions": infoObj(map[string]interface{}{
		"address":              infoStr("address"),
		"chain":                infoStr("chain"),
		"min_value_usd":        infoNum("min_value_usd"),
		"tx_type":              infoStrEnum("tx_type", "all", "transfer", "contract_call", "token_transfer", "all"),
		"time_range":           infoStrEnum("time_range", "", "1h", "24h", "1d", "7d", "30d", "90d"),
		"start_time":           infoInt("start_time"),
		"end_time":             infoInt("end_time"),
		"limit":                infoIntDefaultMax("limit", 50, 200),
		"from_address":         infoStr("from_address"),
		"to_address":           infoStr("to_address"),
		"nonzero_value":        infoBool("nonzero_value"),
		"include_upstream_raw": infoBool("include_upstream_raw"),
		"upstream_raw_mode":    infoStrEnum("upstream_raw_mode", "off", "off", "lite", "full"),
	}, "address"),
	"info_onchain_get_transaction": infoObj(map[string]interface{}{
		"tx_hash":              infoStr("tx_hash"),
		"chain":                infoStr("chain"),
		"include_upstream_raw": infoBool("include_upstream_raw"),
		"upstream_raw_mode":    infoStrEnum("upstream_raw_mode", "off", "off", "lite", "full"),
	}, "tx_hash"),
	"info_onchain_get_token_onchain": infoObj(map[string]interface{}{
		"token":                infoStr("token"),
		"chain":                infoStr("chain"),
		"scope":                infoStrEnum("scope", "full", "holders", "activity", "transfers", "smart_money", "full"),
		"include_upstream_raw": infoBool("include_upstream_raw"),
		"upstream_raw_mode":    infoStrEnum("upstream_raw_mode", "off", "off", "lite", "full"),
	}, "token"),
	"info_compliance_check_token_security": infoObj(map[string]interface{}{
		"token":   infoStr("token"),
		"address": infoStr("address"),
		"chain":   infoStr("chain"),
		"scope":   infoStrEnum("scope", "basic", "basic", "full"),
		"lang":    infoStrEnum("lang", "en", "en", "cn", "tw", "ja", "kr"),
	}, "chain"),
	"info_platformmetrics_get_platform_info": infoObj(map[string]interface{}{
		"platform_name": infoStr("platform_name"),
		"scope":         infoStrEnum("scope", "basic", "basic", "with_chain_breakdown", "full", "detailed"),
	}, "platform_name"),
	"info_platformmetrics_search_platforms": infoObj(map[string]interface{}{
		"platform_type": infoStr("platform_type"),
		"chain":         infoStr("chain"),
		"sort_by":       infoStrEnum("sort_by", "tvl", "tvl", "volume_24h", "volume_spot_24h", "volume_perps_24h", "fees_24h"),
		"sort_order":    infoStrEnum("sort_order", "desc", "asc", "desc"),
		"limit":         infoIntDefaultMax("limit", 20, 100),
	}),
	"info_platformmetrics_get_defi_overview": infoObj(map[string]interface{}{
		"category": infoStrEnum("category", "all", "all", "defi", "de-fi", "cex", "perp", "spot", "stablecoin", "dex", "dexes", "lending", "cdp", "yield", "bridge", "derivatives", "yield aggregator"),
	}),
	"info_platformmetrics_get_stablecoin_info": infoObj(map[string]interface{}{
		"symbol": infoStr("symbol"),
		"chain":  infoStr("chain"),
		"limit":  infoIntDefaultMax("limit", 20, 100),
	}),
	"info_platformmetrics_get_bridge_metrics": infoObj(map[string]interface{}{
		"bridge_name": infoStr("bridge_name"),
		"chain":       infoStr("chain"),
		"sort_by":     infoStrEnum("sort_by", "volume_24h", "volume_24h", "volume_7d", "volume_30d", "deposit_txs_24h"),
		"limit":       infoIntDefaultMax("limit", 20, 100),
	}),
	"info_platformmetrics_get_yield_pools": infoObj(map[string]interface{}{
		"project":     infoStr("project"),
		"chain":       infoStr("chain"),
		"symbol":      infoStr("symbol"),
		"pool_type":   infoStr("pool_type"),
		"sort_by":     infoStrEnum("sort_by", "apy", "apy", "tvl_usd"),
		"limit":       infoIntDefaultMax("limit", 20, 100),
		"min_tvl_usd": infoNum("min_tvl_usd"),
	}),
	"info_platformmetrics_get_platform_history": infoObj(map[string]interface{}{
		"platform_name": infoStr("platform_name"),
		"metrics":       infoArrStrMetricsHints("metrics"),
		"start_date":    infoStr("start_date"),
		"end_date":      infoStr("end_date"),
	}, "platform_name"),
	"info_platformmetrics_get_exchange_reserves": infoObj(map[string]interface{}{
		"exchange": infoStr("exchange"),
		"asset":    infoStr("asset"),
		"period":   infoStrEnum("period", "", "24h", "7d"),
	}),
	"info_platformmetrics_get_liquidation_heatmap": infoObj(map[string]interface{}{
		"symbol":   infoStr("symbol"),
		"exchange": infoStr("exchange"),
		"range":    infoStr("range"),
	}, "symbol"),
	"info_macro_get_macro_indicator": infoObj(map[string]interface{}{
		"mode":         infoStrEnum("mode", "latest", "latest", "timeseries"),
		"indicator":    infoStr("indicator"),
		"country":      infoStr("country"),
		"country_code": infoStr("country_code"),
		"start_time":   infoStr("start_time"),
		"end_time":     infoStr("end_time"),
		"start_date":   infoStr("start_date"),
		"end_date":     infoStr("end_date"),
		"size":         infoIntDefaultMax("size", 50, 200),
	}, "indicator"),
	"info_macro_get_economic_calendar": infoObj(map[string]interface{}{
		"start_date": infoStr("start_date"),
		"end_date":   infoStr("end_date"),
		"event_type": infoStr("event_type"),
		"importance": infoStr("importance"),
		"size":       infoIntDefaultMax("size", 50, 200),
	}),
	"info_macro_get_macro_summary": infoObj(map[string]interface{}{}),
	"info_marketdetail_get_orderbook": infoObj(map[string]interface{}{
		"symbol":      infoStr("symbol"),
		"market_type": infoStrEnum("market_type", "spot", "spot", "futures", "delivery", "options"),
		"depth":       infoIntDefaultMax("depth", 20, 100),
		"settle":      infoStr("settle"),
		"extra":       infoObjAny("extra"),
	}, "symbol"),
	"info_marketdetail_get_recent_trades": infoObj(map[string]interface{}{
		"symbol":      infoStr("symbol"),
		"market_type": infoStrEnum("market_type", "spot", "spot", "futures", "delivery", "options"),
		"limit":       infoIntDefaultMax("limit", 100, 1000),
		"settle":      infoStr("settle"),
		"extra":       infoObjAny("extra"),
	}, "symbol"),
	"info_marketdetail_get_kline": infoObj(map[string]interface{}{
		"symbol":      infoStr("symbol"),
		"market_type": infoStrEnum("market_type", "spot", "spot", "futures", "delivery", "options"),
		"timeframe":   infoStr("timeframe"),
		"start_time":  infoInt("start_time"),
		"end_time":    infoInt("end_time"),
		"limit":       infoIntDefaultMax("limit", 100, 2000),
		"settle":      infoStr("settle"),
		"extra":       infoObjAny("extra"),
	}, "symbol", "timeframe"),
}

func infoStr(desc string) map[string]interface{} {
	return map[string]interface{}{"type": "string", "description": desc}
}

// infoStrEnum builds a string field with enum (and optional default) for CLI flag usage; values follow specs/mcp/info-mcp-tools-inputs-logic.json.
func infoStrEnum(desc, defaultVal string, enum ...string) map[string]interface{} {
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

func infoArrStrMaxItems(desc string, maxItems int) map[string]interface{} {
	return map[string]interface{}{
		"type":        "array",
		"description": desc,
		"items":       map[string]interface{}{"type": "string"},
		"maxItems":    float64(maxItems),
	}
}

func infoArrStrIndicatorHints(desc string) map[string]interface{} {
	return map[string]interface{}{
		"type":        "array",
		"description": desc + " (required); ES _source names e.g. rsi, macd, close_price; not a closed server enum — see specs/mcp/info-mcp-tools-inputs-logic.json",
		"items":       map[string]interface{}{"type": "string"},
	}
}

func infoArrStrMetricsHints(desc string) map[string]interface{} {
	return map[string]interface{}{
		"type":        "array",
		"description": desc + "; typical tvl, volume, fees, revenue; empty -> [tvl] server-side; per-element not strictly validated",
		"items":       map[string]interface{}{"type": "string"},
	}
}

func infoIntDefaultMax(desc string, def, max int) map[string]interface{} {
	return map[string]interface{}{
		"type":        "integer",
		"description": desc,
		"default":     float64(def),
		"maximum":     float64(max),
	}
}

func infoInt(desc string) map[string]interface{} {
	return map[string]interface{}{"type": "integer", "description": desc}
}

func infoNum(desc string) map[string]interface{} {
	return map[string]interface{}{"type": "number", "description": desc}
}

func infoBool(desc string) map[string]interface{} {
	return map[string]interface{}{"type": "boolean", "description": desc}
}

func infoArrStr(desc string) map[string]interface{} {
	return map[string]interface{}{
		"type":        "array",
		"description": desc,
		"items":       map[string]interface{}{"type": "string"},
	}
}

func infoObjAny(desc string) map[string]interface{} {
	return map[string]interface{}{"type": "object", "description": desc}
}

func infoObj(props map[string]interface{}, required ...string) map[string]interface{} {
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

// InfoBaselineInputSchema returns a deep copy of the baseline schema for toolName, or nil.
func InfoBaselineInputSchema(toolName string) map[string]interface{} {
	infoBaselineOnce.Do(initInfoBaselineFrozen)
	raw, ok := infoBaselineFrozen[toolName]
	if !ok || len(raw) == 0 {
		return nil
	}
	return deepCloneSchemaMap(raw)
}

var (
	infoBaselineOnce   sync.Once
	infoBaselineFrozen map[string]map[string]interface{}
)

func initInfoBaselineFrozen() {
	infoBaselineFrozen = make(map[string]map[string]interface{}, len(InfoBaselineInputSchemas))
	freezeToolBaseline(infoBaselineFrozen, InfoBaselineInputSchemas)
}
