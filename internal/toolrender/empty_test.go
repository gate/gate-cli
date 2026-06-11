package toolrender

import "testing"

func TestAppendResultMetaEmpty(t *testing.T) {
	t.Parallel()
	env := map[string]interface{}{
		"data": map[string]interface{}{"items": []interface{}{}},
	}
	AppendResultMeta("news_feed_search_news", env)
	meta := env["meta"].(map[string]interface{})
	if meta["result_hint"] != "EMPTY_RESULT" {
		t.Fatalf("meta=%v", meta)
	}
}
