package futures

import (
	"encoding/json"
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

// Requests must preserve Unix seconds and the legacy active-list query. Only
// the HTTP peer is synthetic; the handler, credentials and SDK are real.
func TestTrailHistoryParameters(t *testing.T) {
	for _, tc := range []struct {
		name, start, end, limit string
		finished                bool
		valid                   bool
	}{
		{"active", "", "", "0", false, true},
		{"active explicit false", "", "", "1", false, true},
		{"history", "1793534400", "1793538000", "1", true, true},
		{"minimum", "1", "2", "100", true, true},
		{"calendar upper bound", "253402300798", "253402300799", "0", true, true},
		{"missing both", "", "", "1", true, false},
		{"missing start", "", "1793538000", "1", true, false},
		{"missing end", "1793534400", "", "1", true, false},
		{"explicit zero", "0", "1793538000", "1", true, false},
		{"negative", "-1", "2", "1", true, false},
		{"equal", "2", "2", "1", true, false},
		{"reversed", "3", "2", "1", true, false},
		{"milliseconds", "1793534400000", "1793538000000", "1", true, false},
		{"unsupported calendar", "253402300799", "253402300800", "1", true, false},
		{"active ignores no filters", "1", "2", "1", false, false},
		{"negative limit", "", "", "-1", false, false},
		{"large limit", "1", "2", "101", true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cmd := confirmedFuturesLeaf(t)
			require.NoError(t, cmd.Flags().Set("finished", boolText(tc.finished)))
			require.NoError(t, cmd.Flags().Set("limit", tc.limit))
			for key, value := range map[string]string{"start-at": tc.start, "end-at": tc.end} {
				if value != "" {
					require.NoError(t, cmd.Flags().Set(key, value))
				}
			}
			var calls atomic.Int32
			peer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.Method != http.MethodGet || r.URL.Path != "/api/v4/futures/usdt/autoorder/v1/trail/list" || r.Header.Get("SIGN") == "" {
					t.Errorf("incorrect signed request: %s %s", r.Method, r.URL)
				}
				q := r.URL.Query()
				if q.Get("contract") != "SYNTH_USDT" || q.Get("start_at") != tc.start || q.Get("end_at") != tc.end {
					t.Errorf("filters/seconds changed: %s", r.URL.RawQuery)
				}
				wantLimit := tc.limit
				if wantLimit == "0" {
					wantLimit = ""
				}
				if q.Get("page_size") != wantLimit || (q.Get("is_finished") == "true") != tc.finished {
					t.Errorf("legacy selectors changed: %s", r.URL.RawQuery)
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, `{"code":0,"data":{"orders":[]}}`)
			}))
			t.Cleanup(peer.Close)
			t.Setenv("GATE_BASE_URL", peer.URL)
			silenceStdout(t)
			err := runFuturesTrailList(cmd, nil)
			if tc.valid {
				require.NoError(t, err)
				require.EqualValues(t, 1, calls.Load())
			} else {
				var detail *output.GateError
				require.ErrorAs(t, err, &detail)
				require.Equal(t, "INVALID_ARGUMENTS", detail.Label)
				require.Zero(t, calls.Load(), "invalid range must fail before HTTP")
			}
		})
	}
}

// False is meaningful on update; omission must not write a default direction.
func TestTrailDirectionParameters(t *testing.T) {
	for _, operation := range []string{"create", "update"} {
		for _, direction := range []string{"omitted", "true", "false"} {
			t.Run(operation+"/"+direction, func(t *testing.T) {
				cmd := confirmedFuturesLeaf(t)
				require.NoError(t, cmd.Flags().Set("price-type", "1"))
				if direction != "omitted" {
					require.NoError(t, cmd.Flags().Set("is-gte", direction))
				}
				var calls atomic.Int32
				peer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					var body map[string]json.RawMessage
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						t.Error(err)
					}
					field, expected := "is_gte", direction
					if operation == "update" {
						field = "is_gte_str"
						if direction != "omitted" {
							expected = `"` + direction + `"`
						}
					} else if direction == "false" {
						expected = "omitted" // SDK omitempty equals the existing false default.
					}
					got, present := body[field]
					if (expected == "omitted" && present) || (expected != "omitted" && string(got) != expected) {
						t.Errorf("%s=%s present=%t want %s", field, got, present, expected)
					}
					if string(body["amount"]) != `"1"` || string(body["price_offset"]) != `"1%"` {
						t.Errorf("unrelated values changed: %v", body)
					}
					w.Header().Set("Content-Type", "application/json")
					_, _ = io.WriteString(w, `{"code":0,"data":{"id":"9007199254740993","order":{"id":"9007199254740993"}}}`)
				}))
				t.Cleanup(peer.Close)
				t.Setenv("GATE_BASE_URL", peer.URL)
				silenceStdout(t)
				run := runFuturesTrailCreate
				if operation == "update" {
					run = runFuturesTrailUpdate
				}
				require.NoError(t, run(cmd, nil))
				require.EqualValues(t, 1, calls.Load())
			})
		}
	}
}

// Inspect registration without mutating the shared command tree used by other
// tests. Parse failures must remain Cobra errors before business execution.
func TestTrailParameterRegistration(t *testing.T) {
	for _, tc := range []struct{ leaf, name, kind string }{
		{"list", "start-at", "int64"}, {"list", "end-at", "int64"},
		{"create", "is-gte", "bool"}, {"update", "is-gte", "bool"},
	} {
		cmd, _, err := trailCmd.Find([]string{tc.leaf})
		require.NoError(t, err)
		flag := cmd.Flags().Lookup(tc.name)
		require.NotNil(t, flag, "%s --%s", tc.leaf, tc.name)
		require.Equal(t, tc.kind, flag.Value.Type())
		if tc.kind == "int64" {
			require.Contains(t, flag.Usage, "Unix seconds")
		} else {
			require.True(t, strings.Contains(flag.Usage, ">=") || strings.Contains(flag.Usage, "unchanged"))
		}
	}
	root := &cobra.Command{Use: "test"}
	root.Flags().Bool("is-gte", false, "")
	require.Error(t, root.ParseFlags([]string{"--is-gte=invalid"}))
	root.Flags().Int64("start-at", 0, "")
	require.Error(t, root.ParseFlags([]string{"--start-at=9223372036854775808"}))
}

// Handler validation must precede RequireAuth so malformed input cannot trigger
// token refresh or become a misleading login failure.
func TestTrailInvalidRangeBeforeAuth(t *testing.T) {
	cmd := confirmedFuturesLeaf(t)
	require.NoError(t, cmd.Flags().Set("finished", "true"))
	t.Setenv("GATE_API_KEY", "")
	t.Setenv("GATE_API_SECRET", "")
	err := runFuturesTrailList(cmd, nil)
	var detail *output.GateError
	if !errors.As(err, &detail) || detail.Label != "INVALID_ARGUMENTS" {
		t.Fatalf("validation must precede auth: %v", err)
	}
}

func boolText(value bool) string {
	if value {
		return "true"
	}
	return "false"
}
