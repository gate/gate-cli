//go:build agentic_e2e

package agentic_test

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gate/gate-cli/internal/agentfeature"
)

// Independent wire fixtures, never derived from the implementation's catalog.
// Trail stop/update/log use the published v178 envelopes and quoted log time.
type futuresConfirmedCase struct {
	name, method, path, query, body, response, identity string
	args                                                []string
}

func futuresConfirmedCases() []futuresConfirmedCase {
	const id = "9007199254740993"
	prefix := "/api/v4/futures/usdt/autoorder/v1/trail/"
	return []futuresConfirmedCase{
		{"trail-create", "POST", prefix + "create", "", `{"contract":"SYNTH_USDT","amount":"1","activation_price":"0","price_type":1,"price_offset":"1%"}`, `{"code":0,"message":"ok","data":{"id":"` + id + `"}}`, id, []string{"trail", "create", "--contract", "SYNTH_USDT", "--amount", "1", "--price-offset", "1%", "--price-type", "1"}},
		{"trail-stop", "POST", prefix + "stop", "", `{"id":` + id + `}`, `{"code":0,"message":"ok","data":{"order":{"id":"` + id + `","contract":"SYNTH_USDT"}}}`, id, []string{"trail", "stop", "--id", id}},
		{"trail-stop-all", "POST", prefix + "stop_all", "", `{"contract":"SYNTH_USDT"}`, `{"orders":[{"id":"` + id + `","contract":"SYNTH_USDT"}]}`, id, []string{"trail", "stop-all", "--contract", "SYNTH_USDT"}},
		{"trail-list", "GET", prefix + "list", "contract=SYNTH_USDT&end_at=1793538000&is_finished=true&page_size=2&start_at=1793534400", "", `{"code":0,"message":"ok","data":{"orders":[{"id":"` + id + `","contract":"SYNTH_USDT"}]},"timestamp":1793582374966743}`, id, []string{"trail", "list", "--contract", "SYNTH_USDT", "--finished", "--start-at", "1793534400", "--end-at", "1793538000", "--limit", "2"}},
		{"trail-get", "GET", prefix + "detail", "id=" + id, "", `{"code":0,"message":"ok","data":{"order":{"id":"` + id + `","contract":"SYNTH_USDT"}}}`, id, []string{"trail", "get", "--id", id}},
		{"trail-update", "POST", prefix + "update", "", `{"id":` + id + `,"amount":"2"}`, `{"code":0,"message":"ok","data":{"order":{"id":"` + id + `","contract":"SYNTH_USDT","amount":"2"}}}`, id, []string{"trail", "update", "--id", id, "--amount", "2"}},
		{"trail-log", "GET", prefix + "change_log", "id=" + id + "&page_num=1&page_size=2", "", `{"code":0,"message":"ok","data":{"change_log":[{"updated_at":"1793527722","amount":"2"}]}}`, "1793527722", []string{"trail", "log", "--id", id, "--limit", "2"}},
		{"price-create", "POST", "/api/v4/futures/usdt/price_orders", "", `{"initial":{"contract":"SYNTH_USDT","size":-10,"price":"0"},"trigger":{"price":"90000","rule":1}}`, `{"id":` + id + `,"id_string":"` + id + `"}`, id, []string{"price-trigger", "create", "--contract", "SYNTH_USDT", "--trigger-price", "90000", "--price", "0", "--size", "-10"}},
		{"price-update", "PUT", "/api/v4/futures/usdt/price_orders/amend/pending-2147483648", "", `{"order_id":` + id + `,"price":"10"}`, `{"id":` + id + `,"id_string":"` + id + `"}`, id, []string{"price-trigger", "update", "--id", "pending-2147483648", "--body-order-id", id, "--price", "10"}},
	}
}

func futuresConfirmedArgs(tc futuresConfirmedCase, format string) []string {
	return append([]string{"--format", format, "cex", "futures"}, tc.args...)
}

// Only preissued credentials and the HTTP peer are mocked; parsing, resource
// binding, SDK, HMAC calculation, output and process exit are the production code.
func futuresConfirmedHome(t *testing.T, base, auth string) string {
	t.Helper()
	home := t.TempDir()
	if auth == "oauth" {
		folder := filepath.Join(home, ".gate-cli")
		if err := os.MkdirAll(folder, 0700); err != nil {
			t.Fatal(err)
		}
		raw := fmt.Sprintf(`{"oauth2":{"token":{"client_id":"fixture-client","access_token":"fixture-at","token_type":"Bearer","resource":%q,"expires_at":"2099-01-01T00:00:00Z","scopes":"base futures_trade"}}}`, base+"/api/v4")
		if err := os.WriteFile(filepath.Join(folder, "config.yaml"), []byte(raw), 0600); err != nil {
			t.Fatal(err)
		}
	}
	return home
}

func futuresConfirmedInvoke(t *testing.T, binary, home, base, auth string, args ...string) ([]byte, []byte, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Env = childEnv(home, base, auth == "hmac")
	var out, stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	err := cmd.Run()
	return out.Bytes(), stderr.Bytes(), err
}

func futuresConfirmedWire(t *testing.T, r *http.Request, tc futuresConfirmedCase, auth string) {
	t.Helper()
	// The released SDK uses json.Encoder.Encode, including its final newline.
	// Keep byte assertions exact instead of trimming away request differences.
	if tc.body != "" {
		tc.body += "\n"
	}
	raw, err := io.ReadAll(io.LimitReader(r.Body, 8193))
	if err != nil {
		t.Error(err)
	}
	if len(raw) > 8192 || r.Method != tc.method || r.URL.Path != tc.path || r.URL.RawQuery != tc.query || string(raw) != tc.body {
		t.Errorf("wire=%s %s?%s %s; want=%s %s?%s %s", r.Method, r.URL.Path, r.URL.RawQuery, raw, tc.method, tc.path, tc.query, tc.body)
	}
	if auth == "oauth" {
		if r.Header.Get("Authorization") != "Bearer fixture-at" || r.Header.Get("KEY") != "" || r.Header.Get("SIGN") != "" {
			t.Error("mixed/lost OAuth credential")
		}
	} else {
		decodedQuery, err := url.QueryUnescape(r.URL.RawQuery)
		if err != nil {
			t.Error(err)
		}
		digest := sha512.Sum512(raw)
		canonical := strings.Join([]string{r.Method, r.URL.Path, decodedQuery, hex.EncodeToString(digest[:]), r.Header.Get("Timestamp")}, "\n")
		sign := hmac.New(sha512.New, []byte("synthetic-secret"))
		_, _ = sign.Write([]byte(canonical))
		if r.Header.Get("KEY") != "synthetic-key" || r.Header.Get("Authorization") != "" || !hmac.Equal([]byte(r.Header.Get("SIGN")), []byte(hex.EncodeToString(sign.Sum(nil)))) {
			t.Error("invalid synthetic HMAC or mixed credentials")
		}
	}
}

func TestFuturesConfirmedSuccess(t *testing.T) {
	for _, auth := range []string{"hmac", "oauth"} {
		for _, tc := range futuresConfirmedCases() {
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
				formats := []string{"json"}
				if tc.name == "trail-list" || tc.name == "price-create" {
					formats = append(formats, "pretty", "table")
				}
				for _, format := range formats {
					args := futuresConfirmedArgs(tc, format)
					out, stderr, err := futuresConfirmedInvoke(t, cliBinary, home, peer.URL, auth, args...)
					if err != nil || len(stderr) != 0 || !bytes.Contains(out, []byte(tc.identity)) {
						t.Fatalf("success lost: %v stdout=%s stderr=%s", err, out, stderr)
					}
					if format == "json" && !json.Valid(out) {
						t.Fatalf("not JSON: %s", out)
					}
					wantCalls := int32(1)
					// Optional test-harness input, never a product config: this run
					// compares an independently built old revision byte for byte.
					if baseline := os.Getenv("CLI_FUTURES_BASELINE_BINARY"); baseline != "" {
						oldOut, oldErr, oldExit := futuresConfirmedInvoke(t, baseline, home, peer.URL, auth, args...)
						if oldExit != nil || !bytes.Equal(oldOut, out) || !bytes.Equal(oldErr, stderr) {
							t.Fatalf("normal behavior changed: old=%v %s %s new=%s %s", oldExit, oldOut, oldErr, out, stderr)
						}
						wantCalls++
					}
					if calls.Swap(0) != wantCalls {
						t.Fatal("unexpected request count")
					}
				}
			})
		}
	}
}

func futuresConfirmedError(t *testing.T, out, stderr []byte, err error, wantExit, status int) struct {
	Code                    *int
	Label, Message, TraceID string
	Retryable               bool
	Request                 struct{ Method, URL string }
	Data                    json.RawMessage
	SuggestedNextAction     string
} {
	t.Helper()
	var exit *exec.ExitError
	if !errors.As(err, &exit) || exit.ExitCode() != wantExit || len(out) != 0 {
		t.Fatalf("failure not propagated: exit=%v stdout=%s stderr=%s", err, out, stderr)
	}
	var envelope struct {
		Error struct {
			Code                    *int
			Label, Message, TraceID string
			Retryable               bool
			Request                 struct{ Method, URL string }
			Data                    json.RawMessage
			SuggestedNextAction     string
		}
	}
	// Wire names use underscores; decode explicitly without changing the DTO.
	var wire struct {
		Error struct {
			Code      *int                         `json:"code"`
			Label     string                       `json:"label"`
			Message   string                       `json:"message"`
			TraceID   string                       `json:"trace_id"`
			Retryable bool                         `json:"retryable"`
			Status    int                          `json:"status"`
			Request   struct{ Method, URL string } `json:"request"`
			Data      json.RawMessage              `json:"data"`
			Action    string                       `json:"suggested_next_action"`
		}
	}
	if json.Unmarshal(stderr, &wire) != nil || wire.Error.Status != status {
		t.Fatalf("invalid error: %s", stderr)
	}
	envelope.Error.Code = wire.Error.Code
	envelope.Error.Label = wire.Error.Label
	envelope.Error.Message = wire.Error.Message
	envelope.Error.TraceID = wire.Error.TraceID
	envelope.Error.Retryable = wire.Error.Retryable
	envelope.Error.Request = wire.Error.Request
	envelope.Error.Data = wire.Error.Data
	envelope.Error.SuggestedNextAction = wire.Error.Action
	return envelope.Error
}

func TestFuturesConfirmedBusinessError(t *testing.T) {
	for _, auth := range []string{"hmac", "oauth"} {
		for _, tc := range futuresConfirmedCases() {
			if tc.name != "trail-create" && tc.name != "trail-list" && tc.name != "price-create" && tc.name != "price-update" {
				continue
			}
			for _, format := range []string{"json", "pretty", "table"} {
				t.Run(auth+"/"+tc.name+"/"+format, func(t *testing.T) {
					code := 1077
					label := "APIOrderNotSupportUpdateTouchOrder"
					message := "original failure"
					if strings.HasPrefix(tc.name, "trail") {
						code = -1
						label = ""
						message = "Invalid Fund password"
					}
					if tc.name == "price-create" {
						code = 1013
						label = "AUTO_INVALID_PARAM_INITIAL_TIF"
					}
					raw := fmt.Sprintf(`{"code":%d,"Label":%q,"Message":%q,"data":{"id":"9007199254740993","token":"secret-canary"},"private_debug":"secret-canary"}`, code, label, message)
					var calls atomic.Int32
					peer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						calls.Add(1)
						futuresConfirmedWire(t, r, tc, auth)
						w.Header().Set("Content-Type", "application/json")
						w.Header().Set("X-Gate-Trace-Id", "fixture-trace")
						_, _ = io.WriteString(w, raw)
					}))
					t.Cleanup(peer.Close)
					out, stderr, err := futuresConfirmedInvoke(t, cliBinary, futuresConfirmedHome(t, peer.URL, auth), peer.URL, auth, futuresConfirmedArgs(tc, format)...)
					if format == "json" {
						detail := futuresConfirmedError(t, out, stderr, err, 1, 200)
						if detail.Code == nil || *detail.Code != code || detail.Label != label || detail.Message != message || detail.TraceID != "fixture-trace" || detail.Request.Method != tc.method || detail.Request.URL != tc.path {
							t.Fatalf("business facts lost: %s", stderr)
						}
						if tc.method != "GET" && (detail.Retryable || !strings.Contains(detail.SuggestedNextAction, "query")) {
							t.Fatalf("unsafe replay advice: %s", stderr)
						}
						if tc.name == "trail-create" && !bytes.Contains(detail.Data, []byte("9007199254740993")) {
							t.Fatal("lost partial identity")
						}
					} else if err == nil || len(out) != 0 || !bytes.Contains(stderr, []byte(message)) {
						t.Fatalf("human error lost: %v %s %s", err, out, stderr)
					}
					if calls.Load() != 1 || bytes.Contains(stderr, []byte("secret-canary")) {
						t.Fatalf("replay/data leak: calls=%d %s", calls.Load(), stderr)
					}
				})
			}
		}
	}
}

// Only the HTTP peer and preissued credentials are replaced. Exact documented
// responses must survive the actual CLI/SDK error path without replaying writes.
func TestFuturesConfirmedStringCodeErrors(t *testing.T) {
	for _, auth := range []string{"hmac", "oauth"} {
		for _, tc := range futuresConfirmedCases() {
			if tc.name != "trail-create" && tc.name != "trail-list" && tc.name != "price-create" && tc.name != "price-update" {
				continue
			}
			for _, failure := range []struct {
				raw, label, message string
				code                int
			}{
				{`{"code":"1013","label":"AUTO_INVALID_PARAM_INITIAL_TIF","message":"invalid argument: market order tif must ioc"}`, "AUTO_INVALID_PARAM_INITIAL_TIF", "invalid argument: market order tif must ioc", 1013},
				{`{"code":"1014","label":"AUTO_INVALID_PARAM_INITIAL_SIZE","message":"invalid argument: initial.size plan close long position size must less zero"}`, "AUTO_INVALID_PARAM_INITIAL_SIZE", "invalid argument: initial.size plan close long position size must less zero", 1014},
			} {
				t.Run(fmt.Sprintf("%s/%s/%d", auth, tc.name, failure.code), func(t *testing.T) {
					var calls atomic.Int32
					peer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						calls.Add(1)
						futuresConfirmedWire(t, r, tc, auth)
						w.Header().Set("Content-Type", "application/json")
						w.Header().Set("X-Gate-Trace-Id", "string-code-trace")
						_, _ = io.WriteString(w, failure.raw)
					}))
					t.Cleanup(peer.Close)
					out, stderr, err := futuresConfirmedInvoke(t, cliBinary, futuresConfirmedHome(t, peer.URL, auth), peer.URL, auth, futuresConfirmedArgs(tc, "json")...)
					detail := futuresConfirmedError(t, out, stderr, err, 1, 200)
					if detail.Code == nil || *detail.Code != failure.code || detail.Label != failure.label || detail.Message != failure.message || detail.TraceID != "string-code-trace" || detail.Request.Method != tc.method || detail.Request.URL != tc.path {
						t.Fatalf("string business facts lost: %s", stderr)
					}
					if tc.method != "GET" && (detail.Retryable || !strings.Contains(detail.SuggestedNextAction, "query")) {
						t.Fatalf("unsafe replay advice: %s", stderr)
					}
					if calls.Load() != 1 {
						t.Fatalf("unexpected replay: %d requests", calls.Load())
					}
				})
			}
		}
	}
}

// Boundary and normal controls guard against broadening successful responses,
// changing non-2xx handling, exposing private data, or accepting ambiguous codes.
func TestFuturesConfirmedStringCodeControls(t *testing.T) {
	var command futuresConfirmedCase
	for _, tc := range futuresConfirmedCases() {
		if tc.name == "price-create" {
			command = tc
		}
	}
	for _, tc := range []struct {
		name, raw, format, label, message, success string
		status, code                               int
	}{
		{"minimum", `{"code":"-2147483648","label":"BAD_REQUEST","message":"original"}`, "json", "BAD_REQUEST", "original", "", 200, -2147483648},
		{"maximum", `{"code":"2147483647","label":"BAD_REQUEST","message":"original"}`, "json", "BAD_REQUEST", "original", "", 200, 2147483647},
		{"pretty-private", `{"Code":"1013","Label":"BAD_REQUEST","Message":" original ","id":1,"data":{"token":"secret-canary"},"private_debug":"secret-canary"}`, "pretty", "BAD_REQUEST", " original ", "", 200, 1013},
		{"table-private", `{"code":"1014","label":"BAD_REQUEST","message":"original","id":1,"private_debug":"secret-canary"}`, "table", "BAD_REQUEST", "original", "", 200, 1014},
		{"zero", `{"code":"0","label":"BAD_REQUEST","message":"original","id":1}`, "json", "INVALID_RESPONSE", "invalid futures response protocol", "", 200, 0},
		{"overflow", `{"code":"2147483648","label":"BAD_REQUEST","message":"original"}`, "json", "INVALID_RESPONSE", "invalid futures response protocol", "", 200, 0},
		{"not-numeric", `{"code":"future","label":"BAD_REQUEST","message":"original"}`, "json", "INVALID_RESPONSE", "invalid futures response protocol", "", 200, 0},
		{"duplicate", `{"code":"1013","Code":0,"label":"BAD_REQUEST","message":"original"}`, "json", "INVALID_RESPONSE", "invalid futures response protocol", "", 200, 0},
		{"http400-1013", `{"code":"1013","label":"AUTO_INVALID_PARAM_INITIAL_TIF","message":"invalid argument: market order tif must ioc"}`, "json", "AUTO_INVALID_PARAM_INITIAL_TIF", "invalid argument: market order tif must ioc", "", 400, 0},
		{"http400-1014", `{"code":"1014","label":"AUTO_INVALID_PARAM_INITIAL_SIZE","message":"invalid argument: initial.size plan close long position size must less zero"}`, "json", "AUTO_INVALID_PARAM_INITIAL_SIZE", "invalid argument: initial.size plan close long position size must less zero", "", 400, 0},
		{"unknown-extension", `{"code":"future","id":9007199254740993,"id_string":"9007199254740993"}`, "json", "", "", `{"id":9007199254740993,"id_string":"9007199254740993"}`, 200, 0},
		{"normal-success", `{"id":9007199254740993,"id_string":"9007199254740993"}`, "json", "", "", `{"id":9007199254740993,"id_string":"9007199254740993"}`, 200, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls atomic.Int32
			peer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				futuresConfirmedWire(t, r, command, "hmac")
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("X-Gate-Trace-Id", "string-code-trace")
				w.WriteHeader(tc.status)
				_, _ = io.WriteString(w, tc.raw)
			}))
			t.Cleanup(peer.Close)
			out, stderr, err := futuresConfirmedInvoke(t, cliBinary, futuresConfirmedHome(t, peer.URL, "hmac"), peer.URL, "hmac", futuresConfirmedArgs(command, tc.format)...)
			if tc.success != "" {
				if err != nil || len(stderr) != 0 || string(out) != tc.success+"\n" {
					t.Fatalf("normal result changed: %v %s %s", err, out, stderr)
				}
			} else if tc.format == "json" {
				detail := futuresConfirmedError(t, out, stderr, err, 1, tc.status)
				if detail.Label != tc.label || detail.Message != tc.message || detail.TraceID != "string-code-trace" || detail.Request.Method != command.method || detail.Request.URL != command.path {
					t.Fatalf("wrong failure: %s", stderr)
				}
				// Non-2xx is outside this adapter: preserve existing invalid-args
				// advice while requiring the guarded 2xx write path to reconcile.
				if tc.status == 200 && (detail.Retryable || !strings.Contains(detail.SuggestedNextAction, "query")) || tc.status == 400 && (!detail.Retryable || !strings.Contains(detail.SuggestedNextAction, "corrected args")) {
					t.Fatalf("error recovery policy changed: %s", stderr)
				}
				if tc.code != 0 && (detail.Code == nil || *detail.Code != tc.code) || tc.code == 0 && detail.Code != nil {
					t.Fatalf("unexpected code: %s", stderr)
				}
			} else {
				var exit *exec.ExitError
				if !errors.As(err, &exit) || exit.ExitCode() != 1 || len(out) != 0 || !bytes.Contains(stderr, []byte(tc.message)) || !bytes.Contains(stderr, []byte(tc.label)) {
					t.Fatalf("human rejection lost: %v %s %s", err, out, stderr)
				}
			}
			if calls.Load() != 1 || bytes.Contains(stderr, []byte("secret-canary")) || bytes.Contains(out, []byte("secret-canary")) {
				t.Fatalf("replay or private-data leak: calls=%d %s %s", calls.Load(), out, stderr)
			}
		})
	}
}

func TestFuturesConfirmedTrailRequestInfo(t *testing.T) {
	for _, tc := range futuresConfirmedCases() {
		if !strings.HasPrefix(tc.name, "trail") {
			continue
		}
		for _, status := range []int{400, 401, 403, 503} {
			t.Run(fmt.Sprintf("%s/%d", tc.name, status), func(t *testing.T) {
				var calls atomic.Int32
				peer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					futuresConfirmedWire(t, r, tc, "hmac")
					w.Header().Set("Content-Type", "application/json")
					w.Header().Set("X-Gate-Trace-Id", "fixture-trace")
					w.WriteHeader(status)
					_, _ = io.WriteString(w, `{"label":"FIXTURE_REJECTED","message":"original"}`)
				}))
				t.Cleanup(peer.Close)
				out, stderr, err := runCLI(t, peer.URL, true, futuresConfirmedArgs(tc, "json")...)
				want := 1
				if status == 401 || status == 403 {
					want = 4
				}
				detail := futuresConfirmedError(t, out, stderr, err, want, status)
				if detail.Request.Method != tc.method || detail.Request.URL != tc.path || detail.Label != "FIXTURE_REJECTED" || !strings.Contains(detail.Message, "original") || detail.TraceID != "fixture-trace" || calls.Load() != 1 {
					t.Fatalf("wrong diagnostics/replay: %s calls=%d", stderr, calls.Load())
				}
			})
		}
	}
}

func TestFuturesConfirmedTIF(t *testing.T) {
	tc := futuresConfirmedCases()[7]
	for _, tif := range []string{"omitted", "ioc", "gtc", "", "IOC", "invalid"} {
		t.Run(tif, func(t *testing.T) {
			valid := tif == "omitted" || tif == "ioc" || tif == "gtc"
			args := futuresConfirmedArgs(tc, "json")
			expected := tc
			if tif != "omitted" {
				args = append(args, "--tif", tif)
				expected.body = strings.Replace(tc.body, `"price":"0"}`, `"price":"0","tif":"`+tif+`"}`, 1)
			}
			var calls atomic.Int32
			peer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				futuresConfirmedWire(t, r, expected, "hmac")
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, tc.response)
			}))
			t.Cleanup(peer.Close)
			out, stderr, err := runCLI(t, peer.URL, true, args...)
			if valid {
				if err != nil || len(stderr) != 0 || !bytes.Contains(out, []byte(tc.identity)) || calls.Load() != 1 {
					t.Fatalf("tif lost: %v %s %s", err, out, stderr)
				}
			} else {
				detail := futuresConfirmedError(t, out, stderr, err, 1, 400)
				if detail.Label != "INVALID_ARGUMENTS" || calls.Load() != 0 {
					t.Fatalf("invalid input sent: %s calls=%d", stderr, calls.Load())
				}
			}
		})
	}
	out, stderr, err := runCLI(t, "http://127.0.0.1:1", false, "cex", "futures", "price-trigger", "create", "--help")
	if err != nil || !bytes.Contains(out, []byte("--tif")) || !bytes.Contains(out, []byte("ioc")) {
		t.Fatalf("missing help: %v %s %s", err, out, stderr)
	}
}

func TestFuturesConfirmedScope(t *testing.T) {
	for _, tc := range []struct {
		args          []string
		raw, expected string
	}{
		{[]string{"price-trigger", "create", "--contract", "SYNTH_USDT", "--price", "0", "--size", "-10", "--trigger-price", "90000"}, `{"code":200,"message":"ok","id":9007199254740993,"id_string":"9007199254740993"}`, `{"id":9007199254740993,"id_string":"9007199254740993"}`},
		{[]string{"price-trigger", "update", "--id", "pending", "--body-order-id", "9007199254740993"}, `{}`, `{}`},
		{[]string{"price-trigger", "get", "--id", "0"}, `{}`, `{"initial":{"contract":"","price":""},"trigger":{"price":"","rule":0}}`},
		{[]string{"trail", "list"}, `{"data":{"orders":[]}}`, `{"data":{}}`},
	} {
		t.Run(strings.Join(tc.args, "/"), func(t *testing.T) {
			peer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, tc.raw)
			}))
			t.Cleanup(peer.Close)
			out, stderr, err := runCLI(t, peer.URL, true, append([]string{"--format", "json", "cex", "futures"}, tc.args...)...)
			if err != nil || len(stderr) != 0 || strings.TrimSpace(string(out)) != tc.expected {
				t.Fatalf("compatibility changed: %v %s %s", err, out, stderr)
			}
		})
	}
}

func TestFuturesConfirmedFailureSafety(t *testing.T) {
	for _, tc := range []futuresConfirmedCase{futuresConfirmedCases()[0], futuresConfirmedCases()[7], futuresConfirmedCases()[8]} {
		for _, mode := range []string{"400", "401", "403", "429", "503", "malformed", "oversize", "disconnect"} {
			t.Run(tc.name+"/"+mode, func(t *testing.T) {
				var calls atomic.Int32
				peer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					futuresConfirmedWire(t, r, tc, "hmac")
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
						_, _ = io.WriteString(w, `{`)
						return
					}
					if mode == "oversize" {
						_, _ = io.WriteString(w, strings.Repeat(" ", 8*1024*1024+1))
						return
					}
					status := 0
					_, _ = fmt.Sscanf(mode, "%d", &status)
					w.WriteHeader(status)
					_, _ = io.WriteString(w, `{"label":"FIXTURE_REJECTED","message":"original"}`)
				}))
				t.Cleanup(peer.Close)
				out, stderr, err := runCLI(t, peer.URL, true, futuresConfirmedArgs(tc, "json")...)
				if err == nil || len(out) != 0 || !json.Valid(stderr) || calls.Load() != 1 {
					t.Fatalf("unsafe failure: %v %s %s calls=%d", err, out, stderr, calls.Load())
				}
				if mode == "disconnect" || mode == "malformed" || mode == "oversize" || mode == "503" {
					if !bytes.Contains(stderr, []byte(`"retryable":false`)) || !strings.Contains(string(stderr), "reconcile") && !strings.Contains(string(stderr), "query") {
						t.Fatalf("missing reconciliation: %s", stderr)
					}
				}
			})
		}
	}
}

// Killing a child is process-lifecycle evidence, not proof that application
// cancellation printed a JSON error. Component tests cover context propagation.
func TestFuturesConfirmedProcessCancellation(t *testing.T) {
	var calls atomic.Int32
	ready := make(chan struct{})
	peer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.(http.Flusher).Flush()
		close(ready)
		<-r.Context().Done()
	}))
	t.Cleanup(peer.Close)
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, cliBinary, futuresConfirmedArgs(futuresConfirmedCases()[0], "json")...)
	cmd.Env = childEnv(t.TempDir(), peer.URL, true)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-ready:
	case <-ctx.Done():
		cancel()
		_ = cmd.Wait()
		t.Fatal("request never reached peer")
	}
	cancel()
	if err := cmd.Wait(); err == nil || calls.Load() != 1 {
		t.Fatal("process did not terminate safely")
	}
}

// The new flag is discoverable only as a precise declaration increment; old
// builds still do not expose agent commands and discovery must perform no I/O.
func TestFuturesConfirmedDiscovery(t *testing.T) {
	var calls atomic.Int32
	peer := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { calls.Add(1) }))
	t.Cleanup(peer.Close)
	args := []string{"agent-index", "--domain", "cex", "--details", "--command", "cex futures price-trigger create", "--format", "json", "--max-output-bytes", "0"}
	home := futuresConfirmedHome(t, peer.URL, "oauth")
	out, stderr, err := futuresConfirmedInvoke(t, cliBinary, home, peer.URL, "oauth", args...)
	if !agentfeature.Commands {
		if err == nil || calls.Load() != 0 {
			t.Fatal("default build exposed agent command or reached network")
		}
		return
	}
	type flag struct {
		Name, Type, Usage string
		Required          bool
	}
	// Keep JSON names faithful so comparing old/new checks every declaration.
	type details struct {
		SchemaVersion int `json:"schema_version"`
		Commands      []struct {
			Path                string `json:"path"`
			Effect              string `json:"effect"`
			FailureAction       string `json:"failure_action"`
			ConstraintsComplete bool   `json:"constraints_complete"`
			Flags               []flag `json:"flags"`
		} `json:"commands"`
	}
	var current details
	if err != nil || len(stderr) != 0 || json.Unmarshal(out, &current) != nil || len(current.Commands) != 1 {
		t.Fatalf("discovery failed: %v %s %s", err, out, stderr)
	}
	if bytes.Contains(out, []byte(`"default":`)) || bytes.Contains(out, []byte(`"value":`)) {
		t.Fatal("discovery exposed default or current flag values")
	}
	if bytes.Contains(out, []byte("fixture-at")) || bytes.Contains(out, []byte("synthetic-secret")) {
		t.Fatal("discovery exposed fixture credentials")
	}
	found := false
	kept := current.Commands[0].Flags[:0]
	for _, f := range current.Commands[0].Flags {
		if f.Name == "tif" {
			found = true
			if f.Type != "string" || f.Required || !strings.Contains(f.Usage, "ioc") {
				t.Fatal("wrong new flag contract")
			}
			continue
		}
		kept = append(kept, f)
	}
	if !found {
		t.Fatal("explicit TIF is undiscoverable")
	}
	current.Commands[0].Flags = kept
	if baseline := os.Getenv("CLI_FUTURES_BASELINE_BINARY"); baseline != "" {
		oldOut, oldStderr, oldErr := futuresConfirmedInvoke(t, baseline, home, peer.URL, "oauth", args...)
		var previous details
		if oldErr != nil || len(oldStderr) != 0 || json.Unmarshal(oldOut, &previous) != nil {
			t.Fatalf("baseline discovery: %v %s", oldErr, oldStderr)
		}
		a, e1 := json.Marshal(current)
		b, e2 := json.Marshal(previous)
		if e1 != nil || e2 != nil || !bytes.Equal(a, b) {
			t.Fatal("discovery changed beyond the optional TIF flag")
		}
	}
	if calls.Load() != 0 {
		t.Fatal("discovery executed a business handler")
	}
}
