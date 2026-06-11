package toolrender

import "testing"

func TestBuildFreshnessSummary(t *testing.T) {
	t.Parallel()
	data := map[string]interface{}{
		"articles": []interface{}{
			map[string]interface{}{
				"published_at":     "2026-06-01T10:00:00Z",
				"freshness_status": "fresh",
			},
			map[string]interface{}{
				"published_at":     "2026-05-20T10:00:00Z",
				"freshness_status": "stale",
			},
		},
	}
	summary := buildFreshnessSummary(data)
	if summary == nil {
		t.Fatal("expected summary")
	}
	if summary["items_with_timestamp"] != 2 {
		t.Fatalf("items=%v", summary["items_with_timestamp"])
	}
	counts, ok := summary["freshness_status_counts"].(map[string]int)
	if !ok || counts["stale"] != 1 || counts["fresh"] != 1 {
		t.Fatalf("counts=%v", summary["freshness_status_counts"])
	}
	if got := freshnessStatusCLI(summary); got != "stale" {
		t.Fatalf("freshness_status_cli=%q", got)
	}
}
