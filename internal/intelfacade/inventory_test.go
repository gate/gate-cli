package intelfacade

import "testing"

func TestBaselineToolCount(t *testing.T) {
	if BaselineToolCount() != 46 {
		t.Fatalf("expected baseline 46, got %d", BaselineToolCount())
	}
}
