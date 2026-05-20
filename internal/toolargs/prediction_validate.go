package toolargs

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

var (
	predictionVenues = map[string]struct{}{
		"polymarket":  {},
		"predict_fun": {},
	}
	searchEventsCategories = map[string]struct{}{
		"crypto_event": {}, "crypto_price": {}, "culture": {}, "earnings": {},
		"elections": {}, "finance": {}, "geopolitics": {}, "macro_economy": {},
		"mentions": {}, "other": {}, "politics": {}, "sports": {}, "tech_ai": {},
		"weather_climate": {}, "world": {},
	}
	searchEventsStatus = map[string]struct{}{
		"active": {}, "closed": {}, "resolved": {}, "all": {},
	}
	searchEventsSortBy = map[string]struct{}{
		"attention": {}, "volume": {}, "liquidity": {}, "recently_listed": {},
		"probability_change": {}, "volume_delta_today": {},
	}
	eventSignalWindows = map[string]struct{}{
		"1h": {}, "24h": {}, "7d": {},
	}
)

func validateNewsPredictionOrderbook(arguments map[string]interface{}) error {
	if missing := missingRequiredStringArgs(arguments, "venue", "market_id"); len(missing) > 0 {
		return errors.New("missing required fields: " + strings.Join(missing, ", "))
	}
	venue := strings.TrimSpace(stringArg(arguments, "venue"))
	if !predictionVenueAllowed(venue) {
		return errors.New("invalid arguments: venue must be polymarket or predict_fun")
	}
	if depth, ok := intArg(arguments, "depth"); ok {
		if depth < 1 || depth > 20 {
			return errors.New("invalid arguments: depth must be between 1 and 20")
		}
	}
	return nil
}

func validateNewsPredictionSearchEvents(arguments map[string]interface{}) error {
	if err := requireAtLeastOneString(arguments, []string{"query", "coin", "category"}, "query, coin, or category"); err != nil {
		return err
	}
	if cat := strings.TrimSpace(stringArg(arguments, "category")); cat != "" {
		if _, ok := searchEventsCategories[cat]; !ok {
			return fmt.Errorf("invalid arguments: category must be one of the supported event_category_primary values (got %q)", cat)
		}
	}
	if st := strings.TrimSpace(stringArg(arguments, "status")); st != "" {
		if _, ok := searchEventsStatus[st]; !ok {
			return fmt.Errorf("invalid arguments: status must be active, closed, resolved, or all (got %q)", st)
		}
	}
	if sortBy := strings.TrimSpace(stringArg(arguments, "sort_by")); sortBy != "" {
		if _, ok := searchEventsSortBy[sortBy]; !ok {
			return fmt.Errorf("invalid arguments: sort_by is not supported (got %q)", sortBy)
		}
	}
	for _, v := range stringSliceArg(arguments, "venue") {
		if v == "" {
			continue
		}
		if !predictionVenueAllowed(v) {
			return fmt.Errorf("invalid arguments: venue must be polymarket or predict_fun (got %q)", v)
		}
	}
	if limit, ok := intArg(arguments, "limit"); ok {
		if limit < 1 || limit > 100 {
			return errors.New("invalid arguments: limit must be between 1 and 100")
		}
	}
	return nil
}

func validateNewsPredictionEventSignal(arguments map[string]interface{}) error {
	if !nonEmptyStringArg(arguments, "event_ref") {
		return errors.New("missing required field: event_ref")
	}
	venue, _, ok := parsePredictionEventRef(stringArg(arguments, "event_ref"))
	if !ok {
		return errors.New("invalid arguments: event_ref must be venue:venue_event_id (first colon separates venue)")
	}
	if !predictionVenueAllowed(venue) {
		return fmt.Errorf("invalid arguments: event_ref venue must be polymarket or predict_fun (got %q)", venue)
	}
	if win := strings.TrimSpace(stringArg(arguments, "window")); win != "" {
		if _, allowed := eventSignalWindows[strings.ToLower(win)]; !allowed {
			return fmt.Errorf("invalid arguments: window must be 1h, 24h, or 7d (got %q)", win)
		}
	}
	for _, v := range stringSliceArg(arguments, "venue") {
		if v == "" {
			continue
		}
		if v != venue {
			return fmt.Errorf("invalid arguments: venue filter %q must match event_ref venue %q", v, venue)
		}
	}
	return nil
}

func predictionVenueAllowed(venue string) bool {
	_, ok := predictionVenues[strings.TrimSpace(venue)]
	return ok
}

func parsePredictionEventRef(eventRef string) (venue, eventID string, ok bool) {
	eventRef = strings.TrimSpace(eventRef)
	i := strings.Index(eventRef, ":")
	if i <= 0 || i >= len(eventRef)-1 {
		return "", "", false
	}
	return strings.TrimSpace(eventRef[:i]), eventRef[i+1:], true
}

func stringArg(arguments map[string]interface{}, key string) string {
	v, ok := arguments[key]
	if !ok || v == nil {
		return ""
	}
	switch s := v.(type) {
	case string:
		return s
	default:
		return fmt.Sprint(v)
	}
}

func stringSliceArg(arguments map[string]interface{}, key string) []string {
	v, ok := arguments[key]
	if !ok || v == nil {
		return nil
	}
	switch xs := v.(type) {
	case []string:
		return xs
	case []interface{}:
		out := make([]string, 0, len(xs))
		for _, item := range xs {
			if item == nil {
				continue
			}
			out = append(out, strings.TrimSpace(fmt.Sprint(item)))
		}
		return out
	default:
		return nil
	}
}

func intArg(arguments map[string]interface{}, key string) (int, bool) {
	v, ok := arguments[key]
	if !ok || v == nil {
		return 0, false
	}
	switch n := v.(type) {
	case int:
		return n, true
	case int64:
		return int(n), true
	case float64:
		return int(n), true
	case json.Number:
		i, err := n.Int64()
		if err != nil {
			return 0, true
		}
		return int(i), true
	default:
		return 0, false
	}
}
