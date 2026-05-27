package toolargs

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"
)

func validateInfoInstitutionalMetrics(arguments map[string]interface{}) error {
	if asset := strings.TrimSpace(stringArg(arguments, "asset")); asset != "" {
		switch strings.ToLower(asset) {
		case "btc", "eth", "all":
		default:
			return errInvalidArgumentsf("asset must be BTC, ETH, or all (got %q)", asset)
		}
	}
	if channel := strings.TrimSpace(stringArg(arguments, "channel")); channel != "" {
		switch strings.ToLower(channel) {
		case "all", "etf", "cme", "cftc":
		default:
			return errInvalidArgumentsf("channel must be all, etf, cme, or cftc (got %q)", channel)
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

	if limit, ok, err := institutionalMetricsLimitArg(arguments); err != nil {
		return err
	} else if ok && (limit < 1 || limit > 366) {
		return errInvalidArguments("limit must be between 1 and 366")
	}
	return nil
}

func institutionalMetricsLimitArg(arguments map[string]interface{}) (int, bool, error) {
	v, ok := arguments["limit"]
	if !ok || v == nil {
		return 0, false, nil
	}
	switch n := v.(type) {
	case int:
		return n, true, nil
	case int64:
		return int(n), true, nil
	case float64:
		if math.Trunc(n) != n {
			return 0, true, errInvalidArgumentsf("limit must be an integer (got %v)", n)
		}
		return int(n), true, nil
	case json.Number:
		i, err := n.Int64()
		if err != nil {
			return 0, true, errInvalidArgumentsf("limit must be an integer (got %q)", n.String())
		}
		return int(i), true, nil
	default:
		return 0, true, errInvalidArgumentsf("limit must be an integer (got %q)", fmt.Sprint(v))
	}
}

func parseOptionalInfoDate(arguments map[string]interface{}, key string) (time.Time, bool, error) {
	raw := strings.TrimSpace(stringArg(arguments, key))
	if raw == "" {
		return time.Time{}, false, nil
	}
	v, err := time.Parse("2006-01-02", raw)
	if err != nil {
		return time.Time{}, false, errInvalidArgumentsf("%s must be YYYY-MM-DD (got %q)", key, raw)
	}
	return v, true, nil
}
