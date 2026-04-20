package intelfacade

import (
	"reflect"
	"testing"
)

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
	sxProps := searchX["properties"].(map[string]interface{})
	timeRange := sxProps["time_range"].(map[string]interface{})
	if def, _ := timeRange["default"].(string); def != "24h" {
		t.Fatalf("time_range default mismatch: want 24h got %q", def)
	}
	enumVals, ok := timeRange["enum"].([]interface{})
	if !ok || len(enumVals) != 3 {
		t.Fatalf("time_range enum mismatch: %#v", timeRange["enum"])
	}

	sxLang := sxProps["lang"].(map[string]interface{})
	if def, _ := sxLang["default"].(string); def != "zh" {
		t.Fatalf("search_x lang default mismatch: want zh got %q", def)
	}
	if enums, ok := sxLang["enum"].([]interface{}); !ok || len(enums) != 3 {
		t.Fatalf("search_x lang enum mismatch: %#v", sxLang["enum"])
	}

	web := NewsBaselineInputSchema("news_feed_web_search")
	wProps := web["properties"].(map[string]interface{})
	wLang := wProps["lang"].(map[string]interface{})
	if def, _ := wLang["default"].(string); def != "en" {
		t.Fatalf("web_search lang default mismatch: want en got %q", def)
	}
	if enums, ok := wLang["enum"].([]interface{}); !ok || len(enums) != 3 {
		t.Fatalf("web_search lang enum mismatch: %#v", wLang["enum"])
	}
}

func TestNewsBaselineBoundKeywordsForCLIHelp(t *testing.T) {
	t.Parallel()
	ugc := NewsBaselineInputSchema("news_feed_search_ugc")
	lim := ugc["properties"].(map[string]interface{})["limit"].(map[string]interface{})
	if lim["maximum"].(float64) != 50 {
		t.Fatalf("ugc limit maximum: got %#v", lim["maximum"])
	}
	if lim["default"].(float64) != 10 {
		t.Fatalf("ugc limit default: got %#v", lim["default"])
	}

	detail := NewsBaselineInputSchema("news_events_get_event_detail")
	ev := detail["properties"].(map[string]interface{})["event_id"].(map[string]interface{})
	if ev["maxLength"].(float64) != 512 {
		t.Fatalf("event_id maxLength: got %#v", ev["maxLength"])
	}
	if pat, _ := ev["pattern"].(string); pat == "" {
		t.Fatal("expected event_id pattern")
	}

	x := NewsBaselineInputSchema("news_feed_search_x")
	ah := x["properties"].(map[string]interface{})["allowed_handles"].(map[string]interface{})
	if ah["maxItems"].(float64) != 10 {
		t.Fatalf("allowed_handles maxItems: got %#v", ah["maxItems"])
	}
}

func TestNewsBaselineInputSchemaDeepCopyIsolation(t *testing.T) {
	t.Parallel()

	schema := NewsBaselineInputSchema("news_feed_search_x")
	if schema == nil {
		t.Fatalf("expected non-nil schema")
	}
	props := schema["properties"].(map[string]interface{})

	allowed := props["allowed_handles"].(map[string]interface{})
	excluded := props["excluded_handles"].(map[string]interface{})

	aItems := allowed["items"].(map[string]interface{})
	bItems := excluded["items"].(map[string]interface{})
	if reflect.ValueOf(aItems).Pointer() == reflect.ValueOf(bItems).Pointer() {
		t.Fatalf("expected distinct nested items maps across sibling fields")
	}

	origType, _ := bItems["type"].(string)
	bItems["type"] = "integer"

	fresh := NewsBaselineInputSchema("news_feed_search_x")
	freshProps := fresh["properties"].(map[string]interface{})
	freshExcluded := freshProps["excluded_handles"].(map[string]interface{})
	freshItems := freshExcluded["items"].(map[string]interface{})
	if typ, _ := freshItems["type"].(string); typ != origType {
		t.Fatalf("mutating a returned copy leaked into subsequent baseline reads: got %q want %q", typ, origType)
	}
}
