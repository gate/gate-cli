package toolargs

import "strings"

func validateInfoExchangeReserves(arguments map[string]interface{}) error {
	scope, _, err := infoScopeBasicOrFull(arguments, "scope")
	if err != nil {
		return err
	}
	if boolArgTrue(arguments, "include_history") && scope != "full" {
		return errInvalidArguments("include_history requires scope=full")
	}
	hw := strings.TrimSpace(stringArg(arguments, "history_window"))
	if hw != "" {
		if !boolArgTrue(arguments, "include_history") {
			return errInvalidArguments("history_window only applies when include_history=true")
		}
		if strings.ToLower(hw) != "quarter" {
			return errInvalidArgumentsf("history_window must be quarter (got %q)", hw)
		}
	}
	if asset := strings.TrimSpace(stringArg(arguments, "asset")); asset != "" {
		u := strings.ToUpper(asset)
		switch u {
		case "BTC", "ETH", "USDT", "USDC":
		default:
			return errInvalidArgumentsf("asset must be BTC, ETH, USDT, or USDC (got %q)", asset)
		}
	}
	return nil
}
