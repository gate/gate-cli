//go:build agentic_e2e

package agentic_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/gate/gate-cli/tests/contracts"
	"gopkg.in/yaml.v3"
)

// Keep one HOME across root config, GitHub Intel and GitLab CEX commands. Only
// the two remote HTTP peers are mocked; process, auth, SDK and persistence run.
func TestGitHubSyncCrossDomainLifecycle(t *testing.T) {
	for _, domain := range []string{"info", "news"} {
		for _, mode := range []string{"normal", "intel error", "CEX error", "invalid JSON"} {
			t.Run(domain+"/"+mode, func(t *testing.T) {
				var rpcCalls, toolCalls, apiCalls atomic.Int32
				tool := "info_coin_get_coin_info"
				if domain == "news" {
					tool = "news_feed_search_news"
				}
				intel := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					rpcCalls.Add(1)
					if r.Header.Get("Authorization") != "Bearer synthetic-intel-token" || r.Header.Get("KEY") != "" || r.Header.Get("SIGN") != "" {
						t.Error("trading auth crossed into Intel transport")
					}
					var req struct {
						ID     json.RawMessage `json:"id"`
						Method string          `json:"method"`
						Params struct {
							Name      string                 `json:"name"`
							Arguments map[string]interface{} `json:"arguments"`
						} `json:"params"`
					}
					if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
						t.Error(err)
						return
					}
					w.Header().Set("Content-Type", "application/json")
					var result interface{}
					switch req.Method {
					case "initialize":
						result = map[string]interface{}{"protocolVersion": "2024-11-05", "capabilities": map[string]interface{}{}, "serverInfo": map[string]string{"name": "isolated-fixture", "version": "1"}}
					case "tools/list":
						result = map[string]interface{}{"tools": []interface{}{map[string]interface{}{"name": tool, "inputSchema": map[string]interface{}{"type": "object"}}}}
					case "tools/call":
						toolCalls.Add(1)
						if req.Params.Name != tool || req.Params.Arguments["query"] != "BTC" {
							t.Error("Intel tool identity or query lost")
						}
						for _, name := range []string{"name", "format", "profile", "debug", "api_key", "api_secret", "max_output_bytes"} {
							if _, exists := req.Params.Arguments[name]; exists {
								t.Errorf("CLI-only argument forwarded to MCP: %s", name)
							}
						}
						result = map[string]interface{}{"isError": mode == "intel error", "content": []interface{}{map[string]string{"type": "text", "text": `{"fixture":"intel-result"}`}}}
					default:
						t.Errorf("unexpected RPC method: %s", req.Method)
						return
					}
					if err := json.NewEncoder(w).Encode(map[string]interface{}{"jsonrpc": "2.0", "id": req.ID, "result": result}); err != nil {
						t.Error(err)
					}
				}))
				t.Cleanup(intel.Close)
				api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					apiCalls.Add(1)
					if r.Method != "GET" || r.URL.Path != "/api/v4/futures/usdt/autoorder/v1/trail/change_log" || r.URL.Query().Get("id") != "9007199254740993" || r.URL.Query().Get("page_num") != "1" {
						t.Errorf("CEX route/query changed: %s %s", r.Method, r.URL)
					}
					if err := contracts.CheckAuth(r, nil, "hmac"); err != nil {
						t.Error(err)
					}
					w.Header().Set("Content-Type", "application/json")
					if mode == "CEX error" {
						_, _ = fmt.Fprint(w, `{"code":-1,"message":"fixture business rejection"}`)
						return
					}
					_, _ = fmt.Fprint(w, `{"code":0,"data":{"change_log":[{"updated_at":"1793627143","amount":"0.01"}]}}`)
				}))
				t.Cleanup(api.Close)
				home := t.TempDir()
				path := filepath.Join(home, ".gate-cli", "config.yaml")
				if err := os.Mkdir(filepath.Dir(path), 0700); err != nil {
					t.Fatal(err)
				}
				original := map[string]interface{}{
					"default_profile": "default", "custom": "preserve",
					"profiles": map[string]interface{}{"default": map[string]interface{}{"api_key": "synthetic-key", "api_secret": "synthetic-secret", "base_url": api.URL}},
					"intel":    map[string]interface{}{"info_mcp_url": intel.URL, "news_mcp_url": intel.URL, "bearer_token": "synthetic-intel-token"},
					"oauth2":   map[string]interface{}{"tokens": []interface{}{map[string]interface{}{"access_token": "synthetic-oauth-at", "refresh_token": "synthetic-oauth-rt"}}},
				}
				raw, err := yaml.Marshal(original)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, raw, 0600); err != nil {
					t.Fatal(err)
				}
				argsJSON := `{"query":"BTC"}`
				if domain == "info" {
					argsJSON = `{"query":"BTC","query_type":"symbol"}`
				}
				if mode == "invalid JSON" {
					argsJSON = `{"query":`
				}
				out, stderr, err := sameHomeCLI(t, home, api.URL, "--format", "json", "--profile", "default", "--api-key", "synthetic-key", "--api-secret", "synthetic-secret", domain, "invoke", "--name", tool, "--args-json", argsJSON)
				failure := mode == "intel error" || mode == "invalid JSON"
				if (err != nil) != failure || failure && (!intelFailedExit(err) || len(out) != 0 || !json.Valid(stderr)) || !failure && (len(stderr) != 0 || !contracts.EqualJSON(out, []byte(`{"fixture":"intel-result"}`))) {
					t.Fatalf("Intel result changed: %v stdout=%s stderr=%s", err, out, stderr)
				}
				if mode == "invalid JSON" {
					if rpcCalls.Load() != 0 {
						t.Fatal("invalid arguments reached MCP")
					}
				} else if toolCalls.Load() != 1 {
					t.Fatalf("MCP calls=%d, want one", toolCalls.Load())
				}
				out, stderr, err = sameHomeCLI(t, home, api.URL, "--format", "json", "cex", "futures", "trail", "log", "--id", "9007199254740993")
				if (err != nil) != (mode == "CEX error") || mode == "CEX error" && (!intelFailedExit(err) || len(out) != 0 || !json.Valid(stderr) || !bytes.Contains(stderr, []byte("fixture business rejection"))) || apiCalls.Load() != 1 {
					t.Fatalf("CEX result changed: %v calls=%d stdout=%s stderr=%s", err, apiCalls.Load(), out, stderr)
				}
				if mode != "CEX error" {
					var result struct {
						Code int `json:"code"`
						Data struct {
							ChangeLog []struct {
								UpdatedAt string `json:"updated_at"`
								Amount    string `json:"amount"`
							} `json:"change_log"`
						} `json:"data"`
					}
					if json.Unmarshal(out, &result) != nil || len(stderr) != 0 || result.Code != 0 || len(result.Data.ChangeLog) != 1 || result.Data.ChangeLog[0].UpdatedAt != "1793627143" || result.Data.ChangeLog[0].Amount != "0.01" {
						t.Fatalf("CEX business fields lost: stdout=%s stderr=%s", out, stderr)
					}
				}
				// The root writer must preserve both domains and existing OAuth state.
				_, stderr, err = sameHomeCLI(t, home, api.URL, "config", "set", "base-url", api.URL, "--profile", "secondary")
				if err != nil || len(stderr) != 0 {
					t.Fatalf("config set: %v %s", err, stderr)
				}
				after, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				var document map[string]interface{}
				if err := yaml.Unmarshal(after, &document); err != nil {
					t.Fatal(err)
				}
				for _, key := range []string{"custom", "intel", "oauth2", "default_profile"} {
					if !reflect.DeepEqual(document[key], original[key]) {
						t.Fatalf("config write erased %s", key)
					}
				}
				profiles, ok := document["profiles"].(map[string]interface{})
				if !ok {
					t.Fatal("config write lost profiles")
				}
				secondary, ok := profiles["secondary"].(map[string]interface{})
				originalProfiles, originalOK := original["profiles"].(map[string]interface{})
				if !ok || !originalOK || secondary["base_url"] != api.URL || !reflect.DeepEqual(profiles["default"], originalProfiles["default"]) {
					t.Fatal("config write was a no-op or changed the original profile")
				}
				listed, stderr, err := sameHomeCLI(t, home, api.URL, "config", "list")
				if err != nil || len(stderr) != 0 {
					t.Fatalf("config list: %v %s", err, stderr)
				}
				for _, secret := range []string{"synthetic-key", "synthetic-secret", "synthetic-intel-token", "synthetic-oauth-at", "synthetic-oauth-rt"} {
					if bytes.Contains(listed, []byte(secret)) {
						t.Fatal("shared config list leaked credentials")
					}
				}
				_, _, err = sameHomeCLI(t, home, api.URL, "config", "set", "unknown-key", "value")
				unchanged, readErr := os.ReadFile(path)
				if !intelFailedExit(err) || readErr != nil || !bytes.Equal(after, unchanged) {
					t.Fatal("failed config write altered shared state")
				}
			})
		}
	}
}
