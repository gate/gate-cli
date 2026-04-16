package intelfacade

import (
	"strings"
)

// ListCapabilitiesPrettyText renders tool inventory for --format pretty (PRD §3.7.5).
func ListCapabilitiesPrettyText(items []ToolSummary) string {
	if len(items) == 0 {
		return "Capabilities\n\n(no entries)\n"
	}
	var b strings.Builder
	b.WriteString("Capabilities\n\n")
	for i, item := range items {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(item.Name)
		b.WriteByte('\n')
		if d := strings.TrimSpace(item.Description); d != "" {
			b.WriteString(d)
			b.WriteByte('\n')
		}
		if item.HasInputSchema {
			b.WriteString("Accepts parameters: yes\n")
		} else {
			b.WriteString("Accepts parameters: no\n")
		}
	}
	return b.String()
}
