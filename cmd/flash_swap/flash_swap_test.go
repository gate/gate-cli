package flashswap

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFlashSwapCommandStructure(t *testing.T) {
	require.Equal(t, "flash-swap", Cmd.Use)

	expectedSubs := []string{
		"pairs", "orders", "order", "preview", "create",
		"preview-v1", "create-v1",
		"preview-many-to-one", "create-many-to-one",
		"preview-one-to-many", "create-one-to-many",
	}

	cmds := Cmd.Commands()
	names := make(map[string]bool, len(cmds))
	for _, c := range cmds {
		names[c.Use] = true
	}

	for _, name := range expectedSubs {
		assert.True(t, names[name], "missing subcommand: %s", name)
	}
	assert.Len(t, cmds, len(expectedSubs), "unexpected number of subcommands")
}
