package mcl

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMclCommandStructure(t *testing.T) {
	subCmds := make(map[string]bool)
	for _, c := range Cmd.Commands() {
		subCmds[c.Name()] = true
	}

	expected := []string{
		"currencies", "quota", "ltv", "current-rate", "fix-rate",
		"orders", "order", "create", "repay", "repay-records",
		"records", "collateral",
	}
	for _, name := range expected {
		assert.True(t, subCmds[name], "missing subcommand: %s", name)
	}
}
