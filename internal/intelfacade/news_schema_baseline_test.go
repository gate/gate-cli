package intelfacade

import "testing"

func TestNewsBaselineInputSchemaCoverage(t *testing.T) {
	t.Parallel()
	if len(NewsBaselineInputSchemas) != len(NewsToolBaseline) {
		t.Fatalf("news baseline size mismatch: schemas=%d inventory=%d", len(NewsBaselineInputSchemas), len(NewsToolBaseline))
	}
	for _, tool := range NewsToolBaseline {
		schema := NewsBaselineInputSchema(tool)
		if schema == nil {
			t.Fatalf("missing baseline schema for %s", tool)
		}
		if _, ok := schema["properties"].(map[string]interface{}); !ok {
			t.Fatalf("missing properties for %s", tool)
		}
	}
}

func TestNewsBaselineInputSchemaCriticalFields(t *testing.T) {
	t.Parallel()
	cases := map[string][]string{
		"news_feed_search_news":                {"query", "coin", "platform", "platform_type", "start_time", "end_time", "similarity_score", "top_total_score"},
		"news_feed_get_exchange_announcements": {"announcement_type", "coin", "platform", "from", "to"},
		"news_events_get_latest_events":        {"event_type", "cursor", "start_time", "end_time"},
		"news_feed_search_x":                   {"allowed_handles", "excluded_handles", "enable_image_understanding", "enable_video_understanding", "platform_type", "time_range"},
	}
	for tool, fields := range cases {
		schema := NewsBaselineInputSchema(tool)
		props := schema["properties"].(map[string]interface{})
		for _, f := range fields {
			if _, ok := props[f]; !ok {
				t.Fatalf("%s missing critical field %q", tool, f)
			}
		}
	}

	searchX := NewsBaselineInputSchema("news_feed_search_x")
	props := searchX["properties"].(map[string]interface{})
	timeRange := props["time_range"].(map[string]interface{})
	if def, _ := timeRange["default"].(string); def != "24h" {
		t.Fatalf("time_range default mismatch: want 24h got %q", def)
	}
	enumVals, ok := timeRange["enum"].([]interface{})
	if !ok || len(enumVals) != 3 {
		t.Fatalf("time_range enum mismatch: %#v", timeRange["enum"])
	}

	for _, tool := range []string{"news_feed_search_x", "news_feed_web_search"} {
		schema := NewsBaselineInputSchema(tool)
		props := schema["properties"].(map[string]interface{})
		lang := props["lang"].(map[string]interface{})
		if def, _ := lang["default"].(string); def != "zh" {
			t.Fatalf("%s lang default mismatch: want zh got %q", tool, def)
		}
		enums, ok := lang["enum"].([]interface{})
		if !ok || len(enums) != 3 {
			t.Fatalf("%s lang enum mismatch: %#v", tool, lang["enum"])
		}
	}
}
