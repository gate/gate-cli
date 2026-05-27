package toolargs

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestValidateForTool_StablecoinSectionsRequireFullScope(t *testing.T) {
	t.Parallel()
	err := ValidateForTool("info_platformmetrics_get_stablecoin_info", map[string]interface{}{
		"scope":    "basic",
		"sections": []string{"issuance_flow"},
	})
	if err == nil || !strings.Contains(err.Error(), "scope=full") {
		t.Fatalf("expected sections_requires_full_scope style error, got %v", err)
	}
	if ValidateForTool("info_platformmetrics_get_stablecoin_info", map[string]interface{}{
		"scope":    "full",
		"sections": []string{"issuance_flow"},
	}) != nil {
		t.Fatal("expected nil for full + issuance_flow")
	}
	if ValidateForTool("info_platformmetrics_get_stablecoin_info", map[string]interface{}{
		"scope":    "full",
		"sections": []string{"usage_structure"},
	}) != nil {
		t.Fatal("expected nil for full + usage_structure")
	}
}

func TestValidateForTool_StablecoinRejectsUnknownSection(t *testing.T) {
	t.Parallel()
	if ValidateForTool("info_platformmetrics_get_stablecoin_info", map[string]interface{}{
		"scope":    "full",
		"sections": []string{"holders"},
	}) == nil {
		t.Fatal("expected error for unknown section")
	}
}

func TestValidateForTool_StablecoinSectionsStringForm(t *testing.T) {
	t.Parallel()
	if ValidateForTool("info_platformmetrics_get_stablecoin_info", map[string]interface{}{
		"scope":    "full",
		"sections": "issuance_flow,usage_structure",
	}) != nil {
		t.Fatal("expected nil for comma-separated string sections")
	}
}

func TestValidateForTool_StablecoinDatesRequireExtensionSection(t *testing.T) {
	t.Parallel()
	if ValidateForTool("info_platformmetrics_get_stablecoin_info", map[string]interface{}{
		"scope":      "full",
		"start_date": "2026-04-01",
	}) == nil {
		t.Fatal("expected error when start_date without sections")
	}
	if ValidateForTool("info_platformmetrics_get_stablecoin_info", map[string]interface{}{
		"scope":      "basic",
		"sections":   []string{"issuance_flow"},
		"start_date": "2026-04-01",
	}) == nil {
		t.Fatal("expected error when start_date with basic scope")
	}
	if ValidateForTool("info_platformmetrics_get_stablecoin_info", map[string]interface{}{
		"scope":      "full",
		"sections":   []string{"issuance_flow"},
		"start_date": "2026-04-01",
		"end_date":   "2026-05-01",
	}) != nil {
		t.Fatal("expected nil for full + issuance_flow + dates")
	}
	if ValidateForTool("info_platformmetrics_get_stablecoin_info", map[string]interface{}{
		"scope":      "full",
		"sections":   []string{"usage_structure"},
		"start_date": "2026-04-01",
	}) != nil {
		t.Fatal("expected nil for full + usage_structure + dates")
	}
}

func TestValidateForTool_StablecoinSymbolWhitelistWithExtensionSections(t *testing.T) {
	t.Parallel()
	if ValidateForTool("info_platformmetrics_get_stablecoin_info", map[string]interface{}{
		"scope":    "full",
		"sections": []string{"issuance_flow"},
		"symbol":   "DAI",
	}) == nil {
		t.Fatal("expected error for non-USDT/USDC symbol with issuance_flow")
	}
	if ValidateForTool("info_platformmetrics_get_stablecoin_info", map[string]interface{}{
		"scope":    "full",
		"sections": []string{"issuance_flow"},
		"symbol":   "usdt",
	}) != nil {
		t.Fatal("expected nil for usdt with issuance_flow")
	}
	if ValidateForTool("info_platformmetrics_get_stablecoin_info", map[string]interface{}{
		"scope":    "full",
		"sections": []string{"usage_structure"},
		"symbol":   "DAI",
	}) != nil {
		t.Fatal("expected nil for DAI with usage_structure")
	}
	if ValidateForTool("info_platformmetrics_get_stablecoin_info", map[string]interface{}{
		"scope":    "full",
		"sections": []string{"usage_structure"},
		"symbol":   "EUR",
	}) == nil {
		t.Fatal("expected error for unsupported usage_structure symbol")
	}
	if ValidateForTool("info_platformmetrics_get_stablecoin_info", map[string]interface{}{
		"scope":    "full",
		"sections": []string{"issuance_flow", "usage_structure"},
		"symbol":   "DAI",
	}) == nil {
		t.Fatal("expected issuance_flow whitelist to apply when both sections are requested")
	}
}

func TestValidateForTool_StablecoinRejectsInvalidScopeAndLimit(t *testing.T) {
	t.Parallel()
	if ValidateForTool("info_platformmetrics_get_stablecoin_info", map[string]interface{}{
		"scope": "detailed",
	}) == nil {
		t.Fatal("expected error for invalid scope")
	}
	if ValidateForTool("info_platformmetrics_get_stablecoin_info", map[string]interface{}{
		"limit": 500,
	}) == nil {
		t.Fatal("expected error when limit > 400")
	}
	if ValidateForTool("info_platformmetrics_get_stablecoin_info", map[string]interface{}{
		"limit": 0,
	}) != nil {
		t.Fatal("expected nil when limit<=0 (server applies default 10)")
	}
	if ValidateForTool("info_platformmetrics_get_stablecoin_info", map[string]interface{}{}) != nil {
		t.Fatal("expected nil for empty args (server defaults)")
	}
}

func TestValidateForTool_StablecoinSectionsJSONString(t *testing.T) {
	t.Parallel()
	if ValidateForTool("info_platformmetrics_get_stablecoin_info", map[string]interface{}{
		"scope":    "full",
		"sections": `["issuance_flow","usage_structure"]`,
	}) != nil {
		t.Fatal("expected nil for JSON-array string sections")
	}
}

func TestValidateForTool_StablecoinExtensionChainWhitelist(t *testing.T) {
	t.Parallel()
	if ValidateForTool("info_platformmetrics_get_stablecoin_info", map[string]interface{}{
		"scope":    "full",
		"sections": []string{"usage_structure"},
		"chain":    "eth",
	}) != nil {
		t.Fatal("expected nil for extension chain alias")
	}
	if ValidateForTool("info_platformmetrics_get_stablecoin_info", map[string]interface{}{
		"scope":    "full",
		"sections": []string{"usage_structure"},
		"chain":    "moonbeam",
	}) == nil {
		t.Fatal("expected error for invalid extension chain")
	}
	if ValidateForTool("info_platformmetrics_get_stablecoin_info", map[string]interface{}{
		"scope": "basic",
		"chain": "moonbeam",
	}) != nil {
		t.Fatal("expected basic stablecoin chain filtering to defer to server")
	}
}

func TestValidateForTool_InstitutionalMetricsEnumsAndBounds(t *testing.T) {
	t.Parallel()
	tool := "info_marketsnapshot_get_institutional_metrics"
	if ValidateForTool(tool, map[string]interface{}{
		"asset":      "eth",
		"channel":    "CME",
		"start_date": "2026-04-01",
		"end_date":   "2026-05-01",
		"limit":      30,
	}) != nil {
		t.Fatal("expected nil for valid institutional metrics arguments")
	}
	if ValidateForTool(tool, map[string]interface{}{"asset": "SOL"}) == nil {
		t.Fatal("expected error for invalid asset")
	}
	if ValidateForTool(tool, map[string]interface{}{"channel": "dex"}) == nil {
		t.Fatal("expected error for invalid channel")
	}
	if ValidateForTool(tool, map[string]interface{}{"limit": 0}) == nil {
		t.Fatal("expected error for limit below range")
	}
	if ValidateForTool(tool, map[string]interface{}{"limit": 367}) == nil {
		t.Fatal("expected error for limit above range")
	}
	if ValidateForTool(tool, map[string]interface{}{"limit": 30.5}) == nil {
		t.Fatal("expected error for fractional limit")
	}
	if ValidateForTool(tool, map[string]interface{}{"limit": "30"}) == nil {
		t.Fatal("expected error for string limit")
	}
}

func TestValidateForTool_InstitutionalMetricsDates(t *testing.T) {
	t.Parallel()
	tool := "info_marketsnapshot_get_institutional_metrics"
	if ValidateForTool(tool, map[string]interface{}{"start_date": "2026/04/01"}) == nil {
		t.Fatal("expected error for invalid date format")
	}
	if ValidateForTool(tool, map[string]interface{}{
		"start_date": "2026-05-02",
		"end_date":   "2026-05-01",
	}) == nil {
		t.Fatal("expected error for start_date after end_date")
	}
}

func TestValidateForTool_ExchangeReservesIncludeHistoryRequiresFull(t *testing.T) {
	t.Parallel()
	if ValidateForTool("info_platformmetrics_get_exchange_reserves", map[string]interface{}{
		"scope":           "basic",
		"include_history": true,
	}) == nil {
		t.Fatal("expected error when include_history without full scope")
	}
	if ValidateForTool("info_platformmetrics_get_exchange_reserves", map[string]interface{}{
		"scope":           "full",
		"include_history": true,
	}) != nil {
		t.Fatal("expected nil for full + include_history")
	}
}

func TestValidateForTool_ExchangeReservesHistoryWindowRules(t *testing.T) {
	t.Parallel()
	if ValidateForTool("info_platformmetrics_get_exchange_reserves", map[string]interface{}{
		"history_window": "year",
	}) == nil {
		t.Fatal("expected error for history_window without include_history")
	}
	if ValidateForTool("info_platformmetrics_get_exchange_reserves", map[string]interface{}{
		"scope":           "full",
		"include_history": true,
		"history_window":  "quarter",
	}) != nil {
		t.Fatal("expected nil for quarter with include_history")
	}
}

func TestValidateForTool_ExchangeReservesAssetEnum(t *testing.T) {
	t.Parallel()
	if ValidateForTool("info_platformmetrics_get_exchange_reserves", map[string]interface{}{
		"asset": "SOL",
	}) == nil {
		t.Fatal("expected error for invalid asset")
	}
}

func TestValidateForTool_PlatformInfoOIRequiresFullScope(t *testing.T) {
	t.Parallel()
	if ValidateForTool("info_platformmetrics_get_platform_info", map[string]interface{}{
		"platform_name":            "binance",
		"scope":                    "basic",
		"include_oi_symbol_detail": true,
	}) == nil {
		t.Fatal("expected error when include_oi_symbol_detail without full scope")
	}
	if ValidateForTool("info_platformmetrics_get_platform_info", map[string]interface{}{
		"platform_name":            "binance",
		"scope":                    "full",
		"include_oi_symbol_detail": true,
		"oi_symbol_limit":          150,
	}) == nil {
		t.Fatal("expected error when oi_symbol_limit > 100")
	}
}

func TestValidateForTool_TokenSecurityRequiresTokenXORAddress(t *testing.T) {
	t.Parallel()
	if ValidateForTool("info_compliance_check_token_security", map[string]interface{}{
		"chain": "eth",
	}) == nil {
		t.Fatal("expected error when token and address are both empty")
	}
	if ValidateForTool("info_compliance_check_token_security", map[string]interface{}{
		"chain":   "eth",
		"token":   "USDT",
		"address": "0xd8dA6BF26964aF9D7eEd9e03E53415dA322193D",
	}) == nil {
		t.Fatal("expected error when token and address are both set")
	}
	if ValidateForTool("info_compliance_check_token_security", map[string]interface{}{
		"chain": "eth",
		"token": "USDT",
	}) != nil {
		t.Fatal("expected nil when token is set")
	}
	if ValidateForTool("info_compliance_check_token_security", map[string]interface{}{
		"chain":   "eth",
		"address": json.Number("12345"),
	}) != nil {
		t.Fatal("expected nil when address is a json.Number from decoder UseNumber")
	}
}

func TestValidateForTool_PlatformHistoryRequiresOneIdentifier(t *testing.T) {
	t.Parallel()
	err := ValidateForTool("info_platformmetrics_get_platform_history", map[string]interface{}{
		"platform_name": "  ",
		"exchange_slug": "",
	})
	if err == nil {
		t.Fatal("expected error when both identifiers empty")
	}

	if ValidateForTool("info_platformmetrics_get_platform_history", map[string]interface{}{
		"platform_name": "uniswap",
	}) != nil {
		t.Fatal("expected nil when platform_name set")
	}
	if ValidateForTool("info_platformmetrics_get_platform_history", map[string]interface{}{
		"exchange_slug": "binance",
	}) != nil {
		t.Fatal("expected nil when exchange_slug set")
	}
	if ValidateForTool("info_platformmetrics_get_platform_history", map[string]interface{}{
		"exchange_slug": json.Number("ok"),
	}) != nil {
		t.Fatal("expected nil when exchange_slug is json.Number from decoder UseNumber")
	}
	if ValidateForTool("info_coin_get_coin_info", map[string]interface{}{}) != nil {
		t.Fatal("expected no static validation for unrelated tool")
	}
}

func TestValidateForTool_SearchUGCRequiresQueryOrCoin(t *testing.T) {
	t.Parallel()
	if err := ValidateForTool("news_feed_search_ugc", map[string]interface{}{}); err == nil {
		t.Fatal("expected error when query and coin are both empty")
	}
	if ValidateForTool("news_feed_search_ugc", map[string]interface{}{"query": "BTC"}) != nil {
		t.Fatal("expected nil when query set")
	}
	if ValidateForTool("news_feed_search_ugc", map[string]interface{}{"coin": "BTC"}) != nil {
		t.Fatal("expected nil when coin set")
	}
}

func TestValidateForTool_SearchEventsRequiresFilter(t *testing.T) {
	t.Parallel()
	if err := ValidateForTool("news_prediction_search_events", map[string]interface{}{}); err == nil {
		t.Fatal("expected error when query, coin, and category are all empty")
	}
	if ValidateForTool("news_prediction_search_events", map[string]interface{}{"category": "crypto_price"}) != nil {
		t.Fatal("expected nil when category set")
	}
	if ValidateForTool("news_prediction_search_events", map[string]interface{}{"coin": "BTC"}) != nil {
		t.Fatal("expected nil when coin set")
	}
}

func TestValidateForTool_WebSearchRequiresQuery(t *testing.T) {
	t.Parallel()
	if err := ValidateForTool("news_feed_web_search", map[string]interface{}{}); err == nil {
		t.Fatal("expected error when query empty")
	}
	if ValidateForTool("news_feed_web_search", map[string]interface{}{"query": "  "}) == nil {
		t.Fatal("expected error when query whitespace only")
	}
	if ValidateForTool("news_feed_web_search", map[string]interface{}{"query": "BTC ETF"}) != nil {
		t.Fatal("expected nil when query set")
	}
}

func TestValidateForTool_EventDetailRequiresEventID(t *testing.T) {
	t.Parallel()
	if err := ValidateForTool("news_events_get_event_detail", map[string]interface{}{}); err == nil {
		t.Fatal("expected error when event_id empty")
	}
	if ValidateForTool("news_events_get_event_detail", map[string]interface{}{"event_id": "evt:1"}) != nil {
		t.Fatal("expected nil when event_id set")
	}
}

func TestValidateForTool_ExplainMarketMoveRequiresQueryAndCoin(t *testing.T) {
	t.Parallel()
	if err := ValidateForTool("news_events_explain_market_move", map[string]interface{}{}); err == nil {
		t.Fatal("expected error when query and coin empty")
	}
	if ValidateForTool("news_events_explain_market_move", map[string]interface{}{"query": "why"}) == nil {
		t.Fatal("expected error when coin missing")
	}
	if ValidateForTool("news_events_explain_market_move", map[string]interface{}{
		"query": "why",
		"coin":  "BTC",
	}) != nil {
		t.Fatal("expected nil when query and coin set")
	}
}

func TestValidateForTool_OrderbookRequiresVenueAndMarketID(t *testing.T) {
	t.Parallel()
	if err := ValidateForTool("news_prediction_get_market_orderbook", map[string]interface{}{}); err == nil {
		t.Fatal("expected error when venue and market_id empty")
	}
	if ValidateForTool("news_prediction_get_market_orderbook", map[string]interface{}{"venue": "polymarket"}) == nil {
		t.Fatal("expected error when market_id missing")
	}
	if ValidateForTool("news_prediction_get_market_orderbook", map[string]interface{}{
		"venue":     "polymarket",
		"market_id": "12345",
	}) != nil {
		t.Fatal("expected nil when venue and market_id set")
	}
}

func TestValidateForTool_EventSignalRequiresEventRef(t *testing.T) {
	t.Parallel()
	if err := ValidateForTool("news_prediction_get_event_signal", map[string]interface{}{}); err == nil {
		t.Fatal("expected error when event_ref empty")
	}
	if ValidateForTool("news_prediction_get_event_signal", map[string]interface{}{
		"event_ref": "polymarket:107711",
	}) != nil {
		t.Fatal("expected nil when event_ref set")
	}
}

func TestValidateForTool_OrderbookRejectsInvalidVenue(t *testing.T) {
	t.Parallel()
	err := ValidateForTool("news_prediction_get_market_orderbook", map[string]interface{}{
		"venue": "opinion", "market_id": "1",
	})
	if err == nil {
		t.Fatal("expected error for invalid venue")
	}
}

func TestValidateForTool_OrderbookRejectsDepthOutOfRange(t *testing.T) {
	t.Parallel()
	if ValidateForTool("news_prediction_get_market_orderbook", map[string]interface{}{
		"venue": "polymarket", "market_id": "1", "depth": 25,
	}) == nil {
		t.Fatal("expected error when depth > 20")
	}
}

func TestValidateForTool_SearchEventsRejectsInvalidCategory(t *testing.T) {
	t.Parallel()
	if ValidateForTool("news_prediction_search_events", map[string]interface{}{
		"category": "not_a_real_category",
	}) == nil {
		t.Fatal("expected error for invalid category")
	}
}

func TestValidateForTool_SearchEventsRejectsInvalidStatus(t *testing.T) {
	t.Parallel()
	if ValidateForTool("news_prediction_search_events", map[string]interface{}{
		"coin": "BTC", "status": "open",
	}) == nil {
		t.Fatal("expected error for invalid status")
	}
}

func TestValidateForTool_SearchEventsRejectsInvalidLimit(t *testing.T) {
	t.Parallel()
	if ValidateForTool("news_prediction_search_events", map[string]interface{}{
		"coin": "BTC", "limit": 200,
	}) == nil {
		t.Fatal("expected error when limit > 100")
	}
}

func TestValidateForTool_EventSignalRejectsBadEventRef(t *testing.T) {
	t.Parallel()
	if ValidateForTool("news_prediction_get_event_signal", map[string]interface{}{
		"event_ref": "no-colon",
	}) == nil {
		t.Fatal("expected error when event_ref has no colon")
	}
}

func TestValidateForTool_EventSignalRejectsVenueMismatch(t *testing.T) {
	t.Parallel()
	if ValidateForTool("news_prediction_get_event_signal", map[string]interface{}{
		"event_ref": "polymarket:1",
		"venue":     []string{"predict_fun"},
	}) == nil {
		t.Fatal("expected error when venue filter mismatches event_ref")
	}
}

func TestValidateForTool_EventSignalAcceptsCaseInsensitiveWindow(t *testing.T) {
	t.Parallel()
	if ValidateForTool("news_prediction_get_event_signal", map[string]interface{}{
		"event_ref": "polymarket:1",
		"window":    "7D",
	}) != nil {
		t.Fatal("expected nil for case-insensitive window")
	}
}
