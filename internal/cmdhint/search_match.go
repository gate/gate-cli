//go:build agent

package cmdhint

import (
	"strings"

	"github.com/gate/gate-cli/internal/cmdindex"
)

// SearchMatch is an agent-search hit with discovery provenance.
type SearchMatch struct {
	Path          string `json:"path"`
	Short         string `json:"short,omitempty"`
	Score         int    `json:"score,omitempty"`
	MatchSource   string `json:"match_source"`
	CuratedIntent string `json:"curated_intent,omitempty"`
	IsShortcut    bool   `json:"is_shortcut,omitempty"`
}

// EnrichSearchMatches annotates cobra leaf hits with curated/shortcut/leaf source.
func EnrichSearchMatches(hits []cmdindex.Entry) []SearchMatch {
	if len(hits) == 0 {
		return nil
	}
	curatedByPath := curatedPathIndex()
	out := make([]SearchMatch, 0, len(hits))
	for _, h := range hits {
		m := SearchMatch{
			Path:  h.Path,
			Short: h.Short,
			Score: h.Score,
		}
		if strings.Contains(h.Path, " +") {
			m.MatchSource = "shortcut"
			m.IsShortcut = true
		} else if intent, ok := curatedByPath[h.Path]; ok {
			m.MatchSource = "curated"
			m.CuratedIntent = intent
		} else {
			m.MatchSource = "leaf"
		}
		out = append(out, m)
	}
	return out
}

func curatedPathIndex() map[string]string {
	out := make(map[string]string, len(AgentLeaves))
	for _, leaf := range AgentLeaves {
		out[catalogCLIPath(leaf.Command)] = leaf.Intent
	}
	return out
}
