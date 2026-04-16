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
		"news_feed_search_x":                   {"allowed_handles", "excluded_handles", "enable_image_understanding", "enable_video_understanding", "platform_type"},
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
}
