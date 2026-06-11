package toolargs

import (
	"errors"
	"strings"
)

var onchainTxTimeRanges = map[string]struct{}{
	"1h": {}, "24h": {}, "1d": {}, "7d": {}, "30d": {}, "90d": {},
}

var macroIndicatorModes = map[string]struct{}{
	"latest": {}, "timeseries": {},
}

var coinRankingTypes = map[string]struct{}{
	"popular": {}, "top_gainers": {}, "top_losers": {}, "twitter_hot": {}, "airdrop": {}, "new_listing": {}, "market_pulse_hot": {},
}

var yieldPoolsScopes = map[string]struct{}{
	"basic": {}, "full": {},
}

var coinRankingTimeRanges = map[string]struct{}{
	"1h": {}, "24h": {}, "7d": {},
}

func validateInfoOnchainGetAddressInfo(arguments map[string]interface{}) error {
	if !nonEmptyStringArg(arguments, "address") {
		return errors.New("missing required field: address")
	}
	return nil
}

func validateInfoOnchainGetAddressTransactions(arguments map[string]interface{}) error {
	if !nonEmptyStringArg(arguments, "address") {
		return errors.New("missing required field: address")
	}
	if tr := strings.TrimSpace(strings.ToLower(stringArg(arguments, "time_range"))); tr != "" {
		if _, ok := onchainTxTimeRanges[tr]; !ok {
			return errInvalidArgumentsf("time_range is not supported (got %q)", stringArg(arguments, "time_range"))
		}
	}
	if limit, ok := intArg(arguments, "limit"); ok && limit > 200 {
		return errInvalidArguments("limit must be at most 200")
	}
	return nil
}

func validateInfoOnchainGetTransaction(arguments map[string]interface{}) error {
	if !nonEmptyStringArg(arguments, "tx_hash") {
		return errors.New("missing required field: tx_hash")
	}
	return nil
}

func validateInfoOnchainGetTokenOnchain(arguments map[string]interface{}) error {
	if !nonEmptyStringArg(arguments, "token") {
		return errors.New("missing required field: token")
	}
	return nil
}

func validateInfoMacroGetMacroIndicator(arguments map[string]interface{}) error {
	if !nonEmptyStringArg(arguments, "indicator") {
		return errors.New("missing required field: indicator")
	}
	if mode := strings.TrimSpace(strings.ToLower(stringArg(arguments, "mode"))); mode != "" {
		if _, ok := macroIndicatorModes[mode]; !ok {
			return errInvalidArgumentsf("mode must be latest or timeseries (got %q)", stringArg(arguments, "mode"))
		}
	}
	if size, ok := intArg(arguments, "size"); ok && size > 200 {
		return errInvalidArguments("size must be at most 200")
	}
	return nil
}

func validateInfoMacroGetEconomicCalendar(arguments map[string]interface{}) error {
	start, hasStart, err := parseOptionalInfoDate(arguments, "start_date")
	if err != nil {
		return err
	}
	end, hasEnd, err := parseOptionalInfoDate(arguments, "end_date")
	if err != nil {
		return err
	}
	if hasStart && hasEnd && start.After(end) {
		return errInvalidArguments("start_date must be on or before end_date")
	}
	if size, ok := intArg(arguments, "size"); ok && size > 200 {
		return errInvalidArguments("size must be at most 200")
	}
	return nil
}

func validateInfoPlatformmetricsDefiOverview(arguments map[string]interface{}) error {
	// Spec: unknown category strings pass through to ES; no closed enum reject.
	return nil
}

func validateInfoPlatformmetricsBridgeMetrics(arguments map[string]interface{}) error {
	if limit, ok := intArg(arguments, "limit"); ok && limit > 100 {
		return errInvalidArguments("limit must be at most 100")
	}
	return nil
}

func validateInfoPlatformmetricsYieldPools(arguments map[string]interface{}) error {
	if scope := strings.TrimSpace(strings.ToLower(stringArg(arguments, "scope"))); scope != "" {
		if _, ok := yieldPoolsScopes[scope]; !ok {
			return errInvalidArgumentsf("scope must be basic or full (got %q)", stringArg(arguments, "scope"))
		}
	}
	if limit, ok := intArg(arguments, "limit"); ok && limit > 100 {
		return errInvalidArguments("limit must be at most 100")
	}
	return nil
}

func validateInfoPlatformmetricsLiquidationHeatmap(arguments map[string]interface{}) error {
	if !nonEmptyStringArg(arguments, "symbol") {
		return errors.New("missing required field: symbol")
	}
	return nil
}

var chainActivityMetricGroups = map[string]struct{}{
	"staking": {},
}

var chainActivityLookbacks = map[string]struct{}{
	"30d": {},
	"90d": {},
	"1y":  {},
}

var chainActivityStakingChains = map[string]struct{}{
	"eth":      {},
	"ethereum": {},
}

func validateInfoPlatformmetricsChainActivity(arguments map[string]interface{}) error {
	mg := strings.TrimSpace(strings.ToLower(stringArg(arguments, "metric_group")))
	if mg == "" {
		return errors.New("missing required field: metric_group")
	}
	if _, ok := chainActivityMetricGroups[mg]; !ok {
		return errInvalidArgumentsf("metric_group is not supported (got %q)", stringArg(arguments, "metric_group"))
	}
	if chain := strings.TrimSpace(strings.ToLower(stringArg(arguments, "chain"))); chain != "" {
		if mg == "staking" {
			if _, ok := chainActivityStakingChains[chain]; !ok {
				return errInvalidArgumentsf("chain must be eth or ethereum for metric_group=staking (got %q)", stringArg(arguments, "chain"))
			}
		}
	}
	if lb := strings.TrimSpace(strings.ToLower(stringArg(arguments, "lookback"))); lb != "" {
		if _, ok := chainActivityLookbacks[lb]; !ok {
			return errInvalidArgumentsf("lookback must be 30d, 90d, or 1y (got %q)", stringArg(arguments, "lookback"))
		}
	}
	start, hasStart, err := parseOptionalInfoDate(arguments, "start_date")
	if err != nil {
		return err
	}
	end, hasEnd, err := parseOptionalInfoDate(arguments, "end_date")
	if err != nil {
		return err
	}
	if hasStart && hasEnd && start.After(end) {
		return errInvalidArguments("start_date must be on or before end_date")
	}
	return nil
}

func validateInfoCoinGetCoinRankings(arguments map[string]interface{}) error {
	rt := strings.TrimSpace(strings.ToLower(stringArg(arguments, "ranking_type")))
	if rt == "" {
		return errors.New("missing required field: ranking_type")
	}
	if _, ok := coinRankingTypes[rt]; !ok {
		return errInvalidArgumentsf("ranking_type is not supported (got %q)", stringArg(arguments, "ranking_type"))
	}
	if tr := strings.TrimSpace(strings.ToLower(stringArg(arguments, "time_range"))); tr != "" {
		if _, ok := coinRankingTimeRanges[tr]; !ok {
			return errInvalidArgumentsf("time_range must be 1h, 24h, or 7d (got %q)", stringArg(arguments, "time_range"))
		}
		if rt != "top_gainers" && rt != "top_losers" {
			return errInvalidArguments("time_range is only valid for ranking_type top_gainers or top_losers")
		}
	}
	if rt != "new_listing" {
		if nonEmptyStringArg(arguments, "listing_query") ||
			arguments["listing_from"] != nil ||
			nonEmptyStringArg(arguments, "listing_tickers") {
			return errInvalidArguments("listing_query, listing_from, and listing_tickers are only valid for ranking_type new_listing")
		}
	}
	if limit, ok := intArg(arguments, "limit"); ok && limit > 100 {
		return errInvalidArguments("limit must be at most 100")
	}
	return nil
}
