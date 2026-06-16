package toolrender

import (
	"strings"
	"time"
)

type freshnessStats struct {
	ItemsWithTimestamp int
	StatusCounts       map[string]int
	Timestamps         []time.Time
}

func buildFreshnessSummary(data interface{}) map[string]interface{} {
	st := scanFreshnessStats(data, freshnessStats{
		StatusCounts: make(map[string]int),
	}, 64)
	if st.ItemsWithTimestamp == 0 && len(st.StatusCounts) == 0 {
		return nil
	}
	summary := map[string]interface{}{
		"items_with_timestamp": st.ItemsWithTimestamp,
	}
	if len(st.StatusCounts) > 0 {
		summary["freshness_status_counts"] = st.StatusCounts
	}
	if len(st.Timestamps) > 0 {
		newest := st.Timestamps[0]
		oldest := st.Timestamps[0]
		for _, ts := range st.Timestamps[1:] {
			if ts.After(newest) {
				newest = ts
			}
			if ts.Before(oldest) {
				oldest = ts
			}
		}
		summary["newest_published_at"] = newest.Format(time.RFC3339)
		summary["oldest_published_at"] = oldest.Format(time.RFC3339)
		age := time.Now().UTC().Sub(newest)
		if age > freshnessStaleAfter {
			summary["newest_age_hours"] = int(age.Hours())
			summary["newest_is_stale"] = true
		} else {
			summary["newest_is_stale"] = false
		}
		if len(st.Timestamps) > 1 && time.Now().UTC().Sub(oldest) > 7*24*time.Hour {
			summary["span_over_7d"] = true
		}
	}
	return summary
}

func scanFreshnessStats(v interface{}, st freshnessStats, budget int) freshnessStats {
	if budget <= 0 {
		return st
	}
	switch x := v.(type) {
	case map[string]interface{}:
		for k, item := range x {
			kl := strings.ToLower(strings.TrimSpace(k))
			if kl == "freshness_status" || kl == "freshnessstatus" {
				if s := strings.ToLower(strings.TrimSpace(stringFromInterface(item))); s != "" {
					st.StatusCounts[s]++
				}
			}
			if isFreshnessKey(k) {
				if ts, ok := parseTimeValue(item); ok {
					st.ItemsWithTimestamp++
					st.Timestamps = append(st.Timestamps, ts)
					budget--
				}
			}
			st = scanFreshnessStats(item, st, budget)
			budget = 64 - len(st.Timestamps)
		}
	case []interface{}:
		for _, item := range x {
			st = scanFreshnessStats(item, st, budget)
			budget = 64 - len(st.Timestamps)
		}
	}
	return st
}

func stringFromInterface(v interface{}) string {
	switch x := v.(type) {
	case string:
		return x
	default:
		return ""
	}
}
