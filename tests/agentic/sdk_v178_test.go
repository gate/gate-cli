//go:build agentic_e2e

package agentic_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

// Keep the real process, flags, authentication, SDK and output. Only Gate's
// remote HTTP peer is mocked; request counts prevent unsafe replay of writes.
func TestSDK178CLITrailResponses(t *testing.T) {
	for _, operation := range []string{"update", "stop", "log"} {
		formats := []string{"json"}
		if operation == "log" {
			formats = append(formats, "table")
		}
		for _, format := range formats {
			for _, mode := range []string{"normal", "empty", "business", "malformed", "400", "503", "unauthenticated", "numeric-time"} {
				if mode == "numeric-time" && operation != "log" {
					continue
				}
				t.Run(operation+"/"+format+"/"+mode, func(t *testing.T) {
					var calls atomic.Int32
					peer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						calls.Add(1)
						endpoint, method := operation, http.MethodPost
						if operation == "log" {
							endpoint, method = "change_log", http.MethodGet
						}
						if r.Method != method || r.URL.Path != "/api/v4/futures/usdt/autoorder/v1/trail/"+endpoint || r.Header.Get("SIGN") == "" {
							t.Errorf("incorrect signed route: %s %s", r.Method, r.URL.Path)
						}
						if operation == "log" {
							if r.URL.Query().Get("id") != "9007199254740993" || r.URL.Query().Get("page_num") != "1" || r.URL.Query().Get("page_size") != "5" {
								t.Errorf("first log page/query lost: %s", r.URL.RawQuery)
							}
						} else {
							var body map[string]json.RawMessage
							if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
								t.Error(err)
							}
							if string(body["id"]) != "9007199254740993" {
								t.Errorf("request ID lost: %s", body["id"])
							}
							if operation == "update" && string(body["activation_price"]) != `"39075.6"` {
								t.Errorf("price changed: %s", body["activation_price"])
							}
						}
						w.Header().Set("Content-Type", "application/json")
						w.Header().Set("X-Gate-Trace-Id", "sdk178-fixture-trace")
						if mode == "400" || mode == "503" {
							status := http.StatusBadRequest
							if mode == "503" {
								status = http.StatusServiceUnavailable
							}
							w.WriteHeader(status)
							_, _ = io.WriteString(w, `{"label":"TRAIL_REJECTED","message":"fixture rejection","private_debug":"sdk178-private-canary"}`)
							return
						}
						if mode == "malformed" {
							_, _ = io.WriteString(w, `{"data":`)
							return
						}
						if mode == "business" {
							_, _ = io.WriteString(w, `{"code":-1,"label":"TRAIL_REJECTED","message":"fixture rejection","data":{},"private_debug":"sdk178-private-canary"}`)
							return
						}
						data := `{"order":{"id":"9007199254740993","amount":"0.01","status":"open","updated_at":"1793627143"}}`
						if operation == "stop" {
							data = strings.Replace(data, `"open"`, `"finished"`, 1)
						}
						if operation == "log" {
							data = `{"change_log":[{"updated_at":"1793627143","amount":"0.01","activation_price":"39075.6","price_offset":"1%"}]}`
						}
						if mode == "numeric-time" {
							data = strings.Replace(data, `"1793627143"`, `1793627143`, 1)
						}
						if mode == "empty" {
							data = `{}`
						}
						_, _ = fmt.Fprintf(w, `{"code":0,"message":"ok","data":%s,"timestamp":1793627143000,"unknown":true}`, data)
					}))
					t.Cleanup(peer.Close)
					args := []string{"--format", format, "cex", "futures", "trail", operation, "--id", "9007199254740993"}
					if operation == "update" {
						args = append(args, "--activation-price", "39075.6")
					}
					if operation == "log" {
						args = append(args, "--limit", "5")
					}
					out, stderr, err := runCLI(t, peer.URL, mode != "unauthenticated", args...)
					wantCalls := int32(1)
					if mode == "unauthenticated" {
						wantCalls = 0
					}
					if calls.Load() != wantCalls {
						t.Errorf("requests=%d want %d", calls.Load(), wantCalls)
					}
					failure := mode != "normal" && mode != "empty"
					if (err != nil) != failure {
						t.Fatalf("exit=%v failure=%v stdout=%s stderr=%s", err, failure, out, stderr)
					}
					if bytes.Contains(out, []byte("sdk178-private-canary")) || bytes.Contains(stderr, []byte("sdk178-private-canary")) {
						t.Fatal("private response leaked")
					}
					if failure {
						if len(bytes.TrimSpace(out)) != 0 || len(bytes.TrimSpace(stderr)) == 0 {
							t.Fatalf("failure did not use stderr: out=%s err=%s", out, stderr)
						}
						// JSON errors are structured; table mode retains the existing
						// human-readable stderr contract rather than changing output.
						if format == "json" && !bytes.Contains(stderr, []byte(`"error"`)) {
							t.Fatalf("JSON error envelope missing: %s", stderr)
						}
						if mode == "unauthenticated" && format == "table" && (!bytes.Contains(stderr, []byte("AUTH_ERROR")) || bytes.Contains(stderr, []byte("fix flags"))) {
							t.Fatalf("misleading auth guidance: %s", stderr)
						}
						if mode == "business" || mode == "400" || mode == "503" {
							if !bytes.Contains(stderr, []byte("fixture rejection")) {
								t.Fatalf("backend message lost: %s", stderr)
							}
						}
						return
					}
					if len(bytes.TrimSpace(stderr)) != 0 {
						t.Fatalf("successful command wrote stderr: %s", stderr)
					}
					if format == "table" {
						if mode == "normal" && (!bytes.Contains(out, []byte("1793627143")) || bytes.Contains(out, []byte("%!"))) {
							t.Fatalf("table lost string timestamp: %s", out)
						}
						if mode == "empty" && bytes.Contains(out, []byte("1793627143")) {
							t.Fatalf("empty table invented a log: %s", out)
						}
						return
					}
					var envelope struct {
						Message   string                     `json:"message"`
						Timestamp int64                      `json:"timestamp"`
						Data      map[string]json.RawMessage `json:"data"`
					}
					if err := json.Unmarshal(out, &envelope); err != nil {
						t.Fatal(err)
					}
					if envelope.Message != "ok" || envelope.Timestamp != 1793627143000 {
						t.Fatalf("response envelope lost: %s", out)
					}
					if mode == "empty" {
						return
					}
					if operation == "log" {
						var entries []struct {
							UpdatedAt string `json:"updated_at"`
							Amount    string `json:"amount"`
						}
						if err := json.Unmarshal(envelope.Data["change_log"], &entries); err != nil {
							t.Fatal(err)
						}
						if len(entries) != 1 || entries[0].UpdatedAt != "1793627143" || entries[0].Amount != "0.01" {
							t.Fatalf("log lost: %s", out)
						}
					} else {
						var order struct {
							ID     string `json:"id"`
							Status string `json:"status"`
						}
						if err := json.Unmarshal(envelope.Data["order"], &order); err != nil {
							t.Fatal(err)
						}
						want := "open"
						if operation == "stop" {
							want = "finished"
						}
						if order.ID != "9007199254740993" || order.Status != want {
							t.Fatalf("order lost: %s", out)
						}
					}
				})
			}
		}
	}
}
