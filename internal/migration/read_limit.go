package migration

import (
	"fmt"
	"io"
)

// readFromReaderLimited reads at most maxBytes from r using a LimitReader(maxBytes+1) sentinel
// so callers can reject payloads larger than maxBytes (CR-211).
func readFromReaderLimited(r io.Reader, maxBytes int64) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(r, maxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxBytes {
		return nil, fmt.Errorf("content exceeds %d bytes", maxBytes)
	}
	return data, nil
}
