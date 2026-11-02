package terminalio

import (
	"context"
	"errors"
	"io"
	"os"
	"testing"
	"time"
)

// Cancellation must join the file reader without closing its caller-owned fd.
func TestCEXCompatCancelableReader(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		for _, file := range []*os.File{r, w} {
			if err := file.Close(); err != nil && !errors.Is(err, os.ErrClosed) {
				t.Error(err)
			}
		}
	})
	ctx, cancel := context.WithCancel(context.Background())
	started := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- WithReader(ctx, r, func(reader io.Reader) error { close(started); _, err := reader.Read(make([]byte, 1)); return err })
	}()
	<-started
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("reader did not join")
	}
	if _, err := r.Stat(); err != nil {
		t.Fatal("reader closed caller descriptor")
	}
}

func TestCEXCompatFileReadAndClosed(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		for _, file := range []*os.File{r, w} {
			if err := file.Close(); err != nil && !errors.Is(err, os.ErrClosed) {
				t.Error(err)
			}
		}
	})
	if _, err = w.Write([]byte("x")); err != nil {
		t.Fatal(err)
	}
	err = WithReader(context.Background(), r, func(reader io.Reader) error {
		var b [1]byte
		_, err := io.ReadFull(reader, b[:])
		if b[0] != 'x' {
			t.Error("byte lost")
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if err = r.Close(); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err = WithReader(ctx, r, func(reader io.Reader) error { _, err := reader.Read(make([]byte, 1)); return err }); err == nil || errors.Is(err, context.DeadlineExceeded) {
		t.Fatal("closed fd did not fail immediately", err)
	}
}
