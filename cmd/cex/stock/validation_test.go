package stock

import (
	"strconv"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// TestInputGuards 测试有来源的数值契约及本地资源保护；不推断三个月的日历边界。
func TestInputGuards(t *testing.T) {
	for _, value := range []string{"0", "0.000", "-1", "1e3", "NaN", " 1", ".5", "1.", strings.Repeat("1", 129)} {
		if positiveDecimal(value) == nil {
			t.Errorf("decimal %q accepted", value)
		}
	}
	for _, value := range []string{"0.00000001", "9007199254740993", "001.01"} {
		if err := positiveDecimal(value); err != nil {
			t.Error(err)
		}
	}
	for _, value := range []string{"../x", "x/y", "x%2fy", "x?y", "x#y", "x\\y", " x", ""} {
		if symbol(value, true) == nil {
			t.Errorf("symbol %q accepted", value)
		}
	}
	for _, value := range []string{"0", "-1", "9223372036854775808", "1,,2", strings.Repeat("1,", 20) + "1"} {
		if validateOrderIDs(value) == nil {
			t.Error("invalid IDs accepted")
		}
	}
	if err := validateOrderIDs("2147483648,9007199254740993,9223372036854775807"); err != nil {
		t.Error(err)
	}
	for _, pair := range [][2]int64{{-1, 0}, {0, -1}, {2, 1}} {
		if validateTimes(pair[0], pair[1]) == nil {
			t.Error("bad time accepted")
		}
	}
	if err := validateTimes(1, 2000000000); err != nil {
		t.Fatal("backend must validate three months", err)
	}
	for _, size := range []int32{0, 501} {
		cmd := &cobra.Command{}
		addPagination(cmd)
		if err := cmd.Flags().Set("page-size", strconv.Itoa(int(size))); err != nil {
			t.Fatal(err)
		}
		if pagination(cmd) == nil {
			t.Error("invalid page size accepted")
		}
	}
	for _, key := range []string{"pnl-calc-type", "pnl-calc-price"} {
		cmd := &cobra.Command{}
		addPnL(cmd)
		if err := cmd.Flags().Set(key, "0"); err != nil {
			t.Fatal(err)
		}
		if _, _, err := pnl(cmd); err == nil {
			t.Error("explicit zero PnL accepted")
		}
		if err := cmd.Flags().Set(key, "2"); err != nil {
			t.Fatal(err)
		}
		if _, _, err := pnl(cmd); err != nil {
			t.Error(err)
		}
	}
	if got := cell([]byte(`9007199254740993`)); got != "9007199254740993" {
		t.Fatal("table lost int64")
	}
}
