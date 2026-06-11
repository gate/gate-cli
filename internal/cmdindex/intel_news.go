package cmdindex

import "strings"

// FilterInfoNewsOnly keeps runnable leaves under info and news (excludes cex, config, agent-*, etc.).
func FilterInfoNewsOnly(entries []Entry) []Entry {
	out := make([]Entry, 0, len(entries))
	for _, e := range entries {
		parts := strings.Fields(strings.TrimSpace(e.Path))
		if len(parts) == 0 {
			continue
		}
		switch parts[0] {
		case "info", "news":
			out = append(out, e)
		}
	}
	return out
}
