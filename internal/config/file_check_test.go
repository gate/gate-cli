package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// Diagnostics read file metadata and readability only; even malformed YAML must
// never be parsed or repaired by this explicit command.
func TestCEXCompatFileCheck(t *testing.T) {
	root := t.TempDir()
	p := filepath.Join(root, "config.yaml")
	if d := CheckFile(p); d.Status != "fail" {
		t.Fatal(d)
	}
	if err := os.WriteFile(p, []byte("broken: [synthetic-secret"), 0600); err != nil {
		t.Fatal(err)
	}
	want := "pass"
	if runtime.GOOS == "windows" {
		want = "warn"
	}
	if d := CheckFile(p); d.Status != want || d.Scope != "file" {
		t.Fatal(d)
	}
	if err := os.Chmod(p, 0644); err != nil {
		t.Fatal(err)
	}
	if d := CheckFile(p); d.Status != "warn" {
		t.Fatal(d)
	}
	if d := CheckFile(root); d.Status != "fail" {
		t.Fatal(d)
	}
	if runtime.GOOS != "windows" {
		link := filepath.Join(root, "link")
		if err := os.Symlink(p, link); err != nil {
			t.Fatal(err)
		}
		if d := CheckFile(link); d.Status != "warn" {
			t.Fatal(d)
		}
		if err := os.Chmod(p, 0000); err != nil {
			t.Fatal(err)
		}
		if os.Geteuid() != 0 {
			if d := CheckFile(p); d.Status != "fail" {
				t.Fatal(d)
			}
		}
	}
}
