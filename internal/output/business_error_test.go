package output

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

// Validation labels on 2xx responses outrank message heuristics without changing HTTP errors.
func TestBusinessErrorClassification(t *testing.T) {
	for _, label := range []string{"BAD_REQUEST", "INVALID_ARGUMENT", "INVALID_ARGUMENTS", "INVALID_PARAM", "INVALID_PARAM_VALUE"} {
		require.Equal(t, "INVALID_ARGS", ClassifyCLIError(200, label, "meme token not found"))
	}
	for _, tc := range []struct {
		status         int
		label, message string
		want           string
	}{
		{200, "OTHER", "rejection", "UNKNOWN"}, {200, "OTHER", "not found", "EMPTY_RESULT"},
		{401, "BAD_REQUEST", "bad", "AUTH_ERROR"}, {403, "BAD_REQUEST", "bad", "PERMISSION_DENIED"},
		{404, "BAD_REQUEST", "not found", "API_NOT_FOUND"}, {429, "BAD_REQUEST", "bad", "NETWORK_ERROR"},
		{503, "INVALID_ARGUMENT", "bad", "NETWORK_ERROR"},
	} {
		require.Equal(t, tc.want, ClassifyCLIError(tc.status, tc.label, tc.message))
	}
}

// A caller may correct read arguments, but a failed write with uncertain outcome must be reconciled.
func TestBusinessErrorRetrySafety(t *testing.T) {
	for _, method := range []string{"GET", "HEAD", "POST", "PUT", "DELETE"} {
		for _, status := range []int{200, 299, 500, 503} {
			for _, action := range []string{"", "reconcile using existing reference"} {
				t.Run(fmt.Sprintf("%s/%d/%s", method, status, action), func(t *testing.T) {
					ge := &GateError{Status: status, Label: "BAD_REQUEST", Message: "meme token not found", Request: &RequestInfo{Method: method}, SuggestedNextAction: action}
					FillAgentErrorConvergence(ge)
					read := method == "GET" || method == "HEAD"
					require.Equal(t, read && status < 300, ge.Retryable)
					if action != "" {
						require.Equal(t, action, ge.SuggestedNextAction)
					} else if !read {
						require.Contains(t, ge.SuggestedNextAction, "query the original")
					} else if status < 300 {
						require.Contains(t, ge.SuggestedNextAction, "corrected args")
					}
					before := *ge
					FillAgentErrorConvergence(ge)
					require.Equal(t, before, *ge, "convergence must be idempotent")
				})
			}
		}
	}
	// A caller that lacks a request method retains the established classification contract.
	for _, request := range []*RequestInfo{nil, {}} {
		ge := &GateError{Status: 200, Label: "BAD_REQUEST", Request: request}
		FillAgentErrorConvergence(ge)
		require.True(t, ge.Retryable)
	}
}
