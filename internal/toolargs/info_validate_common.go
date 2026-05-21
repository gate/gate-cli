package toolargs

import "strings"

func infoScopeBasicOrFull(arguments map[string]interface{}, key string) (normalized string, raw string, err error) {
	raw = strings.TrimSpace(stringArg(arguments, key))
	normalized = strings.ToLower(raw)
	if normalized == "" {
		return "basic", raw, nil
	}
	if normalized != "basic" && normalized != "full" {
		return "", raw, errInfoScopeBasicOrFull(raw)
	}
	return normalized, raw, nil
}

func errInfoScopeBasicOrFull(got string) error {
	return errInvalidArgumentsf("scope must be basic or full (got %q)", got)
}

func boolArgTrue(arguments map[string]interface{}, key string) bool {
	v, ok := arguments[key]
	if !ok || v == nil {
		return false
	}
	switch b := v.(type) {
	case bool:
		return b
	default:
		return false
	}
}
