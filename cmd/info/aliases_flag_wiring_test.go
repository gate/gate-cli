package info

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarkettrendGetKlineUsesFlexBoolForWithIndicators(t *testing.T) {
	cmd, _, err := Cmd.Find([]string{"markettrend", "get-kline"})
	require.NoError(t, err)
	fl := cmd.Flags().Lookup("with-indicators")
	require.NotNil(t, fl, "with-indicators flag should exist")
	assert.Equal(t, "flexBool", fl.Value.Type(), "flexBool keeps custom ParseBool behavior")
	assert.Equal(t, "true", fl.NoOptDefVal, "bare --with-indicators must mean true so the flag does not consume the next argv token")
}

func TestMarkettrendGetKlineParseFlags_BareBoolMeansTrue(t *testing.T) {
	leaf, _, err := Cmd.Find([]string{"markettrend", "get-kline"})
	require.NoError(t, err)
	require.NoError(t, leaf.ParseFlags([]string{
		"--symbol", "ETH", "--timeframe", "4h", "--with-indicators",
	}))
	fl := leaf.Flags().Lookup("with-indicators")
	require.NotNil(t, fl)
	v, err := strconv.ParseBool(fl.Value.String())
	require.NoError(t, err)
	assert.True(t, v)
}

func TestMarkettrendGetKlineParseFlags_EqualsBoolValue(t *testing.T) {
	leaf, _, err := Cmd.Find([]string{"markettrend", "get-kline"})
	require.NoError(t, err)
	require.NoError(t, leaf.ParseFlags([]string{
		"--symbol", "ETH", "--timeframe", "4h", "--with-indicators=false",
	}))
	fl := leaf.Flags().Lookup("with-indicators")
	require.NotNil(t, fl)
	v, err := strconv.ParseBool(fl.Value.String())
	require.NoError(t, err)
	assert.False(t, v)
}

func TestMarkettrendGetKlineParseFlags_BoolThenNextFlagDoesNotConsume(t *testing.T) {
	leaf, _, err := Cmd.Find([]string{"markettrend", "get-kline"})
	require.NoError(t, err)
	require.NoError(t, leaf.ParseFlags([]string{
		"--symbol", "ETH", "--with-indicators", "--timeframe", "4h",
	}))
	fl := leaf.Flags().Lookup("with-indicators")
	require.NotNil(t, fl)
	v, err := strconv.ParseBool(fl.Value.String())
	require.NoError(t, err)
	assert.True(t, v, "bare --with-indicators followed by --next-flag must resolve to true, not error")

	tf := leaf.Flags().Lookup("timeframe")
	require.NotNil(t, tf)
	assert.Equal(t, "4h", tf.Value.String(), "the following --timeframe value must still be parsed")
}
