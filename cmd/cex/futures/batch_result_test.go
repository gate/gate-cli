package futures

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/url"
	"testing"

	"github.com/gate/gate-cli/internal/output"
)

type brokenBatchWriter struct{}

func (brokenBatchWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

// Only decoded false changes the exit outcome; this deliberately does not copy
// Spot's requested-count policy into the Futures contract.
func TestCEXCompatBatchResults(t *testing.T) {
	for _, items := range [][]bool{nil, {}, {true}, {true, true}, {false}, {true, false}} {
		var out bytes.Buffer
		response := &http.Response{StatusCode: 200, Header: http.Header{"X-Gate-Trace-Id": []string{"synthetic"}}, Request: &http.Request{URL: &url.URL{Path: "/api/v4/futures/usdt/batch_orders"}}}
		err := printFuturesBatchResult(output.New(&out, "json"), items, response, func(v bool) bool { return v })
		failed := false
		for _, v := range items {
			failed = failed || !v
		}
		if (err != nil) != failed || out.Len() == 0 {
			t.Fatalf("items=%v err=%v output=%s", items, err, out.String())
		}
		if failed {
			var ge *output.GateError
			if !errors.As(err, &ge) {
				t.Fatal(err)
			}
			output.FillAgentErrorConvergence(ge)
			if ge.Retryable || ge.Label != "BATCH_ORDER_FAILED" || ge.Request.Method != "POST" {
				t.Fatalf("unsafe convergence %+v", ge)
			}
		}
	}
	err := printFuturesBatchResult(output.New(brokenBatchWriter{}, "json"), []bool{false}, nil, func(v bool) bool { return v })
	if !errors.Is(err, io.ErrClosedPipe) {
		t.Fatalf("writer error lost %v", err)
	}
}
