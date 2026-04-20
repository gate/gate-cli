package toolschema

import "fmt"

type VerifyReport struct {
	Backend      string          `json:"backend"`
	ToolCount    int             `json:"tool_count"`
	CacheFresh   bool            `json:"cache_fresh"`
	Status       string          `json:"status"`
	StrictMode   bool            `json:"strict_mode,omitempty"`
	StrictFailed bool            `json:"strict_failed,omitempty"`
	WarningCount int             `json:"warning_count"`
	Warnings     []VerifyWarning `json:"warnings"`
}

type VerifyWarning struct {
	Tool    string `json:"tool,omitempty"`
	Field   string `json:"field,omitempty"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

func ValidateTools(backend string, tools []ToolSummary, cacheFresh bool) VerifyReport {
	report := VerifyReport{
		Backend:    backend,
		ToolCount:  len(tools),
		CacheFresh: cacheFresh,
		Warnings:   []VerifyWarning{},
	}
	seen := map[string]struct{}{}
	for i, tool := range tools {
		if tool.Name == "" {
			report.Warnings = append(report.Warnings, VerifyWarning{
				Code:    "empty_tool_name",
				Message: fmt.Sprintf("tool at index %d has empty name; fix MCP tools/list registration output", i),
			})
			continue
		}
		if _, ok := seen[tool.Name]; ok {
			report.Warnings = append(report.Warnings, VerifyWarning{
				Tool:    tool.Name,
				Code:    "duplicate_tool_name",
				Message: "duplicate tool name in cache; deduplicate MCP tool registration",
			})
		}
		seen[tool.Name] = struct{}{}
		if !tool.HasInputSchema {
			report.Warnings = append(report.Warnings, VerifyWarning{
				Tool:    tool.Name,
				Code:    "missing_input_schema",
				Message: "tool has_input_schema=false; add inputSchema in backend service to enable accurate CLI flags",
			})
			continue
		}
		schema, ok := tool.InputSchema.(map[string]interface{})
		if !ok {
			report.Warnings = append(report.Warnings, VerifyWarning{
				Tool:    tool.Name,
				Code:    "invalid_schema_shape",
				Message: "input_schema must be an object; return JSON schema object from backend service",
			})
			continue
		}
		props, ok := schema["properties"].(map[string]interface{})
		if !ok || len(props) == 0 {
			report.Warnings = append(report.Warnings, VerifyWarning{
				Tool:    tool.Name,
				Code:    "missing_properties",
				Message: "input_schema.properties is missing or empty; add properties for flat CLI flags",
			})
		}
		if req, ok := schema["required"].([]interface{}); ok {
			for _, item := range req {
				key, _ := item.(string)
				if key == "" {
					report.Warnings = append(report.Warnings, VerifyWarning{
						Tool:    tool.Name,
						Code:    "invalid_required_item",
						Message: "required contains non-string/empty item; keep required entries as non-empty strings",
					})
					continue
				}
				if _, found := props[key]; !found {
					report.Warnings = append(report.Warnings, VerifyWarning{
						Tool:    tool.Name,
						Field:   key,
						Code:    "required_not_in_properties",
						Message: "required field is absent from properties; keep required and properties keys consistent",
					})
				}
			}
		}
		for field, raw := range props {
			prop, ok := raw.(map[string]interface{})
			if !ok {
				report.Warnings = append(report.Warnings, VerifyWarning{
					Tool:    tool.Name,
					Field:   field,
					Code:    "invalid_property_shape",
					Message: "property must be an object with schema keywords",
				})
				continue
			}
			t, validType := normalizedType(prop)
			if !validType {
				report.Warnings = append(report.Warnings, VerifyWarning{
					Tool:    tool.Name,
					Field:   field,
					Code:    "invalid_property_type",
					Message: "property type is missing or unsupported",
				})
				continue
			}
			if def, ok := prop["default"]; ok && !valueMatchesType(def, t) {
				report.Warnings = append(report.Warnings, VerifyWarning{
					Tool:    tool.Name,
					Field:   field,
					Code:    "default_type_mismatch",
					Message: "default value type does not match property type",
				})
			}
			if enumVals, ok := prop["enum"].([]interface{}); ok {
				for _, ev := range enumVals {
					if !valueMatchesType(ev, t) {
						report.Warnings = append(report.Warnings, VerifyWarning{
							Tool:    tool.Name,
							Field:   field,
							Code:    "enum_type_mismatch",
							Message: "enum value type does not match property type",
						})
						break
					}
				}
			}
		}
	}
	report.WarningCount = len(report.Warnings)
	if report.WarningCount == 0 {
		report.Status = "ok"
	} else {
		report.Status = "warn"
	}
	return report
}

func normalizedType(spec map[string]interface{}) (string, bool) {
	if t, ok := spec["type"].(string); ok {
		switch t {
		case "string", "integer", "number", "boolean", "array", "object":
			return t, true
		default:
			return "", false
		}
	}
	if ts, ok := spec["type"].([]interface{}); ok {
		for _, item := range ts {
			s, _ := item.(string)
			if s == "" || s == "null" {
				continue
			}
			switch s {
			case "string", "integer", "number", "boolean", "array", "object":
				return s, true
			default:
				return "", false
			}
		}
	}
	return "", false
}

func valueMatchesType(v interface{}, t string) bool {
	switch t {
	case "string":
		_, ok := v.(string)
		return ok
	case "integer":
		switch x := v.(type) {
		case float64:
			return x == float64(int64(x))
		case int:
			return true
		case int64:
			return true
		}
		return false
	case "number":
		switch v.(type) {
		case float64, int, int64:
			return true
		}
		return false
	case "boolean":
		_, ok := v.(bool)
		return ok
	case "array":
		_, ok := v.([]interface{})
		return ok
	case "object":
		_, ok := v.(map[string]interface{})
		return ok
	default:
		return false
	}
}
