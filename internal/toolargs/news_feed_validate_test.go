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
