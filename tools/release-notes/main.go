// Command release-notes extracts the public notes for one release tag without consulting Git history.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"unicode/utf8"
)

const maxChangelogBytes = 1 << 20

var (
	versionPattern = regexp.MustCompile(`^v?[0-9]+\.[0-9]+\.[0-9]+(?:-[0-9A-Za-z]+(?:[.-][0-9A-Za-z]+)*)?(?:\+[0-9A-Za-z]+(?:[.-][0-9A-Za-z]+)*)?$`)
	versionHeading = regexp.MustCompile(`^##[ \t]+\[([^\]]+)\](?:[ \t]+.*)?$`)
)

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return
		}
		fmt.Fprintln(os.Stderr, "release notes:", err)
		os.Exit(1)
	}
}

// run emits nothing on stdout until the entire version selection has been validated.
func run(args []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("release-notes", flag.ContinueOnError)
	flags.SetOutput(stderr)
	version := flags.String("version", "", "Release tag to match (required)")
	path := flags.String("changelog", "CHANGELOG.md", "Public changelog file")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("unexpected positional arguments")
	}
	if len(*version) > 128 || !versionPattern.MatchString(*version) {
		return errors.New("version must be an explicit release tag such as v1.2.3")
	}
	file, err := os.Open(*path)
	if err != nil {
		return fmt.Errorf("open changelog: %w", err)
	}
	notes, parseErr := extract(file, *version)
	if err := errors.Join(parseErr, file.Close()); err != nil {
		return fmt.Errorf("read changelog: %w", err)
	}
	return writeNotes(stdout, notes)
}

func writeNotes(w io.Writer, notes string) error {
	n, err := io.WriteString(w, notes)
	if err != nil {
		return fmt.Errorf("write release notes: %w", err)
	}
	if n != len(notes) {
		return io.ErrShortWrite
	}
	return nil
}

// extract accepts both historical [1.2.3] and [v1.2.3] headings, with exact version equality.
// Missing, duplicate, empty or malformed sections fail closed rather than publishing other material.
func extract(r io.Reader, version string) (string, error) {
	if len(version) > 128 || !versionPattern.MatchString(version) {
		return "", errors.New("invalid release version")
	}
	data, err := io.ReadAll(io.LimitReader(r, maxChangelogBytes+1))
	if err != nil {
		return "", err
	}
	if len(data) > maxChangelogBytes {
		return "", errors.New("changelog exceeds 1 MiB")
	}
	if !utf8.Valid(data) {
		return "", errors.New("changelog is not valid UTF-8")
	}
	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	wanted := strings.TrimPrefix(version, "v")
	start, end, matches := -1, len(lines), 0
	var fence byte
	var fenceLength int
	for i, line := range lines {
		trimmed := strings.TrimLeft(line, " ")
		// Only Markdown fences/headings indented at most three spaces are structural.
		if len(line)-len(trimmed) > 3 {
			continue
		}
		if fence != 0 {
			n := prefixLength(trimmed, fence)
			if n >= fenceLength && strings.TrimSpace(trimmed[n:]) == "" {
				fence, fenceLength = 0, 0
			}
			continue
		}
		if len(trimmed) > 0 && (trimmed[0] == '`' || trimmed[0] == '~') {
			if n := prefixLength(trimmed, trimmed[0]); n >= 3 {
				fence, fenceLength = trimmed[0], n
				continue
			}
		}
		if !strings.HasPrefix(trimmed, "## ") && !strings.HasPrefix(trimmed, "##\t") {
			continue
		}
		// Every second-level heading ends the selected section, not only version headings.
		if start >= 0 && end == len(lines) {
			end = i
		}
		match := versionHeading.FindStringSubmatch(trimmed)
		if len(match) == 2 && strings.TrimPrefix(match[1], "v") == wanted {
			matches++
			if matches == 1 {
				start = i + 1
			}
		}
	}
	if fence != 0 {
		return "", errors.New("unclosed Markdown fence in changelog")
	}
	if matches != 1 {
		return "", fmt.Errorf("expected one section for %s, found %d", version, matches)
	}
	// Trim blank boundary lines without changing indentation or Markdown inside the body.
	for start < end && strings.TrimSpace(lines[start]) == "" {
		start++
	}
	for end > start && strings.TrimSpace(lines[end-1]) == "" {
		end--
	}
	notes := strings.Join(lines[start:end], "\n")
	if strings.TrimSpace(notes) == "" {
		return "", fmt.Errorf("section for %s is empty", version)
	}
	return notes + "\n", nil
}

func prefixLength(s string, marker byte) int {
	n := 0
	for n < len(s) && s[n] == marker {
		n++
	}
	return n
}
