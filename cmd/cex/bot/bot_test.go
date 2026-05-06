package bot

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	gateapi "github.com/gate/gateapi-go/v7"
)

// Test layout for the new cex bot module wired against gateapi-go v7.2.78.
//
// Layer 1 — command-tree wiring (10 commands across 3 trees).
// Layer 2 — SDK type contracts the bot module is locked to (StrategyType
//            enum, v7.2.78 SpotMartingale field rename, InfiniteGrid omitempty).
// Layer 3 — RunE end-to-end against a mock Gate server, asserting the
//            outbound query/body so wire-level regressions surface.
// Layer 4 — auth gate (all 10 commands gated by RequireAuth).
// Layer 5 — input-validation: invalid --json must error before hitting the SDK.

const cobraRequiredFlagAnnotation = "cobra_annotation_bash_completion_one_required_flag"

func newBotTestRoot(t *testing.T) *cobra.Command {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("GATE_API_KEY", "")
	t.Setenv("GATE_API_SECRET", "")
	root := &cobra.Command{Use: "gate-cli"}
	root.PersistentFlags().String("format", "json", "")
	root.PersistentFlags().String("profile", "default", "")
	root.PersistentFlags().Bool("debug", false, "")
	root.PersistentFlags().Bool("verbose", false, "")
	root.PersistentFlags().String("api-key", "", "")
	root.PersistentFlags().String("api-secret", "", "")
	return root
}

func authedBotTestRoot(t *testing.T) *cobra.Command {
	root := newBotTestRoot(t)
	t.Setenv("GATE_API_KEY", "fake-key")
	t.Setenv("GATE_API_SECRET", "fake-secret")
	return root
}

func silenceBotStdout(t *testing.T) {
	t.Helper()
	oldOut := os.Stdout
	devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	require.NoError(t, err)
	os.Stdout = devNull
	t.Cleanup(func() {
		os.Stdout = oldOut
		_ = devNull.Close()
	})
}

// captureRequest spins up a mock server that records the inbound request
// (URL + body) and returns 200 with `respBody`. Subtests inspect the
// captured fields after invoking RunE.
type captured struct {
	URL    *url.URL
	Body   string
	Method string
}

func captureRequest(t *testing.T, respBody string) *captured {
	t.Helper()
	cap := &captured{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cap.URL = r.URL
		cap.Method = r.Method
		if r.ContentLength > 0 {
			buf := make([]byte, r.ContentLength)
			_, _ = r.Body.Read(buf)
			cap.Body = string(buf)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(respBody))
	}))
	t.Cleanup(srv.Close)
	t.Setenv("GATE_BASE_URL", srv.URL)
	return cap
}

func findBotSub(parent *cobra.Command, name string) *cobra.Command {
	for _, c := range parent.Commands() {
		if c.Name() == name {
			return c
		}
	}
	return nil
}

// ===========================================================================
// Layer 1 — Command-tree wiring.
// ===========================================================================

func TestBotRootCommand(t *testing.T) {
	assert.Equal(t, "bot", Cmd.Name())
	assert.NotEmpty(t, Cmd.Short)
}

func TestBotFirstLevelSubcommands(t *testing.T) {
	want := map[string]bool{
		"recommend":  false,
		"running":    false,
		"detail":     false,
		"stop":       false,
		"grid":       false,
		"martingale": false,
	}
	for _, c := range Cmd.Commands() {
		if _, ok := want[c.Name()]; ok {
			want[c.Name()] = true
		}
	}
	for name, found := range want {
		assert.True(t, found, "bot must register %q", name)
	}
}

func TestBotGridSubcommands(t *testing.T) {
	grid := findBotSub(Cmd, "grid")
	require.NotNil(t, grid)

	want := map[string]bool{"spot": false, "margin": false, "infinite": false, "futures": false}
	for _, c := range grid.Commands() {
		if _, ok := want[c.Name()]; ok {
			want[c.Name()] = true
		}
	}
	for name, found := range want {
		assert.True(t, found, "bot grid must register %q", name)
	}
}

func TestBotMartingaleSubcommands(t *testing.T) {
	mart := findBotSub(Cmd, "martingale")
	require.NotNil(t, mart)

	want := map[string]bool{"spot": false, "contract": false}
	for _, c := range mart.Commands() {
		if _, ok := want[c.Name()]; ok {
			want[c.Name()] = true
		}
	}
	for name, found := range want {
		assert.True(t, found, "bot martingale must register %q", name)
	}
}

// Required-flag matrix. detail/stop need both --strategy-id and
// --strategy-type. All 6 create commands need --json.
func TestBotDetail_RequiredFlags(t *testing.T) {
	detail := findBotSub(Cmd, "detail")
	require.NotNil(t, detail)
	for _, name := range []string{"strategy-id", "strategy-type"} {
		f := detail.Flag(name)
		require.NotNil(t, f, "detail must expose --%s", name)
		assert.NotEmpty(t, f.Annotations[cobraRequiredFlagAnnotation],
			"detail --%s must be required", name)
	}
}

func TestBotStop_RequiredFlags(t *testing.T) {
	stop := findBotSub(Cmd, "stop")
	require.NotNil(t, stop)
	for _, name := range []string{"strategy-id", "strategy-type"} {
		f := stop.Flag(name)
		require.NotNil(t, f)
		assert.NotEmpty(t, f.Annotations[cobraRequiredFlagAnnotation])
	}
}

func TestBotRecommend_AllFlagsOptional(t *testing.T) {
	rec := findBotSub(Cmd, "recommend")
	require.NotNil(t, rec)
	for _, name := range []string{"market", "strategy-type", "direction", "invest-amount", "scene", "refresh-recommendation-id", "limit", "max-drawdown-lte", "backtest-apr-gte"} {
		f := rec.Flag(name)
		require.NotNil(t, f, "recommend must expose --%s", name)
		assert.Empty(t, f.Annotations[cobraRequiredFlagAnnotation],
			"recommend --%s must be optional", name)
	}
}

func TestBotRunning_AllFlagsOptional(t *testing.T) {
	run := findBotSub(Cmd, "running")
	require.NotNil(t, run)
	for _, name := range []string{"strategy-type", "market", "page", "page-size"} {
		f := run.Flag(name)
		require.NotNil(t, f)
		assert.Empty(t, f.Annotations[cobraRequiredFlagAnnotation])
	}
}

func TestBotCreateCommands_RequireJSON(t *testing.T) {
	cases := []struct {
		group, leaf string
	}{
		{"grid", "spot"}, {"grid", "margin"}, {"grid", "infinite"}, {"grid", "futures"},
		{"martingale", "spot"}, {"martingale", "contract"},
	}
	for _, tc := range cases {
		grp := findBotSub(Cmd, tc.group)
		require.NotNil(t, grp)
		leaf := findBotSub(grp, tc.leaf)
		require.NotNil(t, leaf, "%s %s must be registered", tc.group, tc.leaf)
		f := leaf.Flag("json")
		require.NotNil(t, f, "%s %s must expose --json", tc.group, tc.leaf)
		assert.NotEmpty(t, f.Annotations[cobraRequiredFlagAnnotation],
			"%s %s --json must be required", tc.group, tc.leaf)
	}
}

// ===========================================================================
// Layer 2 — SDK type contracts the bot module relies on.
// ===========================================================================

// StrategyType enum constants must keep their string values, otherwise the
// `stop` command's gateapi.StrategyType(strategyType) cast would silently
// drift away from server expectations.
func TestStrategyType_EnumStableValues(t *testing.T) {
	assert.Equal(t, "spot_grid", string(gateapi.SPOT_GRID))
	assert.Equal(t, "margin_grid", string(gateapi.MARGIN_GRID))
	assert.Equal(t, "infinite_grid", string(gateapi.INFINITE_GRID))
	assert.Equal(t, "futures_grid", string(gateapi.FUTURES_GRID))
	assert.Equal(t, "spot_martingale", string(gateapi.SPOT_MARTINGALE))
	assert.Equal(t, "contract_martingale", string(gateapi.CONTRACT_MARTINGALE))
}

// v7.2.78 spot martingale wire shape: stop_loss_per_cycle replaces the
// old stop_loss_price; trigger_price is new and optional.
func TestSpotMartingaleCreateParams_NewWireShape(t *testing.T) {
	params := gateapi.SpotMartingaleCreateParams{
		InvestAmount:     "100",
		PriceDeviation:   "0.02",
		MaxOrders:        5,
		TakeProfitRatio:  "0.01",
		StopLossPerCycle: "0.05",
		TriggerPrice:     "70000",
	}
	raw, err := json.Marshal(params)
	require.NoError(t, err)
	s := string(raw)

	assert.Contains(t, s, `"stop_loss_per_cycle":"0.05"`,
		"v7.2.78 spot martingale uses stop_loss_per_cycle")
	assert.Contains(t, s, `"trigger_price":"70000"`,
		"v7.2.78 added trigger_price")
	assert.NotContains(t, s, `"stop_loss_price"`,
		"v7.2.78 removed stop_loss_price from SpotMartingaleCreateParams")
}

// v7.2.78 InfiniteGridCreateParams: grid_num and price_type became omitempty.
// When the user only supplies money/price_floor/profit_per_grid (the
// documented minimum), the optional ints must not show up as 0 on the wire.
func TestInfiniteGridCreateParams_OmitemptyForOptionals(t *testing.T) {
	params := gateapi.InfiniteGridCreateParams{
		Money:         "100",
		PriceFloor:    "60000",
		ProfitPerGrid: "0.005",
	}
	raw, err := json.Marshal(params)
	require.NoError(t, err)
	s := string(raw)

	assert.NotContains(t, s, `"grid_num"`,
		"v7.2.78 made grid_num omitempty; zero must not be sent")
	assert.NotContains(t, s, `"price_type"`,
		"v7.2.78 made price_type omitempty; zero must not be sent")
	assert.Contains(t, s, `"money":"100"`)
	assert.Contains(t, s, `"price_floor":"60000"`)
	assert.Contains(t, s, `"profit_per_grid":"0.005"`)
}

// Conversely, non-zero values still serialize so explicit user choices reach
// the server.
func TestInfiniteGridCreateParams_NonZeroValuesSerialize(t *testing.T) {
	params := gateapi.InfiniteGridCreateParams{
		Money: "100", PriceFloor: "60000", ProfitPerGrid: "0.005",
		GridNum: 50, PriceType: 1,
	}
	raw, err := json.Marshal(params)
	require.NoError(t, err)
	s := string(raw)
	assert.Contains(t, s, `"grid_num":50`)
	assert.Contains(t, s, `"price_type":1`)
}

// AiHubPortfolioStopRequest shape lock; the bot stop CLI builds this struct.
func TestAiHubPortfolioStopRequest_FieldShape(t *testing.T) {
	body := gateapi.AiHubPortfolioStopRequest{
		StrategyId:   "abc-123",
		StrategyType: gateapi.SPOT_GRID,
	}
	raw, err := json.Marshal(body)
	require.NoError(t, err)
	s := string(raw)
	assert.Contains(t, s, `"strategy_id":"abc-123"`)
	assert.Contains(t, s, `"strategy_type":"spot_grid"`)
}

// ===========================================================================
// Layer 3 — RunE end-to-end (query + body capture).
// ===========================================================================

func TestRunBotRecommend_AllFlags_FlowToQuery(t *testing.T) {
	cap := captureRequest(t, `{}`)
	silenceBotStdout(t)
	root := authedBotTestRoot(t)
	cmd := &cobra.Command{Use: "recommend"}
	cmd.Flags().String("market", "BTC_USDT", "")
	cmd.Flags().String("strategy-type", "spot_grid", "")
	cmd.Flags().String("direction", "long", "")
	cmd.Flags().String("invest-amount", "1000", "")
	cmd.Flags().String("scene", "filter", "")
	cmd.Flags().String("refresh-recommendation-id", "spot_grid|BTC_USDT", "")
	cmd.Flags().Int32("limit", 5, "")
	cmd.Flags().String("max-drawdown-lte", "0.3", "")
	cmd.Flags().String("backtest-apr-gte", "0.1", "")
	root.AddCommand(cmd)

	err := runBotRecommend(cmd, nil)
	require.NoError(t, err)

	require.NotNil(t, cap.URL)
	q := cap.URL.Query()
	assert.Equal(t, "BTC_USDT", q.Get("market"))
	assert.Equal(t, "spot_grid", q.Get("strategy_type"))
	assert.Equal(t, "long", q.Get("direction"))
	assert.Equal(t, "1000", q.Get("invest_amount"))
	assert.Equal(t, "filter", q.Get("scene"))
	assert.Equal(t, "spot_grid|BTC_USDT", q.Get("refresh_recommendation_id"))
	assert.Equal(t, "5", q.Get("limit"))
	assert.Equal(t, "0.3", q.Get("max_drawdown_lte"))
	assert.Equal(t, "0.1", q.Get("backtest_apr_gte"))
}

func TestRunBotRecommend_NoFlags_LeavesQueryEmpty(t *testing.T) {
	cap := captureRequest(t, `{}`)
	silenceBotStdout(t)
	root := authedBotTestRoot(t)
	cmd := &cobra.Command{Use: "recommend"}
	cmd.Flags().String("market", "", "")
	cmd.Flags().String("strategy-type", "", "")
	cmd.Flags().String("direction", "", "")
	cmd.Flags().String("invest-amount", "", "")
	cmd.Flags().String("scene", "", "")
	cmd.Flags().String("refresh-recommendation-id", "", "")
	cmd.Flags().Int32("limit", 0, "")
	cmd.Flags().String("max-drawdown-lte", "", "")
	cmd.Flags().String("backtest-apr-gte", "", "")
	root.AddCommand(cmd)

	require.NoError(t, runBotRecommend(cmd, nil))

	require.NotNil(t, cap.URL)
	q := cap.URL.Query()
	for _, k := range []string{"market", "strategy_type", "direction", "invest_amount", "scene", "refresh_recommendation_id", "limit", "max_drawdown_lte", "backtest_apr_gte"} {
		assert.Empty(t, q.Get(k), "zero/empty flag must not leak as %q", k)
	}
}

func TestRunBotRunning_FlowToQuery(t *testing.T) {
	cap := captureRequest(t, `{}`)
	silenceBotStdout(t)
	root := authedBotTestRoot(t)
	cmd := &cobra.Command{Use: "running"}
	cmd.Flags().String("strategy-type", "spot_martingale", "")
	cmd.Flags().String("market", "ETH_USDT", "")
	cmd.Flags().Int32("page", 2, "")
	cmd.Flags().Int32("page-size", 50, "")
	root.AddCommand(cmd)

	require.NoError(t, runBotRunning(cmd, nil))

	q := cap.URL.Query()
	assert.Equal(t, "spot_martingale", q.Get("strategy_type"))
	assert.Equal(t, "ETH_USDT", q.Get("market"))
	assert.Equal(t, "2", q.Get("page"))
	assert.Equal(t, "50", q.Get("page_size"))
}

func TestRunBotDetail_PathAndQuery(t *testing.T) {
	cap := captureRequest(t, `{}`)
	silenceBotStdout(t)
	root := authedBotTestRoot(t)
	cmd := &cobra.Command{Use: "detail"}
	cmd.Flags().String("strategy-id", "strat-001", "")
	cmd.Flags().String("strategy-type", "spot_grid", "")
	root.AddCommand(cmd)

	require.NoError(t, runBotDetail(cmd, nil))

	require.NotNil(t, cap.URL)
	assert.Equal(t, "GET", cap.Method)
	q := cap.URL.Query()
	assert.Equal(t, "strat-001", q.Get("strategy_id"),
		"strategy_id must be sent as a query param on GET /bot/portfolio/detail")
	assert.Equal(t, "spot_grid", q.Get("strategy_type"))
}

func TestRunBotStop_BodyShape(t *testing.T) {
	cap := captureRequest(t, `{}`)
	silenceBotStdout(t)
	root := authedBotTestRoot(t)
	cmd := &cobra.Command{Use: "stop"}
	cmd.Flags().String("strategy-id", "strat-stop-1", "")
	cmd.Flags().String("strategy-type", "futures_grid", "")
	root.AddCommand(cmd)

	require.NoError(t, runBotStop(cmd, nil))

	assert.Equal(t, "POST", cap.Method)
	assert.Contains(t, cap.Body, `"strategy_id":"strat-stop-1"`)
	assert.Contains(t, cap.Body, `"strategy_type":"futures_grid"`)
}

// All 6 create commands share the same JSON-passthrough pattern. The
// table-driven test below avoids 6 near-identical functions while still
// asserting the body actually reaches the server intact.
func TestRunBotCreate_JSONFlowsToBody(t *testing.T) {
	cases := []struct {
		name     string
		fn       func(cmd *cobra.Command, args []string) error
		jsonBody string
		wantKeys []string
	}{
		{
			name:     "grid_spot",
			fn:       runBotGridSpotCreate,
			jsonBody: `{"strategy_type":"spot_grid","market":"BTC_USDT","create_params":{"money":"100","low_price":"60000","high_price":"70000","grid_num":10,"price_type":0}}`,
			wantKeys: []string{`"strategy_type":"spot_grid"`, `"market":"BTC_USDT"`, `"money":"100"`, `"grid_num":10`},
		},
		{
			name:     "grid_margin",
			fn:       runBotGridMarginCreate,
			jsonBody: `{"strategy_type":"margin_grid","market":"BTC_USDT","create_params":{"money":"100","low_price":"60000","high_price":"70000","grid_num":10,"price_type":0,"leverage":"3","direction":"long"}}`,
			wantKeys: []string{`"strategy_type":"margin_grid"`, `"leverage":"3"`, `"direction":"long"`},
		},
		{
			name:     "grid_infinite",
			fn:       runBotGridInfiniteCreate,
			jsonBody: `{"strategy_type":"infinite_grid","market":"BTC_USDT","create_params":{"money":"100","price_floor":"60000","profit_per_grid":"0.005"}}`,
			wantKeys: []string{`"strategy_type":"infinite_grid"`, `"price_floor":"60000"`, `"profit_per_grid":"0.005"`},
		},
		{
			name:     "grid_futures",
			fn:       runBotGridFuturesCreate,
			jsonBody: `{"strategy_type":"futures_grid","market":"BTC_USDT","create_params":{"money":"100","low_price":"60000","high_price":"70000","grid_num":10,"price_type":0,"leverage":"5"}}`,
			wantKeys: []string{`"strategy_type":"futures_grid"`, `"leverage":"5"`},
		},
		{
			name:     "martingale_spot",
			fn:       runBotMartingaleSpotCreate,
			jsonBody: `{"strategy_type":"spot_martingale","market":"BTC_USDT","create_params":{"invest_amount":"100","price_deviation":"0.02","max_orders":5,"take_profit_ratio":"0.01","stop_loss_per_cycle":"0.05","trigger_price":"70000"}}`,
			wantKeys: []string{`"strategy_type":"spot_martingale"`, `"stop_loss_per_cycle":"0.05"`, `"trigger_price":"70000"`},
		},
		{
			name:     "martingale_contract",
			fn:       runBotMartingaleContractCreate,
			jsonBody: `{"strategy_type":"contract_martingale","market":"BTC_USDT","create_params":{"invest_amount":"100","price_deviation":"0.02","max_orders":5,"take_profit_ratio":"0.01","direction":"buy","leverage":"3"}}`,
			wantKeys: []string{`"strategy_type":"contract_martingale"`, `"direction":"buy"`, `"leverage":"3"`},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cap := captureRequest(t, `{}`)
			silenceBotStdout(t)
			root := authedBotTestRoot(t)
			cmd := &cobra.Command{Use: tc.name}
			cmd.Flags().String("json", tc.jsonBody, "")
			root.AddCommand(cmd)

			require.NoError(t, tc.fn(cmd, nil))

			assert.Equal(t, "POST", cap.Method)
			for _, want := range tc.wantKeys {
				assert.Contains(t, cap.Body, want, "case %s expected %s in outbound body", tc.name, want)
			}
		})
	}
}

// Specific guard for the v7.2.78 spot martingale wire-shape change: even
// though Layer 2 already pinned the model, this RunE-level test proves
// the rename survives the full CLI → SDK → HTTP path.
func TestRunBotMartingaleSpot_StopLossPerCycle_OnWire(t *testing.T) {
	cap := captureRequest(t, `{}`)
	silenceBotStdout(t)
	root := authedBotTestRoot(t)
	cmd := &cobra.Command{Use: "spot"}
	cmd.Flags().String("json",
		`{"strategy_type":"spot_martingale","market":"BTC_USDT","create_params":{"invest_amount":"100","price_deviation":"0.02","max_orders":5,"take_profit_ratio":"0.01","stop_loss_per_cycle":"0.05"}}`, "")
	root.AddCommand(cmd)

	require.NoError(t, runBotMartingaleSpotCreate(cmd, nil))

	assert.Contains(t, cap.Body, `"stop_loss_per_cycle":"0.05"`,
		"runBotMartingaleSpotCreate must transmit the v7.2.78 stop_loss_per_cycle field")
	assert.NotContains(t, cap.Body, `"stop_loss_price"`,
		"the legacy stop_loss_price field is gone from SpotMartingaleCreateParams in v7.2.78")
}

// Specific guard for InfiniteGrid omitempty: the user-supplied minimal
// JSON (without grid_num/price_type) must reach the server without
// auto-zero injection.
func TestRunBotGridInfinite_MinimalJSON_OmitsOptionalInts(t *testing.T) {
	cap := captureRequest(t, `{}`)
	silenceBotStdout(t)
	root := authedBotTestRoot(t)
	cmd := &cobra.Command{Use: "infinite"}
	cmd.Flags().String("json",
		`{"strategy_type":"infinite_grid","market":"BTC_USDT","create_params":{"money":"100","price_floor":"60000","profit_per_grid":"0.005"}}`, "")
	root.AddCommand(cmd)

	require.NoError(t, runBotGridInfiniteCreate(cmd, nil))

	assert.NotContains(t, cap.Body, `"grid_num"`,
		"v7.2.78 omitempty: minimal JSON must not auto-inject grid_num=0")
	assert.NotContains(t, cap.Body, `"price_type"`)
}

// ===========================================================================
// Layer 4 — RequireAuth gate. All 10 RunE handlers must trip without creds.
// ===========================================================================

func TestRunBotRecommend_RequiresAuth(t *testing.T) {
	root := newBotTestRoot(t)
	cmd := &cobra.Command{Use: "recommend"}
	for _, n := range []string{"market", "strategy-type", "direction", "invest-amount", "scene", "refresh-recommendation-id", "max-drawdown-lte", "backtest-apr-gte"} {
		cmd.Flags().String(n, "", "")
	}
	cmd.Flags().Int32("limit", 0, "")
	root.AddCommand(cmd)

	err := runBotRecommend(cmd, nil)
	require.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "api key")
}

func TestRunBotRunning_RequiresAuth(t *testing.T) {
	root := newBotTestRoot(t)
	cmd := &cobra.Command{Use: "running"}
	cmd.Flags().String("strategy-type", "", "")
	cmd.Flags().String("market", "", "")
	cmd.Flags().Int32("page", 0, "")
	cmd.Flags().Int32("page-size", 0, "")
	root.AddCommand(cmd)

	err := runBotRunning(cmd, nil)
	require.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "api key")
}

func TestRunBotDetail_RequiresAuth(t *testing.T) {
	root := newBotTestRoot(t)
	cmd := &cobra.Command{Use: "detail"}
	cmd.Flags().String("strategy-id", "x", "")
	cmd.Flags().String("strategy-type", "spot_grid", "")
	root.AddCommand(cmd)

	err := runBotDetail(cmd, nil)
	require.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "api key")
}

func TestRunBotStop_RequiresAuth(t *testing.T) {
	root := newBotTestRoot(t)
	cmd := &cobra.Command{Use: "stop"}
	cmd.Flags().String("strategy-id", "x", "")
	cmd.Flags().String("strategy-type", "spot_grid", "")
	root.AddCommand(cmd)

	err := runBotStop(cmd, nil)
	require.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "api key")
}

// All 6 create commands are gated by RequireAuth; cover them table-style.
func TestRunBotCreateCommands_RequireAuth(t *testing.T) {
	cases := []struct {
		name string
		fn   func(cmd *cobra.Command, args []string) error
	}{
		{"grid_spot", runBotGridSpotCreate},
		{"grid_margin", runBotGridMarginCreate},
		{"grid_infinite", runBotGridInfiniteCreate},
		{"grid_futures", runBotGridFuturesCreate},
		{"martingale_spot", runBotMartingaleSpotCreate},
		{"martingale_contract", runBotMartingaleContractCreate},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := newBotTestRoot(t)
			cmd := &cobra.Command{Use: tc.name}
			cmd.Flags().String("json", `{}`, "")
			root.AddCommand(cmd)

			err := tc.fn(cmd, nil)
			require.Error(t, err)
			assert.Contains(t, strings.ToLower(err.Error()), "api key")
		})
	}
}

// ===========================================================================
// Layer 5 — Input validation. Invalid --json must fail before any HTTP.
// Auth must pass first; otherwise the RequireAuth path returns early and
// we cannot exercise the JSON parse branch.
// ===========================================================================

func TestRunBotCreateCommands_InvalidJSON_Errors(t *testing.T) {
	// No mock server registered — if the JSON parse path were skipped
	// (regression), the SDK would panic-style fail trying to dial. The
	// test asserts a clean parse error.
	cases := []struct {
		name string
		fn   func(cmd *cobra.Command, args []string) error
	}{
		{"grid_spot", runBotGridSpotCreate},
		{"grid_margin", runBotGridMarginCreate},
		{"grid_infinite", runBotGridInfiniteCreate},
		{"grid_futures", runBotGridFuturesCreate},
		{"martingale_spot", runBotMartingaleSpotCreate},
		{"martingale_contract", runBotMartingaleContractCreate},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := authedBotTestRoot(t)
			cmd := &cobra.Command{Use: tc.name}
			cmd.Flags().String("json", `{"strategy_type": not-a-string}`, "")
			root.AddCommand(cmd)

			err := tc.fn(cmd, nil)
			require.Error(t, err)
			assert.Contains(t, strings.ToLower(err.Error()), "invalid --json",
				"invalid JSON must surface as a clear --json error, not a network failure")
		})
	}
}
