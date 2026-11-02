package tradfi

import (
	"github.com/spf13/cobra"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
)

// Assert actual SDK query values, including the registered default, against the API enum.
func TestTradfiKlineWireType(t *testing.T) {
	registered, _, err := Cmd.Find([]string{"market", "kline"})
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		input, want string
		invalid     bool
	}{
		{"<default>", "1m", false}, {"1m", "1m", false}, {"15m", "15m", false}, {"1h", "1h", false}, {"4h", "4h", false}, {"1d", "1d", false}, {"7d", "7d", false}, {"30d", "30d", false},
		{"M1", "1m", false}, {"M15", "15m", false}, {"H1", "1h", false}, {"H4", "4h", false}, {"D1", "1d", false},
		{"", "", true}, {"M5", "", true}, {"M30", "", true}, {"1M", "", true}, {"garbage", "", true},
	} {
		t.Run(tc.input, func(t *testing.T) {
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.Method != "GET" || r.URL.Path != "/api/v4/tradfi/symbols/XAUUSD/klines" || r.URL.Query().Get("kline_type") != tc.want || r.URL.Query().Get("limit") != "3" {
					t.Errorf("unexpected request %s %s", r.Method, r.URL)
				}
				w.Header().Set("Content-Type", "application/json")
				if _, err := io.WriteString(w, `{"code":0,"data":{"list":[]}}`); err != nil {
					t.Error(err)
				}
			}))
			t.Cleanup(server.Close)
			t.Setenv("HOME", t.TempDir())
			t.Setenv("GATE_BASE_URL", server.URL)
			t.Setenv("GATE_API_KEY", "")
			t.Setenv("GATE_API_SECRET", "")
			root := &cobra.Command{Use: "gate-cli"}
			root.PersistentFlags().String("format", "json", "")
			root.PersistentFlags().String("profile", "default", "")
			root.PersistentFlags().Bool("debug", false, "")
			root.PersistentFlags().Bool("verbose", false, "")
			root.PersistentFlags().String("api-key", "", "")
			root.PersistentFlags().String("api-secret", "", "")
			cmd := &cobra.Command{Use: "kline"}
			cmd.Flags().String("symbol", "XAUUSD", "")
			cmd.Flags().String("type", registered.Flags().Lookup("type").DefValue, "")
			cmd.Flags().Int32("limit", 3, "")
			root.AddCommand(cmd)
			if tc.input != "<default>" {
				if err := cmd.Flags().Set("type", tc.input); err != nil {
					t.Fatal(err)
				}
			}
			sink, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
			if err != nil {
				t.Fatal(err)
			}
			old := os.Stdout
			os.Stdout = sink
			t.Cleanup(func() {
				os.Stdout = old
				if err := sink.Close(); err != nil {
					t.Error(err)
				}
			})
			err = runTradfiKline(cmd, nil)
			if (err != nil) != tc.invalid {
				t.Errorf("invalid=%v error=%v", tc.invalid, err)
			}
			want := int32(1)
			if tc.invalid {
				want = 0
			}
			if calls.Load() != want {
				t.Errorf("calls=%d want=%d", calls.Load(), want)
			}
		})
	}
}
