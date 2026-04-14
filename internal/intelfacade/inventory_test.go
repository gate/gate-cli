package intelfacade

import "testing"

func TestBaselineToolCount(t *testing.T) {
	if BaselineToolCount() != 42 {
		t.Fatalf("expected baseline 42, got %d", BaselineToolCount())
	}
}
