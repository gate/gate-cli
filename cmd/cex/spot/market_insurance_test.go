package spot

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"

	"github.com/spf13/cobra"
)

// The registered command must enforce the API's required time range before RunE.
func TestSpotInsuranceRequiredTimeFlags(t *testing.T) {
	cmd, _, err := Cmd.Find([]string{"market", "insurance"})
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"currency", "from", "to"} {
		flag := cmd.Flags().Lookup(name)
		if flag == nil {
			t.Fatalf("missing flag %s", name)
		}
		values := flag.Annotations[cobra.BashCompOneRequiredFlag]
		if len(values) != 1 || values[0] != "true" {
			t.Errorf("--%s must be required", name)
		}
	}
}

// Exercise the actual handler: invalid ranges must fail before client/network work.
func TestSpotInsuranceTimeRange(t *testing.T) {
	for _, tc := range []struct {
		name, from, to string
		valid          bool
	}{
		{"missing", "0", "0", false}, {"zero-from", "0", "1700000001", false},
		{"zero-to", "1700000000", "0", false}, {"negative-from", "-1", "1700000001", false},
		{"negative-to", "1700000000", "-1", false}, {"reversed", "1700000002", "1700000001", false},
		{"equal", "1700000000", "1700000000", true}, {"ordered", "1700000000", "1700000001", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.Method != "GET" || r.URL.Path != "/api/v4/spot/insurance_history" {
					t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
				}
				if r.URL.Query().Get("from") != tc.from || r.URL.Query().Get("to") != tc.to {
					t.Errorf("time range was altered: %s", r.URL.RawQuery)
				}
				w.Header().Set("Content-Type", "application/json")
				if _, err := io.WriteString(w, `[{"currency":"BTC","balance":"1021.21","time":1700000000}]`); err != nil {
					t.Error(err)
				}
			}))
			t.Cleanup(server.Close)
			root := newSpotTestRoot(t)
			t.Setenv("GATE_BASE_URL", server.URL)
			cmd := &cobra.Command{Use: "insurance"}
			cmd.Flags().String("business", "margin", "")
			cmd.Flags().String("currency", "BTC", "")
			cmd.Flags().Int64("from", 0, "")
			cmd.Flags().Int64("to", 0, "")
			cmd.Flags().Int32("limit", 3, "")
			root.AddCommand(cmd)
			if err := cmd.ParseFlags([]string{"--from", tc.from, "--to", tc.to}); err != nil {
				t.Fatal(err)
			}
			// This package's printer targets stdout; avoid printing table data in unit logs.
			old := os.Stdout
			sink, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
			if err != nil {
				t.Fatal(err)
			}
			os.Stdout = sink
			t.Cleanup(func() {
				os.Stdout = old
				if err := sink.Close(); err != nil {
					t.Error(err)
				}
			})
			err = runSpotInsurance(cmd, nil)
			if (err == nil) != tc.valid {
				t.Errorf("valid=%v error=%v", tc.valid, err)
			}
			want := int32(0)
			if tc.valid {
				want = 1
			}
			if calls.Load() != want {
				t.Errorf("requests=%d want=%d", calls.Load(), want)
			}
		})
	}
}
