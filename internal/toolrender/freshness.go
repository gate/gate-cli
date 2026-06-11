package toolrender

import (
	"fmt"
	"strings"
	"time"

	"github.com/gate/gate-cli/internal/cmdhint"
)

const freshnessStaleAfter = 48 * time.Hour

var freshnessTimeKeys = []string{
	"published_at", "publishedat", "created_at", "createdat",
	"event_time", "eventtime", "pub_time", "timestamp",
}

// AppendFreshnessMeta adds meta.freshness_hints for time-sensitive Intel tools (news).
func AppendFreshnessMeta(toolName string, envelope map[string]interface{}) {
	if envelope == nil || !strings.HasPrefix(toolName, "news_") {
		return
	}
	meta, _ := envelope["meta"].(map[string]interface{})
	if meta == nil {
		meta = map[string]interface{}{}
	} else {
		meta = cloneMap(meta)
	}
	summary := buildFreshnessSummary(envelope["data"])
	if summary != nil {
		meta["freshness_summary"] = summary
	}
	statusCLI := freshnessStatusCLI(summary)
	if statusCLI != "" {
		meta["freshness_status_cli"] = statusCLI
	}
	hints := collectFreshnessHints(envelope["data"])
	if len(hints) == 0 {
		if statusCLI == "" || statusCLI == "unknown" {
			if summary == nil {
				meta["freshness_status_cli"] = "unknown"
				hints = []string{"no published_at in payload; treat as unverified for «latest/current» claims (use web_search_verify in Skill)"}
			} else if n, ok := summary["items_with_timestamp"].(int); ok && n == 0 {
				meta["freshness_status_cli"] = "unknown"
				hints = []string{"no published_at in payload; treat as unverified for «latest/current» claims (use web_search_verify in Skill)"}
			}
		}
	}
	if len(hints) > 0 {
		meta["freshness_hints"] = hints
	}
	if cmdhint.AgentModeEnabled() {
		meta["agent_reminder"] = "Separate tool facts (with published_at) from model opinions in the Skill layer; stale items must not be presented as current events."
	}
	envelope["meta"] = meta
}

func freshnessStatusCLI(summary map[string]interface{}) string {
	if summary == nil {
		return ""
	}
	if counts, ok := summary["freshness_status_counts"].(map[string]int); ok && len(counts) > 0 {
		if counts["stale"] > 0 {
			return "stale"
		}
		if counts["fresh"] > 0 && counts["stale"] == 0 {
			return "fresh"
		}
		for k, n := range counts {
			if n > 0 && k != "" {
				return k
			}
		}
	}
	if stale, ok := summary["newest_is_stale"].(bool); ok {
		if stale {
			return "stale"
		}
		if n, ok := summary["items_with_timestamp"].(int); ok && n > 0 {
			return "fresh"
		}
	}
	return ""
}

func collectFreshnessHints(data interface{}) []string {
	times := collectTimestamps(data, 32)
	if len(times) == 0 {
		return nil
	}
	now := time.Now().UTC()
	newest := times[0]
	oldest := times[0]
	for _, ts := range times[1:] {
		if ts.After(newest) {
			newest = ts
		}
		if ts.Before(oldest) {
			oldest = ts
		}
	}
	var hints []string
	if age := now.Sub(newest); age > freshnessStaleAfter {
		hints = append(hints, fmt.Sprintf("newest item is older than 48h (~%dh); do not describe as «latest» without web verification", int(age.Hours())))
	}
	if len(times) > 1 && now.Sub(oldest) > 7*24*time.Hour {
		hints = append(hints, "payload spans more than 7 days; confirm time_range matches the user question")
	}
	return hints
}

func collectTimestamps(v interface{}, max int) []time.Time {
	var out []time.Time
	var walk func(interface{})
	walk = func(node interface{}) {
		if len(out) >= max {
			return
		}
		switch x := node.(type) {
		case map[string]interface{}:
			for k, item := range x {
				if isFreshnessKey(k) {
					if ts, ok := parseTimeValue(item); ok {
						out = append(out, ts)
					}
				}
				walk(item)
			}
		case []interface{}:
			for _, item := range x {
				walk(item)
			}
		}
	}
	walk(v)
	return out
}

func isFreshnessKey(key string) bool {
	kl := strings.ToLower(strings.TrimSpace(key))
	for _, w := range freshnessTimeKeys {
		if kl == w || strings.HasSuffix(kl, w) {
			return true
		}
	}
	return false
}

func parseTimeValue(v interface{}) (time.Time, bool) {
	switch x := v.(type) {
	case string:
		s := strings.TrimSpace(x)
		if s == "" {
			return time.Time{}, false
		}
		for _, layout := range []string{
			time.RFC3339,
			"2006-01-02T15:04:05Z07:00",
			"2006-01-02 15:04:05",
			"2006-01-02",
		} {
			if t, err := time.Parse(layout, s); err == nil {
				return t.UTC(), true
			}
		}
	case float64:
		return unixToTime(int64(x))
	case int:
		return unixToTime(int64(x))
	case int64:
		return unixToTime(x)
	}
	return time.Time{}, false
}

func unixToTime(sec int64) (time.Time, bool) {
	if sec <= 0 {
		return time.Time{}, false
	}
	if sec > 1_000_000_000_000 {
		sec = sec / 1000
	}
	return time.Unix(sec, 0).UTC(), true
}
