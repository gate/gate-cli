package intelfacade

import "testing"

func TestNewsBaselineInputSchema_search_news(t *testing.T) {
	t.Parallel()
	m := NewsBaselineInputSchema("news_feed_search_news")
	if m == nil {
		t.Fatal("nil schema")
	}
	props, ok := m["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("missing properties")
	}
	for _, k := range []string{"query", "coin", "limit", "sort_by"} {
		if _, ok := props[k]; !ok {
			t.Fatalf("missing property %q", k)
		}
	}
	req, _ := m["required"].([]interface{})
	if len(req) != 1 || req[0] != "query" {
		t.Fatalf("unexpected required: %#v", req)
	}
}
