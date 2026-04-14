package news

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewsCommandStructure(t *testing.T) {
	subCmds := make(map[string]bool)
	for _, c := range Cmd.Commands() {
		subCmds[c.Name()] = true
	}
	assert.True(t, subCmds["list"], "missing news list subcommand")
	assert.True(t, subCmds["verify-schema"], "missing news verify-schema subcommand")
}
