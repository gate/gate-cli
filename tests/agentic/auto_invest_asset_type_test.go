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

// Run the built CLI through its original SDK/transport, asserting final HTTP
// bytes at a loopback peer. OAuth/HMAC credentials and the peer are synthetic.
func TestAutoInvestAssetTypeCLI(t *testing.T) {
	for _, mode := range []string{"oauth", "hmac"} {
		for _, assetType := range []int{1, 2} {
			t.Run(fmt.Sprintf("%s/%d", mode, assetType), func(t *testing.T) {
				var hits atomic.Int32
				peer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					hits.Add(1)
					if r.Method != "POST" || r.URL.Path != "/api/v4/earn/autoinvest/min_invest_amount" {
						t.Errorf("wrong route: %s %s", r.Method, r.URL.Path)
					}
					var body struct {
						Money     string                          `json:"money"`
						AssetType int                             `json:"asset_type"`
						Items     []struct{ Asset, Ratio string } `json:"items"`
					}
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						t.Error(err)
					}
					if body.AssetType != assetType || body.Money != "USDT" || len(body.Items) != 2 || body.Items[0].Ratio != "70" || body.Items[1].Ratio != "30" {
						t.Errorf("wire lost input: %+v", body)
					}
					if mode == "oauth" && r.Header.Get("Authorization") != "Bearer synthetic-test-base" {
						t.Error("OAuth header lost")
					}
					if mode == "hmac" && (r.Header.Get("KEY") != "synthetic-key" || r.Header.Get("SIGN") == "") {
						t.Error("HMAC header lost")
					}
					w.Header().Set("Content-Type", "application/json")
					if _, err := io.WriteString(w, `{"min_amount":"0.123456789012345678"}`); err != nil {
						t.Error(err)
					}
				}))
				defer peer.Close()
				body := fmt.Sprintf(`{"money":"USDT","asset_type":%d,"items":[{"asset":"BTC","ratio":"70"},{"asset":"ETH","ratio":"30"}]}`, assetType)
				out, stderr, err := credentialCLI(t, peer.URL, mode, "base", "--format", "json", "cex", "earn", "auto-invest", "min-amount", "--json", body)
				if err != nil || hits.Load() != 1 || !strings.Contains(string(out), "0.123456789012345678") {
					t.Fatalf("exit=%v requests=%d out=%s stderr=%s", err, hits.Load(), out, stderr)
				}
			})
		}
	}
}

// Invalid asset families fail locally; upstream errors must remain nonzero and
// use exactly one request rather than being mistaken for a successful query.
func TestAutoInvestAssetTypeCLIRejection(t *testing.T) {
	for _, value := range []string{"missing", "null", "0", "3", "-1", "2147483648", `"1"`, "true", "1.5"} {
		t.Run(value, func(t *testing.T) {
			var hits atomic.Int32
			peer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hits.Add(1); w.WriteHeader(500) }))
			defer peer.Close()
			body := `{"money":"USDT","items":[{"asset":"BTC","ratio":"100"}]`
			if value != "missing" {
				body += `,"asset_type":` + value
			}
			body += "}"
			out, stderr, err := credentialCLI(t, peer.URL, "oauth", "base", "--format", "json", "cex", "earn", "auto-invest", "min-amount", "--json", body)
			if err == nil || hits.Load() != 0 || len(out) > 0 || !strings.Contains(string(stderr), "asset_type") {
				t.Fatalf("invalid input escaped: exit=%v requests=%d out=%s stderr=%s", err, hits.Load(), out, stderr)
			}
		})
	}
	for _, status := range []int{400, 401, 403, 429, 500} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			var hits atomic.Int32
			peer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				hits.Add(1)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(status)
				if _, err := io.WriteString(w, `{"label":"INVALID_PARAM_VALUE","message":"fixture rejection"}`); err != nil {
					t.Error(err)
				}
			}))
			defer peer.Close()
			out, stderr, err := credentialCLI(t, peer.URL, "oauth", "base", "--format", "json", "cex", "earn", "auto-invest", "min-amount", "--json", `{"money":"USDT","asset_type":1,"items":[{"asset":"BTC","ratio":"100"}]}`)
			if err == nil || hits.Load() != 1 || len(out) > 0 {
				t.Fatalf("error became success: exit=%v requests=%d out=%s stderr=%s", err, hits.Load(), out, stderr)
			}
		})
	}
}
