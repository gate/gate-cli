package intelfacade

import "testing"

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
