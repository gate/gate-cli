package doctor

import "testing"

func TestDoctorFlags(t *testing.T) {
	if Cmd.Flags().Lookup("check") == nil {
		t.Fatalf("missing --check")
	}
	if Cmd.Flags().Lookup("strict") == nil {
		t.Fatalf("missing --strict")
	}
}
