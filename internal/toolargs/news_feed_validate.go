package toolargs

import (
	"errors"
	"strings"
	"time"
	"unicode/utf8"
)

var searchNewsTimeRanges = map[string]struct{}{
	"1h": {}, "24h": {}, "7d": {}, "30d": {},
}

var sentimentTimeRanges = map[string]struct{}{
	"1h": {}, "24h": {}, "7d": {},
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
	socialInsightPlatforms = map[string]struct{}{
		"all": {}, "gate_square": {}, "binance_square": {}, "twitter": {},
		"telegram": {}, "youtube": {}, "reddit": {}, "discord": {},
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

func validateNewsEventsGetMarketMoveReport(arguments map[string]interface{}) error {
	if _, exists := arguments["is_make_new"]; exists {
		return errInvalidArguments("is_make_new is not supported by the read-only MCP/CLI tool")
	}
	return validateMarketMoveReportSymbol(arguments)
}

func validateNewsEventsListMarketMoveReports(arguments map[string]interface{}) error {
	if err := validateMarketMoveReportSymbol(arguments); err != nil {
		return err
	}
	if _, exists := arguments["is_make_new"]; exists {
		return errInvalidArguments("is_make_new is not supported by the read-only MCP/CLI tool")
	}
	if missing := missingRequiredStringArgs(arguments, "start_time", "end_time"); len(missing) > 0 {
		return errors.New("missing required fields: " + strings.Join(missing, ", "))
	}
	start, err := parseMarketMoveReportTime(stringArg(arguments, "start_time"))
	if err != nil {
		return errInvalidArguments("start_time (updated_at lower bound) must be an ISO 8601 or YYYY-MM-DD HH:MM:SS UTC0 time")
	}
	end, err := parseMarketMoveReportTime(stringArg(arguments, "end_time"))
	if err != nil {
		return errInvalidArguments("end_time (updated_at upper bound) must be an ISO 8601 or YYYY-MM-DD HH:MM:SS UTC0 time")
	}
	if start.After(end) {
		return errInvalidArguments("start_time must not be after end_time for updated_at filtering")
	}
	if limit, ok := intArg(arguments, "limit"); ok && (limit < 0 || limit > 100) {
		return errInvalidArgumentsf("limit must be 0 (default 20) or between 1 and 100 (got %d)", limit)
	}
	return nil
}

func validateMarketMoveReportSymbol(arguments map[string]interface{}) error {
	symbol := strings.TrimSpace(stringArg(arguments, "symbol"))
	if symbol == "" {
		return errors.New("missing required field: symbol")
	}
	if utf8.RuneCountInString(symbol) > 20 {
		return errInvalidArguments("symbol must contain at most 20 characters")
	}
	return nil
}

func parseMarketMoveReportTime(raw string) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	for _, layout := range []string{
		time.RFC3339Nano,
		"2006-01-02 15:04:05Z07:00",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
	} {
		var (
			parsed time.Time
			err    error
		)
		if layout == "2006-01-02T15:04:05" || layout == "2006-01-02 15:04:05" {
			parsed, err = time.ParseInLocation(layout, raw, time.UTC)
		} else {
			parsed, err = time.Parse(layout, raw)
		}
		if err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, errors.New("unsupported datetime")
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
		if _, ok := sentimentTimeRanges[tr]; !ok {
			return errInvalidArgumentsf("time_range must be 1h, 24h, or 7d (got %q)", stringArg(arguments, "time_range"))
		}
	}
	return nil
}

func validateNewsFeedMentionBurst(arguments map[string]interface{}) error {
	if !nonEmptyStringArg(arguments, "coin") {
		return errors.New("missing required field: coin")
	}
	if window := strings.TrimSpace(strings.ToLower(stringArg(arguments, "window"))); window != "" && window != "24h" {
		return errInvalidArgumentsf("window only supports 24h (got %q)", stringArg(arguments, "window"))
	}
	return validateSocialInsightPlatforms(arguments)
}

func validateNewsFeedHotTopics(arguments map[string]interface{}) error {
	if !nonEmptyStringArg(arguments, "coin") {
		return errors.New("missing required field: coin")
	}
	if window := strings.TrimSpace(strings.ToLower(stringArg(arguments, "window"))); window != "" && window != "4h" {
		return errInvalidArgumentsf("window only supports 4h (got %q)", stringArg(arguments, "window"))
	}
	if limit, ok := intArg(arguments, "limit"); ok && (limit < 2 || limit > 4) {
		return errInvalidArgumentsf("limit must be between 2 and 4 (got %d)", limit)
	}
	return validateSocialInsightPlatforms(arguments)
}

func validateSocialInsightPlatforms(arguments map[string]interface{}) error {
	raw := strings.ReplaceAll(stringArg(arguments, "platforms"), "，", ",")
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	seen := map[string]struct{}{}
	for _, part := range strings.Split(raw, ",") {
		platform := strings.TrimSpace(strings.ToLower(part))
		if platform == "" {
			continue
		}
		if _, ok := socialInsightPlatforms[platform]; !ok {
			return errInvalidArgumentsf("unsupported platform %q", platform)
		}
		seen[platform] = struct{}{}
	}
	if _, hasAll := seen["all"]; hasAll && len(seen) > 1 {
		return errInvalidArguments("all cannot be combined with another platform")
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
