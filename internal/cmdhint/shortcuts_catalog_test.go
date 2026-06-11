package cmdhint

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInfoNewsShortcutPathsCount(t *testing.T) {
	t.Parallel()
	require.Len(t, InfoNewsShortcutPaths, 10)
	require.Len(t, DeferredInfoShortcutPaths, 1)
	require.Equal(t, "info +address-risk", DeferredInfoShortcutPaths[0])
}
