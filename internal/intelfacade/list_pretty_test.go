package intelfacade

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestListCapabilitiesPrettyText_Empty(t *testing.T) {
	out := ListCapabilitiesPrettyText(nil)
	assert.Contains(t, out, "Capabilities")
	assert.Contains(t, strings.ToLower(out), "no entries")
}

func TestListCapabilitiesPrettyText_Rows(t *testing.T) {
	out := ListCapabilitiesPrettyText([]ToolSummary{
		{Name: "info_coin_get_coin_info", Description: "Coin data.", HasInputSchema: true},
		{Name: "info_marketsnapshot_get_market_snapshot", HasInputSchema: false},
	})
	assert.Contains(t, out, "info_coin_get_coin_info")
	assert.Contains(t, out, "Coin data.")
	assert.Contains(t, out, "Accepts parameters: yes")
	assert.Contains(t, out, "Accepts parameters: no")
}
