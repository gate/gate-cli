package earn

import (
	"testing"
)

func TestEarnSubcommands(t *testing.T) {
	want := map[string]bool{
		"dual":        false,
		"staking":     false,
		"fixed":       false,
		"auto-invest": false,
		"uni":         false,
	}

	for _, sub := range Cmd.Commands() {
		name := sub.Name()
		if _, ok := want[name]; ok {
			want[name] = true
		} else {
			t.Errorf("unexpected subcommand registered: %s", name)
		}
	}

	for name, found := range want {
		if !found {
			t.Errorf("expected subcommand %q not registered on Cmd", name)
		}
	}
}
