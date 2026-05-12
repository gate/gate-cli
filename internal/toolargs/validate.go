package toolargs

import (
	"encoding/json"
	"errors"
	"strings"
)

// ValidateForTool applies static argument rules before MCP tools/call (caller-fixable input).
// Server-side JSON Schema and business validation remain authoritative at runtime.
func ValidateForTool(toolName string, arguments map[string]interface{}) error {
	if arguments == nil {
		arguments = map[string]interface{}{}
	}
	switch toolName {
	case "info_platformmetrics_get_platform_history":
		if !nonEmptyStringArg(arguments, "platform_name") && !nonEmptyStringArg(arguments, "exchange_slug") {
			return errors.New("missing required fields: provide platform_name or exchange_slug (at least one)")
		}
	}
	return nil
}

func nonEmptyStringArg(arguments map[string]interface{}, key string) bool {
	v, ok := arguments[key]
	if !ok || v == nil {
		return false
	}
	switch s := v.(type) {
	case string:
		return strings.TrimSpace(s) != ""
	case json.Number:
		return strings.TrimSpace(s.String()) != ""
	default:
		return false
	}
}
