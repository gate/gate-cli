package info

import "testing"

func TestInfoMCPSpecCommandRegistered(t *testing.T) {
	if Cmd == nil {
		t.Fatal("Cmd is nil")
	}
	sub, _, err := Cmd.Find([]string{"mcp-spec"})
	if err != nil || sub == nil || sub.Name() != "mcp-spec" {
		t.Fatalf("mcp-spec subcommand: err=%v cmd=%v", err, sub)
	}
}
