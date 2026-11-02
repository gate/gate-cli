package earn

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/spf13/cobra"
)

// The command surface must publish the SDK's optional query types without
// turning examples into defaults or making optional filters required.
func TestFixedProductStatusFlags(t *testing.T) {
	cmd, _, err := fixedCmd.Find([]string{"products"})
	if err != nil {
		t.Fatal(err)
	}
	for name, kind := range map[string]string{"status": "int32", "show-status": "string"} {
		f := cmd.Flags().Lookup(name)
		if f == nil {
			t.Fatalf("missing --%s", name)
		}
		if f.Value.Type() != kind || len(f.Annotations[cobra.BashCompOneRequiredFlag]) != 0 {
			t.Fatalf("incorrect optional --%s: %+v", name, f)
		}
	}
}

// Exercise the actual handler and released SDK. The HTTP response is mocked;
// query construction and optional-value presence are deliberately not mocked.
func TestFixedProductStatusSDKWire(t *testing.T) {
	for _, tc := range []struct {
		name               string
		flags              []string
		status, show       string
		hasStatus, hasShow bool
	}{
		{name: "omitted"},
		{"status", []string{"--status", "2"}, "2", "", true, false},
		{"show", []string{"--show-status", "2"}, "", "2", false, true},
		{"both", []string{"--status", "2", "--show-status", "2"}, "2", "2", true, true},
		{"zero-empty", []string{"--status", "0", "--show-status", ""}, "0", "", true, true},
		{"max", []string{"--status", "2147483647"}, "2147483647", "", true, false},
		{"min", []string{"--status", "-2147483648"}, "-2147483648", "", true, false},
		{"string", []string{"--show-status", "visible & scheduled"}, "", "visible & scheduled", false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var hits atomic.Int32
			peer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				hits.Add(1)
				q := r.URL.Query()
				if r.Method != "GET" || r.URL.Path != "/api/v4/earn/fixed-term/product" {
					t.Errorf("route=%s %s", r.Method, r.URL.Path)
				}
				if q.Has("status") != tc.hasStatus || q.Get("status") != tc.status || q.Has("show_status") != tc.hasShow || q.Get("show_status") != tc.show {
					t.Errorf("lost optional values: %s", r.URL.RawQuery)
				}
				if q.Get("page") != "2" || q.Get("limit") != "3" || q.Get("asset") != "USDT" || q.Get("type") != "1" {
					t.Errorf("old filters changed: %s", r.URL.RawQuery)
				}
				w.Header().Set("Content-Type", "application/json")
				if _, err := io.WriteString(w, `{"code":0,"data":{"list":[],"total":0}}`); err != nil {
					t.Error(err)
				}
			}))
			defer peer.Close()
			t.Setenv("GATE_BASE_URL", peer.URL)
			root := newTestRoot(t)
			cmd := &cobra.Command{Use: "products"}
			root.AddCommand(cmd)
			cmd.Flags().Int32("page", 2, "")
			cmd.Flags().Int32("limit", 3, "")
			cmd.Flags().String("asset", "USDT", "")
			cmd.Flags().Int32("type", 0, "")
			cmd.Flags().Int32("status", 0, "")
			cmd.Flags().String("show-status", "", "")
			if err := cmd.ParseFlags(append([]string{"--type", "1"}, tc.flags...)); err != nil {
				t.Fatal(err)
			}
			silenceStdout(t)
			if err := runFixedProducts(cmd, nil); err != nil {
				t.Fatal(err)
			}
			if hits.Load() != 1 {
				t.Fatalf("requests=%d", hits.Load())
			}
		})
	}
}

// Cobra rejects non-int32 status values before the handler/client can run.
func TestFixedProductStatusInvalid(t *testing.T) {
	for _, value := range []string{"2147483648", "-2147483649", "1.5", "null", "two"} {
		t.Run(value, func(t *testing.T) {
			cmd := &cobra.Command{Use: "products"}
			cmd.Flags().Int32("status", 0, "")
			if err := cmd.ParseFlags([]string{"--status", value}); err == nil || !strings.Contains(err.Error(), "status") {
				t.Fatalf("bad status accepted: %v", err)
			}
		})
	}
}
