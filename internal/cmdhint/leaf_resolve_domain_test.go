//go:build agent

package cmdhint

import (
	"strings"
	"testing"
)

func TestResolveAgentIntentDedupesByCLIPath(t *testing.T) {
	t.Parallel()
	got := ResolveAgentIntent("market kline BTC", 3, 3, "")
	paths := make(map[string]int)
	for _, r := range got {
		paths[catalogCLIPath(r.Command)]++
	}
	for path, n := range paths {
		if n > 1 {
			t.Fatalf("duplicate path %q in %#v", path, got)
		}
	}
}

func TestResolveAgentIntentDomainNewsOnly(t *testing.T) {
	t.Parallel()
	got := ResolveAgentIntent("BTC brief kline overview", 5, 5, "news")
	for _, r := range got {
		path := catalogCLIPath(r.Command)
		if !strings.HasPrefix(path, "news ") {
			t.Fatalf("expected news-only, got path=%q source=%s", path, r.MatchSource)
		}
	}
}

func TestResolveAgentIntentDomainIntelAlias(t *testing.T) {
	t.Parallel()
	got := ResolveAgentIntent("market kline", 3, 3, "intel")
	for _, r := range got {
		if !strings.HasPrefix(catalogCLIPath(r.Command), "info ") {
			t.Fatalf("intel alias should map to info paths, got %q", r.Command)
		}
	}
}
