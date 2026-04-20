package square

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSquareCommandStructure(t *testing.T) {
	subCmds := make(map[string]bool)
	for _, c := range Cmd.Commands() {
		subCmds[c.Name()] = true
	}

	expected := []string{"ai-search", "live-replay"}
	for _, name := range expected {
		assert.True(t, subCmds[name], "missing subcommand: %s", name)
	}
}

func TestAiSearchFlags(t *testing.T) {
	for _, c := range Cmd.Commands() {
		if c.Name() == "ai-search" {
			assert.NotNil(t, c.Flag("keyword"))
			assert.NotNil(t, c.Flag("currency"))
			assert.NotNil(t, c.Flag("time-range"))
			assert.NotNil(t, c.Flag("sort"))
			assert.NotNil(t, c.Flag("limit"))
			assert.NotNil(t, c.Flag("page"))
			return
		}
	}
	t.Fatal("ai-search subcommand not found")
}

func TestLiveReplayFlags(t *testing.T) {
	for _, c := range Cmd.Commands() {
		if c.Name() == "live-replay" {
			assert.NotNil(t, c.Flag("tag"))
			assert.NotNil(t, c.Flag("coin"))
			assert.NotNil(t, c.Flag("sort"))
			assert.NotNil(t, c.Flag("limit"))
			return
		}
	}
	t.Fatal("live-replay subcommand not found")
}
