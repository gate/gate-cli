package withdrawal

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithdrawalCommandStructure(t *testing.T) {
	assert.Equal(t, "withdrawal", Cmd.Use)
	assert.NotEmpty(t, Cmd.Short)

	subs := make(map[string]bool)
	for _, c := range Cmd.Commands() {
		subs[c.Use] = true
	}

	for _, name := range []string{"create", "push-order", "cancel"} {
		assert.True(t, subs[name], "missing sub-command: %s", name)
	}
}

func TestWithdrawalRequiredFlags(t *testing.T) {
	tests := []struct {
		cmdName string
		flag    string
	}{
		{"create", "currency"},
		{"create", "amount"},
		{"create", "address"},
		{"push-order", "receive-uid"},
		{"push-order", "currency"},
		{"push-order", "amount"},
		{"cancel", "id"},
	}

	for _, tt := range tests {
		var cmd *cobra.Command
		for _, c := range Cmd.Commands() {
			if c.Name() == tt.cmdName {
				cmd = c
				break
			}
		}
		require.NotNil(t, cmd, "command %q not found", tt.cmdName)

		f := cmd.Flags().Lookup(tt.flag)
		require.NotNil(t, f, "flag %q not found on %q", tt.flag, tt.cmdName)

		annotations := f.Annotations
		_, required := annotations["cobra_annotation_bash_completion_one_required_flag"]
		assert.True(t, required, "flag %q on %q should be required", tt.flag, tt.cmdName)
	}
}
