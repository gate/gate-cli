package intelfacade

import (
	"sort"
	"strings"
)

// DescribePrettyText returns fixed-section human text for describe commands
// (PRD §3.7.5 / §3.4.1: default stdout avoids protocol-oriented dumps).
func DescribePrettyText(tool *ToolSummary) string {
	if tool == nil {
		return ""
	}
	var b strings.Builder
	b.WriteString("Overview\n\n")
	b.WriteString(tool.Name)
	b.WriteByte('\n')
	if d := strings.TrimSpace(tool.Description); d != "" {
		b.WriteString(d)
		b.WriteByte('\n')
	}
	if paramBlock := formatParameterSummary(tool); paramBlock != "" {
		b.WriteString("\nParameters\n\n")
		b.WriteString(paramBlock)
		b.WriteByte('\n')
	}
	b.WriteString("\nNext steps\n\n")
	b.WriteString("- Use --format json for the full definition suitable for automation.\n")
	b.WriteString("- Use --help on the leaf command that maps to this capability for CLI flags.\n")
	return b.String()
}

func formatParameterSummary(tool *ToolSummary) string {
	if !tool.HasInputSchema || tool.InputSchema == nil {
		return ""
	}
	m, ok := tool.InputSchema.(map[string]interface{})
	if !ok {
		return "See --format json for parameter details."
	}
	props, ok := m["properties"].(map[string]interface{})
	if !ok || len(props) == 0 {
		return "See --format json for parameter details."
	}

	var required []string
	if r, ok := m["required"].([]interface{}); ok {
		for _, x := range r {
			if s, ok := x.(string); ok {
				required = append(required, s)
			}
		}
	}
	sort.Strings(required)
	reqSet := make(map[string]struct{}, len(required))
	for _, s := range required {
		reqSet[s] = struct{}{}
	}

	var optional []string
	for k := range props {
		if _, isReq := reqSet[k]; !isReq {
			optional = append(optional, k)
		}
	}
	sort.Strings(optional)

	var lines []string
	if len(required) > 0 {
		lines = append(lines, "Required:")
		for _, k := range required {
			lines = append(lines, "  - "+k)
		}
	}
	if len(optional) > 0 {
		if len(lines) > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, "Optional:")
		for _, k := range optional {
			lines = append(lines, "  - "+k)
		}
	}
	if len(lines) == 0 {
		return "See --format json for parameter details."
	}
	return strings.Join(lines, "\n")
}
