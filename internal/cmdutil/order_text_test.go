package cmdutil

import (
	"github.com/spf13/cobra"
	"strings"
	"testing"
)

// The new optional field must not tighten any existing ordinary-order input.
func TestCEXCompatOrderText(t *testing.T) {
	for _, v := range []string{"", "t-", "t-a_Z.1-", "t-" + strings.Repeat("x", 28), "x", "t-../x", "t-中文", "t-" + strings.Repeat("x", 29)} {
		c := &cobra.Command{}
		c.Flags().String("text", v, "")
		got, err := OrderText(c)
		valid := v == "" || v == "t-" || v == "t-a_Z.1-" || v == "t-"+strings.Repeat("x", 28)
		if (err == nil) != valid || (valid && got != v) {
			t.Errorf("%q got=%q err=%v", v, got, err)
		}
	}
	if v, err := OrderText(&cobra.Command{}); err != nil || v != "" {
		t.Fatal("old direct calls changed")
	}
}
