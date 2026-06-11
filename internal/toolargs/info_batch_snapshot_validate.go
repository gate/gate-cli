package toolargs

import (
	"errors"
	"strings"
)

func validateInfoBatchMarketSnapshot(arguments map[string]interface{}) error {
	syms := stringSliceArg(arguments, "symbols")
	nonEmpty := 0
	for _, s := range syms {
		if strings.TrimSpace(s) != "" {
			nonEmpty++
		}
	}
	if nonEmpty == 0 {
		return errors.New("missing required field: symbols")
	}
	if nonEmpty > 20 {
		return errInvalidArguments("symbols must contain at most 20 pairs")
	}
	return nil
}
