package coupon

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCouponCommandStructure(t *testing.T) {
	subCmds := make(map[string]bool)
	for _, c := range Cmd.Commands() {
		subCmds[c.Name()] = true
	}

	expected := []string{"list", "detail"}
	for _, name := range expected {
		assert.True(t, subCmds[name], "missing subcommand: %s", name)
	}
}

func TestDetailRequiredFlags(t *testing.T) {
	for _, c := range Cmd.Commands() {
		if c.Name() == "detail" {
			f := c.Flag("coupon-type")
			require.NotNil(t, f, "detail should have --coupon-type flag")

			f = c.Flag("detail-id")
			require.NotNil(t, f, "detail should have --detail-id flag")
			return
		}
	}
	t.Fatal("detail subcommand not found")
}
