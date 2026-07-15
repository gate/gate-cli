package intelfacade

import "testing"

func TestBaselineToolCount(t *testing.T) {
	if BaselineToolCount() != 50 {
		t.Fatalf("expected baseline 50, got %d", BaselineToolCount())
	}
}
