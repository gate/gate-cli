package toolrender

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/gate/gate-cli/internal/mcpclient"
)

func TestBuildCLIEnvelopeParsesTextJSON(t *testing.T) {
	env := BuildCLIEnvelope("news_feed_search_news", &mcpclient.CallResult{
		Content: []mcpclient.ContentItem{{"type": "text", "text": `{"ok":true}`}},
	})
	assert.Equal(t, "success", env["status"])
	assert.Equal(t, "news_feed_search_news", env["tool_name"])
	assert.Equal(t, false, env["is_error"])
	assert.Equal(t, "content", env["data_source"])
	data, ok := env["data"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, true, data["ok"])
}

func TestBuildCLIEnvelopeUsesStructuredContentFirst(t *testing.T) {
	env := BuildCLIEnvelope("tool", &mcpclient.CallResult{
		StructuredContent: map[string]interface{}{"x": "y"},
		Content:           []mcpclient.ContentItem{{"type": "text", "text": `{"ok":true}`}},
	})
	assert.Equal(t, "structured_content", env["data_source"])
	data, ok := env["data"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "y", data["x"])
}

func TestBuildCLIEnvelopeNormalizesMultiContent(t *testing.T) {
	env := BuildCLIEnvelope("tool", &mcpclient.CallResult{
		Content: []mcpclient.ContentItem{
			{"type": "text", "text": `{"a":1}`},
			{"type": "text", "text": "plain"},
		},
	})
	assert.Equal(t, "content", env["data_source"])
	arr, ok := env["data"].([]interface{})
	assert.True(t, ok)
	assert.Len(t, arr, 2)
	meta, ok := env["meta"].(map[string]interface{})
	assert.True(t, ok)
	warnings, ok := meta["parse_warnings"].([]string)
	assert.True(t, ok)
	assert.NotEmpty(t, warnings)
}

func TestBuildCLIEnvelopeMergesWarningsWithMeta(t *testing.T) {
	env := BuildCLIEnvelope("tool", &mcpclient.CallResult{
		Content: []mcpclient.ContentItem{
			{"type": "text", "text": "not-json"},
		},
		Meta: map[string]interface{}{
			"duration_ms": float64(12),
		},
	})
	meta, ok := env["meta"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, float64(12), meta["duration_ms"])
	warnings, ok := meta["parse_warnings"].([]string)
	assert.True(t, ok)
	assert.Len(t, warnings, 1)
}

func TestBuildCLIEnvelopeHandlesUnexpectedContentItemType(t *testing.T) {
	env := BuildCLIEnvelope("tool", &mcpclient.CallResult{
		ContentRaw: []interface{}{
			map[string]interface{}{"type": "text", "text": `{"ok":true}`},
			float64(123),
		},
	})
	assert.Equal(t, "content", env["data_source"])
	data, ok := env["data"].([]interface{})
	assert.True(t, ok)
	assert.Len(t, data, 2)
	meta, ok := env["meta"].(map[string]interface{})
	assert.True(t, ok)
	warnings, ok := meta["parse_warnings"].([]string)
	assert.True(t, ok)
	assert.NotEmpty(t, warnings)
}
