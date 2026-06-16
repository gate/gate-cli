package toolrender

import "testing"

func TestMetaToolNameNewsBrief(t *testing.T) {
	t.Parallel()
	if got := MetaToolName("news/+brief"); got != "news_shortcut_brief" {
		t.Fatalf("got %q", got)
	}
}

func TestMetaToolNameLeavesMCPWireName(t *testing.T) {
	t.Parallel()
	name := "news_feed_search_news"
	if got := MetaToolName(name); got != name {
		t.Fatalf("got %q", got)
	}
}

func TestAppendFreshnessMetaNewsShortcut(t *testing.T) {
	t.Parallel()
	stale := "2020-01-01T00:00:00Z"
	env := map[string]interface{}{
		"data": map[string]interface{}{
			"items": []interface{}{
				map[string]interface{}{"published_at": stale},
			},
		},
	}
	AppendFreshnessMeta(MetaToolName("news/+brief"), env)
	meta, _ := env["meta"].(map[string]interface{})
	if meta == nil {
		t.Fatal("expected meta")
	}
	if _, ok := meta["freshness_summary"]; !ok {
		t.Fatalf("meta=%v", meta)
	}
}
