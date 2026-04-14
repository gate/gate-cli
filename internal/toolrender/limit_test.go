package toolrender

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyOutputLimit_NoLimit(t *testing.T) {
	in := map[string]interface{}{"data": map[string]interface{}{"x": "y"}}
	out := ApplyOutputLimit(in, 0)
	assert.Equal(t, in, out)
}

func TestApplyOutputLimit_TruncatesWhenExceeded(t *testing.T) {
	in := map[string]interface{}{
		"status": "success",
		"data": map[string]interface{}{
			"text": "abcdefghijklmnopqrstuvwxyz",
		},
	}
	out := ApplyOutputLimit(in, 10)
	meta, ok := out["meta"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, true, meta["truncated"])
	assert.Equal(t, int64(10), meta["max_output_bytes"])
	data, ok := out["data"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, true, data["truncated"])
}
