package toolargs

import "strings"

var platformInfoScopes = map[string]struct{}{
	"basic": {}, "with_chain_breakdown": {}, "full": {}, "detailed": {},
}

func validateInfoPlatformInfo(arguments map[string]interface{}) error {
	scopeRaw := strings.TrimSpace(stringArg(arguments, "scope"))
	scope := strings.ToLower(scopeRaw)
	if scope == "" {
		scope = "basic"
	} else if _, ok := platformInfoScopes[scope]; !ok {
		return errInvalidArgumentsf("scope must be basic, with_chain_breakdown, full, or detailed (got %q)", scopeRaw)
	}
	if boolArgTrue(arguments, "include_oi_symbol_detail") && scope != "full" {
		return errInvalidArguments("include_oi_symbol_detail requires scope=full")
	}
	if limit, ok := intArg(arguments, "oi_symbol_limit"); ok && limit > 100 {
		return errInvalidArguments("oi_symbol_limit must be at most 100")
	}
	return nil
}
