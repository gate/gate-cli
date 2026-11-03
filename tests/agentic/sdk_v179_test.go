//go:build agentic_e2e

package agentic_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// Actual process/flags/auth/SDK/output remain real; only the final service is
// synthetic. The shared fixture follows an actual response structure, with
// invented IDs, account, times and suborder text; it is not a raw service response.
func TestSDK179CLITrailStopAll(t *testing.T) {
	fixture, err := os.ReadFile(filepath.Join("..", "..", "internal", "client", "testdata", "trail_stop_all.json"))
	if err != nil {
		t.Fatal(err)
	}
	const route = "/api/v4/futures/usdt/autoorder/v1/trail/stop_all"
	for _, auth := range []string{"hmac", "oauth"} {
		for _, format := range []string{"json", "pretty", "table"} {
			for _, mode := range []string{"synthetic-cancellation", "large-id", "empty", "business", "400", "503", "malformed", "unauthenticated"} {
				t.Run(auth+"/"+format+"/"+mode, func(t *testing.T) {
					var calls atomic.Int32
					peer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						calls.Add(1)
						futuresConfirmedWire(t, r, futuresConfirmedCase{method: "POST", path: route, body: `{"contract":"BTC_USDT"}`}, auth)
						w.Header().Set("Content-Type", "application/json")
						w.Header().Set("X-Gate-Trace-Id", "sdk179-trace")
						raw := string(fixture)
						switch mode {
						case "large-id":
							raw = strings.ReplaceAll(raw, `"71001"`, `"9007199254740993"`)
						case "empty":
							raw = `{"code":0,"message":"ok","data":{"orders":[]},"timestamp":1700000001837}`
						case "business":
							raw = `{"code":1077,"message":"fixture rejection","data":{},"private_debug":"sdk179-private-canary"}`
						case "malformed":
							raw = `{"data":`
						case "400", "503":
							status := 400
							if mode == "503" {
								status = 503
							}
							w.WriteHeader(status)
							raw = `{"label":"TRAIL_REJECTED","message":"fixture rejection","private_debug":"sdk179-private-canary"}`
						}
						_, _ = io.WriteString(w, raw)
					}))
					t.Cleanup(peer.Close)
					selectedAuth := auth
					if mode == "unauthenticated" {
						selectedAuth = "none"
					}
					home := futuresConfirmedHome(t, peer.URL, selectedAuth)
					out, stderr, callErr := futuresConfirmedInvoke(t, cliBinary, home, peer.URL, selectedAuth,
						"--format", format, "cex", "futures", "trail", "stop-all", "--contract", "BTC_USDT")
					wantCalls := int32(1)
					if mode == "unauthenticated" {
						wantCalls = 0
					}
					if calls.Load() != wantCalls {
						t.Fatalf("write requests=%d want=%d", calls.Load(), wantCalls)
					}
					failure := mode != "synthetic-cancellation" && mode != "large-id" && mode != "empty"
					if (callErr != nil) != failure {
						t.Fatalf("exit=%v stdout=%s stderr=%s", callErr, out, stderr)
					}
					if bytes.Contains(out, []byte("sdk179-private-canary")) || bytes.Contains(stderr, []byte("sdk179-private-canary")) {
						t.Fatal("private fields leaked")
					}
					if failure {
						if len(bytes.TrimSpace(out)) != 0 || len(bytes.TrimSpace(stderr)) == 0 {
							t.Fatalf("invalid failure streams: %s / %s", out, stderr)
						}
						if format == "json" && !json.Valid(stderr) {
							t.Fatalf("invalid error JSON: %s", stderr)
						}
						if mode == "business" || mode == "400" || mode == "503" {
							if !bytes.Contains(stderr, []byte("fixture rejection")) {
								t.Fatalf("backend message lost: %s", stderr)
							}
						}
						if format == "json" && mode != "unauthenticated" {
							var failure struct {
								Error struct {
									Retryable *bool  `json:"retryable"`
									ErrorType string `json:"error_type"`
									Action    string `json:"suggested_next_action"`
								} `json:"error"`
							}
							if err := json.Unmarshal(stderr, &failure); err != nil || failure.Error.Retryable == nil {
								t.Fatalf("missing error convergence: %s %v", stderr, err)
							}
							// Preserve the established corrected-arguments policy for
							// explicit HTTP400 rejection; uncertain writes stay nonretryable.
							if *failure.Error.Retryable != (mode == "400") {
								t.Fatalf("write retry policy changed: %s", stderr)
							}
							if mode == "400" && (failure.Error.ErrorType != "INVALID_ARGS" || !strings.Contains(failure.Error.Action, "corrected args")) {
								t.Fatalf("HTTP400 correction guidance changed: %s", stderr)
							}
						}
						return
					}
					if len(bytes.TrimSpace(stderr)) != 0 {
						t.Fatalf("success wrote stderr: %s", stderr)
					}
					var result struct {
						Message   string `json:"message"`
						Timestamp int64  `json:"timestamp"`
						Data      struct {
							Orders []struct {
								ID         string `json:"id"`
								Amount     string `json:"amount"`
								StatusCode string `json:"status_code"`
							} `json:"orders"`
						} `json:"data"`
					}
					if err := json.Unmarshal(out, &result); err != nil {
						t.Fatal(err)
					}
					wantTimestamp := int64(1700000001827)
					if mode == "empty" {
						wantTimestamp = 1700000001837
					}
					if result.Message != "ok" || result.Timestamp != wantTimestamp {
						t.Fatalf("envelope lost: %s", out)
					}
					if mode == "empty" {
						if len(result.Data.Orders) != 0 {
							t.Fatalf("invented orders: %s", out)
						}
						return
					}
					wantID := "71001"
					if mode == "large-id" {
						wantID = "9007199254740993"
					}
					if len(result.Data.Orders) != 1 || result.Data.Orders[0].ID != wantID || result.Data.Orders[0].StatusCode != "cancelled" || result.Data.Orders[0].Amount != "0.01" {
						t.Fatalf("cancelled order lost: %s", out)
					}
				})
			}
		}
	}
}

// A killed process proves child/connection ownership, not application-level
// cancellation JSON. The component suite separately checks context causes.
func TestSDK179CLITrailStopAllProcessCancellation(t *testing.T) {
	var calls atomic.Int32
	entered := make(chan struct{}, 1)
	peer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		// Drain the body before waiting for the child connection to close.
		_, _ = io.Copy(io.Discard, r.Body)
		w.WriteHeader(200)
		w.(http.Flusher).Flush()
		entered <- struct{}{}
		<-r.Context().Done()
	}))
	t.Cleanup(peer.Close)
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	child := exec.CommandContext(ctx, cliBinary, "--format", "json", "cex", "futures", "trail", "stop-all", "--contract", "BTC_USDT")
	child.Env = childEnv(t.TempDir(), peer.URL, true)
	if err := child.Start(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-entered:
	case <-ctx.Done():
		cancel()
		_ = child.Wait()
		t.Fatal("write never reached peer")
	}
	cancel()
	if err := child.Wait(); err == nil || calls.Load() != 1 {
		t.Fatalf("unsafe cancellation: calls=%d err=%v", calls.Load(), err)
	}
}
