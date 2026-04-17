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
	assert.Equal(t, "flexBool", fl.Value.Type(), "native bool breaks --with-indicators true (spaced)")
}

func TestMarkettrendGetKlineParseFlags_SpacedBoolValue(t *testing.T) {
	leaf, _, err := Cmd.Find([]string{"markettrend", "get-kline"})
	require.NoError(t, err)
	require.NoError(t, leaf.ParseFlags([]string{
		"--symbol", "ETH", "--timeframe", "4h", "--with-indicators", "true",
	}))
	fl := leaf.Flags().Lookup("with-indicators")
	require.NotNil(t, fl)
	v, err := strconv.ParseBool(fl.Value.String())
	require.NoError(t, err)
	assert.True(t, v)
}
