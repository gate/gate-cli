//go:build agentic_e2e

package agentic_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/gate/gate-cli/internal/agentfeature"
)

// These tests mock only Gate's HTTP response; the child CLI, SDK and error exit
// path are real. A partial write must retain facts and must never be replayed.
func TestCEXCompatFuturesBatch(t *testing.T) {
	for _, operation := range []string{"create", "amend", "cancel"} {
		for _, result := range []string{`[{"id":9007199254740993,"succeeded":true}]`, `[{"succeeded":true,"id":9007199254740993},{"succeeded":false,"label":"BAD_REQUEST","message":"synthetic failure"}]`, `[{"succeeded":false}]`, `[{}]`, `[]`, `null`} {
			t.Run(operation+result, func(t *testing.T) {
				if operation == "cancel" {
					result = strings.ReplaceAll(result, `"id":9007199254740993`, `"id":"9007199254740993"`)
				}
				var calls atomic.Int32
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					if r.Method != "POST" || !strings.Contains(r.URL.Path, "batch_") {
						t.Errorf("wire %s %s", r.Method, r.URL.Path)
					}
					w.Header().Set("Content-Type", "application/json")
					w.Header().Set("X-Gate-Trace-ID", "synthetic-trace")
					_, _ = io.WriteString(w, result)
				}))
				defer server.Close()
				args := []string{"cex", "futures", "order", "batch-" + operation, "--format", "json", "--max-output-bytes", "0"}
				if operation == "cancel" {
					args = append(args, "--ids", "9007199254740993")
				} else {
					args = append(args, "--orders-json", `[{"contract":"BTC_USDT","size":"1","id":9007199254740993}]`)
				}
				out, stderr, err := runCLI(t, server.URL, true, args...)
				failed := strings.Contains(result, `false`) || result == `[{}]`
				if (err != nil) != failed {
					t.Fatalf("failed=%v exit=%v stdout=%s stderr=%s", failed, err, out, stderr)
				}
				if !json.Valid(out) || calls.Load() != 1 {
					t.Fatalf("invalid output/replay: %s calls=%d", out, calls.Load())
				}
				if strings.Contains(result, "9007199254740993") && !bytes.Contains(out, []byte("9007199254740993")) {
					t.Fatal("lost successful ID")
				}
				if failed && (!bytes.Contains(stderr, []byte("BATCH_ORDER_FAILED")) || !bytes.Contains(stderr, []byte(`"retryable":false`)) || !bytes.Contains(stderr, []byte("synthetic-trace"))) {
					t.Fatalf("missing safe error %s", stderr)
				}
			})
		}
	}
}

func TestCEXCompatDetailsAndConfig(t *testing.T) {
	out, stderr, err := runCLI(t, "http://127.0.0.1:1", false, "version")
	if err != nil || len(out) != 0 || !bytes.Contains(stderr, []byte("gate-cli version")) {
		t.Fatalf("legacy version %v %s %s", err, out, stderr)
	}
	out, stderr, err = runCLI(t, "http://127.0.0.1:1", false, "version", "--details", "--format", "json")
	var details map[string]any
	if err != nil || json.Unmarshal(out, &details) != nil || details["agent_commands"] != agentfeature.Commands || details["schema_version"] != float64(1) || details["go"] == "" {
		t.Fatalf("details %v %s %s", err, out, stderr)
	}
	for _, args := range [][]string{{"version", "--details", "--format", "table"}, {"config", "check", "--format", "json"}} {
		out, stderr, err = runCLI(t, "http://127.0.0.1:1", false, args...)
		if err == nil || !bytes.Contains(stderr, []byte("UNSUPPORTED_FORMAT")) && !bytes.Contains(stderr, []byte("CONFIG_CHECK_FAILED")) {
			t.Fatalf("expected new-interface error %v %s %s", err, out, stderr)
		}
	}
	for _, mode := range []os.FileMode{0600, 0644, 0000} {
		t.Run(fmt.Sprint(mode), func(t *testing.T) {
			home := t.TempDir()
			path := filepath.Join(home, ".gate-cli", "config.yaml")
			if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte("broken: [synthetic-secret"), mode); err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command(cliBinary, "config", "check", "--format", "json")
			cmd.Env = childEnv(home, "http://127.0.0.1:1", false)
			var out, stderr bytes.Buffer
			cmd.Stdout = &out
			cmd.Stderr = &stderr
			err := cmd.Run()
			if (err == nil) != (mode == 0600 && runtime.GOOS != "windows") || !json.Valid(out.Bytes()) || bytes.Contains(out.Bytes(), []byte("synthetic-secret")) {
				t.Fatalf("mode=%o err=%v %s %s", mode, err, out.Bytes(), stderr.Bytes())
			}
			var report struct {
				Status string `json:"status"`
				Scope  string `json:"scope"`
			}
			if json.Unmarshal(out.Bytes(), &report) != nil || report.Scope != "file" {
				t.Fatalf("wrong report %s", out.Bytes())
			}
			want := "pass"
			if runtime.GOOS == "windows" || mode == 0644 {
				want = "warn"
			} else if mode == 0 {
				want = "fail"
			}
			if report.Status != want {
				t.Fatalf("mode=%o status=%s want=%s", mode, report.Status, want)
			}
			raw, e := os.ReadFile(path)
			if e != nil && mode != 0 {
				t.Fatal(e)
			}
			if mode != 0 && string(raw) != "broken: [synthetic-secret" {
				t.Fatal("diagnostic mutated file")
			}
		})
	}
}

func TestCEXCompatOrderText(t *testing.T) {
	for _, args := range [][]string{
		{"spot", "order", "buy", "--pair", "BTC_USDT", "--amount", "1", "--price", "10"},
		{"spot", "order", "sell", "--pair", "BTC_USDT", "--amount", "1"},
		{"futures", "order", "long", "--contract", "BTC_USDT", "--size", "1"},
		{"futures", "order", "short", "--contract", "BTC_USDT", "--size", "1"},
		{"futures", "order", "add", "--contract", "BTC_USDT", "--size", "1"},
		{"futures", "order", "remove", "--contract", "BTC_USDT", "--size", "1"},
	} {
		t.Run(strings.Join(args[:3], "/"), func(t *testing.T) {
			for _, text := range []string{"", "t-synthetic.1", "invalid/secret"} {
				var reads, writes atomic.Int32
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					if r.Method == "GET" {
						reads.Add(1)
						_, _ = io.WriteString(w, `{"contract":"BTC_USDT","size":"5","mode":"single"}`)
						return
					}
					writes.Add(1)
					var body map[string]any
					if json.NewDecoder(r.Body).Decode(&body) != nil {
						t.Error("bad body")
					}
					if text == "" {
						if _, ok := body["text"]; ok {
							t.Error("old wire text appeared")
						}
					} else if body["text"] != text {
						t.Error("text lost")
					}
					if args[0] == "futures" {
						size := "1"
						if args[2] == "short" || args[2] == "remove" {
							size = "-1"
						}
						if body["size"] != size {
							t.Errorf("size=%v", body["size"])
						}
						if args[2] == "remove" && body["reduce_only"] != true {
							t.Error("reduce_only lost")
						}
					}
					if args[0] == "spot" {
						_, _ = io.WriteString(w, `{"id":"9007199254740993"}`)
					} else {
						_, _ = io.WriteString(w, `{"id":9007199254740993}`)
					}
				}))
				argv := append([]string{"cex"}, args...)
				if text != "" {
					argv = append(argv, "--text", text)
				}
				_, stderr, err := runCLI(t, server.URL, true, argv...)
				server.Close()
				invalid := strings.HasPrefix(text, "invalid")
				if (err != nil) != invalid {
					t.Fatalf("text=%s err=%v %s", text, err, stderr)
				}
				if invalid {
					if writes.Load()+reads.Load() != 0 {
						t.Fatal("invalid text reached API")
					}
				} else if writes.Load() != 1 {
					t.Fatal("write missing or replayed")
				}
			}
		})
	}
	for _, market := range []string{"spot", "futures"} {
		var calls atomic.Int32
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			if r.Method != "GET" || !strings.HasSuffix(r.URL.Path, "/t-synthetic") {
				t.Errorf("unsafe lookup %s %s", r.Method, r.URL.Path)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{}`)
		}))
		args := []string{"cex", market, "order"}
		if market == "spot" {
			args = append(args, "get", "--id", "t-synthetic", "--pair", "BTC_USDT")
		} else {
			args = append(args, "get-by-text", "--text", "t-synthetic")
		}
		_, stderr, err := runCLI(t, server.URL, true, args...)
		server.Close()
		if err != nil || calls.Load() != 1 {
			t.Fatalf("lookup %v %s calls=%d", err, stderr, calls.Load())
		}
	}
}

func TestCEXCompatHiddenPipe(t *testing.T) {
	for _, hide := range []bool{false, true} {
		home := t.TempDir()
		args := []string{"config", "init"}
		if hide {
			args = append(args, "--hide-input")
		}
		cmd := exec.Command(cliBinary, args...)
		cmd.Env = childEnv(home, "http://127.0.0.1:1", false)
		cmd.Stdin = strings.NewReader("default\r\nsynthetic-key\r\nsynthetic-secret")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("pipe %v %s", err, out)
		}
		raw, err := os.ReadFile(filepath.Join(home, ".gate-cli", "config.yaml"))
		if err != nil || !bytes.Contains(raw, []byte("synthetic-secret")) {
			t.Fatalf("missing saved config %v", err)
		}
	}
}

func TestCEXCompatAgentDetails(t *testing.T) {
	out, stderr, err := runCLI(t, "http://127.0.0.1:1", false, "agent-index", "--domain", "cex", "--details", "--command", "cex spot order buy", "--format", "json", "--max-output-bytes", "0")
	if !agentfeature.Commands {
		if err == nil {
			t.Fatal("default build exposed agent")
		}
		return
	}
	var v struct {
		SchemaVersion int `json:"schema_version"`
		Commands      []struct {
			Path   string           `json:"path"`
			Effect string           `json:"effect"`
			Flags  []map[string]any `json:"flags"`
		} `json:"commands"`
	}
	if err != nil || json.Unmarshal(out, &v) != nil || len(v.Commands) != 1 || v.Commands[0].Effect != "write" {
		t.Fatalf("agent details %v %s %s", err, out, stderr)
	}
	for _, flag := range v.Commands[0].Flags {
		if _, ok := flag["default"]; ok {
			t.Fatal("default credential exposure")
		}
		if _, ok := flag["value"]; ok {
			t.Fatal("value exposure")
		}
	}
}

// New-interface failures must be explicit, while ordinary HTTP and malformed
// response handling still use the original client/error boundary without replay.
func TestCEXCompatNegativeContracts(t *testing.T) {
	for _, status := range []int{401, 403, 429, 503, 200} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(status)
				_, _ = io.WriteString(w, `{"label":"SYNTHETIC_FAILURE","message":"synthetic rejection"}`)
			}))
			defer server.Close()
			out, stderr, err := runCLI(t, server.URL, true, "cex", "futures", "order", "batch-create", "--orders-json", `[{"contract":"BTC_USDT","size":"1"}]`, "--format", "json")
			if err == nil || len(out) != 0 || calls.Load() != 1 || !bytes.Contains(stderr, []byte("SYNTHETIC_FAILURE")) && status != 200 {
				t.Fatalf("failed write/replay %v %s %s calls=%d", err, out, stderr, calls.Load())
			}
		})
	}
	for _, args := range [][]string{
		{"agent-index", "--details"},
		{"agent-index", "--domain", "info", "--details"},
		{"agent-index", "--domain", "cex", "--details", "--command", "cex spot"},
		{"agent-index", "--domain", "cex", "--details", "--command", "cex nonexistent"},
		{"agent-index", "--domain", "cex", "--command", "cex spot order buy"},
		{"agent-index", "--domain", "cex", "--details", "--format", "table"},
		{"config", "check", "--format", "table"},
		{"cex", "futures", "order", "get-by-text", "--text", ""},
		{"cex", "futures", "order", "get-by-text", "--text", "t-unsafe/path"},
		{"cex", "futures", "order", "get"},
	} {
		out, stderr, err := runCLI(t, "http://127.0.0.1:1", true, args...)
		if err == nil || len(out) != 0 || len(stderr) == 0 {
			t.Fatalf("invalid args succeeded %v %s %s", err, out, stderr)
		}
	}
	for _, status := range []int{404, 503} {
		var calls atomic.Int32
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			if r.Method != "GET" {
				t.Error("lookup submitted write")
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(status)
			_, _ = io.WriteString(w, `{"label":"SYNTHETIC_LOOKUP_FAILURE","message":"synthetic rejection"}`)
		}))
		out, stderr, err := runCLI(t, server.URL, true, "cex", "futures", "order", "get-by-text", "--text", "t-synthetic", "--format", "json")
		server.Close()
		if err == nil || len(out) != 0 || calls.Load() != 1 || !bytes.Contains(stderr, []byte("SYNTHETIC_LOOKUP_FAILURE")) {
			t.Fatalf("lookup replay %v %s %s", err, out, stderr)
		}
	}
}

func TestCEXCompatVersionFormats(t *testing.T) {
	for _, format := range []string{"json", "pretty"} {
		for _, cap := range []string{"0", "1"} {
			out, stderr, err := runCLI(t, "http://127.0.0.1:1", false, "version", "--details", "--format", format, "--max-output-bytes", cap)
			if err != nil || !json.Valid(out) {
				t.Fatalf("details %s cap%s %v %s %s", format, cap, err, out, stderr)
			}
			if cap == "0" && !bytes.Contains(out, []byte("schema_version")) {
				t.Fatal("details lost")
			}
			if cap == "1" && !bytes.Contains(out, []byte("truncated")) {
				t.Fatalf("cap changed %s", out)
			}
		}
	}
	out, stderr, err := runCLI(t, "http://127.0.0.1:1", false, "version", "--details=false", "--format", "json")
	if err != nil || len(out) != 0 || !bytes.Contains(stderr, []byte("gate-cli version")) {
		t.Fatal("explicit false changed legacy version")
	}
	out, stderr, err = runCLI(t, "http://127.0.0.1:1", false, "--version")
	if err != nil || !bytes.Contains(out, []byte("version")) || len(stderr) != 0 {
		t.Fatalf("root version stream changed %s %s", out, stderr)
	}
}
