//go:build agentic_e2e

package agentic_test

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/gate/gate-cli/internal/output"
)

// Only the API peer is mocked; real CLI/SDK/error output must preserve the observed rejections.
func TestAlphaBusinessErrorCLI(t *testing.T) {
	for _, tc := range []struct {
		command, flag, value, label, message string
	}{
		{"tickers", "--currency", "USDS", "BAD_REQUEST", "meme token not found"},
		{"currencies", "--currency", "USDS", "BAD_REQUEST", "meme token not found"},
		{"tokens", "--chain", "ETH", "INVALID_ARGUMENT", "json: cannot unmarshal object into Go value of type []*types.AlphaV4CurrencyItem"},
	} {
		for _, format := range []string{"json", "table"} {
			t.Run(tc.command+"/"+format, func(t *testing.T) {
				var calls atomic.Int32
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					if r.Method != "GET" || r.URL.Path != "/api/v4/alpha/"+tc.command || r.URL.Query().Get(strings.TrimPrefix(tc.flag, "--")) != tc.value {
						t.Errorf("request=%s %s", r.Method, r.URL)
					}
					w.Header().Set("Content-Type", "application/json")
					w.Header().Set("X-Gate-Trace-Id", "alpha-business-trace")
					if err := json.NewEncoder(w).Encode(map[string]string{"Label": tc.label, "Message": tc.message, "private_debug": "secret-sentinel"}); err != nil {
						t.Error(err)
					}
				}))
				t.Cleanup(server.Close)
				stdout, stderr, err := runCLI(t, server.URL, false, "cex", "alpha", "market", tc.command, tc.flag, tc.value, "--format", format)
				var exit *exec.ExitError
				if !errors.As(err, &exit) || exit.ExitCode() != 1 || len(stdout) != 0 || calls.Load() != 1 {
					t.Fatalf("exit=%v stdout=%s calls=%d stderr=%s", err, stdout, calls.Load(), stderr)
				}
				if strings.Contains(string(stderr), "secret-sentinel") || strings.Contains(string(stderr), "[]gateapi.") {
					t.Fatalf("lost safe business diagnostic: %s", stderr)
				}
				if format == "json" {
					var envelope struct{ Error output.GateError }
					if err := json.Unmarshal(stderr, &envelope); err != nil {
						t.Fatalf("error envelope: %v %s", err, stderr)
					}
					ge := envelope.Error
					if ge.Status != 200 || ge.Label != tc.label || ge.Message != tc.message || ge.TraceID != "alpha-business-trace" || ge.ErrorType != "INVALID_ARGS" || !ge.Retryable || !strings.Contains(ge.SuggestedNextAction, "corrected args") {
						t.Fatalf("business error=%+v", ge)
					}
				} else {
					for _, want := range []string{tc.label, tc.message, "200", "INVALID_ARGS", "alpha-business-trace", "corrected args"} {
						if !strings.Contains(string(stderr), want) {
							t.Errorf("missing %q in %s", want, stderr)
						}
					}
				}
			})
		}
	}
}

// Bad DTOs remain failures, and accepting an error envelope never prints unrelated data.
func TestAlphaResponseBoundariesCLI(t *testing.T) {
	for _, tc := range []struct {
		name, body, label string
	}{
		{"normal", `[]`, ""},
		{"private-dto", `{"private_debug":"secret-sentinel"}`, "INVALID_RESPONSE"},
		{"html", `<html>secret-sentinel</html>`, "INVALID_RESPONSE"},
		{"duplicate", `{"label":"BAD_REQUEST","Label":"OTHER","message":"bad","private_debug":"secret-sentinel"}`, "INVALID_RESPONSE"},
		{"null", `{"label":"BAD_REQUEST","message":null,"private_debug":"secret-sentinel"}`, "INVALID_RESPONSE"},
		{"trailing", `{"label":"BAD_REQUEST","message":"bad"} {"private_debug":"secret-sentinel"}`, "INVALID_RESPONSE"},
		{"unknown-label", `{"label":"UPSTREAM_OTHER","message":"original rejection","private_debug":"secret-sentinel"}`, "UPSTREAM_OTHER"},
	} {
		for _, format := range []string{"json", "table"} {
			t.Run(tc.name+"/"+format, func(t *testing.T) {
				var calls atomic.Int32
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					w.Header().Set("Content-Type", "application/json")
					if _, err := io.WriteString(w, tc.body); err != nil {
						t.Error(err)
					}
				}))
				t.Cleanup(server.Close)
				stdout, stderr, err := runCLI(t, server.URL, false, "cex", "alpha", "market", "tickers", "--format", format)
				if calls.Load() != 1 || strings.Contains(string(stderr), "secret-sentinel") {
					t.Fatalf("calls=%d stderr=%s", calls.Load(), stderr)
				}
				if tc.label == "" {
					if err != nil || len(stderr) != 0 || (format == "json" && strings.TrimSpace(string(stdout)) != "[]") {
						t.Fatalf("normal array exit=%v stdout=%s stderr=%s", err, stdout, stderr)
					}
				} else if err == nil || len(stdout) != 0 || !strings.Contains(string(stderr), tc.label) {
					t.Fatalf("exit=%v stdout=%s stderr=%s", err, stdout, stderr)
				}
			})
		}
	}
}

// An array-returning write exercises genuine SDK decode failure; no real write leaves loopback.
func TestBusinessErrorWriteCLI(t *testing.T) {
	for _, status := range []int{200, 503} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.Method != "POST" || r.URL.Path != "/api/v4/spot/cancel_batch_orders" {
					t.Errorf("unexpected write: %s %s", r.Method, r.URL)
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(status)
				if _, err := io.WriteString(w, `{"Label":"BAD_REQUEST","Message":"meme token not found"}`); err != nil {
					t.Error(err)
				}
			}))
			t.Cleanup(server.Close)
			stdout, stderr, err := runCLI(t, server.URL, true, "--format", "json", "cex", "spot", "order", "batch-cancel", "--orders-json", `[{"currency_pair":"SYNTH_USDT","id":"123"}]`)
			var envelope struct{ Error output.GateError }
			if err == nil || len(stdout) != 0 || calls.Load() != 1 || json.Unmarshal(stderr, &envelope) != nil {
				t.Fatalf("exit=%v stdout=%s calls=%d stderr=%s", err, stdout, calls.Load(), stderr)
			}
			ge := envelope.Error
			if ge.Status != status || ge.Label != "BAD_REQUEST" || ge.Retryable || !strings.Contains(ge.Message, "outcome unknown") || !strings.Contains(ge.SuggestedNextAction, "query the original") {
				t.Fatalf("unsafe write error: %+v", ge)
			}
			if status == 200 && ge.ErrorType != "INVALID_ARGS" {
				t.Fatalf("business classification lost: %+v", ge)
			}
		})
	}
}
