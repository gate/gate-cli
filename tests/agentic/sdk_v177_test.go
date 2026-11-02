//go:build agentic_e2e

package agentic_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

// The subprocess, argument parser, HMAC transport and published SDK are real;
// only Gate's final HTTP service is replaced by the bounded loopback peer.
func TestSDK177CLITrailCreate(t *testing.T) {
	for _, tc := range []struct {
		value   string
		status  int
		failure bool
		calls   int32
	}{
		{"1", 200, false, 1}, {"2", 200, false, 1}, {"3", 200, false, 1},
		{"omitted", 200, true, 0}, {"0", 200, true, 0}, {"-1", 200, true, 0}, {"4", 200, true, 0},
		{"x", 200, true, 0}, {"1.5", 200, true, 0}, {"2147483648", 200, true, 0},
		{"1", 400, true, 1}, {"1", 503, true, 1},
	} {
		t.Run(fmt.Sprintf("%s/%d", tc.value, tc.status), func(t *testing.T) {
			var calls atomic.Int32
			peer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.Method != "POST" || r.URL.Path != "/api/v4/futures/usdt/autoorder/v1/trail/create" || r.Header.Get("SIGN") == "" {
					t.Errorf("wrong unsigned request: %s %s", r.Method, r.URL.Path)
				}
				var body map[string]json.RawMessage
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
				}
				if string(body["price_type"]) != tc.value {
					t.Errorf("price_type=%s want %s", body["price_type"], tc.value)
				}
				if string(body["amount"]) != `"1"` || string(body["price_offset"]) != `"1%"` {
					t.Errorf("normal request fields changed: %v", body)
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.status)
				bodyText := `{"code":0,"message":"ok","data":{"id":"9007199254740993"}}`
				if tc.failure {
					bodyText = `{"label":"FIXTURE_REJECTED","message":"original rejection"}`
				}
				_, _ = io.WriteString(w, bodyText)
			}))
			t.Cleanup(peer.Close)
			args := []string{"--format", "json", "cex", "futures", "trail", "create", "--contract", "SYNTH_USDT", "--amount", "1", "--price-offset", "1%"}
			if tc.value != "omitted" {
				args = append(args, "--price-type", tc.value)
			}
			out, stderr, err := runCLI(t, peer.URL, true, args...)
			if calls.Load() != tc.calls {
				t.Errorf("requests=%d want %d", calls.Load(), tc.calls)
			}
			if tc.failure {
				// Cobra's existing parse failures can append a help hint or a
				// second error line. Assert the structured error without changing
				// unrelated global error-presentation behavior in this SDK repair.
				var failure struct {
					Error struct {
						Label string `json:"label"`
					} `json:"error"`
				}
				decodeErr := json.NewDecoder(bytes.NewReader(stderr)).Decode(&failure)
				if err == nil || len(out) != 0 || decodeErr != nil || failure.Error.Label == "" {
					t.Fatalf("failure misreported: %v %s %s", err, out, stderr)
				}
				if tc.calls == 0 && failure.Error.Label != "INVALID_ARGUMENTS" {
					t.Fatalf("argument rejection lost: %s", stderr)
				}
				if tc.calls == 1 && !json.Valid(stderr) {
					t.Fatalf("backend error is not a JSON document: %s", stderr)
				}
				if tc.calls == 1 && !bytes.Contains(stderr, []byte("FIXTURE_REJECTED")) {
					t.Fatalf("backend error lost: %s", stderr)
				}
			} else if err != nil || len(stderr) != 0 || !bytes.Contains(out, []byte(`"9007199254740993"`)) {
				t.Fatalf("normal create: %v %s %s", err, out, stderr)
			}
		})
	}
}

func TestSDK177CLITrailRead(t *testing.T) {
	const order = `{"id":"9007199254740993","user_id":"1001","user":"1002","suborder_id":"9223372036854775807","created_at":"1793613819","activated_at":"0","finished_at":"0","create_time":"1793613819","active_time":"0","finish_time":"0","updated_at":"1793613820","contract":"SYNTH_USDT","amount":"1"}`
	for _, operation := range []string{"get", "list"} {
		for _, mode := range []string{"json", "table", "empty", "400", "503", "malformed", "numeric-id"} {
			t.Run(operation+"/"+mode, func(t *testing.T) {
				var calls atomic.Int32
				peer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					leaf := "list"
					if operation == "get" {
						leaf = "detail"
					}
					if r.URL.Path != "/api/v4/futures/usdt/autoorder/v1/trail/"+leaf || r.Method != "GET" || r.Header.Get("SIGN") == "" {
						t.Errorf("wrong read request: %s %s", r.Method, r.URL.Path)
					}
					if operation == "get" && r.URL.Query().Get("id") != "9007199254740993" {
						t.Errorf("query precision lost: %s", r.URL.RawQuery)
					}
					w.Header().Set("Content-Type", "application/json")
					if mode == "400" || mode == "503" {
						status := 400
						if mode == "503" {
							status = 503
						}
						w.WriteHeader(status)
						_, _ = io.WriteString(w, `{"label":"FIXTURE_REJECTED","message":"original"}`)
						return
					}
					if mode == "malformed" {
						_, _ = io.WriteString(w, `{"code":`)
						return
					}
					value := order
					if mode == "numeric-id" {
						value = `{"id":9007199254740993}`
					}
					data := `{"order":` + value + `}`
					if operation == "list" {
						data = `{"orders":[` + value + `]}`
					}
					if mode == "empty" {
						if operation == "list" {
							data = `{"orders":[]}`
						} else {
							data = `{}`
						}
					}
					_, _ = io.WriteString(w, `{"code":0,"message":"ok","data":`+data+`,"timestamp":1793582374966743,"future":true}`)
				}))
				t.Cleanup(peer.Close)
				format := "json"
				if mode == "table" {
					format = "table"
				}
				args := []string{"--format", format, "cex", "futures", "trail", operation}
				if operation == "get" {
					args = append(args, "--id", "9007199254740993")
				}
				out, stderr, err := runCLI(t, peer.URL, true, args...)
				failure := mode == "400" || mode == "503" || mode == "malformed" || mode == "numeric-id"
				if calls.Load() != 1 {
					t.Fatalf("retried: %d", calls.Load())
				}
				if failure {
					if err == nil || len(out) != 0 || !json.Valid(stderr) {
						t.Fatalf("bad response succeeded: %v %s %s", err, out, stderr)
					}
					return
				}
				if err != nil || len(stderr) != 0 {
					t.Fatalf("normal read failed: %v %s %s", err, out, stderr)
				}
				if mode == "empty" {
					return
				}
				if mode == "table" {
					if !bytes.Contains(out, []byte("9007199254740993")) || bytes.Contains(out, []byte("%!")) {
						t.Fatalf("table ID lost: %s", out)
					}
					return
				}
				var result struct {
					Data struct {
						Order  map[string]any   `json:"order"`
						Orders []map[string]any `json:"orders"`
					} `json:"data"`
				}
				if err := json.Unmarshal(out, &result); err != nil {
					t.Fatal(err)
				}
				got := result.Data.Order
				if operation == "list" {
					if len(result.Data.Orders) != 1 {
						t.Fatal("orders lost")
					}
					got = result.Data.Orders[0]
				}
				var want map[string]any
				if err := json.Unmarshal([]byte(order), &want); err != nil {
					t.Fatal(err)
				}
				for key, value := range want {
					if got[key] != value {
						t.Errorf("%s=%#v want %#v", key, got[key], value)
					}
				}
			})
		}
	}
}
