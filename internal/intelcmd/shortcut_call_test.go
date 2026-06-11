package intelcmd

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gate/gate-cli/internal/mcpclient"
)

func TestGateErrorFromShortcutErrToolIsError(t *testing.T) {
	err := &ShortcutToolIsError{
		ToolName: "news_events_get_latest_events",
		Result: &mcpclient.CallResult{
			IsError: true,
			StructuredContent: map[string]interface{}{
				"message": "invalid coin",
			},
		},
	}
	ge := GateErrorFromShortcutErr(err, "news/+brief")
	require.NotNil(t, ge)
	assert.Empty(t, ge.ToolName)
	assert.Equal(t, "news/+brief", ge.Request.URL)
	assert.Contains(t, ge.Message, "invalid coin")
}

func TestGateErrorFromShortcutErrArgs(t *testing.T) {
	ge := GateErrorFromShortcutErr(ShortcutArgsError("coin or query is required"), "news/+brief")
	require.NotNil(t, ge)
	assert.Equal(t, "INVALID_ARGUMENTS", ge.Label)
	assert.Contains(t, ge.Message, "coin or query is required")
}

func TestGateErrorFromShortcutErrHTTP(t *testing.T) {
	ge := GateErrorFromShortcutErr(&ShortcutHTTPError{
		Err:      errors.New("connection reset"),
		ToolName: "info_coin_get_coin_info",
		HTTPResp: &http.Response{StatusCode: 503},
	}, "info/+coin-overview")
	require.NotNil(t, ge)
	assert.Empty(t, ge.ToolName)
	assert.Equal(t, "info/+coin-overview", ge.Request.URL)
}
