package info

import (
	"encoding/json"
	"strings"
)

// resolveTokenAddressAndChain extracts contract address + chain slug from get_coin_info payload.
// querySymbol drives multi-chain preference (aligned with mcp-server rerank / coreSymbolPreferredChain).
func resolveTokenAddressAndChain(coin map[string]interface{}, querySymbol string) (string, string) {
	if coin == nil {
		return "", ""
	}
	symbol := strings.ToUpper(strings.TrimSpace(querySymbol))

	if items := coinItems(coin); len(items) > 0 {
		if addr, chain := contractFromCoinItem(items[0], symbol); addr != "" && chain != "" {
			return addr, chain
		}
		for _, item := range items[1:] {
			if addr, chain := contractFromCoinItem(item, symbol); addr != "" && chain != "" {
				return addr, chain
			}
		}
	}

	if address, chain := contractFromCoinItem(coin, symbol); address != "" && chain != "" {
		return address, chain
	}
	for _, key := range []string{"data", "coin", "result", "project", "token", "asset"} {
		if nested, ok := coin[key].(map[string]interface{}); ok {
			if address, chain := contractFromCoinItem(nested, symbol); address != "" && chain != "" {
				return address, chain
			}
		}
	}
	return "", ""
}

// coinInfoIndicatesNativeAsset reports native / no-contract assets (e.g. BTC) where token security does not apply.
func coinInfoIndicatesNativeAsset(coin map[string]interface{}, querySymbol string) bool {
	if coin == nil {
		return false
	}
	items := coinItems(coin)
	if len(items) == 0 {
		items = []map[string]interface{}{coin}
	}
	symbol := strings.ToUpper(strings.TrimSpace(querySymbol))
	matched := 0
	nativeLike := 0
	for _, item := range items {
		if !symbolMatchesItem(item, symbol) && symbol != "" {
			continue
		}
		matched++
		if coinItemLooksNative(item) {
			nativeLike++
		}
	}
	if matched == 0 {
		return false
	}
	// Only treat as native when every symbol-matching hit lacks a resolvable contract.
	return matched == nativeLike
}

func coinItems(coin map[string]interface{}) []map[string]interface{} {
	for _, key := range []string{"items", "coins", "results", "matches"} {
		arr, ok := coin[key].([]interface{})
		if !ok {
			continue
		}
		out := make([]map[string]interface{}, 0, len(arr))
		for _, raw := range arr {
			if m, ok := raw.(map[string]interface{}); ok {
				out = append(out, m)
			}
		}
		if len(out) > 0 {
			return out
		}
	}
	return nil
}

func symbolMatchesItem(item map[string]interface{}, symbol string) bool {
	if symbol == "" {
		return true
	}
	for _, key := range []string{"symbol", "gate_symbol", "source_id"} {
		if strings.EqualFold(strings.TrimSpace(firstStringByKeys(item, key)), symbol) {
			return true
		}
	}
	return false
}

func coinItemLooksNative(item map[string]interface{}) bool {
	if item == nil {
		return false
	}
	for _, key := range []string{"native_coin", "is_native", "is_main_asset", "is_main"} {
		switch v := item[key].(type) {
		case bool:
			if v {
				return true
			}
		case string:
			s := strings.ToLower(strings.TrimSpace(v))
			if s == "1" || s == "true" || s == "yes" {
				return true
			}
		}
	}
	if tt := strings.ToLower(strings.TrimSpace(firstStringByKeys(item, "token_type"))); tt == "native" || tt == "main" {
		return true
	}
	addr := extractContractAddress(item)
	if addr != "" {
		return false
	}
	if addrs := extractTokenAddressList(item); len(addrs) > 0 {
		return false
	}
	// Empty contract with chain present and no token_address → native / non-EVM token-security path.
	return len(extractChainList(item)) > 0 || firstStringByKeys(item, "symbol") != ""
}

func contractFromCoinItem(item map[string]interface{}, querySymbol string) (string, string) {
	if item == nil {
		return "", ""
	}
	address := extractContractAddress(item)
	if address == "" {
		if addrs := extractTokenAddressList(item); len(addrs) > 0 {
			address = addrs[0]
		}
	}
	if address == "" {
		return "", ""
	}
	chain := pickPreferredChain(extractChainList(item), querySymbol)
	return address, chain
}

func extractContractAddress(m map[string]interface{}) string {
	if addr := firstStringByKeys(m, "canonical_contract", "contract_address", "contract_addr", "address", "contract"); addr != "" {
		return addr
	}
	return ""
}

func extractTokenAddressList(m map[string]interface{}) []string {
	raw, ok := m["token_address"]
	if !ok || raw == nil {
		return nil
	}
	switch v := raw.(type) {
	case string:
		s := strings.TrimSpace(v)
		if s == "" {
			return nil
		}
		if strings.HasPrefix(s, "[") {
			var arr []interface{}
			if err := json.Unmarshal([]byte(s), &arr); err == nil {
				return addressesFromTokenAddressArray(arr)
			}
		}
		return []string{s}
	case []string:
		out := make([]string, 0, len(v))
		for _, s := range v {
			if t := strings.TrimSpace(s); t != "" {
				out = append(out, t)
			}
		}
		return out
	case []interface{}:
		return addressesFromTokenAddressArray(v)
	}
	return nil
}

func addressesFromTokenAddressArray(arr []interface{}) []string {
	out := make([]string, 0, len(arr))
	for _, it := range arr {
		switch x := it.(type) {
		case string:
			if s := strings.TrimSpace(x); s != "" {
				out = append(out, s)
			}
		case map[string]interface{}:
			if s := firstStringByKeys(x, "contract_addr", "contract_address", "address"); s != "" {
				out = append(out, s)
			}
		}
	}
	return out
}

func extractChainList(m map[string]interface{}) []string {
	if chains := stringSliceField(m, "chain"); len(chains) > 0 {
		return chains
	}
	if c := firstStringByKeys(m, "canonical_chain", "network", "primary_chain"); c != "" {
		return []string{c}
	}
	return nil
}

func stringSliceField(m map[string]interface{}, key string) []string {
	raw, ok := m[key]
	if !ok || raw == nil {
		return nil
	}
	switch v := raw.(type) {
	case string:
		s := strings.TrimSpace(v)
		if s == "" {
			return nil
		}
		return []string{s}
	case []string:
		out := make([]string, 0, len(v))
		for _, s := range v {
			if t := strings.TrimSpace(s); t != "" {
				out = append(out, t)
			}
		}
		return out
	case []interface{}:
		out := make([]string, 0, len(v))
		for _, it := range v {
			if s, ok := it.(string); ok {
				if t := strings.TrimSpace(s); t != "" {
					out = append(out, t)
				}
			}
		}
		return out
	default:
		return nil
	}
}

func pickPreferredChain(chains []string, querySymbol string) string {
	if len(chains) == 0 {
		return ""
	}
	if len(chains) == 1 {
		return normalizeChainSlugForTokenSecurity(chains[0])
	}
	symbol := strings.ToUpper(strings.TrimSpace(querySymbol))
	if hint := coreSymbolPreferredChainSlug(symbol); hint != "" {
		for _, c := range chains {
			if chainSlugMatches(c, hint) {
				return normalizeChainSlugForTokenSecurity(c)
			}
		}
	}
	for _, pref := range tokenSecurityChainPriority {
		for _, c := range chains {
			if chainSlugMatches(c, pref) {
				return normalizeChainSlugForTokenSecurity(c)
			}
		}
	}
	return normalizeChainSlugForTokenSecurity(chains[0])
}

// coreSymbolPreferredChainSlug mirrors mcp-server internal/coin/searcher.go coreSymbolPreferredChain.
func coreSymbolPreferredChainSlug(symbol string) string {
	switch strings.ToUpper(strings.TrimSpace(symbol)) {
	case "BTC":
		return "btc"
	case "ETH":
		return "eth"
	case "SOL":
		return "sol"
	case "BNB":
		return "bsc"
	default:
		return ""
	}
}

var tokenSecurityChainPriority = []string{
	"eth", "ethereum",
	"bsc", "bnb",
	"arbitrum", "arb",
	"base",
	"polygon", "matic",
	"optimism", "op",
	"avalanche", "avax",
	"solana", "sol",
	"tron", "trx",
}

func chainSlugMatches(chain, slug string) bool {
	c := normalizeChainSlugForTokenSecurity(chain)
	s := normalizeChainSlugForTokenSecurity(slug)
	if c == s {
		return true
	}
	aliases := chainSlugAliases[c]
	for _, a := range aliases {
		if a == s {
			return true
		}
	}
	aliases = chainSlugAliases[s]
	for _, a := range aliases {
		if a == c {
			return true
		}
	}
	return false
}

var chainSlugAliases = map[string][]string{
	"eth":       {"ethereum"},
	"ethereum":  {"eth"},
	"bsc":       {"bnb", "bnb chain", "binance-smart-chain"},
	"bnb":       {"bsc"},
	"arb":       {"arbitrum", "arbitrum one"},
	"arbitrum":  {"arb"},
	"matic":     {"polygon"},
	"polygon":   {"matic"},
	"op":        {"optimism"},
	"optimism":  {"op"},
	"avax":      {"avalanche", "avalanche-c"},
	"avalanche": {"avax"},
	"sol":       {"solana"},
	"solana":    {"sol"},
	"trx":       {"tron"},
	"tron":      {"trx"},
	"btc":       {"bitcoin"},
	"bitcoin":   {"btc"},
}

func normalizeChainSlugForTokenSecurity(chain string) string {
	c := strings.ToLower(strings.TrimSpace(chain))
	c = strings.NewReplacer(" ", "-", "_", "-").Replace(c)
	switch c {
	case "ethereum", "eth-mainnet":
		return "eth"
	case "bnb-chain", "binance-smart-chain", "binance-smart-chain-mainnet":
		return "bsc"
	case "arbitrum-one", "arbitrum-one-mainnet":
		return "arbitrum"
	case "polygon-pos", "matic-mainnet":
		return "polygon"
	case "optimistic-ethereum", "op-mainnet":
		return "optimism"
	case "avalanche-c-chain", "avax-c":
		return "avax"
	case "solana-mainnet":
		return "sol"
	case "bitcoin", "btc-mainnet":
		return "btc"
	default:
		return c
	}
}
