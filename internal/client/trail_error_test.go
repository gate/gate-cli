package client

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/gate/gate-cli/internal/exitcode"
	"github.com/gate/gate-cli/internal/operations"
	"github.com/gate/gate-cli/internal/output"
	gateapi "github.com/gate/gateapi-go/v7"
	"github.com/stretchr/testify/require"
)

// A missing HTTP response must not prevent operation-level diagnostics; a
// previously classified multi-step failure must keep its original provenance.
func TestTrailCommandError(t *testing.T) {
	for _, settle := range []string{"usdt", "btc", "a b", "a/b%"} {
		err := TrailCommandError(errors.New("fixture"), nil, "stopTrailOrder", settle, `{"id":1}`)
		var detail *output.GateError
		require.True(t, errors.As(err, &detail))
		require.Equal(t, "POST", detail.Request.Method)
		require.Equal(t, "/api/v4/futures/"+url.QueryEscape(settle)+"/autoorder/v1/trail/stop", detail.Request.URL)
		require.Equal(t, `{"id":1}`, detail.Request.Body)
	}
	original := exitcode.New(4, &output.GateError{Status: 403, Label: "ORIGINAL", Message: "original", Request: &output.RequestInfo{Method: "GET", URL: "/original"}})
	require.Same(t, original, TrailCommandError(original, nil, "createTrailOrder", "usdt", ""))
}

func TestTrailCommandErrorMetadataFailure(t *testing.T) {
	for _, status := range []int{200, 400, 401, 403, 503} {
		for _, resolve := range []func(string) (operations.Operation, error){
			func(string) (operations.Operation, error) {
				return operations.Operation{}, fmt.Errorf("private-resolver-canary")
			},
			func(string) (operations.Operation, error) { return operations.Operation{}, nil },
		} {
			response := &http.Response{StatusCode: status, Header: http.Header{"X-Gate-Trace-Id": []string{"original-trace"}}}
			err := trailCommandError(gateapi.GateAPIError{Label: "BAD_REQUEST", Message: "original"}, response, "createTrailOrder", "usdt", "", resolve)
			var detail *output.GateError
			require.True(t, errors.As(err, &detail))
			require.Equal(t, status, detail.Status)
			require.Equal(t, "BAD_REQUEST", detail.Label)
			require.Equal(t, "original", detail.Message)
			require.Equal(t, "original-trace", detail.TraceID)
			require.Nil(t, detail.Request)
			output.FillAgentErrorConvergence(detail)
			require.False(t, detail.Retryable)
			require.NotContains(t, detail.Message+detail.SuggestedNextAction, "private-resolver-canary")
			var coded *exitcode.Error
			require.True(t, errors.As(err, &coded))
			want := 1
			if status == 401 || status == 403 {
				want = 4
			}
			require.Equal(t, want, coded.Code)
		}
	}
	err := TrailCommandError(exitcode.New(4, errors.New("original auth")), nil, "not-a-trail-operation", "usdt", "")
	var coded *exitcode.Error
	require.True(t, errors.As(err, &coded))
	require.Equal(t, 4, coded.Code)
}
