package intelcmd

import (
	"strings"

	"github.com/gate/gate-cli/internal/mcpclient"
)

// MCP info_onchain error codes aligned with gate/mcp-server pkg/errs (NE enhancement).
const (
	mcpCodeInvalidChain            = "INVALID_CHAIN"
	mcpCodeInvalidAddress          = "INVALID_ADDRESS"
	mcpCodePartialUpstreamResponse = "PARTIAL_UPSTREAM_RESPONSE"
	mcpCodeUpstreamNotReady        = "UPSTREAM_NOT_READY"
)

func resolveIntelToolErrorCode(msg string, result *mcpclient.CallResult) string {
	if c := extractIntelToolErrorCode(result); c != "" {
		return c
	}
	return inferMCPErrorCodeFromMessage(msg)
}

func inferMCPErrorCodeFromMessage(msg string) string {
	lower := strings.ToLower(strings.TrimSpace(msg))
	if lower == "" {
		return ""
	}
	// Explicit code tokens in MCP SafeMessage / logs.
	for _, c := range []string{
		"partial_upstream_response",
		"upstream_not_ready",
		"invalid_address",
		"invalid_chain",
	} {
		if strings.Contains(lower, c) {
			return strings.ToUpper(strings.ReplaceAll(c, "-", "_"))
		}
	}
	// Chinese / paraphrased MCP messages.
	switch {
	case strings.Contains(msg, "无效或不受支持"):
		return mcpCodeInvalidChain
	case strings.Contains(msg, "address 格式无效"), strings.Contains(lower, "invalid_address"):
		return mcpCodeInvalidAddress
	case strings.Contains(lower, "partial_upstream"):
		return mcpCodePartialUpstreamResponse
	case strings.Contains(msg, "该链接口未就绪"), strings.Contains(lower, "not in get /chains"):
		return mcpCodeUpstreamNotReady
	default:
		return ""
	}
}

func isOnchainUpstreamErrorCode(code string) bool {
	switch strings.ToUpper(strings.TrimSpace(code)) {
	case mcpCodePartialUpstreamResponse, mcpCodeUpstreamNotReady:
		return true
	default:
		return false
	}
}

func isOnchainClientArgumentCode(code string) bool {
	switch strings.ToUpper(strings.TrimSpace(code)) {
	case mcpCodeInvalidChain, mcpCodeInvalidAddress:
		return true
	default:
		return false
	}
}

func gateErrorLabelForCode(code string) string {
	c := strings.ToUpper(strings.TrimSpace(strings.ReplaceAll(code, "-", "_")))
	if c == "" {
		return "INTEL_RESULT_ERROR"
	}
	return c
}
