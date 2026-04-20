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
	status, label := gateErrorMetaForIntelToolIsError(msg, result)
	ge := &output.GateError{
		Status:   status,
		Label:    label,
		Message:  msg,
		ToolName: toolName,
	}
	if httpResp != nil && httpResp.Header != nil {
		if tid := strings.TrimSpace(httpResp.Header.Get("x-gate-trace-id")); tid != "" {
			ge.TraceID = tid
		}
	}
	return ge
}

// RunToolCall merges argv, validates required fields when schema is available, calls MCP tools/call,
// and renders success output. backend is "info" or "news" (used in ParseError paths only).
func RunToolCall(cmd *cobra.Command, p *output.Printer, svc ToolCaller, name string, reserved map[string]struct{}, backend string, maxOutputBytes int64) error {
	if p.IsTable() {
		return FailLeafUnsupportedTable(p, backend)
	}

	arguments, err := toolargs.MergeFromCommand(cmd, toolargs.MergeOptions{ReservedFlags: reserved})
	if err != nil {
		return FailAfterPrintError(p, &output.GateError{
			Status:  400,
			Label:   "INVALID_ARGUMENTS",
			Message: err.Error(),
		})
	}
	arguments = toolargs.NormalizeForTool(name, arguments)
	if tool, _, derr := svc.DescribeTool(cmd.Context(), name); derr == nil && tool != nil {
		if missing := toolschema.MissingRequiredArguments(arguments, tool.InputSchema); len(missing) > 0 {
			return FailAfterPrintError(p, &output.GateError{
				Status:  400,
				Label:   "INVALID_ARGUMENTS",
				Message: "missing required fields: " + strings.Join(missing, ", "),
			})
		}
	}

	result, httpResp, err := svc.CallTool(cmd.Context(), name, arguments)
	if err != nil {
		return FailAfterPrintError(p, mcpclient.ParseError(err, httpResp, "POST", invokePath(backend), name))
	}
	if result == nil {
		return FailAfterPrintError(p, &output.GateError{
			Status:   502,
			Label:    "INTEL_PROTOCOL_ERROR",
			Message:  "tool returned empty response",
			ToolName: name,
		})
	}
	if result.IsError {
		return FailAfterPrintError(p, GateErrorForIntelToolIsError(name, httpResp, result))
	}
	return toolrender.RenderCallResult(p, name, result, maxOutputBytes)
}

// FailListTransport maps list endpoint failures to stderr + exit 1.
func FailListTransport(p *output.Printer, err error, httpResp *http.Response, backend string) error {
	return FailAfterPrintError(p, mcpclient.ParseError(err, httpResp, "POST", backend+"/list", ""))
}

// FailDescribeTransport maps describe endpoint failures to stderr + exit 1.
func FailDescribeTransport(p *output.Printer, err error, httpResp *http.Response, backend, toolName string) error {
	return FailAfterPrintError(p, mcpclient.ParseError(err, httpResp, "POST", backend+"/describe", toolName))
}

// FailIntelClientInit maps MCP client construction failures (before list/describe/invoke RPC).
func FailIntelClientInit(p *output.Printer, err error, backend, segment, toolName string) error {
	return FailAfterPrintError(p, mcpclient.ParseError(err, nil, "POST", backend+"/"+segment, toolName))
}
