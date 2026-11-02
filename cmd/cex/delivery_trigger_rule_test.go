package cex_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

// Each call owns its arguments because Cobra and the test peer are reused serially.
func deliveryTriggerCommand() []string {
	return []string{"delivery", "price-trigger", "create", "--contract", "SYNTH", "--trigger-price", "20.25", "--size", "3"}
}

// Rules must come from the caller rather than the zero value or an inferred direction.
func TestDeliveryTriggerRuleSDKFields(t *testing.T) {
	for _, tc := range []struct {
		name, rule, price, body string
	}{
		{"greater_equal", "1", "19.5", `{"initial":{"contract":"SYNTH","size":3,"price":"19.5"},"trigger":{"price":"20.25","rule":1}}`},
		{"less_equal", "2", "19.5", `{"initial":{"contract":"SYNTH","size":3,"price":"19.5"},"trigger":{"price":"20.25","rule":2}}`},
		{"keep_omitted_order_price", "2", "", `{"initial":{"contract":"SYNTH","size":3,"price":""},"trigger":{"price":"20.25","rule":2}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			args := append(deliveryTriggerCommand(), "--trigger-rule", tc.rule)
			if tc.price != "" {
				args = append(args, "--order-price", tc.price)
			}
			require.NoError(t, optionalOrderPeer(t, args, 200, `{"id":123}`, "POST /api/v4/delivery/usdt/price_orders", tc.body, true))
		})
	}
}

// Only representation is validated locally; service refusals cannot cause a substitute rule or replay.
func TestDeliveryTriggerRuleSDKFailures(t *testing.T) {
	for _, rule := range []string{"0", "3", "-2147483648", "2147483647"} {
		t.Run("server_rejects_"+rule, func(t *testing.T) {
			args := append(deliveryTriggerCommand(), "--trigger-rule", rule)
			body := fmt.Sprintf(`{"initial":{"contract":"SYNTH","size":3,"price":""},"trigger":{"price":"20.25","rule":%s}}`, rule)
			err := optionalOrderPeer(t, args, 400, `{"label":"FIXTURE_REJECTED","message":"server rule rejected"}`, "POST /api/v4/delivery/usdt/price_orders", body, true)
			require.ErrorContains(t, err, "server rule rejected")
		})
	}
	for _, status := range []int{400, 401, 403, 429, 503, 200} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			response := `{"label":"FIXTURE_REJECTED","message":"server rule rejected"}`
			if status == 200 {
				response = `not-json`
			}
			args := append(deliveryTriggerCommand(), "--trigger-rule", "1")
			err := optionalOrderPeer(t, args, status, response, "POST /api/v4/delivery/usdt/price_orders", `{"initial":{"contract":"SYNTH","size":3,"price":""},"trigger":{"price":"20.25","rule":1}}`, true)
			require.Error(t, err)
			if status != 200 {
				require.Contains(t, err.Error(), "server rule rejected")
			}
		})
	}
}

// Missing, fractional and overflowing rules must stop before any financial HTTP request.
func TestDeliveryTriggerRuleParsingBeforeNetwork(t *testing.T) {
	t.Run("missing", func(t *testing.T) {
		err := optionalOrderPeer(t, deliveryTriggerCommand(), 200, `{}`, "", "", false)
		require.ErrorContains(t, err, "trigger-rule")
	})
	for _, value := range []string{"bad", "1.5", "2147483648", "-2147483649"} {
		t.Run(value, func(t *testing.T) {
			args := append(deliveryTriggerCommand(), "--trigger-rule", value)
			err := optionalOrderPeer(t, args, 200, `{}`, "", "", false)
			require.ErrorContains(t, err, "trigger-rule")
		})
	}
}
