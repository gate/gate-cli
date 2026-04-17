package news

import "testing"

func TestNewsRefreshSchemaFlagRemoved(t *testing.T) {
	if Cmd.PersistentFlags().Lookup("refresh-schema") != nil {
		t.Fatal("refresh-schema flag should be removed; use GATE_INTEL_REFRESH_SCHEMA")
	}
}
