//go:build agent

package cmdhint

import "testing"

func TestMatchLeavesCatalogKline(t *testing.T) {
	t.Parallel()
	got := MatchLeaves(BaselineMCPCatalog(), "indicator history", 2)
	if len(got) == 0 {
		t.Fatal("expected catalog match")
	}
}
