package cex_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/gate/gate-cli/cmd/cex"
	"github.com/gate/gate-cli/internal/oauth2"
	"github.com/stretchr/testify/require"
)

// The peer is the only mock: command parsing, handlers and released SDK encoding run unchanged.
func optionalOrderPeer(t *testing.T, args []string, status int, response, wantPath, wantBody string, wantSend bool) error {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	oauth2.ResetHomeCacheForTest()
	t.Cleanup(oauth2.ResetHomeCacheForTest)
	t.Setenv("GATE_API_KEY", "fixture-key")
	t.Setenv("GATE_API_SECRET", "fixture-secret")
	var sends atomic.Int32
	bodies := make(chan []byte, 1)
	paths := make(chan string, 1)
	peer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sends.Add(1)
		raw, err := io.ReadAll(io.LimitReader(r.Body, 65537))
		if err != nil {
			t.Error(err)
			http.Error(w, "fixture read failed", 500)
			return
		}
		select {
		case bodies <- raw:
		default:
		}
		select {
		case paths <- r.Method + " " + r.URL.Path:
		default:
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = io.WriteString(w, response)
	}))
	defer peer.Close()
	t.Setenv("GATE_BASE_URL", peer.URL)
	root := gapRoot(t, args)
	defer root.RemoveCommand(cex.Cmd)
	err := root.Execute()
	if !wantSend {
		require.Zero(t, sends.Load())
		return err
	}
	require.EqualValues(t, 1, sends.Load(), "a rejected or unconfirmed write must never replay")
	require.Equal(t, wantPath, <-paths)
	raw := <-bodies
	require.True(t, json.Valid(raw))
	require.JSONEq(t, wantBody, string(raw))
	return err
}

// Fixed expected JSON is independent of SDK output and protects old omission semantics.
func TestOrderOptionalSDKFields(t *testing.T) {
	option := []string{"options", "order", "create", "--contract", "SYNTH", "--size", "1", "--price", "100", "--tif", "gtc"}
	tradfi := []string{"tradfi", "order", "create", "--symbol", "SYNTH", "--side", "2", "--volume", "0.01", "--tp", "200", "--sl", "50"}
	for _, tc := range []struct {
		name       string
		args       []string
		path, body string
	}{
		{"mmp_omitted", option, "POST /api/v4/options/orders", `{"contract":"SYNTH","size":1,"price":"100","tif":"gtc"}`},
		{"mmp_false", append(append([]string{}, option...), "--mmp=false"), "POST /api/v4/options/orders", `{"contract":"SYNTH","size":1,"price":"100","tif":"gtc"}`},
		{"mmp_true", append(append([]string{}, option...), "--mmp"), "POST /api/v4/options/orders", `{"contract":"SYNTH","size":1,"price":"100","tif":"gtc","mmp":true}`},
		{"mmp_full_close", []string{"options", "order", "create", "--contract", "SYNTH", "--size", "0", "--close", "--mmp"}, "POST /api/v4/options/orders", `{"contract":"SYNTH","size":0,"close":true,"mmp":true}`},
		{"mmp_reduce_long", []string{"options", "order", "create", "--contract", "SYNTH", "--size", "-1", "--reduce-only", "--mmp"}, "POST /api/v4/options/orders", `{"contract":"SYNTH","size":-1,"reduce_only":true,"mmp":true}`},
		{"mmp_reduce_short", []string{"options", "order", "create", "--contract", "SYNTH", "--size", "1", "--reduce-only", "--mmp"}, "POST /api/v4/options/orders", `{"contract":"SYNTH","size":1,"reduce_only":true,"mmp":true}`},
		{"leverage_omitted", tradfi, "POST /api/v4/tradfi/orders", `{"symbol":"SYNTH","side":2,"volume":"0.01","price":"0","price_type":"market","price_tp":"200","price_sl":"50"}`},
		{"leverage_zero_sdk_omission", append(append([]string{}, tradfi...), "--leverage", "0"), "POST /api/v4/tradfi/orders", `{"symbol":"SYNTH","side":2,"volume":"0.01","price":"0","price_type":"market","price_tp":"200","price_sl":"50"}`},
		{"leverage_explicit", append(append([]string{}, tradfi...), "--leverage", "100"), "POST /api/v4/tradfi/orders", `{"symbol":"SYNTH","side":2,"volume":"0.01","price":"0","price_type":"market","price_tp":"200","price_sl":"50","leverage":100}`},
		{"leverage_max", append(append([]string{}, tradfi...), "--leverage", "2147483647"), "POST /api/v4/tradfi/orders", `{"symbol":"SYNTH","side":2,"volume":"0.01","price":"0","price_type":"market","price_tp":"200","price_sl":"50","leverage":2147483647}`},
		{"leverage_min", append(append([]string{}, tradfi...), "--leverage", "-2147483648"), "POST /api/v4/tradfi/orders", `{"symbol":"SYNTH","side":2,"volume":"0.01","price":"0","price_type":"market","price_tp":"200","price_sl":"50","leverage":-2147483648}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.NoError(t, optionalOrderPeer(t, tc.args, 200, `{}`, tc.path, tc.body, true))
		})
	}
}

// Business eligibility is the server's decision; a refusal must keep its reason and not downgrade the request.
func TestOrderOptionalSDKFailures(t *testing.T) {
	for _, tc := range []struct {
		name       string
		args       []string
		path, body string
	}{
		{"mmp", []string{"options", "order", "create", "--contract", "SYNTH", "--size", "1", "--mmp"}, "POST /api/v4/options/orders", `{"contract":"SYNTH","size":1,"mmp":true}`},
		{"leverage", []string{"tradfi", "order", "create", "--symbol", "SYNTH", "--side", "2", "--volume", "0.01", "--leverage", "-1"}, "POST /api/v4/tradfi/orders", `{"symbol":"SYNTH","side":2,"volume":"0.01","price":"0","price_type":"market","leverage":-1}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, status := range []int{400, 401, 403, 429, 503} {
				t.Run(http.StatusText(status), func(t *testing.T) {
					err := optionalOrderPeer(t, tc.args, status, `{"label":"FIXTURE_REJECTED","message":"server rule rejected"}`, tc.path, tc.body, true)
					require.Error(t, err)
					require.Contains(t, err.Error(), "server rule rejected")
				})
			}
			t.Run("malformed_response", func(t *testing.T) {
				require.Error(t, optionalOrderPeer(t, tc.args, 200, `not-json`, tc.path, tc.body, true))
			})
		})
	}
}

// These are representation errors, not a second copy of trading business rules.
func TestOrderOptionalParsingBeforeNetwork(t *testing.T) {
	for _, args := range [][]string{
		{"options", "order", "create", "--contract", "SYNTH", "--size", "1", "--mmp=invalid"},
		{"options", "order", "create", "--contract", "SYNTH", "--size", "1", "--close", "--mmp"},
		{"tradfi", "order", "create", "--symbol", "SYNTH", "--side", "2", "--volume", "0.01", "--leverage", "1.5"},
		{"tradfi", "order", "create", "--symbol", "SYNTH", "--side", "2", "--volume", "0.01", "--leverage", "bad"},
		{"tradfi", "order", "create", "--symbol", "SYNTH", "--side", "2", "--volume", "0.01", "--leverage", "2147483648"},
		{"tradfi", "order", "create", "--symbol", "SYNTH", "--side", "2", "--volume", "0.01", "--leverage", "-2147483649"},
	} {
		t.Run(args[len(args)-1], func(t *testing.T) { require.Error(t, optionalOrderPeer(t, args, 200, `{}`, "", "", false)) })
	}
}
