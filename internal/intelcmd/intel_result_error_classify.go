package intelcmd

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/gate/gate-cli/internal/mcpclient"
)

// paramSnakeNotSupportedRE matches messages like "time_range not supported" where the
// leading token looks like a snake_case tool parameter (at least one underscore).
var paramSnakeNotSupportedRE = regexp.MustCompile(`(?i)^[a-z][a-z0-9]*(_[a-z0-9]+)+ not supported$`)

// gateErrorMetaForIntelToolIsError picks HTTP status and label for MCP tools/call when
// result.isError is true. Tool-side argument / validation failures map to 400 +
// INVALID_ARGUMENTS so scripts can distinguish them from transport/protocol issues;
// everything else stays 502 + INTEL_RESULT_ERROR.
func gateErrorMetaForIntelToolIsError(msg string, result *mcpclient.CallResult) (status int, label string) {
	if status, ok := intelToolIsErrorClientHTTPStatus(result); ok {
		return status, "INVALID_ARGUMENTS"
	}
	if code := extractIntelToolErrorCode(result); isIntelToolClientArgumentCode(code) {
		return 400, "INVALID_ARGUMENTS"
	}
	if intelToolIsErrorLikelyClientArgs(msg, result) {
		return 400, "INVALID_ARGUMENTS"
	}
	return 502, "INTEL_RESULT_ERROR"
}

func isIntelToolClientArgumentCode(code string) bool {
	switch strings.ToUpper(strings.TrimSpace(strings.ReplaceAll(code, "-", "_"))) {
	case "INVALID_ARGUMENT", "INVALID_ARGUMENTS", "BAD_REQUEST", "VALIDATION_ERROR",
		"ARGUMENT_ERROR", "ILLEGAL_ARGUMENT", "OUT_OF_RANGE":
		return true
	default:
		return false
	}
}

func intelToolIsErrorClientHTTPStatus(result *mcpclient.CallResult) (int, bool) {
	if result == nil {
		return 0, false
	}
	for _, m := range []map[string]interface{}{result.StructuredContent, result.Raw, result.Meta} {
		for _, key := range []string{"http_status", "httpStatus", "status_code", "statusCode"} {
			if n, ok := intFromInterface(m[key]); ok && n >= 400 && n < 500 {
				return n, true
			}
		}
	}
	return 0, false
}

func intFromInterface(v interface{}) (int, bool) {
	if v == nil {
		return 0, false
	}
	switch t := v.(type) {
	case int:
		return t, true
	case int32:
		return int(t), true
	case int64:
		return int(t), true
	case float64:
		if t == float64(int64(t)) {
			return int(t), true
		}
	case string:
		if n, err := strconv.Atoi(strings.TrimSpace(t)); err == nil {
			return n, true
		}
	}
	return 0, false
}

func extractIntelToolErrorCode(result *mcpclient.CallResult) string {
	if result == nil {
		return ""
	}
	for _, m := range []map[string]interface{}{result.StructuredContent, result.Raw, result.Meta} {
		if c := errorCodeFromMap(m); c != "" {
			return c
		}
	}
	return ""
}

func errorCodeFromMap(m map[string]interface{}) string {
	if m == nil {
		return ""
	}
	for _, k := range []string{"code", "error_code", "errorCode", "error_type", "grpc_code"} {
		if c := normalizeCodeString(m[k]); c != "" {
			return c
		}
	}
	if ev, ok := m["error"]; ok {
		if em, ok2 := ev.(map[string]interface{}); ok2 {
			if c := errorCodeFromMap(em); c != "" {
				return c
			}
		}
	}
	return ""
}

func normalizeCodeString(v interface{}) string {
	s, ok := pickTrimmedString(v)
	if !ok {
		return ""
	}
	return strings.ToUpper(strings.ReplaceAll(s, "-", "_"))
}

// intelToolIsErrorLikelyClientArgs uses conservative substring checks on the extracted
// message plus optional structured hints. It intentionally avoids bare "not supported"
// (ambiguous with product limitations).
func intelToolIsErrorLikelyClientArgs(msg string, result *mcpclient.CallResult) bool {
	msg = strings.TrimSpace(msg)
	lower := strings.ToLower(msg)
	if strings.Contains(lower, "upstream") {
		return false
	}
	if result != nil {
		for _, m := range []map[string]interface{}{result.StructuredContent, result.Raw, result.Meta} {
			if m == nil {
				continue
			}
			for _, k := range []string{"invalid_argument", "invalidArgument", "validation_failed", "validationFailed"} {
				if b, ok := m[k].(bool); ok && b {
					return true
				}
			}
		}
	}
	if msg != "" {
		for _, sub := range []string{
			"参数不合法", "非法参数", "无效参数", "缺少必填", "未知参数", "仅支持",
		} {
			if strings.Contains(msg, sub) {
				return true
			}
		}
		for _, sub := range []string{
			"invalid argument",
			"invalid parameter",
			"invalid parameters",
			"invalid value",
			"invalid values",
			"missing required",
			"unknown field",
			"malformed",
			"must be one of",
			"illegal argument",
			"unexpected argument",
			"unexpected parameter",
		} {
			if strings.Contains(lower, sub) {
				return true
			}
		}
	}
	if msg != "" && paramSnakeNotSupportedRE.MatchString(msg) {
		return true
	}
	return false
}
