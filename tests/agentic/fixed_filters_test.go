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

// Reuse the existing isolated HMAC/OAuth CLI fixture; only the HTTP peer is synthetic.
func TestFixedFiltersCLI(t *testing.T) {
	for _, kind := range []string{"lends", "history"} {
		for _, mode := range []string{"oauth", "hmac"} {
			for _, tc := range []struct {
				name        string
				flags       []string
				sub, filter string
			}{
				{"omitted", nil, "", ""}, {"zero", []string{"--sub-business", "0"}, "0", ""},
				{"json", []string{"--business-filter", `[{"business":1,"sub_business":0}]`}, "", `[{"business":1,"sub_business":0}]`},
				{"both", []string{"--sub-business", "3", "--business-filter", `[ {"business":2, "sub_business":0} ]`}, "3", `[ {"business":2, "sub_business":0} ]`},
				{"array", []string{"--business-filter", "[]"}, "", "[]"},
			} {
				t.Run(kind+"/"+mode+"/"+tc.name, func(t *testing.T) {
					var calls atomic.Int32
					server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						calls.Add(1)
						path := "/api/v4/earn/fixed-term/user/lend"
						if kind == "history" {
							path = "/api/v4/earn/fixed-term/user/history"
						}
						if r.Method != "GET" || r.URL.Path != path {
							t.Errorf("request=%s %s", r.Method, r.URL.Path)
						}
						if mode == "oauth" {
							if r.Header.Get("Authorization") != "Bearer synthetic-test-base" || r.Header.Get("KEY") != "" {
								t.Error("OAuth request lost Bearer authentication")
							}
						} else if r.Header.Get("KEY") != "synthetic-key" || r.Header.Get("SIGN") == "" {
							t.Error("HMAC request lost its credentials")
						}
						q := r.URL.Query()
						if q.Get("sub_business") != tc.sub || q.Has("sub_business") != (tc.sub != "") || q.Get("business_filter") != tc.filter || q.Has("business_filter") != (tc.filter != "") {
							t.Error("SDK filters were dropped or changed")
						}
						if q.Get("product_id") != "42" || q.Get("order_id") != "9007199254740993" || q.Get("page") != "2" || q.Get("limit") != "3" || q.Get("asset") != "USDT" {
							t.Error("existing filters changed")
						}
						if kind == "history" && (q.Get("start_at") != "1700000000" || q.Get("end_at") != "1700000010" || q.Get("type") != "3") {
							t.Error("history range/type changed")
						}
						if kind == "lends" && q.Get("order_type") != "2" {
							t.Error("order type changed")
						}
						w.Header().Set("Content-Type", "application/json")
						moneyField := "principal"
						if kind == "history" {
							moneyField = "amount"
						}
						payload := fmt.Sprintf(`{"code":0,"data":{"list":[{"order_id":9007199254740993,"asset":"USDT","%s":"0.123456789012345678"}],"total":1}}`, moneyField)
						if _, err := io.WriteString(w, payload); err != nil {
							t.Error(err)
						}
					}))
					t.Cleanup(server.Close)
					args := []string{"--format", "json", "cex", "earn", "fixed", kind, "--page", "2", "--limit", "3", "--asset", "USDT", "--product-id", "42", "--order-id", "9007199254740993"}
					if kind == "history" {
						args = append(args, "--type", "3", "--start-at", "1700000000", "--end-at", "1700000010")
					} else {
						args = append(args, "--order-type", "2")
					}
					args = append(args, tc.flags...)
					stdout, stderr, err := credentialCLI(t, server.URL, mode, "base", args...)
					if err != nil || calls.Load() != 1 {
						t.Fatalf("exit=%v requests=%d stderr=%s", err, calls.Load(), stderr)
					}
					var result struct {
						Data struct {
							List []struct {
								OrderID                  int64 `json:"order_id"`
								Asset, Principal, Amount string
							}
							Total int
						}
					}
					if err := json.Unmarshal(stdout, &result); err != nil {
						t.Fatal(err)
					}
					if len(result.Data.List) != 1 || result.Data.Total != 1 {
						t.Fatalf("unexpected result %s", stdout)
					}
					row := result.Data.List[0]
					amount := row.Principal
					if kind == "history" {
						amount = row.Amount
					}
					if row.OrderID != 9007199254740993 || row.Asset != "USDT" || amount != "0.123456789012345678" {
						t.Errorf("result lost data/precision: %s", stdout)
					}

				})
			}
		}
	}
}

// Invalid new input fails without HTTP even when synthetic OAuth credentials are available.
func TestFixedFiltersCLIInvalid(t *testing.T) {
	for _, kind := range []string{"lends", "history"} {
		for _, flags := range [][]string{
			{"--sub-business", "2147483648"}, {"--sub-business", "-2147483649"}, {"--sub-business", "x"},
			{"--business-filter", ""}, {"--business-filter", "1"}, {"--business-filter", "["}, {"--business-filter", "null"}, {"--business-filter", "{}"}, {"--business-filter", "[null]"}, {"--business-filter", "[1]"},
		} {
			t.Run(kind+"/"+strings.Join(flags, "/"), func(t *testing.T) {
				var calls atomic.Int32
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					w.Header().Set("Content-Type", "application/json")
					if _, err := io.WriteString(w, `{"code":0,"data":{"list":[],"total":0}}`); err != nil {
						t.Error(err)
					}
				}))
				t.Cleanup(server.Close)
				args := append([]string{"--format", "json", "cex", "earn", "fixed", kind}, flags...)
				stdout, stderr, err := credentialCLI(t, server.URL, "oauth", "base", args...)
				if err == nil || calls.Load() != 0 || len(stdout) != 0 || len(stderr) == 0 {
					t.Errorf("exit=%v requests=%d stdout=%s stderr=%s", err, calls.Load(), stdout, stderr)
				}
			})
		}
	}
}

// Authentication and remote failures must retain the existing exit/error contract.
func TestFixedFiltersCLIErrorsAndAuth(t *testing.T) {
	for _, kind := range []string{"lends", "history"} {
		for _, status := range []int{400, 401, 403, 429, 503} {
			t.Run(fmt.Sprintf("%s/%d", kind, status), func(t *testing.T) {
				var calls atomic.Int32
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					w.Header().Set("Content-Type", "application/json")
					w.Header().Set("X-Gate-Trace-Id", "synthetic-fixed-filter-trace")
					w.WriteHeader(status)
					if _, err := io.WriteString(w, `{"label":"SYNTHETIC_FILTER_ERROR","message":"synthetic error"}`); err != nil {
						t.Error(err)
					}
				}))
				t.Cleanup(server.Close)
				stdout, stderr, err := credentialCLI(t, server.URL, "oauth", "base", "--format", "json", "cex", "earn", "fixed", kind, "--sub-business", "0", "--business-filter", `[{"business":1,"sub_business":0}]`)
				if err == nil || len(stdout) != 0 || calls.Load() != 1 || !strings.Contains(string(stderr), "synthetic-fixed-filter-trace") {
					t.Errorf("exit=%v requests=%d stdout=%s stderr=%s", err, calls.Load(), stdout, stderr)
				}
			})
		}
		t.Run(kind+"/no-credentials", func(t *testing.T) {
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(500) }))
			t.Cleanup(server.Close)
			stdout, stderr, err := credentialCLI(t, server.URL, "anonymous", "", "--format", "json", "cex", "earn", "fixed", kind, "--sub-business", "0")
			if err == nil || calls.Load() != 0 || len(stdout) != 0 || !strings.Contains(strings.ToLower(string(stderr)), "auth") {
				t.Errorf("exit=%v requests=%d stdout=%s stderr=%s", err, calls.Load(), stdout, stderr)
			}
		})
	}
}

func TestFixedFiltersCLIHelp(t *testing.T) {
	for _, kind := range []string{"lends", "history"} {
		t.Run(kind, func(t *testing.T) {
			stdout, stderr, err := runCLI(t, "http://127.0.0.1:1", false, "cex", "earn", "fixed", kind, "--help")
			if err != nil || !strings.Contains(string(stdout), "--sub-business") || !strings.Contains(string(stdout), "--business-filter") {
				t.Errorf("exit=%v stdout=%s stderr=%s", err, stdout, stderr)
			}
		})
	}
}

// Existing invocations without new flags keep the original default query exactly.
func TestFixedFiltersCLIDefaults(t *testing.T) {
	for _, kind := range []string{"lends", "history"} {
		t.Run(kind, func(t *testing.T) {
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				q := r.URL.Query()
				key := "order_type"
				if kind == "history" {
					key = "type"
				}
				if q.Get("page") != "1" || q.Get("limit") != "10" || q.Get(key) != "1" || len(q) != 3 {
					t.Errorf("default query changed: %s", r.URL.RawQuery)
				}
				w.Header().Set("Content-Type", "application/json")
				if _, err := io.WriteString(w, `{"code":0,"data":{"list":[],"total":0}}`); err != nil {
					t.Error(err)
				}
			}))
			t.Cleanup(server.Close)
			stdout, stderr, err := credentialCLI(t, server.URL, "oauth", "base", "--format", "json", "cex", "earn", "fixed", kind)
			if err != nil || calls.Load() != 1 || !json.Valid(stdout) {
				t.Errorf("exit=%v requests=%d stdout=%s stderr=%s", err, calls.Load(), stdout, stderr)
			}
		})
	}
}
