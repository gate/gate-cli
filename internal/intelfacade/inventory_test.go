package intelfacade

import "testing"

func TestBaselineToolCount(t *testing.T) {
	if BaselineToolCount() != 37 {
		t.Fatalf("expected baseline 37, got %d", BaselineToolCount())
	}
}
