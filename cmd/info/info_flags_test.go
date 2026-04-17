package info

import "testing"

func TestInfoHasRefreshSchemaFlag(t *testing.T) {
	if Cmd.PersistentFlags().Lookup("refresh-schema") == nil {
		t.Fatal("missing --refresh-schema on info command")
	}
}
