package toolargs

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// Every spelling of an in-cwd file keeps the same arguments, including false.
func TestArgsFileCompatibility(t *testing.T) {
	root := t.TempDir()
	work := filepath.Join(root, "work")
	home := filepath.Join(work, "home")
	for _, dir := range []string{work, home} {
		if err := os.Mkdir(dir, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	t.Chdir(work)
	t.Setenv("HOME", home)
	for _, file := range []string{filepath.Join(work, "args.json"), filepath.Join(home, "args.json")} {
		if err := os.WriteFile(file, []byte(`{"query":"BTC","enabled":false}`), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	for _, file := range []string{"args.json", filepath.Join(work, "args.json"), filepath.Join(home, "args.json"), "~/args.json"} {
		t.Run(file, func(t *testing.T) {
			cmd := &cobra.Command{Use: "fixture"}
			cmd.Flags().String("args-file", file, "")
			got, err := MergeFromCommand(cmd, MergeOptions{})
			if err != nil || !reflect.DeepEqual(got, map[string]any{"query": "BTC", "enabled": false}) {
				t.Fatalf("explicit file capability changed: %v %v", got, err)
			}
		})
	}
}

// Explicit absolute/home selections must not authorize an external file. Use
// both valid JSON and malformed content to prove rejection precedes parsing.
func TestArgsFileRejectsExternalPaths(t *testing.T) {
	root := t.TempDir()
	work, home := filepath.Join(root, "work"), filepath.Join(root, "work-sibling")
	for _, dir := range []string{work, home} {
		if err := os.Mkdir(dir, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	t.Chdir(work)
	t.Setenv("HOME", home)
	for _, name := range []string{"args.json", "malformed.txt"} {
		content := `{"query":"external-fixture-marker"}`
		if name == "malformed.txt" {
			content = "external-fixture-marker: not JSON"
		}
		if err := os.WriteFile(filepath.Join(home, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		for _, file := range []string{filepath.Join(home, name), "~/" + name, filepath.Join("..", "work-sibling", name)} {
			t.Run(file, func(t *testing.T) {
				cmd := &cobra.Command{Use: "fixture"}
				cmd.Flags().String("args-file", file, "")
				got, err := MergeFromCommand(cmd, MergeOptions{})
				if err == nil || got != nil || !strings.Contains(err.Error(), "must stay within working directory") {
					t.Fatalf("external file must fail containment: args=%v err=%v", got, err)
				}
				if strings.Contains(err.Error(), "external-fixture-marker") {
					t.Fatal("external file contents leaked in error")
				}
			})
		}
	}
}

// Invalid sources must fail before argument delivery and must never disclose
// file contents. The byte bound is exercised on both sides of its boundary.
func TestArgsFileFailureBoundaries(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	for _, tc := range []struct {
		name, content string
		valid         bool
	}{
		{"exact-bound", `{"query":"` + strings.Repeat("a", maxArgsFileBytes-len(`{"query":""}`)) + `"}`, true},
		{"oversized", `{"query":"` + strings.Repeat("a", maxArgsFileBytes) + `"}`, false},
		{"malformed", "sensitive-fixture-marker: not JSON", false},
		{"array", `["sensitive-fixture-marker"]`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			file := filepath.Join(root, tc.name+".json")
			if err := os.WriteFile(file, []byte(tc.content), 0o600); err != nil {
				t.Fatal(err)
			}
			cmd := &cobra.Command{Use: "fixture"}
			cmd.Flags().String("args-file", file, "")
			_, err := MergeFromCommand(cmd, MergeOptions{})
			if (err == nil) != tc.valid {
				t.Fatalf("valid=%v err=%v", tc.valid, err)
			}
			if err != nil && strings.Contains(err.Error(), "sensitive-fixture-marker") {
				t.Fatal("file contents leaked in error")
			}
		})
	}
	for _, file := range []string{"missing.json", "../outside.json"} {
		cmd := &cobra.Command{Use: "fixture"}
		cmd.Flags().String("args-file", file, "")
		if _, err := MergeFromCommand(cmd, MergeOptions{}); err == nil {
			t.Errorf("invalid file %s accepted", file)
		}
	}
}

// The target, not the spelling of a symlink, determines the allowed boundary.
func TestArgsFileSymlinkContainment(t *testing.T) {
	root := t.TempDir()
	work := filepath.Join(root, "work")
	if err := os.Mkdir(work, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Chdir(work)
	t.Setenv("HOME", work)
	inside, outside := filepath.Join(work, "inside.json"), filepath.Join(root, "outside.json")
	for _, file := range []string{inside, outside} {
		if err := os.WriteFile(file, []byte(`{"query":"BTC"}`), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	for _, tc := range []struct {
		name, target string
		valid        bool
	}{
		{"inside-relative", "inside.json", true},
		{"inside-absolute", inside, true},
		{"outside-relative", "../outside.json", false},
		{"outside-absolute", outside, false},
		{"dangling", "missing.json", false},
		{"loop", "loop", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := os.Symlink(tc.target, filepath.Join(work, tc.name)); err != nil {
				t.Fatal(err)
			}
			for _, file := range []string{tc.name, filepath.Join(work, tc.name), "~/" + tc.name} {
				cmd := &cobra.Command{Use: "fixture"}
				cmd.Flags().String("args-file", file, "")
				got, err := MergeFromCommand(cmd, MergeOptions{})
				if (err == nil) != tc.valid {
					t.Fatalf("link %s valid=%v, got=%v err=%v", file, tc.valid, got, err)
				}
				if tc.valid && got["query"] != "BTC" {
					t.Fatalf("inside link changed arguments: %v", got)
				}
			}
		})
	}
	// The direct absolute target must be rejected as well as its linked forms.
	cmd := &cobra.Command{Use: "fixture"}
	cmd.Flags().String("args-file", outside, "")
	if got, err := MergeFromCommand(cmd, MergeOptions{}); err == nil || got != nil {
		t.Fatalf("explicit external file accepted: %v %v", got, err)
	}
}

// A linked cwd/home alias must resolve to the same real root; textual prefix
// comparisons would either reject valid aliases or accept sibling directories.
func TestArgsFileLinkedWorkingDirectory(t *testing.T) {
	root := t.TempDir()
	work, alias := filepath.Join(root, "work"), filepath.Join(root, "alias")
	if err := os.Mkdir(work, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(work, alias); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(work, "args.json"), []byte(`{"query":"BTC"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Chdir(alias)
	t.Setenv("HOME", alias)
	for _, file := range []string{"args.json", filepath.Join(alias, "args.json"), filepath.Join(work, "args.json"), "~/args.json"} {
		cmd := &cobra.Command{Use: "fixture"}
		cmd.Flags().String("args-file", file, "")
		if got, err := MergeFromCommand(cmd, MergeOptions{}); err != nil || got["query"] != "BTC" {
			t.Fatalf("valid cwd alias rejected: %s args=%v err=%v", file, got, err)
		}
	}
}
