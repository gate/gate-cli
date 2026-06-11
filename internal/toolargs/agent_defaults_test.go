//go:build agent

package toolargs

import "testing"

func TestApplyAgentDefaults_SearchNews(t *testing.T) {
	t.Setenv("GATE_CLI_AGENT", "1")
	got := NormalizeForTool("news_feed_search_news", map[string]interface{}{
		"coin": "BTC",
	})
	if got["time_range"] != "24h" {
		t.Fatalf("time_range=%v", got["time_range"])
	}
	if got["limit"] != 20 {
		t.Fatalf("limit=%v", got["limit"])
	}
}

func TestApplyAgentDefaults_KlineSize(t *testing.T) {
	t.Setenv("GATE_CLI_AGENT", "1")
	got := NormalizeForTool("info_markettrend_get_kline", map[string]interface{}{
		"symbol":    "BTC",
		"timeframe": "1h",
	})
	if got["size"] != 200 {
		t.Fatalf("size=%v", got["size"])
	}
}

func TestApplyAgentDefaultsDisabled(t *testing.T) {
	t.Setenv("GATE_CLI_AGENT", "")
	got := NormalizeForTool("news_feed_search_news", map[string]interface{}{"coin": "BTC"})
	if _, ok := got["time_range"]; ok {
		t.Fatal("should not inject time_range without agent mode")
	}
}
