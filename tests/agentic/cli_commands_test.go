//go:build agentic_e2e

package agentic_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/gate/gate-cli/internal/agentfeature"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

var cliBinary string

// Explicit opt-in keeps release-artifact verification separate from source builds.
var publishedCLI = flag.String("published-cli-binary", "", "Absolute path to a trusted, checksum-verified standard CLI release binary")

// Reuse the exact toolchain selected for TestMain's CLI build. Test fixtures
// must not resolve a different Go from PATH (the OAuth issuer owns that PATH).
var cliBuildGo string

// TestMain builds the real CLI by default, or explicitly selects a verified release artifact.
func TestMain(m *testing.M) {
	flag.Parse()
	//nolint:staticcheck // Fixtures stay local and must use the outer test toolchain, even when OAuth tests replace PATH.
	cliBuildGo = filepath.Join(runtime.GOROOT(), "bin", "go")
	if *publishedCLI != "" {
		var err error
		cliBinary, err = publishedBinaryPath(*publishedCLI, agentfeature.Commands)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "published CLI under test: %s (source build disabled)\n", cliBinary)
		os.Exit(m.Run())
	}
	dir, err := os.MkdirTemp("", "gate-agentic-e2e-")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		fmt.Fprintln(os.Stderr, "cannot locate E2E source")
		os.Exit(1)
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(source), "../.."))
	cliBinary = filepath.Join(dir, "gate-cli")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	tags := "agentic_e2e"
	if agentfeature.Commands {
		tags += ",agent"
	}
	buildArgs := []string{"build", "-tags", tags, "-o", cliBinary}
	if os.Getenv("CLI_AGENTIC_COVER_DIR") != "" {
		buildArgs = append(buildArgs, "-cover", "-coverpkg=github.com/gate/gate-cli,github.com/gate/gate-cli/cmd/auth,github.com/gate/gate-cli/cmd/cex/stock,github.com/gate/gate-cli/internal/oauth2,github.com/gate/gate-cli/internal/config,github.com/gate/gate-cli/internal/configcmd,github.com/gate/gate-cli/internal/client,github.com/gate/gate-cli/internal/operations")
	}
	buildArgs = append(buildArgs, ".")
	//nolint:staticcheck // 夹具不分发到其他机器；必须沿用外层测试工具链，不能误用PATH中的其他Go版本。
	build := exec.CommandContext(ctx, filepath.Join(runtime.GOROOT(), "bin", "go"), buildArgs...)
	cliBuildGo = build.Path
	build.Dir = root
	out, err := build.CombinedOutput()
	cancel()
	if err != nil {
		fmt.Fprintf(os.Stderr, "build actual CLI: %v\n%s", err, out)
		if cleanupErr := os.RemoveAll(dir); cleanupErr != nil {
			fmt.Fprintln(os.Stderr, cleanupErr)
		}
		os.Exit(1)
	}
	code := m.Run()
	if err := os.RemoveAll(dir); err != nil {
		fmt.Fprintln(os.Stderr, err)
		code = 1
	}
	os.Exit(code)
}

type commandCase struct {
	name, method, path string
	args               []string
	public             bool
}

// stockCases 固定独立契约，不能从实现的运行时路由表生成预期值。
func stockCases() []commandCase {
	return []commandCase{
		{"exchanges", "GET", "/stock/exchanges", []string{"market", "exchanges"}, true},
		{"fee-rate", "GET", "/stock/fee-rate", []string{"market", "fee-rate"}, true},
		{"orderbook", "GET", "/stock/market/SYNTH/orderbook", []string{"market", "orderbook", "--symbol", "SYNTH"}, true},
		{"symbols", "GET", "/stock/symbols", []string{"market", "symbols"}, true},
		{"symbol", "GET", "/stock/symbols/detail", []string{"market", "symbol", "--symbols", "SYNTH"}, true},
		{"cancel-all", "DELETE", "/stock/orders", []string{"order", "cancel-all"}, false},
		{"orders", "GET", "/stock/orders", []string{"order", "list"}, false},
		{"create", "POST", "/stock/orders", []string{"order", "create", "--symbol", "SYNTH", "--side", "2", "--volume", "1", "--price-type", "market", "--trading-session", "regular", "--time-in-force", "day", "--client-order-id", "synthetic-order"}, false},
		{"history", "GET", "/stock/orders/history", []string{"order", "history"}, false},
		{"cancel", "DELETE", "/stock/orders/9007199254740993", []string{"order", "cancel", "--id", "9007199254740993"}, false},
		{"update", "PUT", "/stock/orders/2147483648", []string{"order", "update", "--id", "2147483648", "--volume", "1", "--price", "10.00000001"}, false},
		{"positions", "GET", "/stock/positions", []string{"position", "list"}, false},
		{"close", "POST", "/stock/positions/close", []string{"position", "close", "--symbol", "SYNTH", "--close-type", "2"}, false},
		{"transactions", "GET", "/stock/transactions", []string{"account", "transactions"}, false},
		{"transfer", "POST", "/stock/transactions", []string{"account", "transfer", "--change", "0.00000001", "--type", "deposit", "--ref-id", "synthetic-transfer"}, false},
		{"assets", "GET", "/stock/users/assets", []string{"account", "assets"}, false},
	}
}

// childEnv 清除宿主凭据，所有账户、配置和交易均为合成夹具。
func childEnv(home, base string, authenticated bool) []string {
	env := make([]string, 0, len(os.Environ())+5)
	for _, item := range os.Environ() {
		key, _, _ := strings.Cut(item, "=")
		if key == "HOME" || key == "USERPROFILE" || key == "XDG_CACHE_HOME" || key == "XDG_CONFIG_HOME" || key == "LOCALAPPDATA" || key == "APPDATA" || key == "CLI_AGENTIC_TEST_ISSUER" || strings.HasPrefix(key, "GATE_") {
			continue
		}
		env = append(env, item)
	}
	if dir := os.Getenv("CLI_AGENTIC_COVER_DIR"); dir != "" {
		env = append(env, "GOCOVERDIR="+dir)
	}
	// 测试夹具元数据位于产品配置之外，只传递给显式测试标签构建。
	if issuer, err := os.ReadFile(filepath.Join(home, "test-issuer")); err == nil {
		env = append(env, "CLI_AGENTIC_TEST_ISSUER="+string(issuer))
	}
	env = append(env, "USERPROFILE="+home, "XDG_CACHE_HOME="+filepath.Join(home, "cache"), "XDG_CONFIG_HOME="+filepath.Join(home, "config"), "LOCALAPPDATA="+home, "APPDATA="+home, "HOME="+home, "GATE_BASE_URL="+base, "GATE_CLI_SUPPRESS_FORMAT_NOTICE=1")
	if authenticated {
		env = append(env, "GATE_API_KEY=synthetic-key", "GATE_API_SECRET=synthetic-secret")
	}
	return env
}

func runCLI(t *testing.T, base string, authenticated bool, args ...string) ([]byte, []byte, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, cliBinary, args...)
	cmd.Env = childEnv(t.TempDir(), base, authenticated)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	return stdout.Bytes(), stderr.Bytes(), err
}

// TestStockCommandWire 验证16个真正命令经过公共SDK，包含大ID与十进制，不Mock SDK方法。
func TestStockCommandWire(t *testing.T) {
	for _, tc := range stockCases() {
		t.Run(tc.name, func(t *testing.T) {
			var mu sync.Mutex
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				calls++
				if r.Method != tc.method || r.URL.Path != "/api/v4"+tc.path {
					t.Errorf("got %s %s, want %s %s", r.Method, r.URL.Path, tc.method, tc.path)
				}
				// 检查真实SDK编码后的payload，不以命令退出0代替参数保真。
				var want map[string]string
				switch tc.name {
				case "create":
					want = map[string]string{"symbol": "SYNTH", "side": "2", "volume": "1", "price_type": "market", "trading_session": "regular", "time_in_force": "day", "client_order_id": "synthetic-order"}
				case "update":
					want = map[string]string{"volume": "1", "price": "10.00000001"}
				case "transfer":
					want = map[string]string{"asset": "USDT", "change": "0.00000001", "type": "deposit", "ref_id": "synthetic-transfer"}
				case "close":
					want = map[string]string{"symbol": "SYNTH", "close_type": "2"}
				}
				if want != nil {
					raw, err := io.ReadAll(io.LimitReader(r.Body, 8192))
					if err != nil {
						t.Error(err)
					}
					var body map[string]json.RawMessage
					if json.Unmarshal(raw, &body) != nil {
						t.Error("invalid wire body")
					}
					for key, value := range want {
						got := strings.Trim(string(body[key]), "\"")
						if got != value {
							t.Errorf("%s=%s want%s", key, got, value)
						}
					}
				}
				if r.Header.Get("Authorization") != "" {
					t.Error("unexpected OAuth credential on HMAC fixture")
				}
				w.Header().Set("Content-Type", "application/json")
				if _, err := w.Write([]byte(`{"code":0,"data":{"list":[],"total":0}}`)); err != nil {
					t.Error(err)
				}
			}))
			t.Cleanup(server.Close)
			args := append([]string{"--format", "json", "cex", "stock"}, tc.args...)
			stdout, stderr, err := runCLI(t, server.URL, true, args...)
			if err != nil {
				t.Fatalf("CLI failed: %v stderr=%s", err, stderr)
			}
			if !json.Valid(bytes.TrimSpace(stdout)) {
				t.Errorf("stdout is not JSON: %s", stdout)
			}
			mu.Lock()
			defer mu.Unlock()
			if calls != 1 {
				t.Errorf("SDK requests=%d, want1", calls)
			}
		})
	}
}

func TestStockPublicAnonymous(t *testing.T) {
	for _, tc := range stockCases() {
		if !tc.public {
			continue
		}
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("Authorization") != "" || r.Header.Get("KEY") != "" {
					t.Error("public call sent credential")
				}
				w.Header().Set("Content-Type", "application/json")
				if _, err := w.Write([]byte(`{"code":0,"data":{"list":[]}}`)); err != nil {
					t.Error(err)
				}
			}))
			t.Cleanup(server.Close)
			args := append([]string{"cex", "stock"}, tc.args...)
			_, stderr, err := runCLI(t, server.URL, false, args...)
			if err != nil {
				t.Fatalf("anonymous public command failed: %v %s", err, stderr)
			}
		})
	}
}

// TestStockErrorIsFailure 确保异常分支不能只打印错误却以0退出。
func TestStockErrorIsFailure(t *testing.T) {
	for _, status := range []int{401, 403, 429, 503} {
		for _, tc := range stockCases() {
			if tc.public {
				continue
			}
			t.Run(fmt.Sprintf("%s/%d", tc.name, status), func(t *testing.T) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(status)
					if _, err := w.Write([]byte(`{"label":"SYNTHETIC_FAILURE","message":"synthetic dependency failure"}`)); err != nil {
						t.Error(err)
					}
				}))
				t.Cleanup(server.Close)
				args := append([]string{"--format", "json", "cex", "stock"}, tc.args...)
				stdout, stderr, err := runCLI(t, server.URL, true, args...)
				if err == nil {
					t.Errorf("HTTP%d incorrectly exited0 stdout=%s stderr=%s", status, stdout, stderr)
				}
				if status == 401 || status == 403 {
					var processErr *exec.ExitError
					if !errors.As(err, &processErr) || processErr.ExitCode() != 4 {
						t.Errorf("auth exit=%v want4", err)
					}
				}
				if strings.Contains(string(stderr), "unknown command") {
					t.Fatalf("missing command is not a valid error-path test: %s", stderr)
				}
				if len(bytes.TrimSpace(stdout)) != 0 {
					t.Errorf("failure leaked success stdout: %s", stdout)
				}
			})
		}
	}
}

func TestStockInvalidParametersDoNotSend(t *testing.T) {
	for _, args := range [][]string{
		{"order", "cancel", "--id", "0"}, {"order", "cancel", "--id", "9223372036854775808"},
		{"order", "history", "--page-size", "501"}, {"order", "history", "--begin", "2147483648"},
		{"order", "create", "--symbol", "SYNTH", "--side", "2", "--volume", "1", "--price-type", "market", "--trading-session", "all", "--time-in-force", "day", "--client-order-id", "synthetic"},
		{"order", "create", "--symbol", "SYNTH", "--side", "2", "--volume", "1", "--price-type", "limit", "--trading-session", "all", "--time-in-force", "day", "--client-order-id", "synthetic"},
		{"position", "close", "--symbol", "SYNTH", "--close-type", "1"},
		{"position", "close", "--symbol", "SYNTH", "--close-type", "2", "--close-volume", "1"},
		{"account", "transfer", "--change", "1", "--type", "deposit"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				t.Error("invalid parameters reached dependency")
				w.WriteHeader(500)
			}))
			t.Cleanup(server.Close)
			command := append([]string{"cex", "stock"}, args...)
			_, stderr, err := runCLI(t, server.URL, true, command...)
			if err == nil || strings.Contains(string(stderr), "unknown command") {
				t.Fatalf("expected implemented validation failure, got %v %s", err, stderr)
			}
		})
	}
}
