//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris

// Package terminalio owns cancelable file input without closing shared stdin.
package terminalio

import (
	"context"
	"golang.org/x/sys/unix"
	"io"
	"os"
)

// Read preserves the existing auth poll/read contract. The session owns the
// reader; no other consumer may race a read on the same terminal descriptor.
func Read(ctx context.Context, file *os.File, p []byte) (int, error) {
	for {
		if err := ctx.Err(); err != nil {
			return 0, err
		}
		// poll ignores a negative fd instead of reporting POLLNVAL. os.File.Fd
		// returns all-bits-one after Close; reject it before an endless poll loop.
		fd := int32(file.Fd())
		if fd < 0 {
			return 0, os.ErrClosed
		}
		fds := []unix.PollFd{{Fd: fd, Events: unix.POLLIN}}
		n, err := unix.Poll(fds, 25)
		if err == unix.EINTR {
			continue
		}
		if err != nil {
			return 0, err
		}
		if n > 0 {
			if fds[0].Revents&unix.POLLNVAL != 0 {
				return 0, os.ErrClosed
			}
			return file.Read(p)
		}
	}
}

type reader struct {
	ctx  context.Context
	file *os.File
}

func (r reader) Read(p []byte) (int, error) { return Read(r.ctx, r.file, p) }

// WithReader completes consumption synchronously before the caller restores raw
// terminal state. Cancellation doesn't close the host's descriptor.
func WithReader(ctx context.Context, file *os.File, consume func(io.Reader) error) error {
	return consume(reader{ctx, file})
}
