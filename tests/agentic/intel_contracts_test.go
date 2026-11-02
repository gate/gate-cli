//go:build agentic_e2e

package agentic_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gate/gate-cli/internal/agentfeature"
	"github.com/gate/gate-cli/tests/contracts"
)

type intelRPCRequest struct {
	ID     json.RawMessage `json:"id"`
	Method string          `json:"method"`
	Params struct {
		Name      string                 `json:"name"`
		Arguments map[string]interface{} `json:"arguments"`
	} `json:"params"`
}

type intelPeerCalls struct {
	atomic.Int32
	RPC atomic.Int32
}

// Only the remote MCP peer is replaced. Strict auth and a finite tools/call
// counter catch CLI credential leakage and accidental replay across all leaves.
func intelPeer(t *testing.T, tool string, call func(http.ResponseWriter, *http.Request, intelRPCRequest)) (string, *intelPeerCalls) {
	t.Helper()
	var calls intelPeerCalls
	peer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.RPC.Add(1)
		if r.Method != http.MethodPost || r.Header.Get("Authorization") != "Bearer synthetic-intel-token" || r.Header.Get("KEY") != "" || r.Header.Get("SIGN") != "" {
			t.Error("unexpected Intel transport or mixed trading credentials")
		}
		var req intelRPCRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Error(err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Gate-Trace-Id", "intel-contract-trace")
		switch req.Method {
		case "initialize":
			intelReply(t, w, req.ID, map[string]interface{}{"protocolVersion": "2024-11-05", "capabilities": map[string]interface{}{}, "serverInfo": map[string]string{"name": "contract-peer", "version": "1"}})
		case "tools/list":
			intelReply(t, w, req.ID, map[string]interface{}{"tools": []interface{}{map[string]interface{}{"name": tool, "inputSchema": map[string]interface{}{"type": "object"}}}})
		case "tools/call":
			calls.Add(1)
			if req.Params.Name != tool {
				t.Errorf("remote tool=%s, want %s", req.Params.Name, tool)
			}
			call(w, r, req)
		default:
			t.Errorf("unexpected RPC method %q", req.Method)
			w.WriteHeader(http.StatusBadRequest)
		}
	}))
	t.Cleanup(peer.Close)
	return peer.URL, &calls
}

func intelReply(t *testing.T, w http.ResponseWriter, id json.RawMessage, result interface{}) {
	t.Helper()
	if err := json.NewEncoder(w).Encode(map[string]interface{}{"jsonrpc": "2.0", "id": id, "result": result}); err != nil {
		t.Error(err)
	}
}

func intelContractHome(t *testing.T, endpoint, timeout string) string {
	t.Helper()
	home := t.TempDir()
	if err := os.Mkdir(filepath.Join(home, ".gate-cli"), 0700); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(map[string]interface{}{"intel": map[string]string{"info_mcp_url": endpoint, "news_mcp_url": endpoint, "bearer_token": "synthetic-intel-token", "http_timeout": timeout}})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".gate-cli", "config.yaml"), data, 0600); err != nil {
		t.Fatal(err)
	}
	return home
}

// cwd is intentionally the isolated HOME, so --args-file exercises the actual
// containment policy without creating input files in the source checkout.
func intelCLI(t *testing.T, home string, env []string, args ...string) ([]byte, []byte, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 12*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, cliBinary, args...)
	cmd.Dir = home
	cmd.Env = append(childEnv(home, "", false), env...)
	var out, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &stderr
	err := cmd.Run()
	return out.Bytes(), stderr.Bytes(), err
}

// Root parse errors retain the historical structured error plus optional human
// hint/Agent diagnostic lines. Validate every part instead of treating the
// entire stderr stream as one JSON document or ignoring its suffix.
func intelParseError(t *testing.T, stderr []byte, path, invalid string) {
	t.Helper()
	decoder := json.NewDecoder(bytes.NewReader(stderr))
	var envelope struct {
		Error struct {
			Status  int    `json:"status"`
			Label   string `json:"label"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := decoder.Decode(&envelope); err != nil || envelope.Error.Status != 400 || envelope.Error.Label != "INVALID_ARGUMENTS" || !strings.Contains(envelope.Error.Message, invalid) {
		t.Fatalf("invalid input lost structured error: %s", stderr)
	}
	tail := strings.TrimSpace(string(stderr[decoder.InputOffset():]))
	hints, diagnostics := 0, 0
	if tail == "" {
		return
	}
	for _, line := range strings.Split(tail, "\n") {
		switch {
		case strings.HasPrefix(line, "Hint: "):
			hints++
			if invalid == "unexpected-positional" && line != "Hint: try gate-cli "+path {
				t.Errorf("incorrect canonical leaf hint: %s", line)
			}
		case agentfeature.Commands && strings.HasPrefix(line, "gate_cli_diagnostic="):
			diagnostics++
			var diagnostic struct {
				ErrorType string `json:"error_type"`
			}
			if json.Unmarshal([]byte(strings.TrimPrefix(line, "gate_cli_diagnostic=")), &diagnostic) != nil || diagnostic.ErrorType == "" {
				t.Errorf("invalid Agent diagnostic: %s", line)
			}
		default:
			t.Errorf("unexpected stderr suffix: %s", line)
		}
	}
	if hints > 1 || diagnostics > 1 {
		t.Fatalf("duplicate diagnostics: %s", stderr)
	}
}

func intelFailedExit(err error) bool {
	var exit *exec.ExitError
	return errors.As(err, &exit) && exit.ExitCode() == 1
}

// Freeze both the command-to-wire mapping and payload; help-only tests and a
// successful hidden invoke cannot prove that all 50 public leaves still work.
func TestIntelAllLeafContracts(t *testing.T) {
	for _, tc := range contracts.IntelCases(t) {
		t.Run(tc.Path, func(t *testing.T) {
			endpoint, calls := intelPeer(t, tc.Tool, func(w http.ResponseWriter, _ *http.Request, req intelRPCRequest) {
				got, err := json.Marshal(req.Params.Arguments)
				want, wantErr := json.Marshal(tc.Arguments)
				if err != nil || wantErr != nil || !contracts.EqualJSON(got, want) {
					t.Errorf("argument contract changed: got=%s want=%s", got, want)
				}
				intelReply(t, w, req.ID, map[string]interface{}{"structuredContent": tc.Reply})
			})
			home := intelContractHome(t, endpoint, "5s")
			keys := make([]string, 0, len(tc.Arguments))
			for key := range tc.Arguments {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			for _, format := range []string{"json", "pretty"} {
				args := []string{"--format", format, "--profile", "default", "--api-key", "synthetic-global-key", "--api-secret", "synthetic-global-secret", "--debug=false", "--verbose=false", "--help=false", "--max-output-bytes", "0"}
				args = append(args, strings.Fields(tc.Path)...)
				for _, key := range keys {
					value := tc.Arguments[key]
					text, ok := value.(string)
					if !ok {
						raw, err := json.Marshal(value)
						if err != nil {
							t.Fatal(err)
						}
						text = string(raw)
					}
					args = append(args, "--"+strings.ReplaceAll(key, "_", "-"), text)
				}
				before := calls.Load()
				out, stderr, err := intelCLI(t, home, nil, args...)
				if err != nil || len(stderr) != 0 || calls.Load() != before+1 {
					t.Fatalf("leaf failed: %v calls=%d stdout=%s stderr=%s", err, calls.Load()-before, out, stderr)
				}
				want, _ := json.Marshal(tc.Reply)
				if format == "json" && !contracts.EqualJSON(out, want) {
					t.Fatalf("business output lost: %s", out)
				}
				if format == "pretty" {
					for _, expected := range tc.PrettyContains {
						if !bytes.Contains(out, []byte(expected)) {
							t.Errorf("pretty output lost %q: %s", expected, out)
						}
					}
				}
			}
			for _, invalid := range []string{"unexpected-positional", "--test-unknown-flag"} {
				before := calls.Load()
				rpcBefore := calls.RPC.Load()
				args := append([]string{"--format", "json"}, strings.Fields(tc.Path)...)
				out, stderr, err := intelCLI(t, home, nil, append(args, invalid)...)
				if !intelFailedExit(err) || len(out) != 0 || calls.Load() != before || calls.RPC.Load() != rpcBefore {
					t.Fatalf("invalid leaf input reached tool: %v stdout=%s stderr=%s", err, out, stderr)
				}
				intelParseError(t, stderr, tc.Path, strings.TrimPrefix(invalid, "--"))
			}
		})
	}
}

// Exercise the same independent boundary inputs through the public leaf, real
// flag parser, validator and renderer. Rejected arguments must cause zero RPC.
func TestIntelLeafArgumentBoundaries(t *testing.T) {
	for _, tc := range contracts.IntelBoundaries(t) {
		t.Run(tc.Name, func(t *testing.T) {
			endpoint, calls := intelPeer(t, tc.Tool, func(w http.ResponseWriter, _ *http.Request, req intelRPCRequest) {
				got, err := json.Marshal(req.Params.Arguments)
				want, wantErr := json.Marshal(tc.Arguments)
				if err != nil || wantErr != nil || !contracts.EqualJSON(got, want) {
					t.Errorf("boundary argument changed: got=%s want=%s", got, want)
				}
				intelReply(t, w, req.ID, map[string]interface{}{"structuredContent": map[string]bool{"accepted": true}})
			})
			home := intelContractHome(t, endpoint, "5s")
			raw, err := json.Marshal(tc.Arguments)
			if err != nil {
				t.Fatal(err)
			}
			args := append([]string{"--format", "json"}, strings.Fields(tc.Path)...)
			out, stderr, err := intelCLI(t, home, nil, append(args, "--args-json", string(raw))...)
			if tc.Invalid {
				if !intelFailedExit(err) || len(out) != 0 || calls.RPC.Load() != 0 || !json.Valid(stderr) {
					t.Fatalf("invalid boundary reached remote: %v RPC=%d stdout=%s stderr=%s", err, calls.RPC.Load(), out, stderr)
				}
			} else if err != nil || len(stderr) != 0 || calls.Load() != 1 || !contracts.EqualJSON(out, []byte(`{"accepted":true}`)) {
				t.Fatalf("valid boundary rejected: %v stdout=%s stderr=%s", err, out, stderr)
			}
		})
	}
}

func TestIntelInvalidJSONFallback(t *testing.T) {
	for _, domain := range []string{"info", "news"} {
		for _, raw := range []string{"null", "[]", `"scalar"`, `{"query":`} {
			t.Run(domain+"/"+raw, func(t *testing.T) {
				tool, path := "info_coin_get_coin_info", "info coin get-coin-info"
				if domain == "news" {
					tool, path = "news_feed_search_news", "news feed search-news"
				}
				endpoint, calls := intelPeer(t, tool, func(http.ResponseWriter, *http.Request, intelRPCRequest) {
					t.Error("invalid JSON reached remote tool")
				})
				home := intelContractHome(t, endpoint, "5s")
				for _, flags := range [][]string{{"--args-json", raw}, {"--params", `{"query":"BTC"}`, "--args-json", raw}} {
					args := append([]string{"--format", "json"}, strings.Fields(path)...)
					out, stderr, err := intelCLI(t, home, nil, append(args, flags...)...)
					if !intelFailedExit(err) || len(out) != 0 || calls.RPC.Load() != 0 || !json.Valid(stderr) {
						t.Fatalf("invalid/mutually-exclusive fallback accepted: %v RPC=%d stdout=%s stderr=%s", err, calls.RPC.Load(), out, stderr)
					}
				}
			})
		}
	}
}

// Explicit JSON keys belong to the business payload even when they have a
// CLI-global name. Flat flags override JSON; all three input forms agree.
func TestIntelJSONInputPrecedence(t *testing.T) {
	for _, tc := range []struct{ domain, tool, path, key string }{
		{"info", "info_coin_get_coin_info", "info coin get-coin-info", "query"},
		{"news", "news_feed_search_news", "news feed search-news", "query"},
	} {
		for _, entry := range []string{"leaf", "invoke", "call"} {
			for _, source := range []string{"params", "args-json", "args-file"} {
				t.Run(tc.domain+"/"+entry+"/"+source, func(t *testing.T) {
					endpoint, calls := intelPeer(t, tc.tool, func(w http.ResponseWriter, _ *http.Request, req intelRPCRequest) {
						for key, want := range map[string]interface{}{tc.key: "ETH", "profile": "business-profile", "format": "business-format", "enabled": false} {
							if req.Params.Arguments[key] != want {
								t.Errorf("explicit JSON/flat precedence lost %s: %#v", key, req.Params.Arguments)
							}
						}
						for _, key := range []string{"api_key", "api_secret", "name", "args_json", "args_file", "params"} {
							if _, leaked := req.Params.Arguments[key]; leaked {
								t.Errorf("CLI-only key forwarded: %s", key)
							}
						}
						intelReply(t, w, req.ID, map[string]interface{}{"structuredContent": map[string]bool{"accepted": true}})
					})
					home := intelContractHome(t, endpoint, "5s")
					args := []string{"--format", "json", "--profile", "default", "--api-key", "synthetic-global-key", "--api-secret", "synthetic-global-secret"}
					if entry == "leaf" {
						args = append(args, strings.Fields(tc.path)...)
					} else {
						args = append(args, tc.domain, entry, "--name", tc.path)
					}
					value := `{"query":"BTC","profile":"business-profile","format":"business-format","enabled":false}`
					if source == "args-file" {
						path := filepath.Join(home, "args.json")
						if err := os.WriteFile(path, []byte(value), 0600); err != nil {
							t.Fatal(err)
						}
						value = path
					}
					args = append(args, "--"+source, value, "--query", "ETH")
					out, stderr, err := intelCLI(t, home, nil, args...)
					if err != nil || len(stderr) != 0 || calls.Load() != 1 || !contracts.EqualJSON(out, []byte(`{"accepted":true}`)) {
						t.Fatalf("input form failed: %v stdout=%s stderr=%s", err, out, stderr)
					}
				})
			}
		}
	}
}

// Faults happen at tools/call after normal initialization/discovery. The call
// is never replayed, failure stays on stderr, and trace IDs survive HTTP errors.
func TestIntelTransportFailures(t *testing.T) {
	for _, domain := range []string{"info", "news"} {
		for _, fault := range []string{"401", "403", "429", "503", "malformed", "wrong-id", "array-result", "oversize", "disconnect", "timeout"} {
			t.Run(domain+"/"+fault, func(t *testing.T) {
				tool, path := "info_coin_get_coin_info", "info coin get-coin-info"
				if domain == "news" {
					tool, path = "news_feed_search_news", "news feed search-news"
				}
				endpoint, calls := intelPeer(t, tool, func(w http.ResponseWriter, r *http.Request, req intelRPCRequest) {
					switch fault {
					case "401", "403", "429", "503":
						var status int
						if _, err := fmt.Sscan(fault, &status); err != nil {
							t.Error(err)
						}
						w.WriteHeader(status)
						_, _ = fmt.Fprint(w, `{"error":"fixture transport rejected"}`)
					case "malformed":
						_, _ = fmt.Fprint(w, `{"result":`)
					case "wrong-id":
						intelReply(t, w, json.RawMessage(`"wrong-id"`), map[string]interface{}{})
					case "array-result":
						intelReply(t, w, req.ID, []string{"invalid-result"})
					case "oversize":
						intelReply(t, w, req.ID, map[string]interface{}{"content": []interface{}{map[string]string{"type": "text", "text": strings.Repeat("x", 4096)}}})
					case "disconnect":
						hijacker, ok := w.(http.Hijacker)
						if !ok {
							t.Error("HTTP fixture cannot hijack its connection")
							return
						}
						conn, _, err := hijacker.Hijack()
						if err != nil {
							t.Error(err)
							return
						}
						if err := conn.Close(); err != nil {
							t.Error(err)
						}
					case "timeout":
						select {
						case <-r.Context().Done():
						case <-time.After(3 * time.Second):
							t.Error("client timeout did not cancel peer request")
						}
					}
				})
				timeout := "5s"
				if fault == "timeout" {
					timeout = "500ms"
				}
				home := intelContractHome(t, endpoint, timeout)
				args := append([]string{"--format", "json"}, strings.Fields(path)...)
				args = append(args, "--query", "BTC")
				env := []string(nil)
				if fault == "oversize" {
					env = []string{"GATE_INTEL_MAX_RESPONSE_BYTES=2048"}
				}
				out, stderr, err := intelCLI(t, home, env, args...)
				if !intelFailedExit(err) || len(out) != 0 || calls.Load() != 1 || !json.Valid(stderr) {
					t.Fatalf("transport failure accepted/replayed: %v calls=%d stdout=%s stderr=%s", err, calls.Load(), out, stderr)
				}
				for _, secret := range []string{"synthetic-intel-token", "synthetic-global-secret"} {
					if bytes.Contains(stderr, []byte(secret)) {
						t.Fatal("transport error exposed credentials")
					}
				}
				if fault == "401" || fault == "403" || fault == "429" || fault == "503" {
					var failure struct {
						Error struct {
							Status  int    `json:"status"`
							TraceID string `json:"trace_id"`
						} `json:"error"`
					}
					if err := json.Unmarshal(stderr, &failure); err != nil || fmt.Sprint(failure.Error.Status) != fault || failure.Error.TraceID != "intel-contract-trace" {
						t.Fatalf("HTTP error diagnostic lost: %s", stderr)
					}
				}
			})
		}
	}
}
