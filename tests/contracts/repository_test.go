package contracts

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSiblingRepository(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "cli", "tests", "agentic", "fixture_test.go")
	_, err := SiblingRepository(source, "oauth-fixture", "go.mod")
	require.ErrorContains(t, err, "not found")
	write := func(dir string) {
		t.Helper()
		require.NoError(t, os.MkdirAll(dir, 0700))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module fixture\n"), 0600))
	}
	fallback := filepath.Join(root, ".worktrees", "oauth-fixture")
	write(fallback)
	got, err := SiblingRepository(source, "oauth-fixture", "go.mod")
	require.NoError(t, err)
	require.Equal(t, fallback, got)
	direct := filepath.Join(root, "oauth-fixture")
	write(direct)
	got, err = SiblingRepository(source, "oauth-fixture", "go.mod")
	require.NoError(t, err)
	require.Equal(t, direct, got)
	for _, name := range []string{"", ".", "..", "../other", "/absolute"} {
		_, err := SiblingRepository(source, name, "go.mod")
		require.Error(t, err)
	}
	for _, sentinel := range []string{"", ".", "..", "../secret", "/absolute"} {
		_, err := SiblingRepository(source, "oauth-fixture", sentinel)
		require.Error(t, err)
	}
	// A directory at the sentinel path is not proof of a usable checkout.
	_, err = SiblingRepository(source, "oauth-fixture", ".")
	require.Error(t, err)
}
