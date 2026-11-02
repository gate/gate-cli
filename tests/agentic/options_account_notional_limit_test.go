//go:build agentic_e2e

package agentic_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

// Run the real CLI process, authentication transport and published SDK decoder.
// Only credentials and API responses are synthetic; no trading is performed.
func TestOptionsAccountNotionalLimitCLI(t *testing.T) {
	for _, mode := range []string{"oauth", "hmac"} {
		for _, value := range []string{"10000000", "0", "9223372036854775808", "10000000.123456789012345678", ""} {
			t.Run(mode+"/"+value, func(t *testing.T) {
				var hits atomic.Int32
				peer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					hits.Add(1)
					if r.Method != "GET" || r.URL.Path != "/api/v4/options/accounts" || r.URL.RawQuery != "" {
						t.Errorf("wrong route: %s %s", r.Method, r.URL)
					}
					if mode == "oauth" && (r.Header.Get("Authorization") != "Bearer synthetic-test-base" || r.Header.Get("KEY") != "") {
						t.Error("OAuth identity changed")
					}
					if mode == "hmac" && (r.Header.Get("KEY") != "synthetic-key" || r.Header.Get("SIGN") == "" || r.Header.Get("Authorization") != "") {
						t.Error("HMAC identity changed")
					}
					w.Header().Set("Content-Type", "application/json")
					body := fmt.Sprintf(`{"user":666,"short_enabled":true,"orders_limit":10,"total":"1.123456789012345678","position_notional_limit":%q}`, value)
					if _, err := io.WriteString(w, body); err != nil {
						t.Error(err)
					}
				}))
				defer peer.Close()
				out, stderr, err := credentialCLI(t, peer.URL, mode, "base", "--format", "json", "cex", "options", "account", "get")
				if err != nil || hits.Load() != 1 {
					t.Fatalf("exit=%v calls=%d out=%s err=%s", err, hits.Load(), out, stderr)
				}
				var result map[string]json.RawMessage
				if err := json.Unmarshal(out, &result); err != nil {
					t.Fatal(err)
				}
				// SDK omitempty retains its existing zero-value behavior for an empty string.
				if value != "" {
					var got string
					if err := json.Unmarshal(result["position_notional_limit"], &got); err != nil || got != value {
						t.Fatalf("string changed: %s, %v", out, err)
					}
				}
				if string(result["short_enabled"]) != "true" || string(result["orders_limit"]) != "10" || string(result["total"]) != `"1.123456789012345678"` {
					t.Fatalf("other fields changed: %s", out)
				}
			})
		}
	}
}

// HTTP 200 with a wrong field type must fail decoding; remote failures stay failures.
func TestOptionsAccountNotionalLimitCLIErrors(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		status     int
	}{
		{"numeric", `{"position_notional_limit":10000000}`, 200},
		{"boolean", `{"position_notional_limit":true}`, 200},
		{"malformed", `{`, 200},
		{"bad_request", `{"label":"INVALID_PARAM_VALUE","message":"fixture"}`, 400},
		{"not_found_user", `{"label":"USER_NOT_FOUND","message":"fixture"}`, 400},
		{"unauthorized", `{"label":"INVALID_CREDENTIALS","message":"fixture"}`, 401},
		{"forbidden", `{"label":"FORBIDDEN","message":"fixture"}`, 403},
		{"rate_limit", `{"label":"TOO_MANY_REQUESTS","message":"fixture"}`, 429},
		{"server_error", `{"label":"SERVER_ERROR","message":"fixture"}`, 500},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var hits atomic.Int32
			peer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				hits.Add(1)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.status)
				if _, err := io.WriteString(w, tc.body); err != nil {
					t.Error(err)
				}
			}))
			defer peer.Close()
			out, stderr, err := credentialCLI(t, peer.URL, "oauth", "base", "--format", "json", "cex", "options", "account", "get")
			if err == nil || hits.Load() != 1 || len(out) != 0 {
				t.Fatalf("failure became success: %v calls=%d out=%s err=%s", err, hits.Load(), out, stderr)
			}
			if tc.status == 200 && !strings.Contains(string(stderr), "INVALID_RESPONSE") {
				t.Fatalf("decode failure hidden: %s", stderr)
			}
			if tc.name == "not_found_user" && !strings.Contains(string(stderr), "USER_NOT_FOUND") {
				t.Fatalf("business error hidden: %s", stderr)
			}
		})
	}
}
