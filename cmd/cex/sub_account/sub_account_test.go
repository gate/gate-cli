package subaccount

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSubAccountCommandStructure(t *testing.T) {
	// Verify root command
	assert.Equal(t, "sub-account", Cmd.Use)
	assert.NotEmpty(t, Cmd.Short)

	// Collect all sub-commands (including nested)
	subs := make(map[string]bool)
	for _, c := range Cmd.Commands() {
		subs[c.Use] = true
		for _, nested := range c.Commands() {
			subs[c.Use+"/"+nested.Use] = true
		}
	}

	// Top-level sub-commands
	for _, name := range []string{"list", "create", "get", "lock", "unlock", "unified-mode", "key"} {
		assert.True(t, subs[name], "missing sub-command: %s", name)
	}

	// Key sub-commands
	for _, name := range []string{"list", "create", "get", "update", "delete"} {
		assert.True(t, subs["key/"+name], "missing key sub-command: %s", name)
	}
}

func TestSubAccountRequiredFlags(t *testing.T) {
	tests := []struct {
		path []string // command path under Cmd
		flag string
	}{
		{[]string{"create"}, "login-name"},
		{[]string{"get"}, "user-id"},
		{[]string{"lock"}, "user-id"},
		{[]string{"unlock"}, "user-id"},
		{[]string{"key", "list"}, "user-id"},
		{[]string{"key", "create"}, "user-id"},
		{[]string{"key", "get"}, "user-id"},
		{[]string{"key", "get"}, "api-key"},
		{[]string{"key", "update"}, "user-id"},
		{[]string{"key", "update"}, "api-key"},
		{[]string{"key", "delete"}, "user-id"},
		{[]string{"key", "delete"}, "api-key"},
	}

	for _, tt := range tests {
		cmd := findSubCmd(t, Cmd, tt.path)
		require.NotNil(t, cmd, "command not found: %v", tt.path)

		f := cmd.Flags().Lookup(tt.flag)
		require.NotNil(t, f, "flag %q not found on %v", tt.flag, tt.path)

		annotations := f.Annotations
		_, required := annotations["cobra_annotation_bash_completion_one_required_flag"]
		assert.True(t, required, "flag %q on %v should be required", tt.flag, tt.path)
	}
}

func TestBuildPerms(t *testing.T) {
	perms := buildPerms([]string{"spot", " futures ", "wallet"})
	require.Len(t, perms, 3)
	assert.Equal(t, "spot", perms[0].Name)
	assert.Equal(t, "futures", perms[1].Name)
	assert.Equal(t, "wallet", perms[2].Name)
}

// findSubCmd traverses the command tree by path segments.
func findSubCmd(t *testing.T, root *cobra.Command, path []string) *cobra.Command {
	t.Helper()
	cur := root
	for _, name := range path {
		var found *cobra.Command
		for _, c := range cur.Commands() {
			if c.Use == name || c.Name() == name {
				found = c
				break
			}
		}
		if found == nil {
			return nil
		}
		cur = found
	}
	return cur
}
