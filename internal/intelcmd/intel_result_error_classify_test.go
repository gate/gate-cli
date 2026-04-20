package intelcmd

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/gate/gate-cli/internal/mcpclient"
)

func TestGateErrorMetaForIntelToolIsError_ChineseValidation(t *testing.T) {
	t.Parallel()
	st, lb := gateErrorMetaForIntelToolIsError("参数不合法: time_range 仅支持 1h、24h、7d", nil)
	assert.Equal(t, 400, st)
	assert.Equal(t, "INVALID_ARGUMENTS", lb)
}

func TestGateErrorMetaForIntelToolIsError_StructuredCode(t *testing.T) {
	t.Parallel()
	r := &mcpclient.CallResult{
		StructuredContent: map[string]interface{}{"code": "INVALID_ARGUMENT"},
	}
	st, lb := gateErrorMetaForIntelToolIsError("something", r)
	assert.Equal(t, 400, st)
	assert.Equal(t, "INVALID_ARGUMENTS", lb)
}

func TestGateErrorMetaForIntelToolIsError_HTTPStatusHint(t *testing.T) {
	t.Parallel()
	r := &mcpclient.CallResult{
		StructuredContent: map[string]interface{}{"http_status": float64(422)},
	}
	st, lb := gateErrorMetaForIntelToolIsError("", r)
	assert.Equal(t, 422, st)
	assert.Equal(t, "INVALID_ARGUMENTS", lb)
}

func TestGateErrorMetaForIntelToolIsError_UpstreamStays502(t *testing.T) {
	t.Parallel()
	st, lb := gateErrorMetaForIntelToolIsError("upstream validation failed", nil)
	assert.Equal(t, 502, st)
	assert.Equal(t, "INTEL_RESULT_ERROR", lb)
}

func TestGateErrorMetaForIntelToolIsError_GenericStays502(t *testing.T) {
	t.Parallel()
	st, lb := gateErrorMetaForIntelToolIsError("internal tool failure", nil)
	assert.Equal(t, 502, st)
	assert.Equal(t, "INTEL_RESULT_ERROR", lb)
}
