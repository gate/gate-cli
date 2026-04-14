package info

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInfoCommandStructure(t *testing.T) {
	subCmds := make(map[string]bool)
	for _, c := range Cmd.Commands() {
		subCmds[c.Name()] = true
	}
	assert.True(t, subCmds["list"], "missing info list subcommand")
	assert.True(t, subCmds["verify-schema"], "missing info verify-schema subcommand")
}
