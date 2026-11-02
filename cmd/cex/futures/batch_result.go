package futures

import (
	"fmt"
	"net/http"

	"github.com/gate/gate-cli/internal/output"
)

// printFuturesBatchResult keeps all returned facts before reporting item failures.
// A successful HTTP write is not safe to replay; do not infer missing outcomes or
// import Spot's different requested-count contract here.
func printFuturesBatchResult[T any](p *output.Printer, results []T, response *http.Response, succeeded func(T) bool) error {
	if err := p.Print(results); err != nil {
		return err
	}
	successful := 0
	for _, item := range results {
		if succeeded(item) {
			successful++
		}
	}
	failed := len(results) - successful
	if failed == 0 {
		return nil
	}
	detail := &output.GateError{Status: http.StatusOK, Label: "BATCH_ORDER_FAILED", Message: fmt.Sprintf("batch result: %d succeeded, %d failed; inspect per-order results and successful IDs", successful, failed), SuggestedNextAction: "inspect per-order results and reconcile outcomes; do not retry the entire batch", Request: &output.RequestInfo{Method: http.MethodPost}}
	if response != nil {
		detail.Status = response.StatusCode
		detail.TraceID = response.Header.Get("X-Gate-Trace-ID")
		detail.RequestID = response.Header.Get("X-Request-ID")
		if response.Request != nil && response.Request.URL != nil {
			detail.Request.URL = response.Request.URL.EscapedPath()
		}
	}
	return detail
}
