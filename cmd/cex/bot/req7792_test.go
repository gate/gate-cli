package bot

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/gate/gate-cli/internal/output"
	"github.com/stretchr/testify/require"
)

// Test real Cobra → client → generated SDK → HTTP; the peer owns business outcomes.
func TestREQ7792PortfolioCommands(t *testing.T) {
	for _, name := range []string{"smart-position", "futures-combo"} {
		for _, code := range []int{200, 400} {
			t.Run(name+fmtInt(code), func(t *testing.T) {
				root := authedBotTestRoot(t)
				silenceBotStdout(t)
				cap := captureRequest(t, `{"code":`+fmtInt(code)+`,"label":"RECOVERY_REQUIRED","message":"verify","trace_id":"fixture-trace","data":{"strategy_id":"9223372036854775807","status":"starting","operation_status":"pending","recovery_required":false}}`)
				root.AddCommand(Cmd)
				typ := "smart_position"
				params := `"asset_info":[{"asset":"BTC","rate":"0.5"},{"asset":"ETH","rate":"0.5"}],"close_flag":0`
				if name == "futures-combo" {
					typ = "futures_combo"
					params = `"leverage":2,"asset_info":[{"market":"BTC_USDT","side":"buy","rate":"0.5"},{"market":"ETH_USDT","side":"sell","rate":"0.5"}]`
				}
				raw := `{"strategy_type":"` + typ + `","create_params":{"begin_money_amount":"100","begin_money_currency":"USDT","period":"1h",` + params + `}}`
				root.SetArgs([]string{"bot", name, "create", "--json", raw})
				err := root.Execute()
				require.Equal(t, "/api/v4/bot/"+name+"/create", cap.URL.Path)
				if code == 200 {
					require.NoError(t, err)
				} else {
					var detail *output.GateError
					require.True(t, errors.As(err, &detail))
					require.Equal(t, "RECOVERY_REQUIRED", detail.Label)
					require.Equal(t, "fixture-trace", detail.TraceID)
					require.Contains(t, string(detail.Data), `"strategy_id":"9223372036854775807"`)
					require.Contains(t, string(detail.Data), `"recovery_required":false`)
				}
				if name == "smart-position" {
					require.Contains(t, cap.Body, `"close_flag":0`)
				}
			})
		}
	}
}

// Helpers test protocol boundaries without contacting any live trading service.
func fmtInt(n int) string { raw, _ := json.Marshal(n); return string(raw) }
func TestREQ7792InvalidIntent(t *testing.T) {
	for _, raw := range []string{`null`, `[]`, `{"strategy_type":"smart_position","extra":1}`, `{} {}`, `{"strategy_type":"smart_position","create_params":{"period":"1h","threshold":"0.1"}}`, `{"strategy_type":"wrong"}`} {
		root := authedBotTestRoot(t)
		root.AddCommand(Cmd)
		cap := captureRequest(t, `{"code":200}`)
		root.SetArgs([]string{"bot", "smart-position", "create", "--json", raw})
		require.Error(t, root.Execute())
		require.Nil(t, cap.URL)
	}
}

// Stop options must retain their product-specific meaning through command flags.
func TestREQ7792StopModesAndRecommendationRefresh(t *testing.T) {
	for _, tc := range []struct {
		kind, mode, options string
		wantError           bool
	}{
		{"smart_position", "limit", `{"price":"100.000000000000000001","expiration":60,"slip":"0.01"}`, false},
		{"smart_position", "no_close", "", false},
		{"smart_position", "market", "", false},
		{"futures_combo", "", "", false},
		{"smart_position", "", "", true},
		{"smart_position", "limit", "", true},
		{"smart_position", "market", `{"price":"1","expiration":60}`, true},
		{"futures_combo", "market", "", true},
	} {
		t.Run(tc.kind+tc.mode+tc.options, func(t *testing.T) {
			root := authedBotTestRoot(t)
			silenceBotStdout(t)
			root.AddCommand(Cmd)
			cap := captureRequest(t, `{"code":200,"data":{"strategy_id":"9223372036854775807","status":"stopping"}}`)
			command := findBotSub(Cmd, "stop")
			// Global Cobra commands persist between tests; reset every tested optional flag.
			require.NoError(t, command.Flags().Set("strategy-id", "9223372036854775807"))
			require.NoError(t, command.Flags().Set("strategy-type", tc.kind))
			require.NoError(t, command.Flags().Set("stop-mode", tc.mode))
			require.NoError(t, command.Flags().Set("stop-options", tc.options))
			err := command.RunE(command, nil)
			if tc.wantError {
				require.Error(t, err)
				require.Nil(t, cap.URL)
				return
			}
			require.NoError(t, err)
			require.Contains(t, cap.Body, `"strategy_id":"9223372036854775807"`)
			if tc.mode == "limit" {
				require.Contains(t, cap.Body, `"price":"100.000000000000000001"`)
			} else {
				require.NotContains(t, cap.Body, `"stop_options"`)
			}
		})
	}
	root := authedBotTestRoot(t)
	silenceBotStdout(t)
	root.AddCommand(Cmd)
	cap := captureRequest(t, `{"code":200,"data":{}}`)
	command := findBotSub(Cmd, "recommend")
	require.NoError(t, command.Flags().Set("scene", "refresh"))
	require.NoError(t, command.Flags().Set("exclude-recommendation-id", "smart_position|USDT|123|a"))
	require.NoError(t, command.RunE(command, nil))
	require.Equal(t, "smart_position|USDT|123|a", cap.URL.Query().Get("exclude_recommendation_id"))
}

// Published SDK request types must reject ambiguous financial intent before HTTP.
func TestREQ7792ContractStopPricePair(t *testing.T) {
	for _, tc := range []struct {
		price, kind string
		invalid     bool
	}{
		{"", "", false},
		{"10.000000000000000001", "last", false},
		{"10", "index", false},
		{"10", "mark", false},
		{"10", "", true},
		{"", "mark", true},
		{"10", "mid", true},
	} {
		t.Run(tc.price+tc.kind, func(t *testing.T) {
			root := authedBotTestRoot(t)
			silenceBotStdout(t)
			root.AddCommand(Cmd)
			cap := captureRequest(t, `{"code":200,"data":{"strategy_id":"1","status":"starting"}}`)
			command := findBotSub(findBotSub(Cmd, "martingale"), "contract")
			raw, err := json.Marshal(map[string]any{"strategy_type": "contract_martingale", "create_params": map[string]any{"stop_loss_price": tc.price, "stop_loss_price_type": tc.kind}})
			require.NoError(t, err)
			require.NoError(t, command.Flags().Set("json", string(raw)))
			err = command.RunE(command, nil)
			if tc.invalid {
				require.Error(t, err)
				require.Nil(t, cap.URL)
				return
			}
			require.NoError(t, err)
			if tc.price != "" {
				require.Contains(t, cap.Body, `"stop_loss_price":"`+tc.price+`"`)
				require.Contains(t, cap.Body, `"stop_loss_price_type":"`+tc.kind+`"`)
			}
		})
	}
}
