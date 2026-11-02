package configcmd

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
)

// Exercise the new parser with no terminal mock; the E2E suite owns real PTYs.
func TestCEXCompatHiddenParser(t *testing.T) {
	for _, tc := range []struct {
		in     string
		ok     bool
		secret string
	}{
		{"default\r\nkey\r\nsecret\r\n", true, "secret"},
		{"\n\n\n", true, ""}, {"p\nk\ns\x04", true, "s"},
		{"p\x04", false, ""}, {"p\nk\x04", false, ""}, {"p\nk\n\x03", false, ""},
		{"p\nk\n中文\x7fx\n", true, "中x"},
		{"p\nk\n" + strings.Repeat("x", 65536) + "\n", true, strings.Repeat("x", 65536)},
		{"p\nk\n" + strings.Repeat("x", 65537) + "\n", false, ""},
		{strings.Repeat("p", 129) + "\nk\ns\n", false, ""},
	} {
		var out bytes.Buffer
		v, err := readHiddenFields(context.Background(), strings.NewReader(tc.in), &out)
		if (err == nil) != tc.ok || (tc.ok && v[2] != tc.secret) {
			t.Fatalf("parser ok=%v err=%v", tc.ok, err)
		}
		if strings.Contains(out.String(), "secret") {
			t.Fatal("input echoed")
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := readHiddenFields(ctx, strings.NewReader("p\nk\ns\n"), &bytes.Buffer{}); err == nil {
		t.Fatal("cancel ignored")
	}
}

// A restoration error is a failed input phase, never permission to save values.
func TestCEXCompatHiddenRestoreFailure(t *testing.T) {
	failure := io.ErrClosedPipe
	if err := finishHiddenInput(context.Background(), func() error { return failure }, nil); !errors.Is(err, failure) {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	called := false
	err := finishHiddenInput(ctx, func() error { called = true; return failure }, context.DeadlineExceeded)
	if !called || !errors.Is(err, failure) || !errors.Is(err, context.Canceled) || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal("restore or failure context lost", err)
	}
}
