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
	for _, c := range Cmd.Commands() {
		if c.Name() == "invoke" {
			assert.True(t, c.Hidden, "news invoke should be hidden from user help")
			assert.Contains(t, c.Aliases, "call")
			return
		}
	}
	t.Fatal("missing news invoke subcommand (hidden)")
}
