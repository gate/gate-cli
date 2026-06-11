package toolrender

import (
	"testing"
	"time"
)

func TestAppendFreshnessMetaStaleNews(t *testing.T) {
	t.Parallel()
	old := time.Now().UTC().Add(-72 * time.Hour).Format(time.RFC3339)
	env := map[string]interface{}{
		"data": map[string]interface{}{
			"items": []interface{}{
				map[string]interface{}{"published_at": old, "title": "vote ended"},
			},
		},
	}
	AppendFreshnessMeta("news_feed_search_news", env)
	meta, ok := env["meta"].(map[string]interface{})
	if !ok {
		t.Fatal("expected meta")
	}
	hints, ok := meta["freshness_hints"].([]string)
	if !ok || len(hints) == 0 {
		t.Fatalf("hints=%v", meta["freshness_hints"])
	}
}

func TestAppendFreshnessMetaSkipsInfoKline(t *testing.T) {
	t.Parallel()
	env := map[string]interface{}{"data": map[string]interface{}{"candles": []interface{}{}}}
	AppendFreshnessMeta("info_markettrend_get_kline", env)
	if _, ok := env["meta"]; ok {
		t.Fatal("kline should not get freshness meta")
	}
}
