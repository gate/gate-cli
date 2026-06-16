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
	if err := ValidateForTool("info_coin_get_coin_info", map[string]interface{}{}); err == nil {
		t.Fatal("expected error when query and symbol are both empty")
	}
	if ValidateForTool("info_coin_get_coin_info", map[string]interface{}{"symbol": "BTC"}) != nil {
		t.Fatal("expected nil when symbol set")
	}
	if err := ValidateForTool("info_marketsnapshot_get_market_snapshot", map[string]interface{}{}); err == nil {
		t.Fatal("expected error when symbol missing")
	}
	if err := ValidateForTool("news_feed_get_exchange_announcements", map[string]interface{}{}); err == nil {
		t.Fatal("expected error when no filter fields set")
	}
	if ValidateForTool("news_feed_get_exchange_announcements", map[string]interface{}{"coin": "BTC"}) != nil {
		t.Fatal("expected nil when coin set")
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

func TestValidateForTool_SearchXRejectsBothHandleLists(t *testing.T) {
	t.Parallel()
	err := ValidateForTool("news_feed_search_x", map[string]interface{}{
		"query":             "btc",
		"allowed_handles":   []string{"a"},
		"excluded_handles":  []string{"b"},
	})
	if err == nil || !strings.Contains(err.Error(), "allowed_handles") {
		t.Fatalf("expected handles conflict error, got %v", err)
	}
}

func TestValidateForTool_SearchXRejectsInvalidTimeRange(t *testing.T) {
	t.Parallel()
	if ValidateForTool("news_feed_search_x", map[string]interface{}{
		"query": "btc", "time_range": "14d",
	}) == nil {
		t.Fatal("expected error for 14d time_range")
	}
}

func TestValidateForTool_OrderbookRejectsUnsupportedParams(t *testing.T) {
	t.Parallel()
	base := map[string]interface{}{"venue": "polymarket", "market_id": "1"}
	cases := []map[string]interface{}{
		{"granularity": "1m"},
		{"start_time": "2026-01-01"},
		{"page_token": "x"},
		{"mode": "history"},
	}
	for _, extra := range cases {
		args := make(map[string]interface{}, len(base)+len(extra))
		for k, v := range base {
			args[k] = v
		}
		for k, v := range extra {
			args[k] = v
		}
		if ValidateForTool("news_prediction_get_market_orderbook", args) == nil {
			t.Fatalf("expected error for extra %#v", extra)
		}
	}
}

func TestValidateForTool_BatchMarketSnapshotSymbolsBounds(t *testing.T) {
	t.Parallel()
	if ValidateForTool("info_marketsnapshot_batch_market_snapshot", map[string]interface{}{}) == nil {
		t.Fatal("expected error when symbols missing")
	}
	syms := make([]string, 21)
	for i := range syms {
		syms[i] = "BTC_USDT"
	}
	if ValidateForTool("info_marketsnapshot_batch_market_snapshot", map[string]interface{}{
		"symbols": syms,
	}) == nil {
		t.Fatal("expected error when symbols > 20")
	}
	if ValidateForTool("info_marketsnapshot_batch_market_snapshot", map[string]interface{}{
		"symbols": []string{"BTC_USDT"},
	}) != nil {
		t.Fatal("expected nil for one symbol")
	}
}

func TestValidateForTool_LatestEventsTimeRangeRules(t *testing.T) {
	t.Parallel()
	if ValidateForTool("news_events_get_latest_events", map[string]interface{}{
		"time_range": "7d", "start_time": "2026-01-01",
	}) == nil {
		t.Fatal("expected error when time_range mixed with start_time")
	}
	if ValidateForTool("news_events_get_latest_events", map[string]interface{}{
		"limit": 101,
	}) == nil {
		t.Fatal("expected error when limit > 100")
	}
}

func TestValidateForTool_PredictionRankingDateAndLimit(t *testing.T) {
	t.Parallel()
	tool := "news_prediction_get_volume_delta_ranking"
	if ValidateForTool(tool, map[string]interface{}{"date_utc": "2026/04/01"}) == nil {
		t.Fatal("expected error for bad date_utc")
	}
	if ValidateForTool(tool, map[string]interface{}{"status": "open"}) == nil {
		t.Fatal("expected error for bad status")
	}
	if ValidateForTool(tool, map[string]interface{}{"venue": []string{"bad"}}) == nil {
		t.Fatal("expected error for bad venue")
	}
	if ValidateForTool(tool, map[string]interface{}{"limit": 0}) == nil {
		t.Fatal("expected error for limit < 1")
	}
}

func TestValidateForTool_SearchUGCEnumAndLimit(t *testing.T) {
	t.Parallel()
	if ValidateForTool("news_feed_search_ugc", map[string]interface{}{
		"query": "x", "platform": "twitter",
	}) == nil {
		t.Fatal("expected error for bad platform")
	}
	if ValidateForTool("news_feed_search_ugc", map[string]interface{}{
		"coin": "BTC", "limit": 51,
	}) == nil {
		t.Fatal("expected error when limit > 50")
	}
}

func TestValidateForTool_WebSearchLimitAndTimeRange(t *testing.T) {
	t.Parallel()
	if ValidateForTool("news_feed_web_search", map[string]interface{}{
		"query": "btc", "limit": 11,
	}) == nil {
		t.Fatal("expected error when limit > 10")
	}
	if ValidateForTool("news_feed_web_search", map[string]interface{}{
		"query": "btc", "time_range": "all",
	}) == nil {
		t.Fatal("expected error for invalid time_range")
	}
}

func TestValidateForTool_CoinRankingsMarketPulseHot(t *testing.T) {
	t.Parallel()
	tool := "info_coin_get_coin_rankings"
	if err := ValidateForTool(tool, map[string]interface{}{
		"ranking_type": "market_pulse_hot",
	}); err != nil {
		t.Fatalf("expected valid market_pulse_hot, got %v", err)
	}
	if ValidateForTool(tool, map[string]interface{}{
		"ranking_type": "not_a_board",
	}) == nil {
		t.Fatal("expected error for unsupported ranking_type")
	}
}

func TestValidateForTool_CoinRankingsCrossFieldRules(t *testing.T) {
	t.Parallel()
	tool := "info_coin_get_coin_rankings"
	if ValidateForTool(tool, map[string]interface{}{
		"ranking_type": "popular",
		"time_range":   "24h",
	}) == nil {
		t.Fatal("expected error when time_range set for non-movers ranking_type")
	}
	if err := ValidateForTool(tool, map[string]interface{}{
		"ranking_type": "top_gainers",
		"time_range":   "24h",
	}); err != nil {
		t.Fatalf("expected valid gainers+time_range, got %v", err)
	}
	if ValidateForTool(tool, map[string]interface{}{
		"ranking_type": "popular",
		"listing_query": "btc",
	}) == nil {
		t.Fatal("expected error when listing_query set for non-new_listing")
	}
}

func TestValidateForTool_EconomicCalendarOptionalDates(t *testing.T) {
	t.Parallel()
	tool := "info_macro_get_economic_calendar"
	if err := ValidateForTool(tool, map[string]interface{}{}); err != nil {
		t.Fatalf("expected zero-arg calendar call, got %v", err)
	}
	if ValidateForTool(tool, map[string]interface{}{
		"start_date": "2026-05-02",
		"end_date":   "2026-05-01",
	}) == nil {
		t.Fatal("expected error when start_date after end_date")
	}
	if err := ValidateForTool(tool, map[string]interface{}{
		"start_date": "2026-04-01",
	}); err != nil {
		t.Fatalf("expected valid start_date only, got %v", err)
	}
}

func TestValidateForTool_YieldPoolsScope(t *testing.T) {
	t.Parallel()
	tool := "info_platformmetrics_get_yield_pools"
	if err := ValidateForTool(tool, map[string]interface{}{
		"scope": "full",
	}); err != nil {
		t.Fatalf("expected valid scope=full, got %v", err)
	}
	if ValidateForTool(tool, map[string]interface{}{
		"scope": "detailed",
	}) == nil {
		t.Fatal("expected error for invalid scope")
	}
}

func TestValidateForTool_CexOrderbookDepthRequiresSymbol(t *testing.T) {
	t.Parallel()
	if ValidateForTool("info_platformmetrics_get_cex_orderbook_depth", map[string]interface{}{}) == nil {
		t.Fatal("expected error when symbol missing")
	}
	if ValidateForTool("info_platformmetrics_get_cex_orderbook_depth", map[string]interface{}{
		"symbol": "BTC_USDT", "market_type": "swap",
	}) == nil {
		t.Fatal("expected error for bad market_type")
	}
	if ValidateForTool("info_platformmetrics_get_cex_orderbook_depth", map[string]interface{}{
		"symbol": "BTC_USDT", "limit": 101,
	}) == nil {
		t.Fatal("expected error when limit > 100")
	}
}

func TestValidateForTool_ChainActivity(t *testing.T) {
	t.Parallel()
	tool := "info_platformmetrics_get_chain_activity"
	if ValidateForTool(tool, map[string]interface{}{}) == nil {
		t.Fatal("expected error when metric_group missing")
	}
	if ValidateForTool(tool, map[string]interface{}{
		"metric_group": "fees",
	}) == nil {
		t.Fatal("expected error for unsupported metric_group")
	}
	if ValidateForTool(tool, map[string]interface{}{
		"metric_group": "staking", "chain": "solana",
	}) == nil {
		t.Fatal("expected error for unsupported chain on staking")
	}
	if ValidateForTool(tool, map[string]interface{}{
		"metric_group": "staking", "lookback": "7d",
	}) == nil {
		t.Fatal("expected error for invalid lookback")
	}
	if ValidateForTool(tool, map[string]interface{}{
		"metric_group": "staking",
		"start_date":   "2026-05-02",
		"end_date":     "2026-05-01",
	}) == nil {
		t.Fatal("expected error when start_date after end_date")
	}
	if ValidateForTool(tool, map[string]interface{}{
		"metric_group": "staking",
		"start_date":   "2026/04/01",
	}) == nil {
		t.Fatal("expected error for invalid start_date format")
	}
	if err := ValidateForTool(tool, map[string]interface{}{
		"metric_group": "staking", "chain": "eth", "lookback": "90d",
	}); err != nil {
		t.Fatalf("expected valid args, got %v", err)
	}
	if err := ValidateForTool(tool, map[string]interface{}{
		"metric_group": "staking",
		"start_date":   "2026-04-01",
	}); err != nil {
		t.Fatalf("expected valid args with start_date only, got %v", err)
	}
}

func TestValidateForTool_SearchEventsPageToken(t *testing.T) {
	t.Parallel()
	tool := "news_prediction_search_events"
	if ValidateForTool(tool, map[string]interface{}{
		"coin": "BTC", "page_token": "not-valid-base64!!!",
	}) == nil {
		t.Fatal("expected error for invalid page_token")
	}
	tokenOK := "eyJzb3J0X2J5Ijoidm9sdW1lIn0=" // {"sort_by":"volume"}
	if ValidateForTool(tool, map[string]interface{}{
		"coin": "BTC", "page_token": tokenOK, "sort_by": "recently_listed",
	}) == nil {
		t.Fatal("expected error for sort_by mismatch with page_token")
	}
	if ValidateForTool(tool, map[string]interface{}{
		"coin": "BTC", "page_token": tokenOK, "sort_by": "volume",
	}) != nil {
		t.Fatal("expected nil when page_token sort_by matches request")
	}
}
