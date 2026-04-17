package migration

import (
	"strings"
	"testing"
)

func TestReadFromReaderLimited(t *testing.T) {
	t.Parallel()
	data, err := readFromReaderLimited(strings.NewReader("hello"), 10)
	if err != nil || string(data) != "hello" {
		t.Fatalf("got %q err=%v", data, err)
	}
	_, err = readFromReaderLimited(strings.NewReader("12345678901"), 10)
	if err == nil {
		t.Fatal("expected over-limit error")
	}
}
