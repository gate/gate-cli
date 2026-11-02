package intelcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/gate/gate-cli/internal/intelfacade"
	"github.com/gate/gate-cli/internal/mcpclient"
	"github.com/gate/gate-cli/internal/output"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// Mock only the service boundary, leaving the real merge, validation, error
// classification and renderer in the unit's complete command orchestration.
type toolCallFixture struct {
	t                    *testing.T
	describe, call       int
	describeErr, callErr error
	schema               interface{}
	result               *mcpclient.CallResult
	context              context.Context
	arguments            map[string]interface{}
}

func (f *toolCallFixture) DescribeTool(ctx context.Context, name string) (*intelfacade.ToolSummary, *http.Response, error) {
	f.describe++
	require.Equal(f.t, "info_coin_get_coin_info", name)
	require.Equal(f.t, f.context, ctx)
	return &intelfacade.ToolSummary{InputSchema: f.schema}, nil, f.describeErr
}

func (f *toolCallFixture) CallTool(ctx context.Context, name string, args map[string]interface{}) (*mcpclient.CallResult, *http.Response, error) {
	f.call++
	require.Equal(f.t, "info_coin_get_coin_info", name)
	require.Equal(f.t, f.context, ctx)
	f.arguments = args
	return f.result, &http.Response{StatusCode: 200, Header: http.Header{"X-Gate-Trace-Id": []string{"unit-trace"}}}, f.callErr
}

func TestRunToolCallBranches(t *testing.T) {
	for _, mode := range []string{"normal", "describe unavailable", "missing schema field", "invalid arguments", "malformed JSON", "nil result", "business error", "transport error", "cancelled", "table"} {
		for _, format := range []output.Format{output.FormatJSON, output.FormatPretty} {
			t.Run(mode+"/"+string(format), func(t *testing.T) {
				ctx, cancel := context.WithCancel(t.Context())
				defer cancel()
				f := &toolCallFixture{t: t, context: ctx, result: &mcpclient.CallResult{StructuredContent: map[string]interface{}{"accepted": true}}}
				cmd := &cobra.Command{Use: "get-coin-info"}
				cmd.SetContext(ctx)
				AddFallbackArgFlags(cmd)
				input := `{"query":"BTC"}`
				wantDescribe, wantCall := 1, 1
				wantError := mode != "normal" && mode != "describe unavailable"
				switch mode {
				case "describe unavailable":
					f.describeErr = errors.New("metadata unavailable")
				case "missing schema field":
					f.schema = map[string]interface{}{"type": "object", "required": []interface{}{"query", "required_fixture"}}
					wantCall = 0
				case "invalid arguments":
					input, wantDescribe, wantCall = `{}`, 0, 0
				case "malformed JSON":
					input, wantDescribe, wantCall = `{"query":`, 0, 0
				case "nil result":
					f.result = nil
				case "business error":
					f.result = &mcpclient.CallResult{IsError: true, StructuredContent: map[string]interface{}{"message": "fixture business rejection"}}
				case "transport error":
					f.callErr = &mcpclient.Error{Kind: mcpclient.ErrorKindTransport, Err: errors.New("fixture transport failure")}
				case "cancelled":
					cancel()
					f.callErr = &mcpclient.Error{Kind: mcpclient.ErrorKindTransport, Err: ctx.Err()}
				case "table":
					format, wantDescribe, wantCall = output.FormatTable, 0, 0
				}
				require.NoError(t, cmd.Flags().Set("args-json", input))
				var out, stderr bytes.Buffer
				printer := output.NewWithStderr(&out, &stderr, format)
				err := RunToolCall(cmd, printer, f, "info coin get-coin-info", ReservedMCPJSONFallbackFlags(), "info", 0)
				require.Equal(t, wantDescribe, f.describe)
				require.Equal(t, wantCall, f.call)
				if wantError {
					require.Error(t, err)
					require.Empty(t, out.String())
					require.NotEmpty(t, stderr.String())
					if format == output.FormatJSON {
						var envelope map[string]json.RawMessage
						require.NoError(t, json.Unmarshal(stderr.Bytes(), &envelope))
						require.NotEmpty(t, envelope["error"])
					}
					if mode == "business error" {
						require.Contains(t, stderr.String(), "fixture business rejection")
						require.Contains(t, stderr.String(), "unit-trace")
					}
					if mode == "nil result" {
						require.Contains(t, stderr.String(), "INTEL_PROTOCOL_ERROR")
					}
					return
				}
				require.NoError(t, err)
				require.Empty(t, stderr.String())
				require.Contains(t, out.String(), "accepted")
				require.Equal(t, map[string]interface{}{"query": "BTC"}, f.arguments)
			})
		}
	}
}
