package intelfacade

import "testing"

func TestBaselineToolCount(t *testing.T) {
	if BaselineToolCount() != 41 {
		t.Fatalf("expected baseline 41, got %d", BaselineToolCount())
	}
}
