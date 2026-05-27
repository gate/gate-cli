package intelfacade

import "testing"

func TestBaselineToolCount(t *testing.T) {
	if BaselineToolCount() != 45 {
		t.Fatalf("expected baseline 45, got %d", BaselineToolCount())
	}
}
