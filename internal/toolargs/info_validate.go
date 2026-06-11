package toolargs

import (
	"encoding/json"
	"errors"
	"strings"
)

var infoMarketdetailMarketTypes = map[string]struct{}{
	"spot": {}, "futures": {}, "delivery": {}, "options": {},
}

var infoSearchPlatformsSortBy = map[string]struct{}{
	"tvl": {}, "volume_24h": {}, "volume_spot_24h": {}, "volume_perps_24h": {},
	"volume_perps_7d": {}, "volume_perps_30d": {}, "volume_perps_qtd": {}, "fees_24h": {},
}

var infoIndicatorTimeframes = map[string]struct{}{
	"15m": {}, "1h": {}, "4h": {}, "1d": {},
}

func validateInfoCoinGetCoinInfo(arguments map[string]interface{}) error {
	if err := requireAtLeastOneString(arguments, []string{"query", "symbol"}, "query or symbol"); err != nil {
		return err
	}
	if limit, ok := intArg(arguments, "limit"); ok && limit > 100 {
		return errInvalidArguments("limit must be at most 100")
	}
	if size, ok := intArg(arguments, "size"); ok && size > 100 {
		return errInvalidArguments("size must be at most 100")
	}
	return nil
}

func validateInfoMarketSnapshot(arguments map[string]interface{}) error {
	if !nonEmptyStringArg(arguments, "symbol") {
		return errors.New("missing required field: symbol")
	}
	return nil
}

func validateInfoMarkettrendGetTechnicalAnalysis(arguments map[string]interface{}) error {
	if !nonEmptyStringArg(arguments, "symbol") {
		return errors.New("missing required field: symbol")
	}
	return nil
}

func validateInfoMarkettrendGetIndicatorHistory(arguments map[string]interface{}) error {
	if missing := missingRequiredStringArgs(arguments, "symbol", "timeframe"); len(missing) > 0 {
		return errors.New("missing required fields: " + strings.Join(missing, ", "))
	}
	if len(stringSliceArg(arguments, "indicators")) == 0 && !nonEmptyStringArg(arguments, "indicators") {
		return errors.New("missing required field: indicators")
	}
	if tf := strings.TrimSpace(strings.ToLower(stringArg(arguments, "timeframe"))); tf != "" {
		if _, ok := infoIndicatorTimeframes[tf]; !ok {
			return errInvalidArgumentsf("timeframe must be 15m, 1h, 4h, or 1d (got %q)", stringArg(arguments, "timeframe"))
		}
	}
	if limit, ok := intArg(arguments, "limit"); ok && limit > 500 {
		return errInvalidArguments("limit must be at most 500")
	}
	return nil
}

func validateInfoMarketdetailOrderbook(arguments map[string]interface{}) error {
	if !nonEmptyStringArg(arguments, "symbol") {
		return errors.New("missing required field: symbol")
	}
	if mt := strings.TrimSpace(strings.ToLower(stringArg(arguments, "market_type"))); mt != "" {
		if _, ok := infoMarketdetailMarketTypes[mt]; !ok {
			return errInvalidArgumentsf("market_type must be spot, futures, delivery, or options (got %q)", stringArg(arguments, "market_type"))
		}
	}
	if depth, ok := intArg(arguments, "depth"); ok && depth > 100 {
		return errInvalidArguments("depth must be at most 100")
	}
	return nil
}

func validateInfoMarketdetailRecentTrades(arguments map[string]interface{}) error {
	if !nonEmptyStringArg(arguments, "symbol") {
		return errors.New("missing required field: symbol")
	}
	if mt := strings.TrimSpace(strings.ToLower(stringArg(arguments, "market_type"))); mt != "" {
		if _, ok := infoMarketdetailMarketTypes[mt]; !ok {
			return errInvalidArgumentsf("market_type must be spot, futures, delivery, or options (got %q)", stringArg(arguments, "market_type"))
		}
	}
	if limit, ok := intArg(arguments, "limit"); ok && limit > 1000 {
		return errInvalidArguments("limit must be at most 1000")
	}
	return nil
}

func validateInfoCoinSearchCoins(arguments map[string]interface{}) error {
	hasFilter := nonEmptyStringArg(arguments, "category") ||
		nonEmptyStringArg(arguments, "chain") ||
		nonEmptyStringArg(arguments, "asset_type") ||
		floatArgPresent(arguments, "market_cap_min") ||
		floatArgPresent(arguments, "market_cap_max")
	if !hasFilter {
		return errors.New("missing required fields: provide at least one filter (category, chain, asset_type, or market_cap_min/max)")
	}
	if limit, ok := intArg(arguments, "limit"); ok && limit > 100 {
		return errInvalidArguments("limit must be at most 100")
	}
	return nil
}

func validateInfoPlatformmetricsSearchPlatforms(arguments map[string]interface{}) error {
	if sb := strings.TrimSpace(strings.ToLower(stringArg(arguments, "sort_by"))); sb != "" {
		if _, ok := infoSearchPlatformsSortBy[sb]; !ok {
			return errInvalidArgumentsf("sort_by is not supported (got %q)", stringArg(arguments, "sort_by"))
		}
	}
	if so := strings.TrimSpace(strings.ToLower(stringArg(arguments, "sort_order"))); so != "" && so != "asc" && so != "desc" {
		return errInvalidArgumentsf("sort_order must be asc or desc (got %q)", stringArg(arguments, "sort_order"))
	}
	if limit, ok := intArg(arguments, "limit"); ok && limit > 100 {
		return errInvalidArguments("limit must be at most 100")
	}
	return nil
}

func floatArgPresent(arguments map[string]interface{}, key string) bool {
	v, ok := arguments[key]
	if !ok || v == nil {
		return false
	}
	switch x := v.(type) {
	case float64:
		return true
	case int:
		return true
	case int64:
		return true
	case json.Number:
		return strings.TrimSpace(x.String()) != ""
	default:
		return false
	}
}
