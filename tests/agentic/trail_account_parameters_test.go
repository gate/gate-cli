//go:build agentic_e2e

package agentic_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/gate/gate-cli/internal/output"
)

// Reuse the historical real-process/HMAC/OAuth fixtures. Only Gate's final
// HTTP peer is replaced; these tests do not place orders in any live account.
func TestTrailParametersCLIWire(t *testing.T) {
	const prefix = "/api/v4/futures/usdt/autoorder/v1/trail/"
	const id = "9007199254740993"
	const order = `{"id":"` + id + `","contract":"SYNTH_USDT","amount":"1","activation_price":"100000","price_offset":"1%","status":"pending"}`
	cases := []futuresConfirmedCase{
		{"history", "GET", prefix + "list", "contract=SYNTH_USDT&end_at=1793538000&is_finished=true&page_size=1&start_at=1793534400", "", `{"code":0,"data":{"orders":[` + order + `]}}`, id, []string{"trail", "list", "--finished", "--start-at", "1793534400", "--end-at", "1793538000", "--contract", "SYNTH_USDT", "--limit", "1"}},
		{"empty history", "GET", prefix + "list", "end_at=2&is_finished=true&start_at=1", "", `{"code":0,"data":{"orders":[]}}`, "", []string{"trail", "list", "--finished", "--start-at", "1", "--end-at", "2"}},
		{"active unchanged", "GET", prefix + "list", "contract=SYNTH_USDT", "", `{"code":0,"data":{"orders":[` + order + `]}}`, id, []string{"trail", "list", "--contract", "SYNTH_USDT", "--limit", "0"}},
	}
	for _, direction := range []string{"omitted", "true", "false"} {
		createArgs := []string{"trail", "create", "--contract", "SYNTH_USDT", "--amount", "1", "--activation-price", "100000", "--price-type", "1", "--price-offset", "1%"}
		updateArgs := []string{"trail", "update", "--id", id, "--price-offset", "1%"}
		createField, updateField := "", ""
		if direction != "omitted" {
			createArgs = append(createArgs, "--is-gte="+direction)
			updateArgs = append(updateArgs, "--is-gte="+direction)
			updateField = `,"is_gte_str":"` + direction + `"`
		}
		if direction == "true" {
			createField = `,"is_gte":true`
		}
		cases = append(cases,
			futuresConfirmedCase{"create " + direction, "POST", prefix + "create", "", `{"contract":"SYNTH_USDT","amount":"1","activation_price":"100000"` + createField + `,"price_type":1,"price_offset":"1%"}`, `{"code":0,"data":{"id":"` + id + `"}}`, id, createArgs},
			futuresConfirmedCase{"update " + direction, "POST", prefix + "update", "", `{"id":` + id + updateField + `,"price_offset":"1%"}`, `{"code":0,"data":{"order":` + order + `}}`, id, updateArgs})
	}
	// Changing direction alone must not require or inject an unrelated field.
	cases = append(cases, futuresConfirmedCase{"direction only", "POST", prefix + "update", "", `{"id":` + id + `,"is_gte_str":"false"}`, `{"code":0,"data":{"order":` + order + `}}`, id, []string{"trail", "update", "--id", id, "--is-gte=false"}})
	// Native bool space forms and a bare true flag must pass through the
	// existing shared argument policy, without adding a second parser here.
	for _, tc := range append([]futuresConfirmedCase(nil), cases...) {
		last := tc.args[len(tc.args)-1]
		if !strings.HasPrefix(last, "--is-gte=") {
			continue
		}
		value := strings.TrimPrefix(last, "--is-gte=")
		spaced := tc
		spaced.name += " spaced"
		spaced.args = append(append([]string(nil), tc.args[:len(tc.args)-1]...), "--is-gte", value)
		cases = append(cases, spaced)
		if value == "true" {
			bare := tc
			bare.name += " bare"
			bare.args = append(append([]string(nil), tc.args[:len(tc.args)-1]...), "--is-gte")
			cases = append(cases, bare)
		}
	}
	for _, auth := range []string{"hmac", "oauth"} {
		for _, tc := range cases {
			t.Run(auth+"/"+tc.name, func(t *testing.T) {
				var calls atomic.Int32
				peer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					futuresConfirmedWire(t, r, tc, auth)
					w.Header().Set("Content-Type", "application/json")
					_, _ = io.WriteString(w, tc.response)
				}))
				t.Cleanup(peer.Close)
				home := futuresConfirmedHome(t, peer.URL, auth)
				for _, format := range []string{"json", "pretty", "table"} {
					out, stderr, err := futuresConfirmedInvoke(t, cliBinary, home, peer.URL, auth, futuresConfirmedArgs(tc, format)...)
					if err != nil || len(stderr) != 0 || calls.Swap(0) != 1 || len(bytes.TrimSpace(out)) == 0 {
						t.Fatalf("%s failed: %v stdout=%s stderr=%s", format, err, out, stderr)
					}
					if tc.identity != "" && !bytes.Contains(out, []byte(tc.identity)) {
						t.Fatalf("large ID lost: %s", out)
					}
					if format == "json" && !json.Valid(out) {
						t.Fatalf("invalid output: %s", out)
					}
					if tc.name == "empty history" && bytes.Contains(out, []byte(id)) {
						t.Fatal("empty history invented an order")
					}
				}
			})
		}
	}
}

func TestTrailParametersCLIRejectsBeforeHTTP(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
	}{
		{"history missing both", []string{"list", "--finished"}},
		{"history missing start", []string{"list", "--finished", "--end-at=2"}},
		{"history missing end", []string{"list", "--finished", "--start-at=1"}},
		{"zero", []string{"list", "--finished", "--start-at=0", "--end-at=2"}},
		{"negative", []string{"list", "--finished", "--start-at=-1", "--end-at=2"}},
		{"negative end", []string{"list", "--finished", "--start-at=1", "--end-at=-2"}},
		{"equal", []string{"list", "--finished", "--start-at=2", "--end-at=2"}},
		{"reverse", []string{"list", "--finished", "--start-at=3", "--end-at=2"}},
		{"milliseconds", []string{"list", "--finished", "--start-at=1793534400000", "--end-at=1793538000000"}},
		{"overflow", []string{"list", "--finished", "--start-at=1", "--end-at=9223372036854775808"}},
		{"non integer", []string{"list", "--finished", "--start-at=1.5", "--end-at=2"}},
		{"active filters", []string{"list", "--start-at=1", "--end-at=2"}},
		{"large limit", []string{"list", "--limit=101"}},
		{"negative limit", []string{"list", "--limit=-1"}},
		{"invalid create direction", []string{"create", "--contract=SYNTH_USDT", "--amount=1", "--price-type=1", "--price-offset=1%", "--is-gte=invalid"}},
		{"invalid update direction", []string{"update", "--id=123", "--is-gte=invalid"}},
		{"create unexpected positional", []string{"create", "--contract=SYNTH_USDT", "--amount=1", "--price-type=1", "--price-offset=1%", "--is-gte=false", "unexpected"}},
		{"update unexpected positional", []string{"update", "--id=123", "--is-gte=false", "unexpected"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls atomic.Int32
			peer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { calls.Add(1); w.WriteHeader(500) }))
			t.Cleanup(peer.Close)
			for _, authenticated := range []bool{false, true} {
				args := append([]string{"--format", "json", "cex", "futures", "trail"}, tc.args...)
				out, stderr, err := runCLI(t, peer.URL, authenticated, args...)
				var exit *exec.ExitError
				var envelope struct{ Error output.GateError }
				decodeErr := json.NewDecoder(bytes.NewReader(stderr)).Decode(&envelope)
				if !errors.As(err, &exit) || exit.ExitCode() != 1 || len(out) != 0 || calls.Load() != 0 || decodeErr != nil || envelope.Error.Label != "INVALID_ARGUMENTS" {
					t.Fatalf("validation/auth ordering wrong: exit=%v calls=%d out=%s err=%s", err, calls.Load(), out, stderr)
				}
			}
		})
	}
}

// Legitimate parameters must not weaken the established business rejection and
// uncertain-write safeguards. Existing suites cover HTTP400/503 on old flags.
func TestTrailParametersCLIFailures(t *testing.T) {
	for _, operation := range []string{"list", "create", "update"} {
		for _, mode := range []string{"unauthenticated", "business", "malformed", "disconnect"} {
			t.Run(operation+"/"+mode, func(t *testing.T) {
				var calls atomic.Int32
				peer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					calls.Add(1)
					if mode == "disconnect" {
						conn, _, err := w.(http.Hijacker).Hijack()
						if err != nil {
							t.Error(err)
							return
						}
						_ = conn.Close()
						return
					}
					w.Header().Set("Content-Type", "application/json")
					if mode == "malformed" {
						_, _ = io.WriteString(w, `{"data":`)
					} else {
						_, _ = io.WriteString(w, `{"code":-1,"label":"TRAIL_REJECTED","message":"fixture rejection","private_debug":"trail-private-canary"}`)
					}
				}))
				t.Cleanup(peer.Close)
				args := []string{"--format", "json", "cex", "futures", "trail", operation}
				switch operation {
				case "list":
					args = append(args, "--finished", "--start-at=1", "--end-at=2")
				case "create":
					args = append(args, "--contract=SYNTH_USDT", "--amount=1", "--price-offset=1%", "--price-type=1", "--is-gte=true")
				case "update":
					args = append(args, "--id=123", "--is-gte=false")
				}
				out, stderr, err := runCLI(t, peer.URL, mode != "unauthenticated", args...)
				wantCalls := int32(1)
				if mode == "unauthenticated" {
					wantCalls = 0
				}
				var envelope struct{ Error output.GateError }
				if err == nil || len(out) != 0 || calls.Load() != wantCalls || json.Unmarshal(stderr, &envelope) != nil || bytes.Contains(stderr, []byte("trail-private-canary")) {
					t.Fatalf("failure hidden/replayed/leaked: %v calls=%d out=%s err=%s", err, calls.Load(), out, stderr)
				}
				if operation != "list" && mode != "unauthenticated" && envelope.Error.Retryable {
					t.Fatalf("unsafe write retry: %s", stderr)
				}
			})
		}
	}
}

func TestAccountPreconditionCLI(t *testing.T) {
	for _, tc := range []struct {
		name, path, kind string
		args             []string
	}{
		{"futures account", "/api/v4/futures/usdt/accounts", "ACCOUNT_NOT_READY", []string{"futures", "account", "get"}},
		{"futures orders", "/api/v4/futures/usdt/orders", "ACCOUNT_NOT_READY", []string{"futures", "order", "list", "--status=open"}},
		{"delivery account", "/api/v4/delivery/usdt/accounts", "ACCOUNT_NOT_READY", []string{"delivery", "account", "get"}},
		{"delivery orders", "/api/v4/delivery/usdt/orders", "ACCOUNT_NOT_READY", []string{"delivery", "order", "list", "--status=open"}},
		{"options unknown prerequisite", "/api/v4/options/accounts", "UNKNOWN", []string{"options", "account", "get"}},
	} {
		for _, message := range []string{"please transfer funds first to create futures account", "", "账户未初始化"} {
			for _, format := range []string{"json", "pretty", "table"} {
				t.Run(tc.name+"/"+message+"/"+format, func(t *testing.T) {
					var calls atomic.Int32
					peer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						calls.Add(1)
						if r.Method != "GET" || r.URL.Path != tc.path || r.Header.Get("SIGN") == "" {
							t.Errorf("incorrect signed request: %s %s", r.Method, r.URL)
						}
						w.Header().Set("Content-Type", "application/json")
						w.Header().Set("X-Gate-Trace-Id", "account-fixture-trace")
						w.WriteHeader(400)
						_ = json.NewEncoder(w).Encode(map[string]string{"label": "USER_NOT_FOUND", "message": message})
					}))
					t.Cleanup(peer.Close)
					args := append([]string{"--format", format, "cex"}, tc.args...)
					out, stderr, err := runCLI(t, peer.URL, true, args...)
					var exit *exec.ExitError
					if !errors.As(err, &exit) || exit.ExitCode() != 1 || len(out) != 0 || calls.Load() != 1 || bytes.Contains(stderr, []byte("fix flags")) {
						t.Fatalf("account rejection hidden/retried: %v calls=%d out=%s err=%s", err, calls.Load(), out, stderr)
					}
					if format == "json" {
						var envelope struct{ Error output.GateError }
						if json.Unmarshal(stderr, &envelope) != nil {
							t.Fatalf("invalid error: %s", stderr)
						}
						ge := envelope.Error
						if ge.ErrorType != tc.kind || ge.Retryable || ge.Status != 400 || ge.Label != "USER_NOT_FOUND" || ge.Message != message || ge.TraceID != "account-fixture-trace" || ge.Request == nil || ge.Request.URL != tc.path {
							t.Fatalf("lost account diagnostic: %+v", ge)
						}
					} else if !strings.Contains(string(stderr), tc.kind) || !strings.Contains(string(stderr), "USER_NOT_FOUND") || !strings.Contains(string(stderr), message) {
						t.Fatalf("missing human diagnostic: %s", stderr)
					}
				})
			}
		}
	}
}

// Exit/status precedence is exercised end to end, not just in the classifier.
func TestAccountPreconditionCLIStatusPrecedence(t *testing.T) {
	for _, status := range []int{401, 403, 404, 429, 500, 503} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			peer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(status)
				_, _ = io.WriteString(w, `{"label":"USER_NOT_FOUND","message":"user not found"}`)
			}))
			t.Cleanup(peer.Close)
			out, stderr, err := runCLI(t, peer.URL, true, "--format=json", "cex", "futures", "account", "get")
			wantExit, kind := 1, "NETWORK_ERROR"
			switch status {
			case 401:
				wantExit, kind = 4, "AUTH_ERROR"
			case 403:
				wantExit, kind = 4, "PERMISSION_DENIED"
			case 404:
				kind = "API_NOT_FOUND"
			}
			var exit *exec.ExitError
			var envelope struct{ Error output.GateError }
			if !errors.As(err, &exit) || exit.ExitCode() != wantExit || len(out) != 0 || json.Unmarshal(stderr, &envelope) != nil || envelope.Error.ErrorType != kind || envelope.Error.Retryable {
				t.Fatalf("status precedence changed: exit=%v out=%s err=%s", err, out, stderr)
			}
		})
	}
}
