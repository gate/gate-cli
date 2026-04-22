package intelfacade

import "testing"

func TestBaselineToolCount(t *testing.T) {
	if BaselineToolCount() != 38 {
		t.Fatalf("expected baseline 38, got %d", BaselineToolCount())
	}
}
