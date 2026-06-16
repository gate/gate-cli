package intelcmd

import (
	"context"
	"net/http"
	"strings"

	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/intelfacade"
	"github.com/gate/gate-cli/internal/mcpclient"
	"github.com/gate/gate-cli/internal/output"
	"github.com/gate/gate-cli/internal/toolargs"
	"github.com/gate/gate-cli/internal/toolrender"
	"github.com/gate/gate-cli/internal/toolschema"
)

// ToolCaller is the subset of Intel services needed for invoke / group-leaf tool runs.
type ToolCaller interface {
	DescribeTool(ctx context.Context, name string) (*intelfacade.ToolSummary, *http.Response, error)
	CallTool(ctx context.Context, name string, arguments map[string]interface{}) (*mcpclient.CallResult, *http.Response, error)
}

func invokePath(backend string) string {
	return backend + "/invoke"
}

// GateErrorForIntelToolIsError is the stderr shape for tools/call when result.isError is true.
// When the payload looks like an argument/validation problem (structured error codes, 4xx
// hints in content, or typical validation wording), status is 400 with label INVALID_ARGUMENTS;
// otherwise status 502 with label INTEL_RESULT_ERROR.
// When httpResp carries x-gate-trace-id, it is copied for support without exposing MCP content.
// When result carries structuredContent or text content, a trimmed summary is used as Message.
func GateErrorForIntelToolIsError(toolName string, httpResp *http.Response, result *mcpclient.CallResult) *output.GateError {
	msg := messageFromIntelToolIsError(result)
	if msg == "" {
		msg = "tool returned isError=true"
	}
	code := resolveIntelToolErrorCode(msg, result)
	if toolName == "info_onchain_get_address_transactions" && code == mcpCodePartialUpstreamResponse {
		msg = "upstream returned total but no parseable transaction list; use --format json for details or --debug / trace_id for support"
	}
	status, label := gateErrorMetaForIntelToolIsError(msg, result)
	ge := &output.GateError{
		Status:  status,
		Label:   label,
		Message: msg,
	}
	ge.ErrorType = output.ClassifyCLIError(status, label, msg)
	output.FillAgentErrorConvergence(ge)
	if httpResp != nil && httpResp.Header != nil {
		if tid := strings.TrimSpace(httpResp.Header.Get("x-gate-trace-id")); tid != "" {
			ge.TraceID = tid
		}
	}
	return ge
}

// GateErrorForIntelToolIsErrorLeaf is GateErrorForIntelToolIsError with user-facing command path (no MCP tool_name).
func GateErrorForIntelToolIsErrorLeaf(backend, toolName string, httpResp *http.Response, result *mcpclient.CallResult) *output.GateError {
	ge := GateErrorForIntelToolIsError(toolName, httpResp, result)
	SanitizeUserFacingGateError(ge, backend, "", toolName)
	return ge
}

// RunToolCall merges argv, validates required fields when schema is available, calls MCP tools/call,
// and renders success output. backend is "info" or "news" (used in ParseError paths only).
func RunToolCall(cmd *cobra.Command, p *output.Printer, svc ToolCaller, name string, reserved map[string]struct{}, backend string, maxOutputBytes int64) error {
	if p.IsTable() {
		return FailLeafUnsupportedTable(p, backend)
	}
	name = ResolveMCPToolName(backend, name)

	arguments, err := toolargs.MergeFromCommand(cmd, toolargs.MergeOptions{ReservedFlags: reserved})
	if err != nil {
		return FailAfterPrintError(p, output.InvalidArgsError(err.Error()))
	}
	arguments = toolargs.NormalizeForTool(name, arguments)
	if err := toolargs.ValidateForTool(name, arguments); err != nil {
		return FailAfterPrintError(p, output.InvalidArgsError(err.Error()))
	}
	if tool, _, derr := svc.DescribeTool(cmd.Context(), name); derr == nil && tool != nil {
		schema := InputSchemaForMissingRequiredCheck(backend, name, tool.InputSchema)
		if missing := toolschema.MissingRequiredArguments(arguments, schema); len(missing) > 0 {
			return FailAfterPrintError(p, output.InvalidArgsError("missing required fields: "+strings.Join(missing, ", ")))
		}
	}

	result, httpResp, err := svc.CallTool(cmd.Context(), name, arguments)
	if err != nil {
		ge := mcpclient.ParseError(err, httpResp, "POST", invokePath(backend), "")
		SanitizeUserFacingGateError(ge, backend, "", name)
		return FailAfterPrintError(p, ge)
	}
	if result == nil {
		ge := &output.GateError{
			Status:  502,
			Label:   "INTEL_PROTOCOL_ERROR",
			Message: "tool returned empty response",
		}
		SanitizeUserFacingGateError(ge, backend, "", name)
		return FailAfterPrintError(p, ge)
	}
	if result.IsError {
		return FailAfterPrintError(p, GateErrorForIntelToolIsErrorLeaf(backend, name, httpResp, result))
	}
	return toolrender.RenderCallResult(p, backend, name, result, maxOutputBytes)
}

// FailListTransport maps list endpoint failures to stderr + exit 1.
func FailListTransport(p *output.Printer, err error, httpResp *http.Response, backend string) error {
	return FailAfterPrintError(p, mcpclient.ParseError(err, httpResp, "POST", backend+"/list", ""))
}

// FailDescribeTransport maps describe endpoint failures to stderr + exit 1.
func FailDescribeTransport(p *output.Printer, err error, httpResp *http.Response, backend, toolName string) error {
	ge := mcpclient.ParseError(err, httpResp, "POST", backend+"/describe", "")
	SanitizeUserFacingGateError(ge, backend, "", toolName)
	return FailAfterPrintError(p, ge)
}

// FailIntelClientInit maps MCP client construction failures (before list/describe/invoke RPC).
func FailIntelClientInit(p *output.Printer, err error, backend, segment, toolName string) error {
	ge := mcpclient.ParseError(err, nil, "POST", backend+"/"+segment, "")
	SanitizeUserFacingGateError(ge, backend, "", toolName)
	return FailAfterPrintError(p, ge)
}
