package welfare

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWelfareCommandStructure(t *testing.T) {
	subCmds := make(map[string]bool)
	for _, c := range Cmd.Commands() {
		subCmds[c.Name()] = true
	}

	expected := []string{"identity", "beginner-tasks"}
	for _, name := range expected {
		assert.True(t, subCmds[name], "missing subcommand: %s", name)
	}
}
