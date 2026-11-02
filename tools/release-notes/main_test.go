package main

import (
	"bytes"
	"errors"
	"flag"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Extraction must isolate one public version even when neighboring material or code contains headings.
func TestExtract(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, changelog, version, want string
	}{
		{"exact", "# Changelog\n\n## [v1.2.30]\n- other\n## [v1.2.3] - 2026-11-02\n\n### Added\n\n- public\n\n## [v1.2.2]\n- older\n", "v1.2.3", "### Added\n\n- public\n"},
		{"indented body", "## [v1.2.3]\n \n    code\n- public\n \n", "v1.2.3", "    code\n- public\n"},
		{"legacy", "## [1.2.3]\n- public\n", "v1.2.3", "- public\n"},
		{"no tag prefix", "## [v1.2.3]\n- public\n", "1.2.3", "- public\n"},
		{"prerelease", "## [v1.2.3-rc.1]\n- preview\n## [v1.2.3]\n- stable\n", "v1.2.3-rc.1", "- preview\n"},
		{"crlf", "## [v1.2.3]\r\n\r\n- public\r\n", "v1.2.3", "- public\n"},
		{"fenced headings", "```md\n## [v1.2.3]\nnot a version\n```\n## [v1.2.3]\n```md\n## [v9.9.9]\nexample\n```\n- public\n## Internal\nprivate\n", "v1.2.3", "```md\n## [v9.9.9]\nexample\n```\n- public\n"},
		{"tilde fence", "## [v1.2.3]\n~~~text\n## [v1.2.3]\n~~~\n- public\n", "v1.2.3", "~~~text\n## [v1.2.3]\n~~~\n- public\n"},
		{"long fence", "## [v1.2.3]\n````md\n```\n## example\n````\n- public\n", "v1.2.3", "````md\n```\n## example\n````\n- public\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := extract(strings.NewReader(tc.changelog), tc.version)
			if err != nil || got != tc.want {
				t.Fatalf("extract() = %q, %v; want %q", got, err, tc.want)
			}
		})
	}
}

// Invalid input must return no notes; there is no first-section or Git-history fallback.
func TestExtractRejects(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ name, changelog, version string }{
		{"missing", "## [v1.2.30]\n- public\n", "v1.2.3"},
		{"empty", "## [v1.2.3]\n\n## [v1.2.2]\n- older\n", "v1.2.3"},
		{"duplicate", "## [v1.2.3]\n- one\n## [1.2.3]\n- two\n", "v1.2.3"},
		{"no version", "## [v1.2.3]\n- public\n", ""},
		{"invalid version", "## [latest]\n- public\n", "latest"},
		{"whitespace version", "## [v1.2.3]\n- public\n", " v1.2.3"},
		{"unclosed fence", "## [v1.2.3]\n```\n## [v1.2.2]\n- older\n", "v1.2.3"},
		{"too large", "## [v1.2.3]\n" + strings.Repeat("x", 1<<20), "v1.2.3"},
		{"invalid utf8", "## [v1.2.3]\n\xff", "v1.2.3"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := extract(strings.NewReader(tc.changelog), tc.version)
			if err == nil || got != "" {
				t.Fatalf("extract() = %q, %v; want no output and an error", got, err)
			}
		})
	}
	if got, err := extract(failedReader{}, "v1.2.3"); err == nil || got != "" {
		t.Fatalf("read failure returned %q, %v", got, err)
	}
}

type failedReader struct{}

func (failedReader) Read([]byte) (int, error) { return 0, errors.New("read failed") }

// Output errors are propagated so a broken pipe cannot look like a successful extraction.
func TestRunFailures(t *testing.T) {
	for _, args := range [][]string{nil, {"--unknown"}, {"--version"}, {"--version", "v1.2.3", "extra"}, {"--version", "v1.2.3", "--changelog", "missing-changelog"}} {
		var stdout, stderr bytes.Buffer
		if err := run(args, &stdout, &stderr); err == nil || stdout.Len() != 0 {
			t.Fatalf("run(%q) = %v, stdout %q", args, err, stdout.String())
		}
	}
	if err := writeNotes(failedWriter{}, "- public\n"); err == nil {
		t.Fatal("write failure ignored")
	}
}

type failedWriter struct{}

func (failedWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

// The reader's inclusive bound must not reject a valid changelog at exactly 1 MiB.
func TestExactSizeLimit(t *testing.T) {
	t.Parallel()
	header := "## [v1.2.3]\n"
	body := strings.Repeat("x", (1<<20)-len(header))
	got, err := extract(strings.NewReader(header+body), "v1.2.3")
	if err != nil || got != body+"\n" {
		t.Fatalf("valid limit rejected: %v", err)
	}
}

func TestRunSuccessAndHelp(t *testing.T) {
	path := filepath.Join(t.TempDir(), "CHANGELOG.md")
	if err := os.WriteFile(path, []byte("## [v1.2.3]\n- public\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if err := run([]string{"--version", "v1.2.3", "--changelog", path}, &stdout, &stderr); err != nil || stdout.String() != "- public\n" || stderr.Len() != 0 {
		t.Fatalf("run() = %v, stdout=%q, stderr=%q", err, stdout.String(), stderr.String())
	}
	stdout.Reset()
	if err := os.WriteFile(path, []byte("## [v9.9.9]\n- other version\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"--version", "v1.2.3", "--changelog", path}, &stdout, &stderr); err == nil || stdout.Len() != 0 {
		t.Fatalf("invalid selection = %v, stdout=%q", err, stdout.String())
	}
	if err := os.WriteFile(path, []byte("## [v1.2.3]\n- public\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"-h"}, &stdout, &stderr); !errors.Is(err, flag.ErrHelp) || stdout.Len() != 0 {
		t.Fatalf("help = %v, stdout=%q", err, stdout.String())
	}
	if err := run([]string{"--version", "v1.2.3", "--changelog", path}, failedWriter{}, &stderr); err == nil {
		t.Fatal("run ignored an output failure")
	}
	if err := writeNotes(shortWriter{}, "- public\n"); !errors.Is(err, io.ErrShortWrite) {
		t.Fatalf("short write = %v", err)
	}
}

type shortWriter struct{}

func (shortWriter) Write(p []byte) (int, error) { return len(p) - 1, nil }
