package toolargs

import (
	"errors"
	"strings"
)

var searchNewsTimeRanges = map[string]struct{}{
	"1h": {}, "24h": {}, "7d": {}, "30d": {},
}

var searchXTimeRanges = map[string]struct{}{
	"1h": {}, "24h": {}, "7d": {},
}

var explainMarketMoveTimeRanges = map[string]struct{}{
	"30m": {}, "1h": {}, "2h": {}, "4h": {}, "24h": {},
}

var latestEventsTimeRanges = map[string]struct{}{
	"1h": {}, "24h": {}, "7d": {},
}

var (
	ugcPlatforms = map[string]struct{}{
		"reddit": {}, "discord": {}, "telegram": {}, "youtube": {}, "all": {},
	}
	ugcDomains = map[string]struct{}{
		"crypto": {}, "defi": {}, "finance": {}, "macro": {}, "ai_agent": {}, "web3_dev": {}, "all": {},
	}
	ugcQualityTiers = map[string]struct{}{
		"a": {}, "b": {}, "all": {},
	}
	ugcTimeRanges = map[string]struct{}{
		"1h": {}, "24h": {}, "7d": {}, "30d": {}, "all": {},
	}
	webSearchTimeRanges = map[string]struct{}{
		"1h": {}, "24h": {}, "7d": {}, "30d": {},
	}
)

func validateNewsFeedSearchNews(arguments map[string]interface{}) error {
	if err := requireAtLeastOneString(arguments, []string{"query", "coin"}, "query or coin"); err != nil {
		return err
	}
	if tr := strings.TrimSpace(strings.ToLower(stringArg(arguments, "time_range"))); tr != "" {
		if _, ok := searchNewsTimeRanges[tr]; !ok {
			return errInvalidArgumentsf("time_range must be 1h, 24h, 7d, or 30d (got %q)", stringArg(arguments, "time_range"))
		}
	}
	if limit, ok := intArg(arguments, "limit"); ok && limit > 100 {
		return errInvalidArguments("limit must be at most 100")
	}
	return nil
}

func validateNewsEventsExplainMarketMove(arguments map[string]interface{}) error {
	if missing := missingRequiredStringArgs(arguments, "query", "coin"); len(missing) > 0 {
		return errors.New("missing required fields: " + strings.Join(missing, ", "))
	}
	if tr := strings.TrimSpace(strings.ToLower(stringArg(arguments, "time_range"))); tr != "" {
		if _, ok := explainMarketMoveTimeRanges[tr]; !ok {
			return errInvalidArgumentsf("time_range must be 30m, 1h, 2h, 4h, or 24h (got %q)", stringArg(arguments, "time_range"))
		}
	}
	return nil
}

func validateNewsFeedSearchX(arguments map[string]interface{}) error {
	if !nonEmptyStringArg(arguments, "query") {
		allowed := nonEmptyStringSlice(arguments, "allowed_handles")
		excluded := nonEmptyStringSlice(arguments, "excluded_handles")
		if len(allowed) == 0 && len(excluded) == 0 {
			return errors.New("missing required field: query (or set allowed_handles / excluded_handles)")
		}
	}
	allowed := nonEmptyStringSlice(arguments, "allowed_handles")
	excluded := nonEmptyStringSlice(arguments, "excluded_handles")
	if len(allowed) > 0 && len(excluded) > 0 {
		return errInvalidArguments("allowed_handles and excluded_handles cannot both be set")
	}
	if tr := strings.TrimSpace(strings.ToLower(stringArg(arguments, "time_range"))); tr != "" {
		if _, ok := searchXTimeRanges[tr]; !ok {
			return errInvalidArgumentsf("time_range must be 1h, 24h, or 7d (got %q)", stringArg(arguments, "time_range"))
		}
	}
	return nil
}

func validateNewsFeedSearchUGC(arguments map[string]interface{}) error {
	if err := requireAtLeastOneString(arguments, []string{"query", "coin"}, "query or coin"); err != nil {
		return err
	}
	if p := strings.TrimSpace(strings.ToLower(stringArg(arguments, "platform"))); p != "" {
		if _, ok := ugcPlatforms[p]; !ok {
			return errInvalidArgumentsf("platform must be reddit, discord, telegram, youtube, or all (got %q)", stringArg(arguments, "platform"))
		}
	}
	if d := strings.TrimSpace(strings.ToLower(stringArg(arguments, "domain"))); d != "" {
		if _, ok := ugcDomains[d]; !ok {
			return errInvalidArgumentsf("domain is not supported (got %q)", stringArg(arguments, "domain"))
		}
	}
	if q := strings.TrimSpace(strings.ToUpper(stringArg(arguments, "quality_tier"))); q != "" {
		if _, ok := ugcQualityTiers[strings.ToLower(q)]; !ok {
			return errInvalidArgumentsf("quality_tier must be A, B, or all (got %q)", stringArg(arguments, "quality_tier"))
		}
	}
	if tr := strings.TrimSpace(strings.ToLower(stringArg(arguments, "time_range"))); tr != "" {
		if _, ok := ugcTimeRanges[tr]; !ok {
			return errInvalidArgumentsf("time_range is not supported (got %q)", stringArg(arguments, "time_range"))
		}
	}
	if limit, ok := intArg(arguments, "limit"); ok && limit > 50 {
		return errInvalidArguments("limit must be at most 50")
	}
	return nil
}

func validateNewsFeedWebSearch(arguments map[string]interface{}) error {
	if !nonEmptyStringArg(arguments, "query") {
		return errors.New("missing required field: query")
	}
	if tr := strings.TrimSpace(strings.ToLower(stringArg(arguments, "time_range"))); tr != "" {
		if _, ok := webSearchTimeRanges[tr]; !ok {
			return errInvalidArgumentsf("time_range must be 1h, 24h, 7d, or 30d (got %q)", stringArg(arguments, "time_range"))
		}
	}
	if limit, ok := intArg(arguments, "limit"); ok && limit > 10 {
		return errInvalidArguments("limit must be at most 10")
	}
	return nil
}

func validateNewsFeedExchangeAnnouncements(arguments map[string]interface{}) error {
	if err := requireAtLeastOneString(arguments, []string{"coin", "query", "exchange", "platform"}, "coin, query, exchange, or platform"); err != nil {
		return err
	}
	if limit, ok := intArg(arguments, "limit"); ok && limit > 100 {
		return errInvalidArguments("limit must be at most 100")
	}
	return nil
}

func validateNewsFeedSocialSentiment(arguments map[string]interface{}) error {
	if tr := strings.TrimSpace(strings.ToLower(stringArg(arguments, "time_range"))); tr != "" {
		if _, ok := searchNewsTimeRanges[tr]; !ok {
			return errInvalidArgumentsf("time_range must be 1h, 24h, 7d, or 30d (got %q)", stringArg(arguments, "time_range"))
		}
	}
	return nil
}

func validateNewsEventsGetLatestEvents(arguments map[string]interface{}) error {
	tr := strings.TrimSpace(strings.ToLower(stringArg(arguments, "time_range")))
	hasStart := nonEmptyStringArg(arguments, "start_time")
	hasEnd := nonEmptyStringArg(arguments, "end_time")
	if tr != "" {
		if _, ok := latestEventsTimeRanges[tr]; !ok {
			return errInvalidArgumentsf("time_range must be 1h, 24h, or 7d (got %q)", stringArg(arguments, "time_range"))
		}
		if hasStart || hasEnd {
			return errInvalidArguments("time_range cannot be used with start_time or end_time")
		}
	}
	if limit, ok := intArg(arguments, "limit"); ok && limit > 100 {
		return errInvalidArguments("limit must be at most 100")
	}
	return nil
}

func nonEmptyStringSlice(arguments map[string]interface{}, key string) []string {
	raw := stringSliceArg(arguments, key)
	if len(raw) == 0 {
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, s := range raw {
		if t := strings.TrimSpace(s); t != "" {
			out = append(out, t)
		}
	}
	return out
}
