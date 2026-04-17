package toolargs

import "testing"

func TestNormalizeForTool_AliasToCanonical(t *testing.T) {
	got := NormalizeForTool("info_coin_get_coin_info", map[string]interface{}{
		"symbol": "BTC",
	})
	if got["query"] != "BTC" {
		t.Fatalf("expected query=BTC, got %#v", got["query"])
	}
	if _, ok := got["symbol"]; ok {
		t.Fatal("expected symbol alias removed")
	}
}

func TestNormalizeForTool_KeepCanonicalWhenBothProvided(t *testing.T) {
	got := NormalizeForTool("info_coin_get_coin_info", map[string]interface{}{
		"symbol": "BTC",
		"query":  "ETH",
	})
	if got["query"] != "ETH" {
		t.Fatalf("expected canonical query to win, got %#v", got["query"])
	}
}

func TestNormalizeForTool_IntervalAliasToTimeframe(t *testing.T) {
	got := NormalizeForTool("info_marketdetail_get_kline", map[string]interface{}{
		"interval": "1h",
	})
	if got["timeframe"] != "1h" {
		t.Fatalf("expected timeframe=1h, got %#v", got["timeframe"])
	}
	if _, ok := got["interval"]; ok {
		t.Fatal("expected interval alias removed")
	}
}

func TestNormalizeForTool_IndicatorAliasWrapsSlice(t *testing.T) {
	got := NormalizeForTool("info_markettrend_get_indicator_history", map[string]interface{}{
		"indicator": "rsi",
	})
	val, ok := got["indicators"].([]string)
	if !ok {
		t.Fatalf("expected []string indicators, got %#v", got["indicators"])
	}
	if len(val) != 1 || val[0] != "rsi" {
		t.Fatalf("unexpected indicators %#v", val)
	}
	if _, ok := got["indicator"]; ok {
		t.Fatal("expected indicator alias removed")
	}
}

func TestNormalizeForTool_PlatformAliasToPlatformName(t *testing.T) {
	got := NormalizeForTool("info_platformmetrics_get_platform_info", map[string]interface{}{
		"platform": "uniswap",
	})
	if got["platform_name"] != "uniswap" {
		t.Fatalf("expected platform_name=uniswap, got %#v", got["platform_name"])
	}
}
