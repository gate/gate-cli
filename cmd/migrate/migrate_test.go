package migrate

import "testing"

func TestMigrateFlags(t *testing.T) {
	for _, name := range []string{"dry-run", "apply", "yes", "provider", "backup-dir"} {
		if Cmd.Flags().Lookup(name) == nil {
			t.Fatalf("missing --%s", name)
		}
	}
}
