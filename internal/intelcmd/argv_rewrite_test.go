package intelcmd

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gate/gate-cli/internal/toolschema"
)

// flexBoolStub mirrors the toolschema.flexBool pflag.Value surface for argv rewriting
// tests. It reuses toolschema.FlexBoolTypeName so the stub stays compile-time bound to
// the production constant — if the real flexBool ever renames its Type(), this test
// breaks loudly instead of silently passing while the rewriter ignores production flags.
type flexBoolStub struct{ v bool }

func (b *flexBoolStub) Set(s string) error { b.v = s == "true" || s == "1" || s == "t"; return nil }
func (b *flexBoolStub) String() string {
	if b.v {
		return "true"
	}
	return "false"
}
func (b *flexBoolStub) Type() string { return toolschema.FlexBoolTypeName }

func newLeafWithFlexBool() *cobra.Command {
	root := &cobra.Command{Use: "root"}
	sub := &cobra.Command{Use: "sub", Run: func(*cobra.Command, []string) {}}
	fb := &flexBoolStub{}
	sub.Flags().Var(fb, "with-indicators", "boolean flag")
	sub.Flags().Lookup("with-indicators").NoOptDefVal = "true"
	sub.Flags().Int("limit", 0, "int flag")
	sub.Flags().String("symbol", "", "string flag")
	root.AddCommand(sub)
	return root
}

func TestRewriteFlexBoolSpaceArgs_CollapsesSpacedTrue(t *testing.T) {
	root := newLeafWithFlexBool()
	in := []string{"sub", "--with-indicators", "true", "--limit", "5"}
	out, rewritten := RewriteFlexBoolSpaceArgs(root, in)
	assert.True(t, rewritten)
	assert.Equal(t, []string{"sub", "--with-indicators=true", "--limit", "5"}, out)
}

func TestRewriteFlexBoolSpaceArgs_CollapsesSpacedFalse(t *testing.T) {
	root := newLeafWithFlexBool()
	in := []string{"sub", "--with-indicators", "False", "--symbol", "ETH"}
	out, rewritten := RewriteFlexBoolSpaceArgs(root, in)
	assert.True(t, rewritten)
	assert.Equal(t, []string{"sub", "--with-indicators=False", "--symbol", "ETH"}, out)
}

func TestRewriteFlexBoolSpaceArgs_BareFlagFollowedByAnotherFlagUntouched(t *testing.T) {
	root := newLeafWithFlexBool()
	in := []string{"sub", "--with-indicators", "--limit", "5"}
	out, rewritten := RewriteFlexBoolSpaceArgs(root, in)
	assert.False(t, rewritten)
	assert.Equal(t, in, out)
}

func TestRewriteFlexBoolSpaceArgs_EqualsFormUntouched(t *testing.T) {
	root := newLeafWithFlexBool()
	in := []string{"sub", "--with-indicators=false", "--limit", "5"}
	out, rewritten := RewriteFlexBoolSpaceArgs(root, in)
	assert.False(t, rewritten)
	assert.Equal(t, in, out)
}

func TestRewriteFlexBoolSpaceArgs_NonBoolNextTokenUntouched(t *testing.T) {
	root := newLeafWithFlexBool()
	in := []string{"sub", "--with-indicators", "maybe"}
	out, rewritten := RewriteFlexBoolSpaceArgs(root, in)
	assert.False(t, rewritten)
	assert.Equal(t, in, out)
}

func TestRewriteFlexBoolSpaceArgs_OnlyRewritesFlexBoolNames(t *testing.T) {
	root := newLeafWithFlexBool()
	// "--symbol" is a string flag; its argv must stay as separate tokens even when the
	// next token happens to look like a boolean literal.
	in := []string{"sub", "--symbol", "true"}
	out, rewritten := RewriteFlexBoolSpaceArgs(root, in)
	assert.False(t, rewritten)
	assert.Equal(t, in, out)
}

func TestRewriteFlexBoolSpaceArgs_DashDashTerminatorRespected(t *testing.T) {
	root := newLeafWithFlexBool()
	in := []string{"sub", "--", "--with-indicators", "true"}
	out, rewritten := RewriteFlexBoolSpaceArgs(root, in)
	assert.False(t, rewritten)
	assert.Equal(t, in, out)
}

func TestRewriteFlexBoolSpaceArgs_NoLeafReturnsArgsUnchanged(t *testing.T) {
	root := newLeafWithFlexBool()
	// Unknown subcommand: cobra.Find returns the root itself; root has no flexBool flags,
	// so the rewriter must short-circuit without touching args.
	in := []string{"missing-subcommand", "--with-indicators", "true"}
	out, rewritten := RewriteFlexBoolSpaceArgs(root, in)
	assert.False(t, rewritten)
	assert.Equal(t, in, out)
}

func TestRewriteFlexBoolSpaceArgs_NilRootSafe(t *testing.T) {
	out, rewritten := RewriteFlexBoolSpaceArgs(nil, []string{"sub"})
	assert.False(t, rewritten)
	assert.Equal(t, []string{"sub"}, out)
}

func TestRewriteFlexBoolSpaceArgs_EmptyArgs(t *testing.T) {
	root := newLeafWithFlexBool()
	out, rewritten := RewriteFlexBoolSpaceArgs(root, nil)
	assert.False(t, rewritten)
	assert.Nil(t, out)
}

func TestRewriteFlexBoolSpaceArgs_TrailingBoolFlagWithoutValue(t *testing.T) {
	root := newLeafWithFlexBool()
	in := []string{"sub", "--with-indicators"}
	out, rewritten := RewriteFlexBoolSpaceArgs(root, in)
	assert.False(t, rewritten)
	assert.Equal(t, in, out)
}

func TestRewriteFlexBoolSpaceArgs_TwoFlexBoolsInARow(t *testing.T) {
	root := &cobra.Command{Use: "root"}
	sub := &cobra.Command{Use: "sub", Run: func(*cobra.Command, []string) {}}
	a := &flexBoolStub{}
	b := &flexBoolStub{}
	sub.Flags().Var(a, "with-a", "")
	sub.Flags().Var(b, "with-b", "")
	sub.Flags().Lookup("with-a").NoOptDefVal = "true"
	sub.Flags().Lookup("with-b").NoOptDefVal = "true"
	root.AddCommand(sub)

	in := []string{"sub", "--with-a", "true", "--with-b", "false"}
	out, rewritten := RewriteFlexBoolSpaceArgs(root, in)
	assert.True(t, rewritten)
	assert.Equal(t, []string{"sub", "--with-a=true", "--with-b=false"}, out)
}

func TestRewriteFlexBoolSpaceArgs_DoesNotTouchNativeBoolFlags(t *testing.T) {
	root := &cobra.Command{Use: "root"}
	sub := &cobra.Command{Use: "sub", Run: func(*cobra.Command, []string) {}}
	// Native pflag bool also carries NoOptDefVal="true" but Type() == "bool", not "flexBool";
	// the rewriter must preserve the historical pflag behavior for non-flexBool boolean flags
	// to avoid accidentally widening its scope.
	sub.Flags().Bool("debug", false, "")
	root.AddCommand(sub)

	in := []string{"sub", "--debug", "true"}
	out, rewritten := RewriteFlexBoolSpaceArgs(root, in)
	assert.False(t, rewritten)
	assert.Equal(t, in, out)
}

// TestRewriteFlexBoolSpaceArgs_EndToEndParse verifies that after rewriting, cobra parses
// the spaced form to true and still binds the next flag value to its real flag.
func TestRewriteFlexBoolSpaceArgs_EndToEndParse(t *testing.T) {
	root := newLeafWithFlexBool()
	in := []string{"sub", "--with-indicators", "true", "--limit", "5"}
	out, rewritten := RewriteFlexBoolSpaceArgs(root, in)
	require.True(t, rewritten)

	leaf, _, err := root.Find(out)
	require.NoError(t, err)
	require.NoError(t, leaf.ParseFlags(out[1:]))
	assert.Equal(t, "true", leaf.Flags().Lookup("with-indicators").Value.String())
	lim, err := leaf.Flags().GetInt("limit")
	require.NoError(t, err)
	assert.Equal(t, 5, lim)
}
