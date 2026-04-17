package news

import "testing"

func TestNewsHasRefreshSchemaFlag(t *testing.T) {
	if Cmd.PersistentFlags().Lookup("refresh-schema") == nil {
		t.Fatal("missing --refresh-schema on news command")
	}
}
