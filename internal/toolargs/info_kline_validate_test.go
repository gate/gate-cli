package toolargs

import "testing"

func TestValidateForTool_MarkettrendKlineRejectsOversize(t *testing.T) {
	t.Parallel()
	err := ValidateForTool("info_markettrend_get_kline", map[string]interface{}{
		"symbol":    "BTC",
		"timeframe": "1h",
		"size":      2000,
	})
	if err == nil {
		t.Fatal("expected error for size > 500")
	}
}

func TestValidateForTool_MarketdetailKlineRejectsOversize(t *testing.T) {
	t.Parallel()
	err := ValidateForTool("info_marketdetail_get_kline", map[string]interface{}{
		"symbol":    "BTC",
		"timeframe": "1h",
		"limit":     900,
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestValidateForTool_MarkettrendKlineAllowsDefaultWindow(t *testing.T) {
	t.Parallel()
	if err := ValidateForTool("info_markettrend_get_kline", map[string]interface{}{
		"symbol":    "BTC",
		"timeframe": "1h",
		"size":      200,
	}); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
}
