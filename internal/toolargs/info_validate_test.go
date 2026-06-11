package toolargs

import "testing"

func TestValidateInfoIndicatorHistory(t *testing.T) {
	t.Parallel()
	if err := ValidateForTool("info_markettrend_get_indicator_history", map[string]interface{}{}); err == nil {
		t.Fatal("expected error")
	}
	args := map[string]interface{}{
		"symbol": "BTC", "timeframe": "1h", "indicators": []interface{}{"rsi"},
	}
	if ValidateForTool("info_markettrend_get_indicator_history", args) != nil {
		t.Fatal("expected nil")
	}
}

func TestValidateInfoMarketdetailOrderbook(t *testing.T) {
	t.Parallel()
	if ValidateForTool("info_marketdetail_get_orderbook", map[string]interface{}{"symbol": "BTC_USDT"}) != nil {
		t.Fatal("expected nil")
	}
	if err := ValidateForTool("info_marketdetail_get_orderbook", map[string]interface{}{"depth": 200}); err == nil {
		t.Fatal("expected depth/symbol errors")
	}
}

func TestValidateInfoSearchCoinsRequiresFilter(t *testing.T) {
	t.Parallel()
	if err := ValidateForTool("info_coin_search_coins", map[string]interface{}{}); err == nil {
		t.Fatal("expected filter error")
	}
	if ValidateForTool("info_coin_search_coins", map[string]interface{}{"category": "defi"}) != nil {
		t.Fatal("expected nil")
	}
}
