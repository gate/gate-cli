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
)

// Full CLI -> flag validation -> SDK -> HTTP -> output; only the API peer is fake.
func TestSpotInsuranceRangeCLI(t *testing.T) {
	for _, tc := range []struct {
		name  string
		flags []string
		valid bool
	}{
		{"missing-both", nil, false}, {"missing-from", []string{"--to", "1700000001"}, false},
		{"missing-to", []string{"--from", "1700000000"}, false},
		{"zero", []string{"--from", "0", "--to", "1700000001"}, false},
		{"negative", []string{"--from", "-1", "--to", "1700000001"}, false},
		{"reversed", []string{"--from", "1700000002", "--to", "1700000001"}, false},
		{"equal", []string{"--from", "1700000000", "--to", "1700000000"}, true},
		{"ordered", []string{"--from", "1700000000", "--to", "1700000001"}, true},
		{"overflow", []string{"--from", "9223372036854775808", "--to", "1700000001"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.Method != "GET" || r.URL.Path != "/api/v4/spot/insurance_history" {
					t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
				}
				if r.Header.Get("Authorization") != "" || r.Header.Get("KEY") != "" {
					t.Error("public query sent credentials")
				}
				q := r.URL.Query()
				if q.Get("currency") != "BTC" || q.Get("business") != "margin" || q.Get("limit") != "3" || q.Get("from") != "1700000000" {
					t.Errorf("query=%s", r.URL.RawQuery)
				}
				if tc.name == "equal" && q.Get("to") != "1700000000" || tc.name == "ordered" && q.Get("to") != "1700000001" {
					t.Errorf("to=%q", q.Get("to"))
				}
				w.Header().Set("Content-Type", "application/json")
				if _, err := io.WriteString(w, `[{"currency":"BTC","balance":"1021.21","time":1700000000}]`); err != nil {
					t.Error(err)
				}
			}))
			t.Cleanup(server.Close)
			args := append([]string{"cex", "spot", "market", "insurance", "--currency", "BTC", "--limit", "3", "--format", "json"}, tc.flags...)
			stdout, stderr, err := runCLI(t, server.URL, false, args...)
			if (err == nil) != tc.valid {
				t.Errorf("valid=%v exit=%v stderr=%s", tc.valid, err, stderr)
			}
			want := int32(0)
			if tc.valid {
				want = 1
			}
			if calls.Load() != want {
				t.Errorf("requests=%d want=%d", calls.Load(), want)
			}
			if !tc.valid {
				if len(strings.TrimSpace(string(stdout))) != 0 {
					t.Error("invalid input emitted successful data")
				}
				if len(stderr) == 0 {
					t.Error("missing validation diagnostic")
				}
				return
			}
			var records []struct {
				Currency, Balance string
				Time              int64
			}
			if err := json.Unmarshal(stdout, &records); err != nil {
				t.Fatalf("invalid JSON: %v stdout=%s", err, stdout)
			}
			if len(records) != 1 || records[0].Currency != "BTC" || records[0].Balance != "1021.21" || records[0].Time != 1700000000 {
				t.Errorf("records=%+v", records)
			}
		})
	}
}

// Query validation must happen before the API peer can return a false green result.
func TestTradfiKlineCLI(t *testing.T) {
	for _, tc := range []struct {
		name  string
		flags []string
		want  string
	}{
		{"default", nil, "1m"}, {"native", []string{"--type", "1m"}, "1m"}, {"legacy", []string{"--type", "M1"}, "1m"}, {"invalid", []string{"--type", "M5"}, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.URL.Query().Get("kline_type") != tc.want {
					t.Errorf("kline_type=%q want=%q", r.URL.Query().Get("kline_type"), tc.want)
				}
				w.Header().Set("Content-Type", "application/json")
				if _, err := io.WriteString(w, `{"code":0,"data":{"list":[{"t":1700000000,"o":"1","c":"2","h":"3","l":"0.5"}]}}`); err != nil {
					t.Error(err)
				}
			}))
			t.Cleanup(server.Close)
			args := append([]string{"cex", "tradfi", "market", "kline", "--symbol", "XAUUSD", "--limit", "3", "--format", "json"}, tc.flags...)
			stdout, stderr, err := runCLI(t, server.URL, false, args...)
			if tc.want == "" {
				if err == nil || calls.Load() != 0 || len(stdout) != 0 {
					t.Errorf("invalid type exit=%v calls=%d stdout=%s", err, calls.Load(), stdout)
				}
				return
			}
			if err != nil || calls.Load() != 1 {
				t.Fatalf("exit=%v calls=%d stdout=%s stderr=%s", err, calls.Load(), stdout, stderr)
			}
			var result struct {
				Data struct {
					List []struct {
						T          int64
						O, C, H, L string
					}
				}
			}
			if err := json.Unmarshal(stdout, &result); err != nil {
				t.Fatal(err)
			}
			if len(result.Data.List) != 1 {
				t.Fatalf("klines=%d want1", len(result.Data.List))
			}
			candle := result.Data.List[0]
			if candle.T != 1700000000 || candle.O != "1" || candle.C != "2" || candle.H != "3" || candle.L != "0.5" {
				t.Errorf("candle=%+v", candle)
			}
		})
	}
}

// SDK types and decimal output are exercised through both actual CLI commands.
func TestInsuranceBalanceCLI(t *testing.T) {
	for _, kind := range []string{"futures", "delivery"} {
		for _, format := range []string{"json", "table"} {
			t.Run(kind+"/"+format, func(t *testing.T) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					if _, err := io.WriteString(w, `[{"b":-42542.804042324,"t":1793232000},{"b":9007199254740993.123456789012345678,"t":1793232001}]`); err != nil {
						t.Error(err)
					}
				}))
				t.Cleanup(server.Close)
				stdout, stderr, err := runCLI(t, server.URL, false, "cex", kind, "market", "insurance", "--limit", "3", "--format", format)
				if err != nil || !strings.Contains(string(stdout), "-42542.804042324") || !strings.Contains(string(stdout), "9007199254740993.123456789012345678") {
					t.Errorf("exit=%v stdout=%s stderr=%s", err, stdout, stderr)
				}
			})
		}
	}
}

// Successful help must prove a valid path, without calling an API or changing flags.
func TestUnknownCommandHelpCLI(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	t.Cleanup(server.Close)
	invalid := [][]string{{"nosuchcmd", "--help"}, {"nosuchcmd", "-h"}, {"cex", "cross-ex", "market", "symbols", "--help"}, {"cex", "nosuchcmd", "--help"}, {"cex", "nosuchcmd", "-h"}, {"cex", "nosuchcmd"}, {"--format", "json", "cex", "nosuchcmd", "--help"}}
	// Explicit help topics must fail in both output formats and before Cobra's flag-help early return.
	for _, topic := range [][]string{{"nosuch"}, {"cex", "cross-ex", "market", "symbols"}, {"cex", "alpha", "market", "tickers", "extra"}} {
		for _, format := range []string{"json", "table"} {
			for _, flag := range []string{"", "--help", "-h"} {
				args := append([]string{"--format", format, "help"}, topic...)
				if flag != "" {
					args = append(args, flag)
				}
				invalid = append(invalid, args)
			}
		}
	}
	for _, args := range invalid {
		t.Run(strings.Join(args, "/"), func(t *testing.T) {
			stdout, stderr, err := runCLI(t, server.URL, false, args...)
			var exit *exec.ExitError
			if !errors.As(err, &exit) || exit.ExitCode() != 1 || !strings.Contains(string(stderr), "unknown command") || len(stdout) != 0 {
				t.Errorf("exit=%v stdout=%s stderr=%s", err, stdout, stderr)
			}
		})
	}
	for _, args := range [][]string{{"cex", "--help"}, {"cex", "spot", "market", "ticker", "--help"}, {"cex", "spot", "market", "ticker", "--pair", "nosuchcmd", "--help"}, {"--profile", "nosuchcmd", "cex", "--help"}, {"help"}, {"help", "cex"}, {"help", "cex", "alpha", "market", "tickers"}, {"help", "cex", "alpha", "order", "place"}, {"help", "info", "coin-overview"}, {"--profile", "nosuchcmd", "help", "cex", "alpha", "market", "tickers"}, {"help", "cex", "alpha", "market", "tickers", "--help=false"}, {"help", "--", "cex", "alpha", "market", "tickers"}} {
		t.Run("valid/"+strings.Join(args, "/"), func(t *testing.T) {
			stdout, stderr, err := runCLI(t, server.URL, false, args...)
			if err != nil || !strings.Contains(string(stdout), "Usage:") {
				t.Errorf("exit=%v stdout=%s stderr=%s", err, stdout, stderr)
			}
		})
	}
	if calls.Load() != 0 {
		t.Fatalf("help made %d API calls", calls.Load())
	}
}

// The transport receive bound and user-selected stdout bound are independent.
func TestCurrenciesLargeResponseCLI(t *testing.T) {
	row := `{"currency":"SYNTH","name":"` + strings.Repeat("x", 4096) + `"}`
	raw := "[" + strings.Repeat(row+",", 529) + row + "]"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if _, err := io.WriteString(w, raw); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(server.Close)
	stdout, stderr, err := runCLI(t, server.URL, false, "cex", "spot", "market", "currencies", "--format", "json", "--max-output-bytes", "0")
	var rows []map[string]interface{}
	if err != nil {
		t.Fatalf("exit=%v stderr=%s", err, stderr)
	}
	if err := json.Unmarshal(stdout, &rows); err != nil || len(rows) != 530 {
		t.Errorf("valid full list not returned: count=%d error=%v", len(rows), err)
	}
	stdout, stderr, err = runCLI(t, server.URL, false, "cex", "spot", "market", "currencies", "--format", "json", "--max-output-bytes", "1024")
	var placeholder struct {
		Truncated bool `json:"truncated"`
		Max       int  `json:"max_output_bytes"`
	}
	if err != nil {
		t.Fatalf("bounded output exit=%v stderr=%s", err, stderr)
	}
	if err := json.Unmarshal(stdout, &placeholder); err != nil || !placeholder.Truncated || placeholder.Max != 1024 {
		t.Errorf("stdout bound ignored: %s error=%v", stdout, err)
	}
}

// Valid input still preserves real HTTP and decode failures rather than claiming success.
func TestInsuranceErrorsCLI(t *testing.T) {
	for _, kind := range []string{"futures", "delivery"} {
		for _, tc := range []struct {
			name, body string
			status     int
		}{
			{"invalid-balance", `[{"b":true,"t":1793232000}]`, 200}, {"invalid-json", "[", 200},
			{"rejected", `{"label":"INVALID_ARGUMENT","message":"synthetic invalid request"}`, 400},
			{"unavailable", `{"label":"SERVICE_UNAVAILABLE","message":"synthetic unavailable"}`, 503},
		} {
			t.Run(kind+"/"+tc.name, func(t *testing.T) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.Header().Set("X-Gate-Trace-Id", "synthetic-insurance-trace")
					w.WriteHeader(tc.status)
					if _, err := io.WriteString(w, tc.body); err != nil {
						t.Error(err)
					}
				}))
				t.Cleanup(server.Close)
				stdout, stderr, err := runCLI(t, server.URL, false, "cex", kind, "market", "insurance", "--limit", "3", "--format", "json")
				if err == nil || len(stdout) != 0 || !strings.Contains(string(stderr), "synthetic-insurance-trace") {
					t.Errorf("exit=%v stdout=%s stderr=%s", err, stdout, stderr)
				}
				if tc.status == 200 && (!strings.Contains(string(stderr), "decode") || tc.body != "[" && strings.Contains(string(stderr), tc.body)) {
					t.Errorf("decoding cause was hidden: %s", stderr)
				}
			})
		}
	}
}
