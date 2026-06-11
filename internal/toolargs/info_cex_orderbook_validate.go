package toolargs

import (
	"errors"
	"strings"
)

var cexOrderbookMarketTypes = map[string]struct{}{
	"spot": {}, "perp": {}, "perps": {}, "futures": {}, "future": {},
}

var cexOrderbookDataScopes = map[string]struct{}{
	"exchange": {}, "market": {},
}

func validateInfoCexOrderbookDepth(arguments map[string]interface{}) error {
	if !nonEmptyStringArg(arguments, "symbol") {
		return errors.New("missing required field: symbol")
	}
	if mt := strings.TrimSpace(strings.ToLower(stringArg(arguments, "market_type"))); mt != "" {
		if _, ok := cexOrderbookMarketTypes[mt]; !ok {
			return errInvalidArgumentsf("market_type must be spot, perp, perps, futures, or future (got %q)", stringArg(arguments, "market_type"))
		}
	}
	if ds := strings.TrimSpace(strings.ToLower(stringArg(arguments, "data_scope"))); ds != "" {
		if _, ok := cexOrderbookDataScopes[ds]; !ok {
			return errInvalidArgumentsf("data_scope must be exchange or market (got %q)", stringArg(arguments, "data_scope"))
		}
	}
	if limit, ok := intArg(arguments, "limit"); ok {
		if limit < 1 || limit > 100 {
			return errInvalidArguments("limit must be between 1 and 100")
		}
	}
	return nil
}
