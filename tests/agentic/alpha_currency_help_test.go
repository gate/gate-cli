//go:build agentic_e2e

package agentic_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

// Help must work offline, including required-currency commands without trade flags.
func TestAlphaCurrencyHelpCLI(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()
	for _, path := range [][]string{{"market", "currencies"}, {"market", "tickers"}, {"order", "list"}, {"order", "quote"}, {"order", "place"}} {
		for _, flag := range []string{"--help", "-h"} {
			t.Run(strings.Join(path, "/")+flag, func(t *testing.T) {
				args := append([]string{"cex", "alpha"}, path...)
				stdout, stderr, err := runCLI(t, server.URL, false, append(args, flag)...)
				if err != nil || len(stderr) != 0 {
					t.Fatalf("help exit=%v stderr=%s", err, stderr)
				}
				for _, want := range []string{"Alpha currency ID", "USDS02b5a8dd2f59a098", "not a symbol", "currency field", "gate-cli cex alpha market currencies --format json", "omit --currency", "--page"} {
					if !strings.Contains(string(stdout), want) {
						t.Errorf("help lacks %q: %s", want, stdout)
					}
				}
				if strings.Contains(string(stdout), "Filter by currency symbol") || strings.Contains(string(stdout), "Currency symbol (required)") {
					t.Errorf("obsolete currency guidance: %s", stdout)
				}
			})
		}
	}
	if calls.Load() != 0 {
		t.Fatalf("offline help made %d API calls", calls.Load())
	}
}

// Only the HTTP peer is mocked: execute real CLI -> SDK -> metadata -> ID ticker.
func TestAlphaCurrencyMetadataToTickerCLI(t *testing.T) {
	const id = "USDS02b5a8dd2f59a098"
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Method != http.MethodGet || r.Header.Get("KEY") != "" || r.Header.Get("Authorization") != "" {
			t.Errorf("public metadata request method/credentials changed")
		}
		w.Header().Set("Content-Type", "application/json")
		var body string
		switch r.URL.Path {
		case "/api/v4/alpha/currencies":
			if r.URL.Query().Has("currency") || r.URL.Query().Get("page") != "2" || r.URL.Query().Get("limit") != "1" {
				t.Errorf("metadata discovery must omit currency and preserve pagination: %s", r.URL.RawQuery)
			}
			body = `[{"currency":"USDS02b5a8dd2f59a098","name":"USDS","chain":"ETH","address":"synthetic-contract","status":1}]`
		case "/api/v4/alpha/tickers":
			if r.URL.Query().Get("currency") != id {
				t.Errorf("ticker currency=%q want ID", r.URL.Query().Get("currency"))
			}
			body = `[{"currency":"USDS02b5a8dd2f59a098","last":"0.9999"}]`
		default:
			t.Errorf("unexpected API request: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
		if _, err := w.Write([]byte(body)); err != nil {
			t.Error(err)
		}
	}))
	defer server.Close()
	stdout, stderr, err := runCLI(t, server.URL, false, "cex", "alpha", "market", "currencies", "--format", "json", "--limit", "1", "--page", "2")
	if err != nil {
		t.Fatalf("metadata exit=%v stderr=%s", err, stderr)
	}
	var currencies []struct{ Currency, Name, Chain, Address string }
	if err := json.Unmarshal(stdout, &currencies); err != nil || len(currencies) != 1 {
		t.Fatalf("metadata decode=%v stdout=%s", err, stdout)
	}
	asset := currencies[0]
	if asset.Currency != id || asset.Name != "USDS" || asset.Chain != "ETH" || asset.Address != "synthetic-contract" {
		t.Fatalf("metadata=%+v", asset)
	}
	stdout, stderr, err = runCLI(t, server.URL, false, "cex", "alpha", "market", "tickers", "--currency", asset.Currency, "--format", "json")
	if err != nil {
		t.Fatalf("ticker exit=%v stderr=%s", err, stderr)
	}
	var tickers []struct{ Currency, Last string }
	if err := json.Unmarshal(stdout, &tickers); err != nil || len(tickers) != 1 || tickers[0].Currency != id || tickers[0].Last != "0.9999" {
		t.Fatalf("ticker decode=%v stdout=%s", err, stdout)
	}
	if calls.Load() != 2 {
		t.Errorf("metadata/ticker calls=%d want=2", calls.Load())
	}
}

// The help correction must not add resolution requests or change API-error handling.
func TestAlphaCurrencyPassThroughCLI(t *testing.T) {
	for _, tc := range []struct {
		name, currency string
		flags          []string
		status         int
	}{
		{"id", "USDS02b5a8dd2f59a098", []string{"--currency", "USDS02b5a8dd2f59a098"}, 200},
		{"symbol", "USDS", []string{"--currency", "USDS"}, 200},
		{"omitted", "", nil, 200},
		{"empty", "", []string{"--currency", ""}, 200},
		{"api-error", "USDS", []string{"--currency", "USDS"}, 400},
		{"missing-value", "", []string{"--currency"}, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.URL.Path != "/api/v4/alpha/tickers" || r.URL.Query().Get("currency") != tc.currency {
					t.Errorf("unexpected request: %s", r.URL)
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.status)
				body := `[]`
				if tc.status != 200 {
					body = `{"label":"SYNTHETIC_NOT_FOUND","message":"synthetic rejection"}`
				}
				if _, err := w.Write([]byte(body)); err != nil {
					t.Error(err)
				}
			}))
			defer server.Close()
			args := append([]string{"cex", "alpha", "market", "tickers", "--format", "json"}, tc.flags...)
			stdout, stderr, err := runCLI(t, server.URL, false, args...)
			wantCalls := int32(1)
			if tc.status == 0 {
				wantCalls = 0
			}
			if calls.Load() != wantCalls || (err == nil) != (tc.status == 200) {
				t.Fatalf("exit=%v calls=%d want=%d stderr=%s", err, calls.Load(), wantCalls, stderr)
			}
			if tc.status != 200 && (len(strings.TrimSpace(string(stdout))) != 0 || len(stderr) == 0) {
				t.Errorf("failure must have diagnostic and no successful output: stdout=%s stderr=%s", stdout, stderr)
			}
		})
	}
}
