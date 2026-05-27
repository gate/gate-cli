package toolargs

import (
	"strings"
)

// validateInfoStablecoinInfo mirrors MCP spec cross-field rules for extension sections
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
		hasIssuanceFlow := false
		hasUsageStructure := false
		for _, sec := range sections {
			switch strings.ToLower(strings.TrimSpace(sec)) {
			case "issuance_flow":
				hasIssuanceFlow = true
			case "usage_structure":
				hasUsageStructure = true
			default:
				return errInvalidArgumentsf("sections must be issuance_flow or usage_structure (got %q)", sec)
			}
		}
		if sym := strings.TrimSpace(stringArg(arguments, "symbol")); sym != "" {
			u := strings.ToUpper(sym)
			if hasIssuanceFlow && u != "USDT" && u != "USDC" {
				return errInvalidArgumentsf("symbol must be USDT or USDC when requesting issuance_flow (got %q)", sym)
			}
			if hasUsageStructure && !stablecoinUsageStructureSymbolAllowed(u) {
				return errInvalidArgumentsf("symbol must be USDT, USDC, DAI, FDUSD, or PYUSD when requesting usage_structure (got %q)", sym)
			}
		}
		if chain := strings.TrimSpace(stringArg(arguments, "chain")); chain != "" && !stablecoinExtensionChainAllowed(chain) {
			return errInvalidArgumentsf("chain is invalid for stablecoin extension sections (got %q)", chain)
		}
	}

	if nonEmptyStringArg(arguments, "start_date") || nonEmptyStringArg(arguments, "end_date") {
		if scope != "full" || !stablecoinSectionsHasExtension(sections) {
			return errInvalidArguments("start_date and end_date require scope=full and sections=issuance_flow or usage_structure")
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

func stablecoinSectionsHasExtension(sections []string) bool {
	for _, sec := range sections {
		switch strings.ToLower(strings.TrimSpace(sec)) {
		case "issuance_flow", "usage_structure":
			return true
		}
	}
	return false
}

func stablecoinUsageStructureSymbolAllowed(symbol string) bool {
	switch symbol {
	case "USDT", "USDC", "DAI", "FDUSD", "PYUSD":
		return true
	default:
		return false
	}
}

func stablecoinExtensionChainAllowed(chain string) bool {
	switch strings.ToLower(strings.TrimSpace(chain)) {
	case "all", "ethereum", "omni", "tron", "solana", "bsc", "arbitrum", "optimism", "polygon", "avalanche",
		"eth", "sol", "bnb", "arb", "op", "matic", "avax":
		return true
	default:
		return false
	}
}
