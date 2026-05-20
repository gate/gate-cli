package toolargs

import (
	"encoding/json"
	"testing"
)

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
