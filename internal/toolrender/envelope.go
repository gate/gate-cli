package toolrender

import (
	"encoding/json"
	"fmt"

	"github.com/gate/gate-cli/internal/mcpclient"
)

// BuildCLIEnvelope normalizes a tool call result for stable JSON output.
// This is used by info/news commands to keep CLI contract consistent.
func BuildCLIEnvelope(toolName string, result *mcpclient.CallResult) map[string]interface{} {
	data, source, warnings := extractData(result)
	payload := map[string]interface{}{
		"status":      "success",
		"tool_name":   toolName,
		"is_error":    result.IsError,
		"data_source": source,
		"data":        data,
	}
	if result.IsError {
		payload["status"] = "error"
	}
	if meta := mergeMeta(result.Meta, warnings); meta != nil {
		payload["meta"] = meta
	}
	return payload
}

func extractData(result *mcpclient.CallResult) (interface{}, string, []string) {
	if result.StructuredContent != nil {
		return result.StructuredContent, "structured_content", nil
	}
	if len(result.ContentRaw) > 0 {
		if normalized, warnings, ok := normalizeContentRaw(result.ContentRaw); ok {
			return normalized, "content", warnings
		}
		return result.ContentRaw, "content", []string{"content present but failed to normalize; using raw content"}
	}
	if len(result.Content) > 0 {
		if normalized, warnings, ok := normalizeContent(result.Content); ok {
			return normalized, "content", warnings
		}
		return result.Content, "content", []string{"content present but failed to normalize; using raw content"}
	}
	if result.Raw != nil {
		return result.Raw, "raw", nil
	}
	return map[string]interface{}{}, "empty", nil
}

func normalizeContentRaw(items []interface{}) (interface{}, []string, bool) {
	if len(items) == 0 {
		return nil, nil, false
	}
	out := make([]interface{}, 0, len(items))
	var warnings []string
	for _, raw := range items {
		m, ok := raw.(map[string]interface{})
		if !ok {
			out = append(out, raw)
			warnings = append(warnings, fmt.Sprintf("content item has unexpected type %T; keeping raw item", raw))
			continue
		}
		if text, ok := m["text"].(string); ok {
			var parsed interface{}
			if json.Unmarshal([]byte(text), &parsed) == nil {
				out = append(out, parsed)
			} else {
				out = append(out, text)
				warnings = append(warnings, "content.text is not valid json; keeping original text")
			}
			continue
		}
		if _, hasType := m["type"]; hasType {
			warnings = append(warnings, fmt.Sprintf("content item type=%v has no text field; keeping raw item", m["type"]))
		}
		out = append(out, m)
	}
	if len(out) == 1 {
		return out[0], warnings, true
	}
	return out, warnings, true
}

func normalizeContent(items []mcpclient.ContentItem) (interface{}, []string, bool) {
	if len(items) == 0 {
		return nil, nil, false
	}
	out := make([]interface{}, 0, len(items))
	var warnings []string
	for _, item := range items {
		if text, ok := item["text"].(string); ok {
			var parsed interface{}
			if json.Unmarshal([]byte(text), &parsed) == nil {
				out = append(out, parsed)
			} else {
				out = append(out, text)
				warnings = append(warnings, "content.text is not valid json; keeping original text")
			}
			continue
		}
		if _, hasType := item["type"]; hasType {
			warnings = append(warnings, fmt.Sprintf("content item type=%v has no text field; keeping raw item", item["type"]))
		}
		out = append(out, item)
	}
	if len(out) == 1 {
		return out[0], warnings, true
	}
	return out, warnings, true
}

func mergeMeta(meta map[string]interface{}, warnings []string) map[string]interface{} {
	if len(warnings) == 0 {
		return meta
	}
	out := make(map[string]interface{})
	for k, v := range meta {
		out[k] = v
	}
	out["parse_warnings"] = warnings
	return out
}
