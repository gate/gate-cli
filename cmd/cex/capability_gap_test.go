package cex_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/gate/gate-cli/cmd/cex"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/require"
)

type gapCase struct {
	Name     string            `json:"name"`
	Args     []string          `json:"args"`
	Method   string            `json:"method"`
	Path     string            `json:"path"`
	Query    map[string]string `json:"query"`
	Body     json.RawMessage   `json:"body"`
	Response json.RawMessage   `json:"response"`
	Private  bool              `json:"private"`
}

// A fresh root and reset leaf flags isolate Cobra's package-level command objects.
func gapRoot(t *testing.T, args []string) *cobra.Command {
	t.Helper()
	leaf, _, err := cex.Cmd.Find(args)
	require.NoError(t, err)
	leaf.Flags().VisitAll(func(f *pflag.Flag) {
		if v, ok := f.Value.(pflag.SliceValue); ok {
			require.NoError(t, v.Replace(nil))
		} else {
			require.NoError(t, f.Value.Set(f.DefValue))
		}
		f.Changed = false
	})
	root := &cobra.Command{Use: "gate-cli", SilenceErrors: true, SilenceUsage: true}
	root.PersistentFlags().String("format", "json", "")
	root.PersistentFlags().String("profile", "default", "")
	root.PersistentFlags().String("api-key", "", "")
	root.PersistentFlags().String("api-secret", "", "")
	root.AddCommand(cex.Cmd)
	root.SetArgs(append([]string{"cex"}, args...))
	return root
}
func TestCapabilityGapSDKWire(t *testing.T) {
	raw, err := os.ReadFile("../../testdata/cex-capability-gaps.json")
	require.NoError(t, err)
	var cases []gapCase
	require.NoError(t, json.Unmarshal(raw, &cases))
	require.Len(t, cases, 21)
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			for _, status := range []int{200, 403, 503} {
				t.Run(http.StatusText(status), func(t *testing.T) {
					t.Setenv("HOME", t.TempDir())
					t.Setenv("GATE_API_KEY", "fixture-key")
					t.Setenv("GATE_API_SECRET", "fixture-secret")
					var count atomic.Int32
					type request struct {
						method, path string
						query        map[string][]string
						body         []byte
						key          string
					}
					seen := make(chan request, 1)
					peer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						body, readErr := io.ReadAll(io.LimitReader(r.Body, 65537))
						if readErr != nil {
							http.Error(w, "read", 500)
							return
						}
						count.Add(1)
						select {
						case seen <- request{r.Method, r.URL.Path, r.URL.Query(), body, r.Header.Get("KEY")}:
						default:
						}
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(status)
						if status == 200 {
							_, _ = w.Write(tc.Response)
						} else {
							_, _ = w.Write([]byte(`{"label":"INSUFFICIENT_SCOPE","message":"fixture rejected"}`))
						}
					}))
					defer peer.Close()
					t.Setenv("GATE_BASE_URL", peer.URL)
					err := gapRoot(t, tc.Args).Execute()
					if status == 200 {
						require.NoError(t, err)
					} else {
						require.Error(t, err)
					}
					require.EqualValues(t, 1, count.Load(), "each invocation must send once, without replaying writes")
					r := <-seen
					require.Equal(t, tc.Method, r.method)
					require.Equal(t, tc.Path, r.path)
					if tc.Private {
						require.Equal(t, "fixture-key", r.key)
					} else {
						require.Empty(t, r.key)
					}
					require.Len(t, r.query, len(tc.Query))
					for k, v := range tc.Query {
						require.Equal(t, []string{v}, r.query[k], k)
					}
					if string(tc.Body) != "null" {
						require.JSONEq(t, string(tc.Body), string(r.body))
					} else {
						require.Empty(t, r.body)
					}
				})
			}
		})
	}
}
func TestCapabilityGapRejectsInvalidBeforeNetwork(t *testing.T) {
	// No credentials or reachable server are needed to reject malformed intent.
	t.Setenv("HOME", t.TempDir())
	t.Setenv("GATE_API_KEY", "")
	t.Setenv("GATE_API_SECRET", "")
	t.Setenv("GATE_BASE_URL", "http://127.0.0.1:1")
	for _, args := range []string{
		"spot pov create --pair BTC_USDT --side bad --amount 1 --participation-rate 10 --ttl 1h",
		"spot pov create --pair BTC_USDT --side buy --amount 0 --participation-rate 10 --ttl 1h",
		"spot pov create --pair BTC_USDT --side buy --amount 1 --participation-rate 1 --ttl 1h",
		"spot pov create --pair BTC_USDT --side buy --amount 1 --participation-rate 10 --ttl 8d",
		"spot pov list --status bad", "spot pov list --page 101", "spot pov list --limit 1001", "spot pov get --id ../x",
		"spot market rpi-orderbook --pair BTC_USDT --limit 0", "futures market rpi-orderbook --contract BTC_USDT --limit 51",
		"futures market contracts-all --offset -1", "futures market contracts-all --limit 1001", "futures market adl-risk-states --settle ../x",
		"futures chase get --id ../x", "futures chase list --finished", "futures chase list --from 2 --to 1",
		"futures chase list --sort-by 0", "futures chase list --reduce-only 3", "futures chase list --side 3", "futures chase list --page 0", "futures chase list --limit 101",
		"futures chase stop", "futures chase stop --id 1 --text t-one", "futures chase stop-all --pos-margin-mode bad",
		"unified config delta-neutral-set", "unified config leverage-set-all --leverage 0",
		"unified loan quick-repay --debt-currencies USDT,USDT --available-currencies BTC",
		"tradfi order log --log-id -1", "tradfi order log --log-id 9223372036854775808",
	} {
		t.Run(args, func(t *testing.T) {
			err := gapRoot(t, strings.Fields(args)).Execute()
			require.Error(t, err)
			require.NotContains(t, err.Error(), "connection refused")
			require.NotContains(t, err.Error(), "authentication required")
			require.NotContains(t, err.Error(), "authorization or API key required")
		})
	}
}

// HTTP 200 can contain partial failure; this must not look like successful execution.
func TestUnifiedLeveragePartialFailure(t *testing.T) {
	for _, payload := range []string{`[{"currency":"BTC","reason":"limit exceeded"}]`, `null`} {
		t.Run(payload, func(t *testing.T) {
			t.Setenv("HOME", t.TempDir())
			t.Setenv("GATE_API_KEY", "fixture-key")
			t.Setenv("GATE_API_SECRET", "fixture-secret")
			var sends atomic.Int32
			peer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				sends.Add(1)
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(payload))
			}))
			defer peer.Close()
			t.Setenv("GATE_BASE_URL", peer.URL)
			err := gapRoot(t, []string{"unified", "config", "leverage-set-all", "--leverage", "3"}).Execute()
			require.Error(t, err)
			if payload != "null" {
				require.Contains(t, err.Error(), "BTC")
				require.Contains(t, err.Error(), "limit exceeded")
				require.Contains(t, err.Error(), "already")
			} else {
				require.Contains(t, err.Error(), "missing")
			}
			require.EqualValues(t, 1, sends.Load())
		})
	}
}
