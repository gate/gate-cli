package intelcmd

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gate/gate-cli/internal/mcpclient"
	"github.com/gate/gate-cli/internal/output"
)

func TestUserFacingCLICommandShortcutPath(t *testing.T) {
	assert.Equal(t, "info/+token-risk", UserFacingCLICommand("info", "info/+token-risk", ""))
}

func TestUserFacingCLICommandLeafPath(t *testing.T) {
	got := UserFacingCLICommand("info", "", "info_coin_get_coin_info")
	assert.Equal(t, "info coin get-coin-info", got)
}

func TestSanitizeUserFacingGateErrorClearsMCPToolName(t *testing.T) {
	ge := &output.GateError{
		ToolName: "info_coin_get_coin_info",
		Request:  &output.RequestInfo{Method: "POST", URL: "info/invoke"},
	}
	SanitizeUserFacingGateError(ge, "info", "", "info_coin_get_coin_info")
	assert.Empty(t, ge.ToolName)
	assert.Equal(t, "info coin get-coin-info", ge.Request.URL)
}

func TestGateErrorFromShortcutErrOmitsMCPToolName(t *testing.T) {
	err := &ShortcutToolIsError{
		ToolName: "info_compliance_check_token_security",
		Result: &mcpclient.CallResult{
			IsError: true,
			StructuredContent: map[string]interface{}{
				"message": "upstream failed",
			},
		},
	}
	ge := GateErrorFromShortcutErr(err, "info/+token-risk")
	require.NotNil(t, ge)
	assert.Empty(t, ge.ToolName)
	assert.Equal(t, "info/+token-risk", ge.Request.URL)
	assert.Contains(t, ge.Message, "upstream failed")
}

func TestGateErrorFromShortcutErrHTTPOmitsMCPToolName(t *testing.T) {
	ge := GateErrorFromShortcutErr(&ShortcutHTTPError{
		Err:      errors.New("connection reset"),
		ToolName: "info_coin_get_coin_info",
		HTTPResp: &http.Response{StatusCode: 503},
	}, "info/+coin-overview")
	require.NotNil(t, ge)
	assert.Empty(t, ge.ToolName)
	assert.Equal(t, "info/+coin-overview", ge.Request.URL)
}
