package earn

import (
	"github.com/spf13/cobra"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

// Both published commands must expose the optional fields already present in SDK v167.
func TestFixedFilterFlags(t *testing.T) {
	for _, name := range []string{"lends", "history"} {
		t.Run(name, func(t *testing.T) {
			cmd, _, err := fixedCmd.Find([]string{name})
			if err != nil {
				t.Fatal(err)
			}
			for flag, kind := range map[string]string{"sub-business": "int32", "business-filter": "string"} {
				value := cmd.Flags().Lookup(flag)
				if value == nil {
					t.Errorf("missing --%s", flag)
					continue
				}
				if value.Value.Type() != kind {
					t.Errorf("--%s type=%s want=%s", flag, value.Value.Type(), kind)
				}
				if len(value.Annotations[cobra.BashCompOneRequiredFlag]) != 0 {
					t.Errorf("optional --%s marked required", flag)
				}
			}
		})
	}
}

// Full SDK request assertions distinguish omission from explicit zero/empty-array input.
func TestFixedFilterSDKWire(t *testing.T) {
	for _, kind := range []string{"lends", "history"} {
		for _, tc := range []struct {
			name        string
			args        []string
			sub, filter string
			bad         bool
		}{
			{"omitted", nil, "", "", false},
			{"zero", []string{"--sub-business", "0"}, "0", "", false},
			{"nonzero", []string{"--sub-business", "3"}, "3", "", false},
			{"int32-max", []string{"--sub-business", "2147483647"}, "2147483647", "", false},
			{"int32-min", []string{"--sub-business", "-2147483648"}, "-2147483648", "", false},
			{"json", []string{"--business-filter", `[{"business":1,"sub_business":0}]`}, "", `[{"business":1,"sub_business":0}]`, false},
			{"combined", []string{"--sub-business", "0", "--business-filter", `[ {"business":2, "sub_business":0} ]`}, "0", `[ {"business":2, "sub_business":0} ]`, false},
			{"empty-array", []string{"--business-filter", "[]"}, "", "[]", false},
			{"boundary", []string{"--business-filter", "[" + strings.Repeat(" ", 65534) + "]"}, "", "[" + strings.Repeat(" ", 65534) + "]", false},
			{"oversized", []string{"--business-filter", "[" + strings.Repeat(" ", 65535) + "]"}, "", "", true},
			{"empty", []string{"--business-filter", ""}, "", "", true},
			{"malformed", []string{"--business-filter", "["}, "", "", true},
			{"object", []string{"--business-filter", "{}"}, "", "", true},
			{"null", []string{"--business-filter", "null"}, "", "", true},
			{"number", []string{"--business-filter", "1"}, "", "", true},
			{"scalar-item", []string{"--business-filter", "[1]"}, "", "", true},
			{"null-item", []string{"--business-filter", "[null]"}, "", "", true},
		} {
			t.Run(kind+"/"+tc.name, func(t *testing.T) {
				var calls atomic.Int32
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					path := "/api/v4/earn/fixed-term/user/lend"
					if kind == "history" {
						path = "/api/v4/earn/fixed-term/user/history"
					}
					if r.Method != "GET" || r.URL.Path != path {
						t.Errorf("request=%s %s", r.Method, r.URL.Path)
					}
					q := r.URL.Query()
					if q.Get("sub_business") != tc.sub || q.Has("sub_business") != (tc.sub != "") {
						t.Errorf("sub_business=%q presence=%v", q.Get("sub_business"), q.Has("sub_business"))
					}
					if q.Get("business_filter") != tc.filter || q.Has("business_filter") != (tc.filter != "") {
						t.Errorf("business_filter was omitted or altered")
					}
					if q.Get("asset") != "USDT" || q.Get("product_id") != "42" || q.Get("order_id") != "9007199254740993" || q.Get("page") != "2" || q.Get("limit") != "3" {
						t.Error("existing filters changed")
					}
					w.Header().Set("Content-Type", "application/json")
					if _, err := io.WriteString(w, `{"code":0,"data":{"list":[],"total":0}}`); err != nil {
						t.Error(err)
					}
				}))
				t.Cleanup(server.Close)
				root := newTestRoot(t)
				t.Setenv("GATE_BASE_URL", server.URL)
				t.Setenv("GATE_API_KEY", "synthetic-key")
				t.Setenv("GATE_API_SECRET", "synthetic-secret")
				cmd := &cobra.Command{Use: kind}
				cmd.Flags().Int32("page", 2, "")
				cmd.Flags().Int32("limit", 3, "")
				cmd.Flags().Int32("product-id", 42, "")
				cmd.Flags().String("asset", "USDT", "")
				cmd.Flags().Int32("sub-business", 0, "")
				cmd.Flags().String("business-filter", "", "")
				if kind == "lends" {
					cmd.Flags().String("order-type", "2", "")
					cmd.Flags().Int64("order-id", 9007199254740993, "")
				} else {
					cmd.Flags().String("type", "3", "")
					cmd.Flags().String("order-id", "9007199254740993", "")
					cmd.Flags().Int32("start-at", 1700000000, "")
					cmd.Flags().Int32("end-at", 1700000010, "")
				}
				root.AddCommand(cmd)
				if err := cmd.ParseFlags(tc.args); err != nil {
					t.Fatal(err)
				}
				silenceStdout(t)
				var err error
				if kind == "lends" {
					err = runFixedLends(cmd, nil)
				} else {
					err = runFixedHistory(cmd, nil)
				}
				if (err != nil) != tc.bad {
					t.Errorf("bad=%v error=%v", tc.bad, err)
				}
				want := int32(1)
				if tc.bad {
					want = 0
				}
				if calls.Load() != want {
					t.Errorf("requests=%d want=%d", calls.Load(), want)
				}
			})
		}
	}
}

// Expose the additions only on the two SDK read methods being completed.
func TestFixedFilterScope(t *testing.T) {
	for _, name := range []string{"products", "products-asset", "create", "pre-redeem"} {
		cmd, _, err := fixedCmd.Find([]string{name})
		if err != nil {
			t.Fatal(err)
		}
		for _, flag := range []string{"sub-business", "business-filter"} {
			if cmd.Flags().Lookup(flag) != nil {
				t.Errorf("unexpected --%s on %s", flag, name)
			}
		}
	}
}
