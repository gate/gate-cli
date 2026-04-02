package activity

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestActivityCommandStructure(t *testing.T) {
	subCmds := make(map[string]bool)
	for _, c := range Cmd.Commands() {
		subCmds[c.Name()] = true
	}

	expected := []string{"get-entry", "list", "types"}
	for _, name := range expected {
		assert.True(t, subCmds[name], "missing subcommand: %s", name)
	}
}

func TestListFlags(t *testing.T) {
	var listCmd = Cmd.Commands()
	for _, c := range listCmd {
		if c.Name() == "list" {
			assert.NotNil(t, c.Flag("recommend-type"))
			assert.NotNil(t, c.Flag("type-ids"))
			assert.NotNil(t, c.Flag("keywords"))
			assert.NotNil(t, c.Flag("page"))
			assert.NotNil(t, c.Flag("page-size"))
			assert.NotNil(t, c.Flag("sort-by"))
			return
		}
	}
	t.Fatal("list subcommand not found")
}
