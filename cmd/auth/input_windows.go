//go:build windows

package auth

import (
	"context"
	"github.com/gate/gate-cli/internal/terminalio"
	"io"
	"os"
)

// Keep auth's scanner and private signature; only file cancellation is shared.
func (input *inputSession) readFileLine(ctx context.Context, file *os.File, max int) (string, error) {
	var line string
	err := terminalio.WithReader(ctx, file, func(reader io.Reader) error { var err error; line, err = input.scanLine(ctx, reader, max); return err })
	if err != nil {
		return "", err
	}
	return line, nil
}
