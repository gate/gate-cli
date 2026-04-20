package mcpspec

import (
	"strings"
	"testing"
)

func TestInfoLeafLongAppendCoinInfo_defaultCompact(t *testing.T) {
	t.Setenv("GATE_INTEL_LEAF_HELP", "")
	s := InfoLeafLongAppend("info_coin_get_coin_info")
	if strings.Contains(s, "Parameters:") {
		t.Fatalf("default should omit Parameters (flags list types; use GATE_INTEL_LEAF_HELP=full for field notes): %s", s)
	}
	if !strings.Contains(s, "Required fields (JSON):") || !strings.Contains(s, "query") {
		t.Fatalf("expected required JSON line: %s", s)
	}
	if !strings.Contains(s, "Logic:") || !strings.Contains(s, "coinSearcher") {
		t.Fatalf("expected logic from spec: %s", s)
	}
}

func TestInfoLeafLongAppendCoinInfo_fullParams(t *testing.T) {
	t.Setenv("GATE_INTEL_LEAF_HELP", "full")
	s := InfoLeafLongAppend("info_coin_get_coin_info")
	if !strings.Contains(s, "Parameters:") || !strings.Contains(s, "query") {
		t.Fatalf("full mode should list parameters: %s", s)
	}
}

func TestNewsLeafLongAppendSearchNews_defaultCompact(t *testing.T) {
	t.Setenv("GATE_INTEL_LEAF_HELP", "")
	s := NewsLeafLongAppend("news_feed_search_news")
	if !strings.Contains(s, "Search platform news") {
		t.Fatalf("missing description: %s", s)
	}
	if strings.Contains(s, "Parameters:") {
		t.Fatalf("default should omit Parameters: %s", s)
	}
	if !strings.Contains(s, "Logic:") || !strings.Contains(s, "time_range") {
		t.Fatalf("expected logic mentioning time window: %s", s)
	}
}

func TestFormatNewsToolLongCompactOmitsParams(t *testing.T) {
	t.Parallel()
	tm := map[string]interface{}{
		"description": "One-line summary.",
		"category":    "news_events",
		"input_rules": map[string]interface{}{
			"required_policy": "none",
			"params": []interface{}{
				map[string]interface{}{"name": "limit", "type": "integer", "notes": "verbose note not in flags"},
			},
		},
		"logic":  []interface{}{"step one"},
		"errors": []interface{}{map[string]interface{}{"code": "bad", "when": "never"}},
	}
	full := formatNewsToolLong(tm, false)
	if !strings.Contains(full, "Parameters:") || !strings.Contains(full, "verbose note") {
		t.Fatalf("full mode should list parameters: %s", full)
	}
	comp := formatNewsToolLong(tm, true)
	if strings.Contains(comp, "Parameters:") || strings.Contains(comp, "verbose note") {
		t.Fatalf("compact should omit parameters block: %s", comp)
	}
	if !strings.Contains(comp, "Logic:") || !strings.Contains(comp, "Errors:") || !strings.Contains(comp, "bad") {
		t.Fatalf("compact should keep logic and errors: %s", comp)
	}
}
