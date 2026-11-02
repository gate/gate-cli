package options

import (
	"github.com/stretchr/testify/require"
	"math"
	"testing"
)

// Full closes must use zero size; signed partial closes retain exact int64 values.
func TestValidateOptionsClose(t *testing.T) {
	for _, tc := range []struct {
		size      int64
		close, ok bool
	}{{0, true, true}, {1, true, false}, {-1, true, false}, {0, false, false}, {1, false, true}, {-1, false, true}, {math.MaxInt64, false, true}, {math.MinInt64, false, true}} {
		require.Equal(t, tc.ok, validateOptionsClose(tc.size, tc.close) == nil)
	}
}
