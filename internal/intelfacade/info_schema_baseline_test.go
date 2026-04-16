package intelfacade

import "testing"

func TestInfoBaselineInputSchema_get_coin_info(t *testing.T) {
	t.Parallel()
	m := InfoBaselineInputSchema("info_coin_get_coin_info")
	if m == nil {
		t.Fatal("nil schema")
	}
	props := m["properties"].(map[string]interface{})
	for _, k := range []string{"query", "query_type", "size", "fields"} {
		if _, ok := props[k]; !ok {
			t.Fatalf("missing %q", k)
		}
	}
}
