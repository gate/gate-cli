package operations

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestReleaseAvailability audits every catalog entry against the independently
// specified endpoint boundary, including APIs without a published CLI command.
func TestReleaseAvailability(t *testing.T) {
	var c catalog
	if err := json.Unmarshal(rawCatalog, &c); err != nil {
		t.Fatal(err)
	}
	retired := 0
	for _, op := range c.Operations {
		want := strings.HasPrefix(op.Path, "/api/v4/p2p/") ||
			strings.HasPrefix(op.Path, "/api/v4/launch/") ||
			strings.HasPrefix(op.Path, "/api/v4/crossex/") ||
			strings.HasPrefix(op.Path, "/api/v4/coupon/") ||
			strings.HasPrefix(op.Path, "/api/v4/ann/") ||
			strings.HasPrefix(op.Path, "/api/v4/rewards/activity/")
		switch op.Method + " " + op.Path {
		case "GET /api/v4/rewards/getUserIdentity", "GET /api/v4/rewards/getBeginnerTaskList",
			"POST /api/v4/rewards/claimTask", "POST /api/v4/rewards/claimReward":
			want = true
		}
		// Concrete path values exercise catalog template matching, not just exact keys.
		path := tmplVarRe.ReplaceAllString(op.Path, "synthetic")
		got, err := IsDisabled(op.Method, path)
		if err != nil || got != want {
			t.Errorf("%s %s disabled=%v want=%v err=%v", op.Method, path, got, want, err)
		}
		if got {
			retired++
		}
	}
	if retired != 81 {
		t.Fatalf("retired catalog operations=%d want81", retired)
	}
	if !IsPublic("POST", "/api/v4/ann/list_article") {
		t.Fatal("release policy must not rewrite Announcement API authentication")
	}
}
