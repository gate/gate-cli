package cmdutil

import (
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

// Inputs are bounded and parsed without a floating-point round trip.
func TestTradingDecimal(t *testing.T) {
	for _, tc := range []struct {
		value        string
		positive, ok bool
	}{
		{"0.000000000000000001", true, true}, {"-1.5", false, true}, {"0", false, true},
		{"-0", true, false}, {"-1", true, false}, {"1e3", true, false}, {"NaN", false, false}, {"1/2", false, false}, {"", false, false}, {" 1", true, false}, {strings.Repeat("1", 81), true, false},
	} {
		t.Run(tc.value, func(t *testing.T) {
			n, err := TradingDecimal("amount", tc.value, tc.positive)
			require.Equal(t, tc.ok, err == nil)
			if tc.ok {
				require.NotNil(t, n)
			}
		})
	}
	n, err := TradingDecimal("amount", "0.000000000000000001", true)
	require.NoError(t, err)
	require.Equal(t, "1/1000000000000000000", n.RatString())
}
func TestTradingToken(t *testing.T) {
	for _, v := range []string{"BTC_USDT", "t-id.1", "9223372036854775807"} {
		require.NoError(t, TradingToken("id", v))
	}
	for _, v := range []string{"", ".", "..", "../a", "a/b", "a?x=y", "a%2fb", "a\\b", strings.Repeat("a", 129)} {
		require.Error(t, TradingToken("id", v))
	}
}
func TestDecodeTradingObject(t *testing.T) {
	for _, v := range []string{`null`, `[]`, `{"amount":1}`, `{"amount":null}`, `{"is_dual_mode":true,"iſ_dual_mode":false}`, `{"i\u017f_dual_mode":false}`, `{"amount":"1","amount":"2"}`, `{"amount":"1","Amount":"2"}`, `{"unknown":"1"}`, `{"amount":"1"} {}`, `{"amount":"1"} garbage`, "{" + strings.Repeat(" ", 65536) + "}"} {
		var target struct {
			Amount string `json:"amount"`
		}
		require.Error(t, DecodeTradingObject("json", v, &target), v)
	}
	var target struct {
		Amount string `json:"amount"`
	}
	require.NoError(t, DecodeTradingObject("json", `{"amount":"0.000000000000000001"}`, &target))
	require.Equal(t, "0.000000000000000001", target.Amount)
}

func TestTradingObjectDepthAndExactTypes(t *testing.T) {
	var target map[string]any
	require.Error(t, DecodeTradingObject("json", `{"a":`+strings.Repeat("[", 34)+`1`+strings.Repeat("]", 34)+`}`, &target))
	require.NoError(t, DecodeTradingObject("json", `{"a":[{"b":true},2]}`, &target))
	require.Error(t, DecodeTradingObject("json", `{"a":[null]}`, &target))
}
