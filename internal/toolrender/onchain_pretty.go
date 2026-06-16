package toolrender

import (
	"fmt"
	"sort"
	"strings"
)

const (
	toolInfoOnchainGetAddressInfo         = "info_onchain_get_address_info"
	toolInfoOnchainGetAddressTransactions = "info_onchain_get_address_transactions"
)

func prettyOnchainToolResult(toolName string, data map[string]interface{}) (string, bool) {
	if data == nil {
		return "", false
	}
	switch toolName {
	case toolInfoOnchainGetAddressInfo:
		return formatAddressInfoPretty(data), true
	case toolInfoOnchainGetAddressTransactions:
		return formatAddressTransactionsPretty(data), true
	default:
		return "", false
	}
}

func formatAddressInfoPretty(data map[string]interface{}) string {
	var b strings.Builder
	addr := stringField(data, "address")
	chain := stringField(data, "chain")
	if addr != "" || chain != "" {
		b.WriteString("Address Profile\n\n")
		if addr != "" {
			fmt.Fprintf(&b, "  address: %s\n", sanitizeTerminalText(addr))
		}
		if chain != "" {
			fmt.Fprintf(&b, "  chain: %s\n", sanitizeTerminalText(chain))
		}
		if dc := stringSliceField(data, "detected_chains"); len(dc) > 0 {
			fmt.Fprintf(&b, "  detected_chains: %s\n", strings.Join(dc, ", "))
		}
		b.WriteByte('\n')
	}

	if sum, ok := data["asset_summary"].(map[string]interface{}); ok && len(sum) > 0 {
		b.WriteString("Asset Summary\n\n")
		writeKV(&b, "  ", "total_usd_value", formatFloatish(sum["total_usd_value"]))
		writeKV(&b, "  ", "native_usd_value", formatFloatish(sum["native_usd_value"]))
		writeKV(&b, "  ", "token_usd_value", formatFloatish(sum["token_usd_value"]))
		writeKV(&b, "  ", "token_num", formatFloatish(sum["token_num"]))
		writeKV(&b, "  ", "have_multi_chain_asset", formatFloatish(sum["have_multi_chain_asset"]))
		writeKV(&b, "  ", "native_balance", stringField(sum, "native_balance"))
		writeKV(&b, "  ", "token_value_usd", formatFloatish(sum["token_value_usd"]))
		writeKV(&b, "  ", "lamport", stringField(sum, "lamport"))
		writeKV(&b, "  ", "lamport_usd", formatFloatish(sum["lamport_usd"]))
		writeKV(&b, "  ", "total_value_usd", formatFloatish(sum["total_value_usd"]))
		writeKV(&b, "  ", "exist_address", formatFloatish(sum["exist_address"]))
		b.WriteByte('\n')
	}

	if rows := tokenBalanceRows(data); len(rows) > 0 {
		b.WriteString("Top Token Balances\n\n")
		limit := 5
		if len(rows) < limit {
			limit = len(rows)
		}
		for i := 0; i < limit; i++ {
			writeTokenBalanceRow(&b, rows[i])
		}
		b.WriteByte('\n')
	}

	if mc := multiChainRows(data); len(mc) > 0 {
		b.WriteString("Multi-chain Assets\n\n")
		limit := 5
		if len(mc) < limit {
			limit = len(mc)
		}
		for i := 0; i < limit; i++ {
			r := mc[i]
			fmt.Fprintf(&b, "  - chain=%s symbol=%s amount=%s usd_value=%s",
				r.chain, r.symbol, r.amount, r.usdValue)
			if r.pct != "" {
				fmt.Fprintf(&b, " percentage=%s", r.pct)
			}
			b.WriteByte('\n')
		}
		b.WriteByte('\n')
	}

	if isSolanaChain(chain) {
		if solRows := solanaTokenAccountRows(data); len(solRows) > 0 {
			b.WriteString("Solana Token Accounts\n\n")
			limit := 5
			if len(solRows) < limit {
				limit = len(solRows)
			}
			for i := 0; i < limit; i++ {
				r := solRows[i]
				fmt.Fprintf(&b, "  - token_account=%s mint_account=%s token_address=%s\n",
					r.tokenAccount, r.mintAccount, r.tokenAddress)
			}
			b.WriteByte('\n')
		}
	}

	rest := compactJSONFallback(data)
	if rest != "" {
		b.WriteString("Full JSON\n\n")
		b.WriteString(rest)
	}
	return strings.TrimSpace(b.String())
}

func formatAddressTransactionsPretty(data map[string]interface{}) string {
	var b strings.Builder
	b.WriteString("Address Transactions\n\n")
	writeKV(&b, "", "address", stringField(data, "address"))
	writeKV(&b, "", "chain", stringField(data, "chain"))
	writeKV(&b, "", "tx_type", stringField(data, "tx_type"))
	writeKV(&b, "", "total", formatFloatish(data["total"]))
	writeKV(&b, "", "count", formatFloatish(data["count"]))
	b.WriteByte('\n')

	items := txItems(data)
	if len(items) == 0 {
		b.WriteString("No transactions in this page.\n")
		return strings.TrimSpace(b.String())
	}

	b.WriteString("Items\n\n")
	limit := 5
	if len(items) < limit {
		limit = len(items)
	}
	for i := 0; i < limit; i++ {
		it := items[i]
		hash := firstNonEmpty(stringField(it, "hash"), stringField(it, "signature"))
		fmt.Fprintf(&b, "  - hash=%s from=%s to=%s value=%s value_usd=%s tx_time=%s\n",
			hash,
			truncateField(stringField(it, "from"), 18),
			truncateField(stringField(it, "to"), 18),
			stringField(it, "value"),
			stringField(it, "value_usd"),
			formatFloatish(it["tx_time"]),
		)
		if st := stringField(it, "tx_status"); st != "" {
			fmt.Fprintf(&b, "    tx_status=%s\n", st)
		}
		if n := len(utxoSlice(it, "inputs")); n > 0 {
			fmt.Fprintf(&b, "    inputs: %d utxo(s)\n", n)
		}
		if n := len(utxoSlice(it, "outputs")); n > 0 {
			fmt.Fprintf(&b, "    outputs: %d utxo(s)\n", n)
		}
	}
	if len(items) > limit {
		fmt.Fprintf(&b, "\n  … and %d more (use --format json)\n", len(items)-limit)
	}
	return strings.TrimSpace(b.String())
}

type tokenBalRow struct {
	usdSort float64
	line    string
}

func tokenBalanceRows(data map[string]interface{}) []tokenBalRow {
	raw, ok := data["token_balances"].([]interface{})
	if !ok || len(raw) == 0 {
		return nil
	}
	rows := make([]tokenBalRow, 0, len(raw))
	for _, x := range raw {
		m, ok := x.(map[string]interface{})
		if !ok {
			continue
		}
		usd, sortKey := usdSortKey(m["value_usd"])
		sym := stringField(m, "symbol")
		addr := stringField(m, "token_address")
		if addr == "" {
			addr = "null"
		}
		price := firstNonEmpty(stringField(m, "price"), stringField(m, "price_usd"))
		vpct := stringField(m, "value_percent")
		line := fmt.Sprintf("symbol=%s token_address=%s amount=%s value_usd=%s",
			sym, addr, stringField(m, "amount"), usd)
		if price != "" {
			line += fmt.Sprintf(" price=%s", price)
		}
		if vpct != "" {
			line += fmt.Sprintf(" value_percent=%s", vpct)
		}
		rows = append(rows, tokenBalRow{
			usdSort: sortKey,
			line:    line,
		})
	}
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].usdSort > rows[j].usdSort })
	return rows
}

type multiChainRow struct {
	usdSort  float64
	chain    string
	symbol   string
	amount   string
	usdValue string
	pct      string
}

func multiChainRows(data map[string]interface{}) []multiChainRow {
	raw, ok := data["multi_chain_token_balances"].([]interface{})
	if !ok || len(raw) == 0 {
		return nil
	}
	out := make([]multiChainRow, 0, len(raw))
	for _, x := range raw {
		m, ok := x.(map[string]interface{})
		if !ok {
			continue
		}
		_, sortKey := usdSortKey(m["usd_value"])
		out = append(out, multiChainRow{
			usdSort:  sortKey,
			chain:    stringField(m, "chain"),
			symbol:   firstNonEmpty(stringField(m, "token_symbol"), stringField(m, "symbol")),
			amount:   stringField(m, "amount"),
			usdValue: formatFloatish(m["usd_value"]),
			pct:      formatFloatish(m["percentage"]),
		})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].usdSort > out[j].usdSort })
	return out
}

type solanaAcctRow struct {
	tokenAccount string
	mintAccount  string
	tokenAddress string
}

func solanaTokenAccountRows(data map[string]interface{}) []solanaAcctRow {
	raw, ok := data["token_balances"].([]interface{})
	if !ok {
		return nil
	}
	var out []solanaAcctRow
	for _, x := range raw {
		m, ok := x.(map[string]interface{})
		if !ok {
			continue
		}
		ta := stringField(m, "token_account")
		ma := stringField(m, "mint_account")
		if ta == "" && ma == "" {
			continue
		}
		addr := stringField(m, "token_address")
		if addr == "" {
			addr = "null"
		}
		out = append(out, solanaAcctRow{tokenAccount: ta, mintAccount: ma, tokenAddress: addr})
	}
	return out
}

func txItems(data map[string]interface{}) []map[string]interface{} {
	if raw, ok := data["items"].([]interface{}); ok && len(raw) > 0 {
		return mapsFromSlice(raw)
	}
	if raw, ok := data["transactions"].([]interface{}); ok && len(raw) > 0 {
		return mapsFromSlice(raw)
	}
	return nil
}

func mapsFromSlice(raw []interface{}) []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(raw))
	for _, x := range raw {
		if m, ok := x.(map[string]interface{}); ok {
			out = append(out, m)
		}
	}
	return out
}

func writeTokenBalanceRow(b *strings.Builder, r tokenBalRow) {
	b.WriteString("  - ")
	b.WriteString(r.line)
	b.WriteByte('\n')
}

func writeKV(b *strings.Builder, prefix, key, val string) {
	if val == "" {
		return
	}
	fmt.Fprintf(b, "%s%s: %s\n", prefix, key, sanitizeTerminalText(val))
}

func stringField(m map[string]interface{}, key string) string {
	if m == nil {
		return ""
	}
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	case float64:
		if t == float64(int64(t)) {
			return fmt.Sprintf("%d", int64(t))
		}
		return fmt.Sprintf("%v", t)
	case int:
		return fmt.Sprintf("%d", t)
	case int64:
		return fmt.Sprintf("%d", t)
	case bool:
		if t {
			return "true"
		}
		return "false"
	default:
		return strings.TrimSpace(fmt.Sprint(v))
	}
}

func stringSliceField(m map[string]interface{}, key string) []string {
	raw, ok := m[key].([]interface{})
	if !ok {
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, x := range raw {
		if s, ok := x.(string); ok && strings.TrimSpace(s) != "" {
			out = append(out, strings.TrimSpace(s))
		}
	}
	return out
}

func formatFloatish(v interface{}) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	case float64:
		return fmt.Sprintf("%v", t)
	case int:
		return fmt.Sprintf("%d", t)
	case int64:
		return fmt.Sprintf("%d", t)
	case bool:
		if t {
			return "true"
		}
		return "false"
	default:
		return strings.TrimSpace(fmt.Sprint(v))
	}
}

func usdSortKey(v interface{}) (string, float64) {
	s := formatFloatish(v)
	if s == "" {
		return "", 0
	}
	var f float64
	if _, err := fmt.Sscanf(s, "%f", &f); err == nil {
		return s, f
	}
	return s, 0
}

func isSolanaChain(chain string) bool {
	c := strings.ToLower(strings.TrimSpace(chain))
	return c == "sol" || c == "solana"
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func truncateField(s string, max int) string {
	s = strings.TrimSpace(s)
	if max <= 0 || len(s) <= max {
		return s
	}
	return s[:max] + "…"
}

func utxoSlice(it map[string]interface{}, key string) []interface{} {
	raw, ok := it[key].([]interface{})
	if !ok {
		return nil
	}
	return raw
}

func compactJSONFallback(data map[string]interface{}) string {
	// Shown only when no structured sections were written (empty profile).
	return ""
}
