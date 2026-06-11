package info

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveTokenAddressAndChainNested(t *testing.T) {
	coin := map[string]interface{}{
		"data": map[string]interface{}{
			"canonical_contract": "0xabc",
			"canonical_chain":    "eth",
		},
	}
	addr, chain := resolveTokenAddressAndChain(coin, "")
	assert.Equal(t, "0xabc", addr)
	assert.Equal(t, "eth", chain)
}

func TestResolveTokenAddressAndChainItemsChainArray(t *testing.T) {
	coin := map[string]interface{}{
		"items": []interface{}{
			map[string]interface{}{
				"symbol":           "USDT",
				"contract_address": "0xdac17f958d2ee523a2206206994597c13d831ec7",
				"chain":            []interface{}{"Ethereum", "BNB Chain"},
			},
		},
	}
	addr, chain := resolveTokenAddressAndChain(coin, "USDT")
	assert.Equal(t, "0xdac17f958d2ee523a2206206994597c13d831ec7", addr)
	assert.Equal(t, "eth", chain)
}

func TestResolveTokenAddressAndChainTokenAddressNested(t *testing.T) {
	coin := map[string]interface{}{
		"items": []interface{}{
			map[string]interface{}{
				"symbol": "PEPE",
				"chain":  []interface{}{"Ethereum"},
				"token_address": []interface{}{
					map[string]interface{}{"contract_addr": "0x6982508145454ce325ddbe47a25d4ec3d2311933"},
				},
			},
		},
	}
	addr, chain := resolveTokenAddressAndChain(coin, "PEPE")
	assert.Equal(t, "0x6982508145454ce325ddbe47a25d4ec3d2311933", addr)
	assert.Equal(t, "eth", chain)
}

func TestResolveTokenAddressAndChainPrefersItemsZero(t *testing.T) {
	coin := map[string]interface{}{
		"items": []interface{}{
			map[string]interface{}{
				"symbol":           "USDT",
				"contract_address": "0xfirst",
				"chain":            []interface{}{"Ethereum"},
			},
			map[string]interface{}{
				"symbol":           "USDT",
				"contract_address": "0xsecond",
				"chain":            []interface{}{"BNB Chain"},
			},
		},
	}
	addr, chain := resolveTokenAddressAndChain(coin, "USDT")
	assert.Equal(t, "0xfirst", addr)
	assert.Equal(t, "eth", chain)
}

func TestCoinInfoIndicatesNativeAssetMixedItemsNotNative(t *testing.T) {
	coin := map[string]interface{}{
		"items": []interface{}{
			map[string]interface{}{
				"symbol":           "USDT",
				"contract_address": "",
				"chain":            []interface{}{"goat"},
			},
			map[string]interface{}{
				"symbol":           "USDT",
				"contract_address": "0xdac17f958d2ee523a2206206994597c13d831ec7",
				"chain":            []interface{}{"Ethereum"},
			},
		},
	}
	assert.False(t, coinInfoIndicatesNativeAsset(coin, "USDT"))
	addr, chain := resolveTokenAddressAndChain(coin, "USDT")
	assert.Equal(t, "0xdac17f958d2ee523a2206206994597c13d831ec7", addr)
	assert.Equal(t, "eth", chain)
}

func TestCoinInfoIndicatesNativeAssetAllItemsNative(t *testing.T) {
	coin := map[string]interface{}{
		"items": []interface{}{
			map[string]interface{}{
				"symbol":           "BTC",
				"contract_address": "",
				"chain":            []interface{}{"botanix"},
			},
			map[string]interface{}{
				"symbol":           "BTC",
				"contract_address": "",
				"chain":            []interface{}{"goat"},
			},
		},
	}
	assert.True(t, coinInfoIndicatesNativeAsset(coin, "BTC"))
}

func TestCoinInfoIndicatesNativeAssetBTCShape(t *testing.T) {
	coin := map[string]interface{}{
		"query": "BTC",
		"items": []interface{}{
			map[string]interface{}{
				"symbol":           "BTC",
				"contract_address": "",
				"chain":            []interface{}{"botanix"},
			},
		},
	}
	assert.True(t, coinInfoIndicatesNativeAsset(coin, "BTC"))
	addr, chain := resolveTokenAddressAndChain(coin, "BTC")
	assert.Empty(t, addr)
	assert.Empty(t, chain)
}

func TestNormalizeChainSlugForTokenSecurity(t *testing.T) {
	assert.Equal(t, "eth", normalizeChainSlugForTokenSecurity("Ethereum"))
	assert.Equal(t, "bsc", normalizeChainSlugForTokenSecurity("BNB Chain"))
	assert.Equal(t, "arbitrum", normalizeChainSlugForTokenSecurity("Arbitrum One"))
}

func TestPickPreferredChainBNBSymbol(t *testing.T) {
	chain := pickPreferredChain([]string{"Ethereum", "BNB Chain"}, "BNB")
	require.Equal(t, "bsc", chain)
}
