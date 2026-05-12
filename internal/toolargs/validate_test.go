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
