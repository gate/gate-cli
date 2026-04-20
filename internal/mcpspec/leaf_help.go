// Leaf help text for info/news cobra Long is built from embedded MCP JSON.
//
// Default is compact: omits the Parameters block (cobra Flags already list type/default/enum/max).
// Set GATE_INTEL_LEAF_HELP=full or detailed to embed per-field notes from the MCP JSON spec.
package mcpspec

import (
	"fmt"
	"os"
	"strings"
	"sync"
)

var (
	infoLongMu        sync.Mutex
	newsLongMu        sync.Mutex
	infoLongBy        map[string]string
	newsLongBy        map[string]string
	infoLongCacheMode string
	newsLongCacheMode string
)

// InfoLeafLongAppend returns MCP-spec narrative for an Info tool (required JSON, logic; per-field list only when GATE_INTEL_LEAF_HELP=full).
// Empty string if the tool is absent from the embedded document. Safe for cobra.Command.Long.
func InfoLeafLongAppend(toolName string) string {
	mode := leafHelpCacheMode()
	infoLongMu.Lock()
	if infoLongBy == nil || infoLongCacheMode != mode {
		infoLongBy = buildInfoLongByTool(leafHelpOmitForMode(mode))
		infoLongCacheMode = mode
	}
	m := infoLongBy
	infoLongMu.Unlock()
	if s := m[toolName]; s != "" {
		return s
	}
	return ""
}

// NewsLeafLongAppend returns MCP-spec narrative for a News tool (description, policy, logic, errors; Parameters block only when GATE_INTEL_LEAF_HELP=full).
func NewsLeafLongAppend(toolName string) string {
	mode := leafHelpCacheMode()
	newsLongMu.Lock()
	if newsLongBy == nil || newsLongCacheMode != mode {
		newsLongBy = buildNewsLongByTool(leafHelpOmitForMode(mode))
		newsLongCacheMode = mode
	}
	m := newsLongBy
	newsLongMu.Unlock()
	if s := m[toolName]; s != "" {
		return s
	}
	return ""
}

func buildInfoLongByTool(omitParamDetails bool) map[string]string {
	doc, err := InfoInputsLogic()
	if err != nil {
		return nil
	}
	m, ok := doc.(map[string]interface{})
	if !ok {
		return nil
	}
	raw, ok := m["tools"].([]interface{})
	if !ok {
		return nil
	}
	omit := omitParamDetails
	out := make(map[string]string, len(raw))
	for _, t := range raw {
		tm, ok := t.(map[string]interface{})
		if !ok {
			continue
		}
		name, _ := tm["tool_name"].(string)
		if name == "" {
			continue
		}
		out[name] = formatInfoToolLong(tm, omit)
	}
	return out
}

func buildNewsLongByTool(omitParamDetails bool) map[string]string {
	doc, err := NewsToolsArgs()
	if err != nil {
		return nil
	}
	m, ok := doc.(map[string]interface{})
	if !ok {
		return nil
	}
	raw, ok := m["tools"].([]interface{})
	if !ok {
		return nil
	}
	omit := omitParamDetails
	out := make(map[string]string, len(raw))
	for _, t := range raw {
		tm, ok := t.(map[string]interface{})
		if !ok {
			continue
		}
		name, _ := tm["name"].(string)
		if name == "" {
			continue
		}
		out[name] = formatNewsToolLong(tm, omit)
	}
	return out
}

// leafHelpCacheMode buckets env for help-text caching (compact vs full parameter blocks).
func leafHelpCacheMode() string {
	v := strings.TrimSpace(strings.ToLower(os.Getenv("GATE_INTEL_LEAF_HELP")))
	if v == "full" || v == "detailed" {
		return "full"
	}
	return "compact"
}

func leafHelpOmitForMode(mode string) bool {
	return mode != "full"
}

func formatInfoToolLong(tm map[string]interface{}, omitParamDetails bool) string {
	var b strings.Builder
	if v, ok := tm["domain"].(string); ok && strings.TrimSpace(v) != "" {
		fmt.Fprintf(&b, "Domain: %s\n", strings.TrimSpace(v))
	}
	if v, ok := tm["request_type"].(string); ok && strings.TrimSpace(v) != "" {
		fmt.Fprintf(&b, "Request type: %s\n\n", strings.TrimSpace(v))
	}
	if rq := tm["required"]; rq != nil {
		fmt.Fprintf(&b, "Required fields (JSON): %s\n\n", compactJSON(rq))
	}
	if cr := tm["conditional_required"]; cr != nil {
		fmt.Fprintf(&b, "Conditional required: %s\n\n", compactJSON(cr))
	}
	if !omitParamDetails {
		if fields, ok := tm["fields"].([]interface{}); ok && len(fields) > 0 {
			b.WriteString("Parameters:\n")
			for _, f := range fields {
				fm, ok := f.(map[string]interface{})
				if !ok {
					continue
				}
				if line := formatInfoFieldLong(fm); line != "" {
					b.WriteString(line)
					b.WriteString("\n")
				}
			}
			b.WriteString("\n")
		}
	}
	if logic := tm["logic"]; logic != nil {
		b.WriteString("Logic:\n")
		b.WriteString(formatInfoLogicLong(logic))
	}
	return strings.TrimSpace(b.String())
}

func formatInfoFieldLong(fm map[string]interface{}) string {
	name, _ := fm["name"].(string)
	if name == "" {
		return ""
	}
	typ, _ := fm["type"].(string)
	req := ""
	if v, ok := fm["required"].(bool); ok && v {
		req = ", required"
	}
	var lines []string
	lines = append(lines, fmt.Sprintf("- %s (%s%s)", name, typ, req))
	if ev := fieldEnumStrings(fm); len(ev) > 0 {
		lines = append(lines, fmt.Sprintf("  enum: %s", strings.Join(ev, ", ")))
	}
	if s, _ := fm["enum_note"].(string); strings.TrimSpace(s) != "" {
		lines = append(lines, fmt.Sprintf("  enum_note: %s", strings.TrimSpace(s)))
	}
	if s, _ := fm["notes"].(string); strings.TrimSpace(s) != "" {
		lines = append(lines, fmt.Sprintf("  notes: %s", strings.TrimSpace(s)))
	}
	for _, key := range []string{"default", "max", "min", "max_items"} {
		if v, ok := fm[key]; ok {
			lines = append(lines, fmt.Sprintf("  %s: %v", key, v))
		}
	}
	if cv, ok := fm["common_values"].([]interface{}); ok && len(cv) > 0 {
		lines = append(lines, fmt.Sprintf("  common_values: %s", joinIfaceStrings(cv)))
	}
	return strings.Join(lines, "\n")
}

func fieldEnumStrings(fm map[string]interface{}) []string {
	raw, ok := fm["enum"].([]interface{})
	if !ok || len(raw) == 0 {
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, v := range raw {
		out = append(out, fmt.Sprint(v))
	}
	return out
}

func formatInfoLogicLong(v interface{}) string {
	switch x := v.(type) {
	case string:
		return strings.TrimSpace(x)
	case []interface{}:
		var lines []string
		for _, item := range x {
			lines = append(lines, fmt.Sprintf("- %v", item))
		}
		return strings.Join(lines, "\n")
	default:
		return strings.TrimSpace(fmt.Sprint(v))
	}
}

func formatNewsToolLong(tm map[string]interface{}, omitParamDetails bool) string {
	var b strings.Builder
	if d, _ := tm["description"].(string); strings.TrimSpace(d) != "" {
		fmt.Fprintf(&b, "%s\n\n", strings.TrimSpace(d))
	}
	if c, _ := tm["category"].(string); strings.TrimSpace(c) != "" {
		fmt.Fprintf(&b, "Category: %s\n\n", strings.TrimSpace(c))
	}
	if ir, ok := tm["input_rules"].(map[string]interface{}); ok {
		if pol, _ := ir["required_policy"].(string); strings.TrimSpace(pol) != "" {
			fmt.Fprintf(&b, "Required policy: %s\n", strings.TrimSpace(pol))
		}
		if !omitParamDetails {
			if params, ok := ir["params"].([]interface{}); ok && len(params) > 0 {
				b.WriteString("\nParameters:\n")
				for _, p := range params {
					pm, ok := p.(map[string]interface{})
					if !ok {
						continue
					}
					b.WriteString(formatNewsParamLong(pm))
					b.WriteString("\n")
				}
			}
		}
	}
	if logic, ok := tm["logic"].([]interface{}); ok && len(logic) > 0 {
		b.WriteString("\nLogic:\n")
		for _, line := range logic {
			fmt.Fprintf(&b, "- %v\n", line)
		}
	}
	if errs, ok := tm["errors"].([]interface{}); ok && len(errs) > 0 {
		b.WriteString("\nErrors:\n")
		for _, e := range errs {
			if em, ok := e.(map[string]interface{}); ok {
				code, _ := em["code"].(string)
				when, _ := em["when"].(string)
				if code != "" || when != "" {
					fmt.Fprintf(&b, "- %s: %s\n", code, when)
					continue
				}
			}
			fmt.Fprintf(&b, "- %v\n", e)
		}
	}
	return strings.TrimSpace(b.String())
}

func formatNewsParamLong(pm map[string]interface{}) string {
	name, _ := pm["name"].(string)
	if name == "" {
		return ""
	}
	typ, _ := pm["type"].(string)
	req := ""
	if v, ok := pm["required"].(bool); ok && v {
		req = ", required"
	}
	var lines []string
	lines = append(lines, fmt.Sprintf("- %s (%s%s)", name, typ, req))
	if ev := fieldEnumStrings(pm); len(ev) > 0 {
		lines = append(lines, fmt.Sprintf("  enum: %s", strings.Join(ev, ", ")))
	}
	if s, _ := pm["notes"].(string); strings.TrimSpace(s) != "" {
		lines = append(lines, fmt.Sprintf("  notes: %s", strings.TrimSpace(s)))
	}
	for _, key := range []string{"default", "max", "min", "max_items"} {
		if v, ok := pm[key]; ok {
			lines = append(lines, fmt.Sprintf("  %s: %v", key, v))
		}
	}
	return strings.Join(lines, "\n")
}

func joinIfaceStrings(v []interface{}) string {
	parts := make([]string, 0, len(v))
	for _, x := range v {
		parts = append(parts, fmt.Sprint(x))
	}
	return strings.Join(parts, ", ")
}

func compactJSON(v interface{}) string {
	// Short, stable enough for help text (no sensitive data in these specs).
	switch x := v.(type) {
	case []interface{}:
		return joinIfaceStrings(x)
	case string:
		return x
	default:
		return fmt.Sprint(v)
	}
}
