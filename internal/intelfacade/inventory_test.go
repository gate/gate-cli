package intelfacade

import "testing"

func TestBaselineToolCount(t *testing.T) {
	if BaselineToolCount() != 40 {
		t.Fatalf("expected baseline 40, got %d", BaselineToolCount())
	}
}
