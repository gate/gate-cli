package futures

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/gate/gate-cli/internal/output"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// Build a standalone handler with the same flags; no globals or real credentials
// are mutated outside t.Setenv's lifetime. The production SDK encoding is real.
func confirmedFuturesLeaf(t *testing.T) *cobra.Command {
	t.Helper()
	root := newTestRoot(t)
	cmd := &cobra.Command{Use: "fixture"}
	for k, v := range map[string]string{"contract": "SYNTH_USDT", "amount": "1", "activation-price": "0", "price-offset": "1%", "trigger-price": "90000", "price": "0", "settle": "usdt", "tif": "", "text": ""} {
		cmd.Flags().String(k, v, "")
	}
	cmd.Flags().Int64("size", -10, "")
	cmd.Flags().Int64("id", 9007199254740993, "")
	cmd.Flags().Int32("trigger-rule", 1, "")
	cmd.Flags().Int32("price-type", 0, "")
	cmd.Flags().Int32("limit", 2, "")
	cmd.Flags().Bool("finished", false, "")
	// Mirror Trail's optional parameters so handler tests exercise real SDK
	// encoding without mutating the process-wide registered command tree.
	cmd.Flags().Int64("start-at", 0, "")
	cmd.Flags().Int64("end-at", 0, "")
	cmd.Flags().Bool("is-gte", false, "")
	root.AddCommand(cmd)
	t.Setenv("GATE_API_KEY", "fixture-key")
	t.Setenv("GATE_API_SECRET", "fixture-secret")
	return cmd
}

func TestFuturesPriceTriggerTIF(t *testing.T) {
	for _, value := range []string{"omitted", "ioc", "gtc", "", "IOC", "invalid"} {
		t.Run(value, func(t *testing.T) {
			cmd := confirmedFuturesLeaf(t)
			if value != "omitted" {
				require.NoError(t, cmd.Flags().Set("tif", value))
			}
			valid := value == "omitted" || value == "ioc" || value == "gtc"
			var calls atomic.Int32
			peer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				raw, err := io.ReadAll(r.Body)
				if err != nil {
					t.Error(err)
				}
				expected := `{"initial":{"contract":"SYNTH_USDT","size":-10,"price":"0"`
				if value != "omitted" {
					expected += `,"tif":"` + value + `"`
				}
				expected += "},\"trigger\":{\"price\":\"90000\",\"rule\":1}}\n"
				if string(raw) != expected {
					t.Errorf("wire=%s want=%s", raw, expected)
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(400)
				_, _ = io.WriteString(w, `{"label":"FIXTURE_REJECTED","message":"fixture"}`)
			}))
			t.Cleanup(peer.Close)
			t.Setenv("GATE_BASE_URL", peer.URL)
			err := runFuturesPriceTriggerCreate(cmd, nil)
			require.Error(t, err)
			var detail *output.GateError
			require.True(t, errors.As(err, &detail))
			if valid {
				require.EqualValues(t, 1, calls.Load())
				require.Equal(t, "FIXTURE_REJECTED", detail.Label)
			} else {
				require.Zero(t, calls.Load())
				require.Equal(t, "INVALID_ARGUMENTS", detail.Label)
			}
		})
	}
}

func TestFuturesPriceTriggerTIFRegistration(t *testing.T) {
	for _, c := range priceTriggerCmd.Commands() {
		if c.Name() != "create" {
			require.Nil(t, c.Flags().Lookup("tif"))
			continue
		}
		f := c.Flags().Lookup("tif")
		require.NotNil(t, f)
		require.Equal(t, "", f.DefValue)
		require.Empty(t, f.Annotations[cobraBashCompOneRequiredFlag])
		require.Contains(t, strings.ToLower(f.Usage), "ioc")
	}
}

func TestFuturesTrailRequestInfo(t *testing.T) {
	for _, tc := range []struct {
		name, method, leaf string
		run                func(*cobra.Command, []string) error
	}{
		{"create", "POST", "create", runFuturesTrailCreate}, {"stop", "POST", "stop", runFuturesTrailStop},
		{"stop-all", "POST", "stop_all", runFuturesTrailStopAll}, {"list", "GET", "list", runFuturesTrailList},
		{"get", "GET", "detail", runFuturesTrailGet}, {"update", "POST", "update", runFuturesTrailUpdate}, {"log", "GET", "change_log", runFuturesTrailLog},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cmd := confirmedFuturesLeaf(t)
			if tc.name == "create" {
				require.NoError(t, cmd.Flags().Set("price-type", "1"))
			}
			path := "/api/v4/futures/usdt/autoorder/v1/trail/" + tc.leaf
			peer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != tc.method || r.URL.Path != path {
					t.Errorf("wire=%s %s", r.Method, r.URL.Path)
				}
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("X-Gate-Trace-Id", "fixture-trace")
				w.WriteHeader(400)
				_, _ = io.WriteString(w, `{"label":"FIXTURE_REJECTED","message":"original"}`)
			}))
			t.Cleanup(peer.Close)
			t.Setenv("GATE_BASE_URL", peer.URL)
			err := tc.run(cmd, nil)
			require.Error(t, err)
			var detail *output.GateError
			require.True(t, errors.As(err, &detail))
			require.Equal(t, tc.method, detail.Request.Method)
			require.Equal(t, path, detail.Request.URL)
			require.Equal(t, "original", detail.Message)
			require.Equal(t, "fixture-trace", detail.TraceID)
		})
	}
}
