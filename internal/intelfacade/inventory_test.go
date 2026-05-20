package intelfacade

import "testing"

func TestBaselineToolCount(t *testing.T) {
	if BaselineToolCount() != 44 {
		t.Fatalf("expected baseline 44, got %d", BaselineToolCount())
	}
}
