package info

import "testing"

func TestInfoRefreshSchemaFlagRemoved(t *testing.T) {
	if Cmd.PersistentFlags().Lookup("refresh-schema") != nil {
		t.Fatal("refresh-schema flag should be removed; use GATE_INTEL_REFRESH_SCHEMA")
	}
}
