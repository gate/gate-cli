package toolargs

import "testing"

func TestValidateForTool_SearchNewsLimit(t *testing.T) {
	t.Parallel()
	if err := ValidateForTool("news_feed_search_news", map[string]interface{}{
		"coin":  "BTC",
		"limit": 101,
	}); err == nil {
		t.Fatal("expected limit cap error")
	}
	if err := ValidateForTool("news_feed_search_news", map[string]interface{}{
		"coin":       "BTC",
		"time_range": "90d",
	}); err == nil {
		t.Fatal("expected time_range error")
	}
	if err := ValidateForTool("news_feed_search_news", map[string]interface{}{
		"coin": "BTC",
	}); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
}

func TestValidateForTool_SocialSentimentTimeRange(t *testing.T) {
	t.Parallel()
	if err := ValidateForTool("news_feed_get_social_sentiment", map[string]interface{}{
		"coin":       "BTC",
		"time_range": "30d",
	}); err == nil {
		t.Fatal("expected sentiment time_range to reject 30d")
	}
	if err := ValidateForTool("news_feed_get_social_sentiment", map[string]interface{}{
		"coin":       "BTC",
		"time_range": "7d",
	}); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
}

func TestValidateForTool_MentionBurst(t *testing.T) {
	t.Parallel()
	valid := map[string]interface{}{
		"coin": "BTC", "window": "24h", "platforms": "twitter,reddit",
	}
	if err := ValidateForTool("news_feed_get_mention_burst", valid); err != nil {
		t.Fatalf("valid mention burst args: %v", err)
	}
	for _, args := range []map[string]interface{}{
		{},
		{"coin": "BTC", "window": "12h"},
		{"coin": "BTC", "platforms": "all,twitter"},
		{"coin": "BTC", "platforms": "unknown"},
	} {
		if err := ValidateForTool("news_feed_get_mention_burst", args); err == nil {
			t.Fatalf("expected mention burst validation error for %#v", args)
		}
	}
}

func TestValidateForTool_HotTopics(t *testing.T) {
	t.Parallel()
	valid := map[string]interface{}{
		"coin": "ETH", "window": "4h", "limit": int64(3), "platforms": "all",
	}
	if err := ValidateForTool("news_feed_get_hot_topics", valid); err != nil {
		t.Fatalf("valid hot topics args: %v", err)
	}
	for _, args := range []map[string]interface{}{
		{},
		{"coin": "ETH", "window": "24h"},
		{"coin": "ETH", "limit": int64(1)},
		{"coin": "ETH", "limit": int64(5)},
		{"coin": "ETH", "platforms": "twitter,unknown"},
	} {
		if err := ValidateForTool("news_feed_get_hot_topics", args); err == nil {
			t.Fatalf("expected hot topics validation error for %#v", args)
		}
	}
}

func TestValidateForTool_ExplainMarketMoveTimeRange(t *testing.T) {
	t.Parallel()
	if err := ValidateForTool("news_events_explain_market_move", map[string]interface{}{
		"query":      "why pump",
		"coin":       "BTC",
		"time_range": "7d",
	}); err == nil {
		t.Fatal("expected invalid time_range")
	}
	if err := ValidateForTool("news_events_explain_market_move", map[string]interface{}{
		"query":      "why pump",
		"coin":       "BTC",
		"time_range": "2h",
	}); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
}

func TestValidateForTool_GetMarketMoveReport(t *testing.T) {
	t.Parallel()
	if err := ValidateForTool("news_events_get_market_move_report", map[string]interface{}{"symbol": "TAIKO", "report_id": "r1"}); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	for _, args := range []map[string]interface{}{
		{},
		{"symbol": "THIS_SYMBOL_IS_LONGER_THAN_TWENTY"},
		{"symbol": "TAIKO", "is_make_new": true},
	} {
		if err := ValidateForTool("news_events_get_market_move_report", args); err == nil {
			t.Fatalf("expected validation error for %#v", args)
		}
	}
}

func TestValidateForTool_ListMarketMoveReports(t *testing.T) {
	t.Parallel()
	valid := map[string]interface{}{
		"symbol": "ETH", "start_time": "2026-07-09 22:00:00", "end_time": "2026-07-10 03:15:00", "limit": int64(10),
	}
	if err := ValidateForTool("news_events_list_market_move_reports", valid); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	withDefaultLimit := map[string]interface{}{
		"symbol": "ETH", "start_time": "2026-07-10T06:00:00+08:00", "end_time": "2026-07-10T11:15:00+08:00", "limit": int64(0),
	}
	if err := ValidateForTool("news_events_list_market_move_reports", withDefaultLimit); err != nil {
		t.Fatalf("limit=0 and explicit timezone offsets must be valid: %v", err)
	}
	for _, args := range []map[string]interface{}{
		{"symbol": "ETH", "start_time": valid["start_time"]},
		{"symbol": "ETH", "start_time": "bad", "end_time": valid["end_time"]},
		{"symbol": "ETH", "start_time": valid["end_time"], "end_time": valid["start_time"]},
		{"symbol": "ETH", "start_time": valid["start_time"], "end_time": valid["end_time"], "limit": int64(-1)},
		{"symbol": "ETH", "start_time": valid["start_time"], "end_time": valid["end_time"], "limit": int64(101)},
		{"symbol": "ETH", "start_time": valid["start_time"], "end_time": valid["end_time"], "is_make_new": false},
	} {
		if err := ValidateForTool("news_events_list_market_move_reports", args); err == nil {
			t.Fatalf("expected validation error for %#v", args)
		}
	}
}
