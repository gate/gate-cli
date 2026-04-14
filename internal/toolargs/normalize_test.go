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
