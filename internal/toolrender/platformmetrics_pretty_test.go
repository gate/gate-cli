package toolrender

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gate/gate-cli/internal/mcpclient"
	"github.com/gate/gate-cli/internal/output"
)

func TestFormatChainActivityPretty_LatestAndSeries(t *testing.T) {
	t.Parallel()
	out := formatChainActivityPretty(map[string]interface{}{
		"chain":            "ethereum",
		"normalized_chain": "ethereum",
		"metric_group":     "staking",
		"lookback":         "30d",
		"total":            30,
		"count":            2,
		"data_status":      "ok",
		"staking_metrics": map[string]interface{}{
			"series": []interface{}{
				map[string]interface{}{
					"date":                   "2026-06-03",
					"validator_active":       1000000,
					"total_value_staked_eth": "32000000",
					"staking_rate":           28.1,
					"eth_supply":             "120000000",
					"staking_apr_7d":         3.2,
					"entry_queue_eth":        "1024",
					"exit_queue_eth":         "512",
					"entry_wait_days":        5,
					"exit_wait_days":         2,
					"data_status":            "ok",
				},
				map[string]interface{}{
					"date":             "2026-06-02",
					"validator_active": 999000,
					"data_status":      "partial",
					"missing_fields":   []interface{}{"staking_apr_7d"},
				},
			},
		},
	})
	assert.Contains(t, out, "Chain Activity (Staking)")
	assert.Contains(t, out, "Latest Snapshot")
	assert.Contains(t, out, "eth_supply")
	assert.Contains(t, out, "staking_apr_7d")
	assert.Contains(t, out, "entry_wait_days")
	assert.Contains(t, out, "exit_wait_days")
	assert.Contains(t, out, "Recent Series")
	assert.Contains(t, out, "date=2026-06-03")
	assert.Contains(t, out, "date=2026-06-02")
}

func TestFormatChainActivityPretty_EmptySeries(t *testing.T) {
	t.Parallel()
	out := formatChainActivityPretty(map[string]interface{}{
		"metric_group": "staking",
		"staking_metrics": map[string]interface{}{
			"series": []interface{}{},
		},
	})
	assert.Contains(t, out, "No series points")
}

func TestRenderCallResult_PrettyChainActivity(t *testing.T) {
	t.Parallel()
	var out bytes.Buffer
	var errOut bytes.Buffer
	p := output.NewWithStderr(&out, &errOut, output.FormatPretty)

	err := RenderCallResult(p, "info", toolInfoPlatformmetricsGetChainActivity, &mcpclient.CallResult{
		StructuredContent: map[string]interface{}{
			"metric_group": "staking",
			"staking_metrics": map[string]interface{}{
				"series": []interface{}{
					map[string]interface{}{
						"date":             "2026-06-03",
						"validator_active": 1,
						"eth_supply":       "120",
					},
				},
			},
		},
	}, 0)
	require.NoError(t, err)
	body := out.String()
	assert.Contains(t, body, "Latest Snapshot")
	assert.Contains(t, body, "eth_supply")
	assert.NotContains(t, body, `"tool_name"`)
}
