package spot

import (
	"fmt"
	"net/http"

	"github.com/gate/gate-cli/internal/output"
)

// printSpotBatchResult preserves the SDK result array on stdout. HTTP success
// alone does not prove any item succeeded; a failed/partial batch must also fail
// the process without replaying writes or hiding successful order identifiers.
func printSpotBatchResult[T any](p *output.Printer, results []T, requested int, response *http.Response, succeeded func(T) bool) error {
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
	label := "BATCH_ORDER_FAILED"
	message := fmt.Sprintf("batch result: %d succeeded, %d failed of %d orders; inspect per-order results", successful, failed, requested)
	// Missing/extra results cannot establish the outcome of the requested batch.
	// Keep the returned facts and require reconciliation rather than inventing success.
	if requested <= 0 || len(results) != requested {
		label = "INVALID_RESPONSE"
		message = fmt.Sprintf("batch response incomplete: requested %d, returned %d (%d succeeded, %d failed); reconcile order outcomes", requested, len(results), successful, failed)
	} else if failed == 0 {
		return nil
	}
	detail := &output.GateError{
		Status:              http.StatusOK,
		Label:               label,
		Message:             message,
		Retryable:           false,
		SuggestedNextAction: "inspect per-order results and reconcile missing outcomes; do not retry the entire batch",
	}
	if response != nil {
		detail.Status = response.StatusCode
		detail.TraceID = response.Header.Get("X-Gate-Trace-ID")
		detail.RequestID = response.Header.Get("X-Request-ID")
	}
	return detail
}
