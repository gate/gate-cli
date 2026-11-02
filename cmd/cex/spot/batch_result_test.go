package spot

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/gate/gate-cli/internal/output"
	gateapi "github.com/gate/gateapi-go/v7"
)

// TestSpotBatchResults checks business failures separately from a successful HTTP
// exchange and preserves the SDK's complete array for reconciliation.
func TestSpotBatchResults(t *testing.T) {
	good := gateapi.BatchOrder{Succeeded: true, Id: "9223372036854775807"}
	bad := gateapi.BatchOrder{Label: "INVALID_PARAM_VALUE", Message: "size too small"}
	cases := []struct {
		name          string
		results       []gateapi.BatchOrder
		requested     int
		label, counts string
	}{
		{"success", []gateapi.BatchOrder{good, good}, 2, "", ""},
		{"one success", []gateapi.BatchOrder{good}, 1, "", ""},
		{"partial", []gateapi.BatchOrder{good, bad}, 2, "BATCH_ORDER_FAILED", "1 succeeded, 1 failed"},
		{"all failed", []gateapi.BatchOrder{bad, bad}, 2, "BATCH_ORDER_FAILED", "0 succeeded, 2 failed"},
		{"false without label", []gateapi.BatchOrder{{}}, 1, "BATCH_ORDER_FAILED", "0 succeeded, 1 failed"},
		{"contradictory", []gateapi.BatchOrder{{Succeeded: true, Label: "REJECTED"}}, 1, "BATCH_ORDER_FAILED", "0 succeeded, 1 failed"},
		{"nil response", nil, 1, "INVALID_RESPONSE", "returned 0"},
		{"empty response", []gateapi.BatchOrder{}, 1, "INVALID_RESPONSE", "returned 0"},
		{"missing response", []gateapi.BatchOrder{good}, 2, "INVALID_RESPONSE", "requested 2, returned 1"},
		{"extra response", []gateapi.BatchOrder{good, good}, 1, "INVALID_RESPONSE", "requested 1, returned 2"},
		{"empty request", nil, 0, "INVALID_RESPONSE", "requested 0"},
		{"invalid count", nil, -1, "INVALID_RESPONSE", "requested -1"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var stdout bytes.Buffer
			response := &http.Response{StatusCode: 200, Header: make(http.Header)}
			response.Header.Set("X-Gate-Trace-ID", "trace-fixture")
			response.Header.Set("X-Request-ID", "request-fixture")
			err := printSpotBatchResult(output.New(&stdout, output.FormatJSON), tc.results, tc.requested, response,
				func(item gateapi.BatchOrder) bool { return item.Succeeded && item.Label == "" })
			want, marshalErr := json.Marshal(tc.results)
			if marshalErr != nil {
				t.Fatal(marshalErr)
			}
			if stdout.String() != string(want)+"\n" {
				t.Errorf("SDK array changed: %s", &stdout)
			}
			if tc.label == "" {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			var detail *output.GateError
			if !errors.As(err, &detail) {
				t.Fatalf("expected business error, got %v", err)
			}
			if detail.Label != tc.label || !strings.Contains(detail.Message, tc.counts) {
				t.Errorf("unexpected error: %+v", detail)
			}
			if detail.Status != 200 || detail.Retryable || detail.TraceID != "trace-fixture" || detail.RequestID != "request-fixture" {
				t.Errorf("response/retry metadata changed: %+v", detail)
			}
		})
	}
}

// The cancellation SDK uses a separate result type, with the same business contract.
func TestSpotBatchCancelResult(t *testing.T) {
	var stdout bytes.Buffer
	err := printSpotBatchResult(output.New(&stdout, output.FormatJSON),
		[]gateapi.CancelOrderResult{{Id: "kept-id", Succeeded: true}, {Id: "bad-id", Label: "ORDER_NOT_FOUND"}}, 2, nil,
		func(item gateapi.CancelOrderResult) bool { return item.Succeeded && item.Label == "" })
	var detail *output.GateError
	if !errors.As(err, &detail) || detail.Status != 200 || !strings.Contains(detail.Message, "1 succeeded, 1 failed") {
		t.Fatalf("unexpected cancellation outcome: %v", err)
	}
	if !strings.Contains(stdout.String(), "kept-id") || !strings.Contains(stdout.String(), "ORDER_NOT_FOUND") {
		t.Fatal("SDK cancellation facts lost")
	}
}

type batchBrokenWriter struct{ err error }

func (w batchBrokenWriter) Write(_ []byte) (int, error) { return 0, w.err }

// Output failures must not be swallowed after orders may already have been submitted.
func TestSpotBatchOutputFailure(t *testing.T) {
	want := errors.New("output unavailable")
	err := printSpotBatchResult(output.New(batchBrokenWriter{want}, output.FormatJSON),
		[]gateapi.BatchOrder{{Succeeded: true}}, 1, nil, func(item gateapi.BatchOrder) bool { return item.Succeeded })
	if !errors.Is(err, want) {
		t.Fatalf("output failure lost: %v", err)
	}
}
