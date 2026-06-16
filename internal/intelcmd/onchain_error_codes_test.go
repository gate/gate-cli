package intelcmd

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/gate/gate-cli/internal/mcpclient"
)

func TestGateErrorMetaForIntelToolIsError_OnchainInvalidChain(t *testing.T) {
	t.Parallel()
	st, lb := gateErrorMetaForIntelToolIsError("无效或不受支持的 chain: random", nil)
	assert.Equal(t, 400, st)
	assert.Equal(t, "INVALID_CHAIN", lb)
}

func TestGateErrorMetaForIntelToolIsError_OnchainInvalidAddress(t *testing.T) {
	t.Parallel()
	st, lb := gateErrorMetaForIntelToolIsError("address 格式无效", nil)
	assert.Equal(t, 400, st)
	assert.Equal(t, "INVALID_ADDRESS", lb)
}

func TestGateErrorMetaForIntelToolIsError_PartialUpstream502(t *testing.T) {
	t.Parallel()
	r := &mcpclient.CallResult{
		StructuredContent: map[string]interface{}{"code": "partial_upstream_response"},
	}
	st, lb := gateErrorMetaForIntelToolIsError("partial_upstream_response", r)
	assert.Equal(t, 502, st)
	assert.Equal(t, "PARTIAL_UPSTREAM_RESPONSE", lb)
}

func TestGateErrorForIntelToolIsError_PartialUpstreamFriendlyMessage(t *testing.T) {
	t.Parallel()
	ge := GateErrorForIntelToolIsError("info_onchain_get_address_transactions", nil, &mcpclient.CallResult{
		StructuredContent: map[string]interface{}{
			"code":    "partial_upstream_response",
			"message": "partial_upstream_response",
		},
	})
	assert.Equal(t, 502, ge.Status)
	assert.Equal(t, "PARTIAL_UPSTREAM_RESPONSE", ge.Label)
	assert.Contains(t, ge.Message, "upstream returned total")
	assert.NotContains(t, ge.Message, "暂无交易")
}
