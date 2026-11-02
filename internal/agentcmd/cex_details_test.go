//go:build agent

package agentcmd

import (
	"bytes"
	"encoding/json"
	"github.com/spf13/cobra"
	"testing"
)

// Discovery must not execute a target or read flag values (which may contain
// credentials); unknown actions remain explicitly unknown.
func TestCEXCompatMetadata(t *testing.T) {
	root := &cobra.Command{Use: "gate-cli"}
	cex := &cobra.Command{Use: "cex"}
	leaf := &cobra.Command{Use: "read", RunE: func(*cobra.Command, []string) error { t.Fatal("executed target"); return nil }}
	root.PersistentFlags().String("api-secret", "synthetic-secret", "")
	leaf.Flags().String("pair", "", "")
	_ = leaf.MarkFlagRequired("pair")
	cex.AddCommand(leaf)
	root.AddCommand(cex)
	entries, err := cexDetails(root, "cex read")
	if err != nil || len(entries) != 1 {
		t.Fatalf("metadata %v %v", entries, err)
	}
	if entries[0].Effect != "unknown" || entries[0].ConstraintsComplete {
		t.Fatal("invented safety")
	}
	found := false
	for _, f := range entries[0].Flags {
		if f.Name == "pair" {
			found = f.Required
		}
	}
	if !found || leaf.Flags().Changed("pair") {
		t.Fatal("required/mutation")
	}
	if _, err := cexDetails(root, "cex"); err == nil {
		t.Fatal("accepted parent")
	}
	if _, err := cexDetails(root, "cex missing"); err == nil {
		t.Fatal("accepted unknown")
	}
}

// Required means the Cobra annotation only, not a claim that handler-level
// conditional requirements are complete. Unknown effects remain unknown.
func TestCEXCompatMetadataWhitelist(t *testing.T) {
	root := &cobra.Command{Use: "gate-cli"}
	cex := &cobra.Command{Use: "cex"}
	leaf := &cobra.Command{Use: "write", Run: func(*cobra.Command, []string) { t.Fatal("executed") }, Annotations: map[string]string{"gate.effect": "write"}}
	root.PersistentFlags().String("api-secret", "synthetic-secret", "")
	cex.AddCommand(leaf)
	root.AddCommand(cex)
	got, err := cexDetails(root, "")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(raw, []byte("synthetic-secret")) || bytes.Contains(raw, []byte(`"value"`)) || bytes.Contains(raw, []byte(`"default"`)) {
		t.Fatal("credential defaults exposed")
	}
	if got[0].Effect != "write" || got[0].ConstraintsComplete {
		t.Fatal("wrong constraints")
	}
}
