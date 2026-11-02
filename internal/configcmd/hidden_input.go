package configcmd

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/gate/gate-cli/internal/terminalio"
	"golang.org/x/term"
)

// readHiddenFields has one buffered reader and EOF/CRLF state for all fields;
// pasted input cannot echo or be lost between profile/key/secret prompts.
func readHiddenFields(ctx context.Context, in io.Reader, out io.Writer) ([3]string, error) {
	var fields [3]string
	reader := bufio.NewReader(in)
	afterCR, ended := false, false
	for i, prompt := range []string{"Profile name [default] (hidden): ", "API Key (hidden): ", "API Secret (hidden): "} {
		if _, err := fmt.Fprint(out, prompt); err != nil {
			return fields, err
		}
		max := 65536
		if i == 0 {
			max = 128
		}
		line := make([]byte, 0, 128)
		complete := false
		for !complete {
			if err := ctx.Err(); err != nil {
				return fields, err
			}
			if ended {
				return fields, fmt.Errorf("read configuration input: %w", io.EOF)
			}
			b, err := reader.ReadByte()
			if err != nil {
				if err != io.EOF {
					return fields, fmt.Errorf("read configuration input: %w", err)
				}
				ended = true
				if len(line) == 0 {
					return fields, fmt.Errorf("read configuration input: %w", io.EOF)
				}
				break
			}
			pairedLF := afterCR && b == '\n'
			afterCR = false
			if pairedLF {
				continue
			}
			switch b {
			case '\r':
				afterCR = true
				complete = true
			case '\n':
				complete = true
			case 3:
				return fields, context.Canceled
			case 4:
				ended = true
				if len(line) == 0 {
					return fields, fmt.Errorf("read configuration input: %w", io.EOF)
				}
				complete = true
			case 8, 127:
				if len(line) > 0 {
					_, size := utf8.DecodeLastRune(line)
					line = line[:len(line)-size]
				}
			default:
				if len(line) >= max {
					return fields, fmt.Errorf("configuration input exceeds %d bytes", max)
				}
				line = append(line, b)
			}
		}
		fields[i] = strings.TrimSpace(string(line))
	}
	return fields, nil
}

// readHiddenInput restores the terminal before returning values to the original
// config transaction. RunInit keeps its signal subscription until exit, after
// this final ctx check. SIGKILL cannot be recovered by this process.
func readHiddenInput(parent context.Context, file *os.File, out io.Writer) (fields [3]string, resultErr error) {
	ctx := parent
	state, err := term.MakeRaw(int(file.Fd()))
	if err != nil {
		return fields, fmt.Errorf("prepare hidden input: %w", err)
	}
	defer func() {
		resultErr = finishHiddenInput(ctx, func() error { return term.Restore(int(file.Fd()), state) }, resultErr)
	}()
	resultErr = terminalio.WithReader(ctx, file, func(reader io.Reader) error {
		var err error
		fields, err = readHiddenFields(ctx, reader, out)
		return err
	})
	return fields, resultErr
}

// finishHiddenInput keeps restore failure observable and evaluates cancellation
// before the original write phase. Its restore argument is a local test seam,
// not a mutable global terminal hook.
func finishHiddenInput(ctx context.Context, restore func() error, inputErr error) error {
	return errors.Join(inputErr, restore(), ctx.Err())
}
