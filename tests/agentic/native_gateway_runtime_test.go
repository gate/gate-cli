//go:build agentic_e2e

package agentic_test

import (
	"strings"
	"sync"
	"testing"
)

// nativeDiagnostics bounds fixture output even if a worker repeatedly fails.
// It never receives production credentials; all dependencies are isolated peers.
type nativeDiagnostics struct {
	mu   sync.Mutex
	data []byte
}

func (b *nativeDiagnostics) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if room := 65536 - len(b.data); room > 0 {
		b.data = append(b.data, p[:min(room, len(p))]...)
	}
	return len(p), nil
}

func (b *nativeDiagnostics) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return string(b.data)
}

// The bounded sink must acknowledge all bytes so exec can always drain pipes.
func TestNativeDiagnosticsBound(t *testing.T) {
	var output nativeDiagnostics
	for _, size := range []int{0, 65535, 2, 70000} {
		n, err := output.Write([]byte(strings.Repeat("x", size)))
		if err != nil || n != size || len(output.String()) > 65536 {
			t.Fatalf("invalid bounded write: %d %v", n, err)
		}
	}
	if len(output.String()) != 65536 {
		t.Fatal("diagnostic bound lost")
	}
}
