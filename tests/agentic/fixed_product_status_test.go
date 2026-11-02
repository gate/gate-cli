//go:build agentic_e2e

package agentic_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

// Full CLI process -> command -> released SDK -> transport -> HTTP peer.
// Only credentials/business responses are synthetic; flags/query are real.
// Product listing is public: existing credential stripping must remain intact.
func TestFixedProductStatusCLI(t *testing.T) {
	for _, mode := range []string{"oauth", "hmac"} {
		for _, tc := range []struct {
			name               string
			flags              []string
			status, show       string
			hasStatus, hasShow bool
		}{
			{name: "omitted"},
			{"status", []string{"--status", "2"}, "2", "", true, false},
			{"show", []string{"--show-status", "2"}, "", "2", false, true},
			{"both", []string{"--status", "2", "--show-status", "2"}, "2", "2", true, true},
			{"zero-empty", []string{"--status", "0", "--show-status", ""}, "0", "", true, true},
			{"bounds", []string{"--status", "-2147483648", "--show-status", "visible & scheduled"}, "-2147483648", "visible & scheduled", true, true},
		} {
			t.Run(mode+"/"+tc.name, func(t *testing.T) {
				var hits atomic.Int32
				peer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					hits.Add(1)
					q := r.URL.Query()
					if r.Method != "GET" || r.URL.Path != "/api/v4/earn/fixed-term/product" {
						t.Errorf("wrong route %s %s", r.Method, r.URL.Path)
					}
					if q.Get("status") != tc.status || q.Has("status") != tc.hasStatus || q.Get("show_status") != tc.show || q.Has("show_status") != tc.hasShow {
						t.Errorf("filters changed: %s", r.URL.RawQuery)
					}
					if q.Get("asset") != "USDT" || q.Get("page") != "2" || q.Get("limit") != "3" || q.Get("type") != "1" {
						t.Errorf("old filters changed: %s", r.URL.RawQuery)
					}
					if r.Header.Get("Authorization") != "" || r.Header.Get("KEY") != "" || r.Header.Get("SIGN") != "" {
						t.Error("public product listing leaked credentials")
					}
					w.Header().Set("Content-Type", "application/json")
					if _, err := io.WriteString(w, `{"code":0,"data":{"list":[{"asset":"USDT","year_rate":"0.123456789012345678"}],"total":1}}`); err != nil {
						t.Error(err)
					}
				}))
				defer peer.Close()
				args := append([]string{"--format", "json", "cex", "earn", "fixed", "products", "--asset", "USDT", "--page", "2", "--limit", "3", "--type", "1"}, tc.flags...)
				out, stderr, err := credentialCLI(t, peer.URL, mode, "base", args...)
				if err != nil || hits.Load() != 1 || !strings.Contains(string(out), "0.123456789012345678") {
					t.Fatalf("exit=%v requests=%d out=%s stderr=%s", err, hits.Load(), out, stderr)
				}
			})
		}
	}
}

// Invalid numeric flags must never reach HTTP; API errors remain command errors.
func TestFixedProductStatusCLIRejection(t *testing.T) {
	for _, value := range []string{"2147483648", "-2147483649", "1.5", "null", "two"} {
		t.Run(value, func(t *testing.T) {
			var hits atomic.Int32
			peer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hits.Add(1); w.WriteHeader(500) }))
			defer peer.Close()
			out, stderr, err := credentialCLI(t, peer.URL, "oauth", "base", "--format", "json", "cex", "earn", "fixed", "products", "--status", value)
			if err == nil || hits.Load() != 0 || len(out) > 0 || !strings.Contains(string(stderr), "status") {
				t.Fatalf("exit=%v requests=%d out=%s stderr=%s", err, hits.Load(), out, stderr)
			}
		})
	}
	for _, status := range []int{400, 401, 403, 429, 500} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			var hits atomic.Int32
			peer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				hits.Add(1)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(status)
				if _, err := io.WriteString(w, `{"label":"INVALID_PARAM_VALUE","message":"fixture"}`); err != nil {
					t.Error(err)
				}
			}))
			defer peer.Close()
			out, stderr, err := credentialCLI(t, peer.URL, "oauth", "base", "--format", "json", "cex", "earn", "fixed", "products", "--status", "2", "--show-status", "2")
			if err == nil || hits.Load() != 1 || len(out) > 0 {
				t.Fatalf("exit=%v requests=%d out=%s stderr=%s", err, hits.Load(), out, stderr)
			}
		})
	}
}

// Help is an Agent-facing discovery surface and must advertise both new flags.
func TestFixedProductStatusCLIHelp(t *testing.T) {
	out, stderr, err := credentialCLI(t, "http://127.0.0.1:1", "oauth", "base", "cex", "earn", "fixed", "products", "--help")
	if err != nil || !strings.Contains(string(out), "--status") || !strings.Contains(string(out), "--show-status") {
		t.Fatalf("help missing filters: exit=%v out=%s stderr=%s", err, out, stderr)
	}
}
