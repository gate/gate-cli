package intelfacade

import "sync"

// InfoBaselineInputSchemas are static JSON-Schema-shaped objects: the stable source for CLI flat flags.
// MCP tools/list may add additional flags (non-colliding) on top; --params / --args-json remain JSON fallback.
// Align with specs/mcp/info-mcp-tools-inputs-logic.json.
var InfoBaselineInputSchemas = map[string]map[string]interface{}{
	"info_coin_get_coin_info": infoObj(map[string]interface{}{
		"query":      infoStr("query"),
		"query_type": infoStr("query_type"),
		"chain":      infoStr("chain"),
		"scope":      infoStr("scope"),
		"size":       infoInt("size"),
		"fields":     infoArrStr("fields"),
	}, "query"),
	"info_coin_search_coins": infoObj(map[string]interface{}{
		"category":       infoStr("category"),
		"chain":          infoStr("chain"),
		"market_cap_min": infoNum("market_cap_min"),
		"market_cap_max": infoNum("market_cap_max"),
		"asset_type":     infoStr("asset_type"),
		"sort_by":        infoStr("sort_by"),
		"limit":          infoInt("limit"),
		"offset":         infoInt("offset"),
	}),
	"info_coin_get_coin_rankings": infoObj(map[string]interface{}{
		"ranking_type":    infoStr("ranking_type"),
		"time_range":      infoStr("time_range"),
		"limit":           infoInt("limit"),
		"listing_query":   infoStr("listing_query"),
		"listing_from":    infoInt("listing_from"),
		"listing_tickers": infoStr("listing_tickers"),
	}, "ranking_type"),
	"info_markettrend_get_kline": infoObj(map[string]interface{}{
		"symbol":          infoStr("symbol"),
		"timeframe":       infoStr("timeframe"),
		"period":          infoStr("period"),
		"size":            infoInt("size"),
		"limit":           infoInt("limit"),
		"start_time":      infoStr("start_time"),
		"end_time":        infoStr("end_time"),
		"with_indicators": infoBool("with_indicators"),
	}, "symbol", "timeframe"),
	"info_markettrend_get_indicator_history": infoObj(map[string]interface{}{
		"symbol":     infoStr("symbol"),
		"indicators": infoArrStr("indicators"),
		"timeframe":  infoStr("timeframe"),
		"start_time": infoStr("start_time"),
		"end_time":   infoStr("end_time"),
		"limit":      infoInt("limit"),
	}, "symbol", "indicators", "timeframe"),
	"info_markettrend_get_technical_analysis": infoObj(map[string]interface{}{
		"symbol":     infoStr("symbol"),
		"period":     infoStr("period"),
		"start_time": infoStr("start_time"),
		"end_time":   infoStr("end_time"),
	}, "symbol"),
	"info_marketsnapshot_get_market_snapshot": infoObj(map[string]interface{}{
		"symbol":              infoStr("symbol"),
		"timeframe":           infoStr("timeframe"),
		"indicator_timeframe": infoStr("indicator_timeframe"),
		"source":              infoStr("source"),
		"quote":               infoStr("quote"),
		"scope":               infoStr("scope"),
	}, "symbol"),
	"info_marketsnapshot_batch_market_snapshot": infoObj(map[string]interface{}{
		"symbols":   infoArrStr("symbols"),
		"timeframe": infoStr("timeframe"),
		"source":    infoStr("source"),
		"quote":     infoStr("quote"),
		"scope":     infoStr("scope"),
	}, "symbols"),
	"info_marketsnapshot_get_market_overview": infoObj(map[string]interface{}{}),
	"info_onchain_get_address_info": infoObj(map[string]interface{}{
		"address":              infoStr("address"),
		"chain":                infoStr("chain"),
		"scope":                infoStr("scope"),
		"min_value_usd":        infoNum("min_value_usd"),
		"include_upstream_raw": infoBool("include_upstream_raw"),
		"upstream_raw_mode":    infoStr("upstream_raw_mode"),
	}, "address"),
	"info_onchain_get_address_transactions": infoObj(map[string]interface{}{
		"address":              infoStr("address"),
		"chain":                infoStr("chain"),
		"min_value_usd":        infoNum("min_value_usd"),
		"tx_type":              infoStr("tx_type"),
		"time_range":           infoStr("time_range"),
		"start_time":           infoInt("start_time"),
		"end_time":             infoInt("end_time"),
		"limit":                infoInt("limit"),
		"from_address":         infoStr("from_address"),
		"to_address":           infoStr("to_address"),
		"nonzero_value":        infoBool("nonzero_value"),
		"include_upstream_raw": infoBool("include_upstream_raw"),
		"upstream_raw_mode":    infoStr("upstream_raw_mode"),
	}, "address"),
	"info_onchain_get_transaction": infoObj(map[string]interface{}{
		"tx_hash":              infoStr("tx_hash"),
		"chain":                infoStr("chain"),
		"include_upstream_raw": infoBool("include_upstream_raw"),
		"upstream_raw_mode":    infoStr("upstream_raw_mode"),
	}, "tx_hash"),
	"info_onchain_get_token_onchain": infoObj(map[string]interface{}{
		"token":                infoStr("token"),
		"chain":                infoStr("chain"),
		"scope":                infoStr("scope"),
		"include_upstream_raw": infoBool("include_upstream_raw"),
		"upstream_raw_mode":    infoStr("upstream_raw_mode"),
	}, "token"),
	"info_compliance_check_token_security": infoObj(map[string]interface{}{
		"token":   infoStr("token"),
		"address": infoStr("address"),
		"chain":   infoStr("chain"),
		"scope":   infoStr("scope"),
		"lang":    infoStr("lang"),
	}, "token", "chain"),
	"info_platformmetrics_get_platform_info": infoObj(map[string]interface{}{
		"platform_name": infoStr("platform_name"),
		"scope":         infoStr("scope"),
	}, "platform_name"),
	"info_platformmetrics_search_platforms": infoObj(map[string]interface{}{
		"platform_type": infoStr("platform_type"),
		"chain":         infoStr("chain"),
		"sort_by":       infoStr("sort_by"),
		"limit":         infoInt("limit"),
	}),
	"info_platformmetrics_get_defi_overview": infoObj(map[string]interface{}{
		"category": infoStr("category"),
	}),
	"info_platformmetrics_get_stablecoin_info": infoObj(map[string]interface{}{
		"symbol": infoStr("symbol"),
		"chain":  infoStr("chain"),
		"limit":  infoInt("limit"),
	}),
	"info_platformmetrics_get_bridge_metrics": infoObj(map[string]interface{}{
		"bridge_name": infoStr("bridge_name"),
		"chain":       infoStr("chain"),
		"sort_by":     infoStr("sort_by"),
		"limit":       infoInt("limit"),
	}),
	"info_platformmetrics_get_yield_pools": infoObj(map[string]interface{}{
		"project":     infoStr("project"),
		"chain":       infoStr("chain"),
		"symbol":      infoStr("symbol"),
		"pool_type":   infoStr("pool_type"),
		"sort_by":     infoStr("sort_by"),
		"limit":       infoInt("limit"),
		"min_tvl_usd": infoNum("min_tvl_usd"),
	}),
	"info_platformmetrics_get_platform_history": infoObj(map[string]interface{}{
		"platform_name": infoStr("platform_name"),
		"metrics":       infoArrStr("metrics"),
		"start_date":    infoStr("start_date"),
		"end_date":      infoStr("end_date"),
	}, "platform_name"),
	"info_platformmetrics_get_exchange_reserves": infoObj(map[string]interface{}{
		"exchange": infoStr("exchange"),
		"asset":    infoStr("asset"),
		"period":   infoStr("period"),
	}),
	"info_platformmetrics_get_liquidation_heatmap": infoObj(map[string]interface{}{
		"symbol":   infoStr("symbol"),
		"exchange": infoStr("exchange"),
		"range":    infoStr("range"),
	}, "symbol"),
	"info_macro_get_macro_indicator": infoObj(map[string]interface{}{
		"mode":         infoStr("mode"),
		"indicator":    infoStr("indicator"),
		"country":      infoStr("country"),
		"country_code": infoStr("country_code"),
		"start_time":   infoStr("start_time"),
		"end_time":     infoStr("end_time"),
		"start_date":   infoStr("start_date"),
		"end_date":     infoStr("end_date"),
		"size":         infoInt("size"),
	}, "indicator"),
	"info_macro_get_economic_calendar": infoObj(map[string]interface{}{
		"start_date": infoStr("start_date"),
		"end_date":   infoStr("end_date"),
		"event_type": infoStr("event_type"),
		"importance": infoStr("importance"),
		"size":       infoInt("size"),
	}),
	"info_macro_get_macro_summary": infoObj(map[string]interface{}{}),
	"info_marketdetail_get_orderbook": infoObj(map[string]interface{}{
		"symbol":      infoStr("symbol"),
		"market_type": infoStr("market_type"),
		"depth":       infoInt("depth"),
		"settle":      infoStr("settle"),
		"extra":       infoObjAny("extra"),
	}, "symbol"),
	"info_marketdetail_get_recent_trades": infoObj(map[string]interface{}{
		"symbol":      infoStr("symbol"),
		"market_type": infoStr("market_type"),
		"limit":       infoInt("limit"),
		"settle":      infoStr("settle"),
		"extra":       infoObjAny("extra"),
	}, "symbol"),
	"info_marketdetail_get_kline": infoObj(map[string]interface{}{
		"symbol":      infoStr("symbol"),
		"market_type": infoStr("market_type"),
		"timeframe":   infoStr("timeframe"),
		"start_time":  infoInt("start_time"),
		"end_time":    infoInt("end_time"),
		"limit":       infoInt("limit"),
		"settle":      infoStr("settle"),
		"extra":       infoObjAny("extra"),
	}, "symbol", "timeframe"),
	"info_onchain_get_smart_money": infoObj(map[string]interface{}{
		"query":   infoStr("query"),
		"symbol":  infoStr("symbol"),
		"limit":   infoInt("limit"),
		"address": infoStr("address"),
	}),
	"info_onchain_get_entity_profile": infoObj(map[string]interface{}{
		"query":   infoStr("query"),
		"symbol":  infoStr("symbol"),
		"limit":   infoInt("limit"),
		"address": infoStr("address"),
	}),
	"info_onchain_trace_fund_flow": infoObj(map[string]interface{}{
		"query":   infoStr("query"),
		"symbol":  infoStr("symbol"),
		"limit":   infoInt("limit"),
		"address": infoStr("address"),
	}),
	"info_compliance_check_address_risk": infoObj(map[string]interface{}{
		"query":   infoStr("query"),
		"symbol":  infoStr("symbol"),
		"limit":   infoInt("limit"),
		"address": infoStr("address"),
	}),
	"info_compliance_search_regulatory_updates": infoObj(map[string]interface{}{
		"query":   infoStr("query"),
		"symbol":  infoStr("symbol"),
		"limit":   infoInt("limit"),
		"address": infoStr("address"),
	}),
}

func infoStr(desc string) map[string]interface{} {
	return map[string]interface{}{"type": "string", "description": desc}
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
	for k, v := range InfoBaselineInputSchemas {
		infoBaselineFrozen[k] = deepCloneSchemaMap(v)
	}
}
