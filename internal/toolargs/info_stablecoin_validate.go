package toolargs

import (
	"strings"
)

// validateInfoStablecoinInfo mirrors MCP spec cross-field rules for issuance_flow
// (see specs/mcp/info-mcp-tools-inputs-logic.json info_platformmetrics_get_stablecoin_info).
func validateInfoStablecoinInfo(arguments map[string]interface{}) error {
	scope, _, err := infoScopeBasicOrFull(arguments, "scope")
	if err != nil {
		return err
	}

	sections := stablecoinSectionsArg(arguments)
	if len(sections) > 0 {
		if scope != "full" {
			return errInvalidArguments("sections requires scope=full")
		}
		for _, sec := range sections {
			if strings.ToLower(strings.TrimSpace(sec)) != "issuance_flow" {
				return errInvalidArgumentsf("sections only supports issuance_flow (got %q)", sec)
			}
		}
		if sym := strings.TrimSpace(stringArg(arguments, "symbol")); sym != "" {
			u := strings.ToUpper(sym)
			if u != "USDT" && u != "USDC" {
				return errInvalidArgumentsf("symbol must be USDT or USDC when requesting issuance_flow (got %q)", sym)
			}
		}
	}

	if nonEmptyStringArg(arguments, "start_date") || nonEmptyStringArg(arguments, "end_date") {
		if scope != "full" || !stablecoinSectionsHasIssuanceFlow(sections) {
			return errInvalidArguments("start_date and end_date require scope=full and sections=issuance_flow")
		}
	}

	// Spec: omit or <=0 -> server default 10; only reject explicit out-of-range positives.
	if limit, ok := intArg(arguments, "limit"); ok && limit > 0 && limit > 400 {
		return errInvalidArguments("limit must be between 1 and 400")
	}
	return nil
}

func stablecoinSectionsArg(arguments map[string]interface{}) []string {
	v, ok := arguments["sections"]
	if !ok || v == nil {
		return nil
	}
	switch s := v.(type) {
	case string:
		return normalizeFlagStringList([]string{s})
	default:
		return stringSliceArg(arguments, "sections")
	}
}

func stablecoinSectionsHasIssuanceFlow(sections []string) bool {
	for _, sec := range sections {
		if strings.ToLower(strings.TrimSpace(sec)) == "issuance_flow" {
			return true
		}
	}
	return false
}
