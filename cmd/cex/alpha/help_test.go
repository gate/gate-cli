package alpha

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// Keep every Alpha currency flag aligned without changing optional/required semantics.
func TestCurrencyIDHelp(t *testing.T) {
	for _, tc := range []struct {
		path     []string
		required bool
	}{
		{[]string{"market", "currencies"}, false},
		{[]string{"market", "tickers"}, false},
		{[]string{"order", "list"}, false},
		{[]string{"order", "quote"}, true},
		{[]string{"order", "place"}, true},
	} {
		t.Run(strings.Join(tc.path, "/"), func(t *testing.T) {
			cmd, remaining, err := Cmd.Find(tc.path)
			if err != nil || len(remaining) != 0 {
				t.Fatalf("find command: remaining=%v err=%v", remaining, err)
			}
			flag := cmd.Flags().Lookup("currency")
			if flag == nil {
				t.Fatal("missing --currency")
			}
			for _, want := range []string{"Alpha currency ID", "USDS02b5a8dd2f59a098", "not a symbol", "currency field", "gate-cli cex alpha market currencies --format json", "omit --currency", "--page"} {
				if !strings.Contains(flag.Usage, want) {
					t.Errorf("help lacks %q: %s", want, flag.Usage)
				}
			}
			if flag.DefValue != "" || flag.Value.Type() != "string" || flag.Hidden {
				t.Errorf("currency flag contract changed: %+v", flag)
			}
			required := len(flag.Annotations[cobra.BashCompOneRequiredFlag]) != 0
			if required != tc.required || strings.Contains(flag.Usage, "(required)") != tc.required {
				t.Errorf("required=%v want=%v help=%s", required, tc.required, flag.Usage)
			}
		})
	}
}
