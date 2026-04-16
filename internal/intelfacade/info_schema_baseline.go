package intelfacade

// InfoBaselineInputSchemas are static JSON-Schema-shaped objects: the stable source for CLI flat flags.
// MCP tools/list may add additional flags (non-colliding) on top; --params / --args-json remain JSON fallback.
// Align with specs/qc/info-news-command-checklist.md §4.2 where applicable.
var InfoBaselineInputSchemas = map[string]map[string]interface{}{
	"info_coin_get_coin_info": infoObj(map[string]interface{}{
		"query":       infoStr("Coin or text query"),
		"query_type":  infoStr("e.g. auto"),
		"size":        infoInt("Max rows"),
		"fields":      infoArrStr("Restrict fields returned"),
	}, "query"),
	"info_coin_search_coins": infoObj(map[string]interface{}{
		"query": infoStr("Search query"),
		"limit": infoInt("Result limit"),
	}, "query"),
	"info_coin_get_coin_rankings": infoObj(map[string]interface{}{
		"sort_by": infoStr("Sort key e.g. market_cap"),
		"limit":   infoInt("Result limit"),
	}, "sort_by"),
	"info_marketsnapshot_get_market_snapshot": infoObj(map[string]interface{}{
		"symbol": infoStr("Pair symbol e.g. BTC_USDT"),
	}, "symbol"),
	"info_marketsnapshot_batch_market_snapshot": infoObj(map[string]interface{}{
		"symbols": infoStr("Comma-separated symbols"),
	}, "symbols"),
	"info_marketsnapshot_get_market_overview": infoObj(map[string]interface{}{}),
	"info_markettrend_get_kline": infoObj(map[string]interface{}{
		"symbol":   infoStr("Pair symbol"),
		"interval": infoStr("Candle interval e.g. 1h"),
		"limit":    infoInt("Candle count"),
	}, "symbol"),
	"info_markettrend_get_indicator_history": infoObj(map[string]interface{}{
		"symbol":    infoStr("Pair symbol"),
		"indicator": infoStr("Indicator id e.g. rsi"),
		"limit":     infoInt("Point count"),
	}, "symbol"),
	"info_markettrend_get_technical_analysis": infoObj(map[string]interface{}{
		"symbol": infoStr("Pair symbol"),
	}, "symbol"),
	"info_onchain_get_address_info": infoObj(map[string]interface{}{
		"address": infoStr("EVM address"),
		"chain":   infoStr("Chain id e.g. eth"),
	}, "address"),
	"info_onchain_get_address_transactions": infoObj(map[string]interface{}{
		"address": infoStr("EVM address"),
		"chain":   infoStr("Chain id"),
		"limit":   infoInt("Tx count"),
	}, "address"),
	"info_onchain_get_transaction": infoObj(map[string]interface{}{
		"tx_hash": infoStr("Transaction hash"),
		"chain":   infoStr("Chain id"),
	}, "tx_hash"),
	"info_onchain_get_token_onchain": infoObj(map[string]interface{}{
		"symbol": infoStr("Token symbol"),
		"chain":  infoStr("Chain id"),
	}, "symbol"),
	"info_platformmetrics_get_platform_info": infoObj(map[string]interface{}{
		"platform": infoStr("Platform id e.g. uniswap"),
	}, "platform"),
	"info_platformmetrics_search_platforms": infoObj(map[string]interface{}{
		"query": infoStr("Search query"),
		"limit": infoInt("Result limit"),
	}, "query"),
	"info_platformmetrics_get_defi_overview": infoObj(map[string]interface{}{
		"platform": infoStr("Platform id"),
	}, "platform"),
	"info_platformmetrics_get_stablecoin_info": infoObj(map[string]interface{}{
		"symbol": infoStr("Stable symbol e.g. USDT"),
	}, "symbol"),
	"info_platformmetrics_get_bridge_metrics": infoObj(map[string]interface{}{
		"bridge": infoStr("Bridge id e.g. wormhole"),
	}, "bridge"),
	"info_platformmetrics_get_yield_pools": infoObj(map[string]interface{}{
		"platform": infoStr("Platform id"),
		"asset":    infoStr("Asset symbol"),
		"limit":    infoInt("Pool count"),
	}, "platform"),
	"info_platformmetrics_get_platform_history": infoObj(map[string]interface{}{
		"platform": infoStr("Platform id"),
	}, "platform"),
	"info_platformmetrics_get_exchange_reserves": infoObj(map[string]interface{}{
		"exchange": infoStr("Exchange id e.g. binance"),
	}, "exchange"),
	"info_platformmetrics_get_liquidation_heatmap": infoObj(map[string]interface{}{
		"symbol": infoStr("Contract or pair symbol"),
	}, "symbol"),
	"info_marketdetail_get_orderbook": infoObj(map[string]interface{}{
		"symbol": infoStr("Pair symbol"),
	}, "symbol"),
	"info_marketdetail_get_recent_trades": infoObj(map[string]interface{}{
		"symbol": infoStr("Pair symbol"),
		"limit":  infoInt("Trade count"),
	}, "symbol"),
	"info_marketdetail_get_kline": infoObj(map[string]interface{}{
		"symbol":   infoStr("Pair symbol"),
		"interval": infoStr("Interval e.g. 1h"),
		"limit":    infoInt("Candle count"),
	}, "symbol"),
	"info_macro_get_macro_indicator": infoObj(map[string]interface{}{
		"indicator": infoStr("Macro indicator id e.g. cpi"),
	}, "indicator"),
	"info_macro_get_economic_calendar": infoObj(map[string]interface{}{
		"region": infoStr("Region code e.g. US"),
	}, "region"),
	"info_macro_get_macro_summary": infoObj(map[string]interface{}{
		"region": infoStr("Region code e.g. US"),
	}, "region"),
	"info_onchain_get_smart_money": infoObj(map[string]interface{}{
		"symbol": infoStr("Token symbol"),
	}, "symbol"),
	"info_onchain_get_entity_profile": infoObj(map[string]interface{}{
		"query": infoStr("Entity search text"),
	}, "query"),
	"info_onchain_trace_fund_flow": infoObj(map[string]interface{}{
		"address": infoStr("Seed address"),
		"chain":   infoStr("Chain id"),
		"depth":   infoInt("Hop depth"),
	}, "address"),
	"info_compliance_check_token_security": infoObj(map[string]interface{}{
		"symbol": infoStr("Token symbol"),
		"chain":  infoStr("Chain id"),
	}, "symbol"),
	"info_compliance_check_address_risk": infoObj(map[string]interface{}{
		"address": infoStr("Address to check"),
		"chain":   infoStr("Chain id"),
	}, "address"),
	"info_compliance_search_regulatory_updates": infoObj(map[string]interface{}{
		"query": infoStr("Search query"),
		"limit": infoInt("Result limit"),
	}, "query"),
}

func infoStr(desc string) map[string]interface{} {
	return map[string]interface{}{"type": "string", "description": desc}
}

func infoInt(desc string) map[string]interface{} {
	return map[string]interface{}{"type": "integer", "description": desc}
}

func infoArrStr(desc string) map[string]interface{} {
	return map[string]interface{}{
		"type":        "array",
		"description": desc,
		"items":       map[string]interface{}{"type": "string"},
	}
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

// InfoBaselineInputSchema returns a shallow copy of the baseline schema for toolName, or nil.
func InfoBaselineInputSchema(toolName string) map[string]interface{} {
	raw, ok := InfoBaselineInputSchemas[toolName]
	if !ok || len(raw) == 0 {
		return nil
	}
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
