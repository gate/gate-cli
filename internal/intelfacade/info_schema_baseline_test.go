package intelfacade

import (
	"reflect"
	"testing"
)

func TestInfoBaselineInputSchemaCoverage(t *testing.T) {
	t.Parallel()
	if len(InfoBaselineInputSchemas) != len(InfoToolBaseline) {
		t.Fatalf("info baseline size mismatch: schemas=%d inventory=%d", len(InfoBaselineInputSchemas), len(InfoToolBaseline))
	}
	for _, tool := range InfoToolBaseline {
		schema := InfoBaselineInputSchema(tool)
		if schema == nil {
			t.Fatalf("missing baseline schema for %s", tool)
		}

		if _, ok := schema["properties"].(map[string]interface{}); !ok {
			t.Fatalf("missing properties for %s", tool)
		}
	}
}

func TestInfoBaselineInputSchemaCriticalFields(t *testing.T) {
	t.Parallel()
	cases := map[string][]string{
		"info_markettrend_get_kline":                {"symbol", "timeframe", "with_indicators"},
		"info_markettrend_get_indicator_history":    {"symbol", "indicators", "timeframe"},
		"info_marketsnapshot_batch_market_snapshot": {"symbols", "timeframe"},
		"info_onchain_get_address_transactions":     {"from_address", "to_address", "nonzero_value"},
		"info_compliance_check_token_security":      {"token", "address", "chain"},
		"info_marketdetail_get_kline":               {"symbol", "timeframe", "extra"},
		"info_compliance_search_regulatory_updates": {"query", "symbol", "address"},
	}
	for tool, fields := range cases {
		schema := InfoBaselineInputSchema(tool)
		props := schema["properties"].(map[string]interface{})
		for _, f := range fields {
			if _, ok := props[f]; !ok {
				t.Fatalf("%s missing critical field %q", tool, f)
			}
		}
	}

	// Explicitly keep symbols as array to match MCP batch input shape.
	batch := InfoBaselineInputSchema("info_marketsnapshot_batch_market_snapshot")
	props := batch["properties"].(map[string]interface{})
	symbols := props["symbols"].(map[string]interface{})
	if typ, _ := symbols["type"].(string); typ != "array" {
		t.Fatalf("symbols type mismatch: want array got %q", typ)
	}
}

func TestInfoBaselineInputSchemaDeepCopyIsolation(t *testing.T) {
	t.Parallel()

	a := InfoBaselineInputSchema("info_coin_get_coin_info")
	b := InfoBaselineInputSchema("info_markettrend_get_indicator_history")
	if a == nil || b == nil {
		t.Fatalf("expected non-nil schemas")
	}

	aProps := a["properties"].(map[string]interface{})
	bProps := b["properties"].(map[string]interface{})

	aFields := aProps["fields"].(map[string]interface{})
	bIndicators := bProps["indicators"].(map[string]interface{})

	aItems := aFields["items"].(map[string]interface{})
	bItems := bIndicators["items"].(map[string]interface{})

	if reflect.ValueOf(aItems).Pointer() == reflect.ValueOf(bItems).Pointer() {
		t.Fatalf("expected distinct nested items maps across tools")
	}

	origType, _ := bItems["type"].(string)
	bItems["type"] = "integer"

	fresh := InfoBaselineInputSchema("info_markettrend_get_indicator_history")
	freshProps := fresh["properties"].(map[string]interface{})
	freshIndicators := freshProps["indicators"].(map[string]interface{})
	freshItems := freshIndicators["items"].(map[string]interface{})
	if typ, _ := freshItems["type"].(string); typ != origType {
		t.Fatalf("mutating a returned copy leaked into subsequent baseline reads: got %q want %q", typ, origType)
	}
}
