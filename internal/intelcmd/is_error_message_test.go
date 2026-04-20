package intelcmd

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gate/gate-cli/internal/mcpclient"
)

func TestMessageFromIntelToolIsError_StructuredContentMessage(t *testing.T) {
	t.Parallel()
	r := &mcpclient.CallResult{
		StructuredContent: map[string]interface{}{
			"message": "invalid time_range for this tool",
		},
	}
	assert.Equal(t, "invalid time_range for this tool", messageFromIntelToolIsError(r))
}

func TestMessageFromIntelToolIsError_StructuredContentNestedError(t *testing.T) {
	t.Parallel()
	r := &mcpclient.CallResult{
		StructuredContent: map[string]interface{}{
			"error": map[string]interface{}{
				"message": "upstream validation failed",
			},
		},
	}
	assert.Equal(t, "upstream validation failed", messageFromIntelToolIsError(r))
}

func TestMessageFromIntelToolIsError_ContentTextJSON(t *testing.T) {
	t.Parallel()
	r := &mcpclient.CallResult{
		ContentRaw: []interface{}{
			map[string]interface{}{
				"type": "text",
				"text": `{"message":"bad argument"}`,
			},
		},
	}
	assert.Equal(t, "bad argument", messageFromIntelToolIsError(r))
}

func TestMessageFromIntelToolIsError_RawMap(t *testing.T) {
	t.Parallel()
	r := &mcpclient.CallResult{
		Raw: map[string]interface{}{
			"isError": true,
			"message": "from raw only",
		},
	}
	assert.Equal(t, "from raw only", messageFromIntelToolIsError(r))
}

func TestMessageFromIntelToolIsError_Truncates(t *testing.T) {
	t.Parallel()
	long := strings.Repeat("x", maxIntelToolIsErrorMessageRunes+50)
	r := &mcpclient.CallResult{
		StructuredContent: map[string]interface{}{"message": long},
	}
	got := messageFromIntelToolIsError(r)
	require.Greater(t, len(got), 10)
	assert.Contains(t, got, "…")
	assert.LessOrEqual(t, utf8.RuneCountInString(got), maxIntelToolIsErrorMessageRunes+1) // + ellipsis
}

func TestMessageFromIntelToolIsError_RedactsBearer(t *testing.T) {
	t.Parallel()
	r := &mcpclient.CallResult{
		StructuredContent: map[string]interface{}{
			"message": "auth failed bearer secret-token-value please retry",
		},
	}
	got := messageFromIntelToolIsError(r)
	assert.Contains(t, got, "Bearer [redacted]")
	assert.NotContains(t, got, "secret-token-value")
}

func TestGateErrorForIntelToolIsErrorUsesExtractedMessage(t *testing.T) {
	t.Parallel()
	ge := GateErrorForIntelToolIsError("news_events_get_latest_events", nil, &mcpclient.CallResult{
		StructuredContent: map[string]interface{}{
			"message": "time_range not supported",
		},
	})
	assert.Equal(t, "time_range not supported", ge.Message)
	assert.Equal(t, 400, ge.Status)
	assert.Equal(t, "INVALID_ARGUMENTS", ge.Label)
}
