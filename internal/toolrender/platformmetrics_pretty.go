package toolrender

import (
	"fmt"
	"strings"
)

const toolInfoPlatformmetricsGetChainActivity = "info_platformmetrics_get_chain_activity"

func prettyPlatformmetricsToolResult(toolName string, data map[string]interface{}) (string, bool) {
	if data == nil {
		return "", false
	}
	switch toolName {
	case toolInfoPlatformmetricsGetChainActivity:
		return formatChainActivityPretty(data), true
	default:
		return "", false
	}
}

func formatChainActivityPretty(data map[string]interface{}) string {
	var b strings.Builder
	b.WriteString("Chain Activity (Staking)\n\n")

	chain := firstNonEmpty(stringField(data, "normalized_chain"), stringField(data, "chain"))
	writeKV(&b, "", "chain", chain)
	writeKV(&b, "", "metric_group", stringField(data, "metric_group"))
	writeKV(&b, "", "start_date", stringField(data, "start_date"))
	writeKV(&b, "", "end_date", stringField(data, "end_date"))
	writeKV(&b, "", "lookback", stringField(data, "lookback"))
	writeKV(&b, "", "total", stringField(data, "total"))
	writeKV(&b, "", "count", stringField(data, "count"))
	writeKV(&b, "", "data_status", stringField(data, "data_status"))
	writePrettyBoolFlag(&b, data, "range_truncated")
	writePrettyBoolFlag(&b, data, "start_date_capped")
	writePrettyBoolFlag(&b, data, "end_date_capped")
	if strings.TrimSpace(b.String()) != "Chain Activity (Staking)\n\n" {
		b.WriteByte('\n')
	}

	series := chainActivitySeries(data)
	if len(series) == 0 {
		b.WriteString("Staking Series\n\n")
		b.WriteString("  No series points in response.\n")
		return strings.TrimSpace(b.String())
	}

	b.WriteString("Latest Snapshot\n\n")
	writeChainActivityPoint(&b, "  ", series[0])
	b.WriteByte('\n')

	b.WriteString("Recent Series\n\n")
	limit := 7
	if len(series) < limit {
		limit = len(series)
	}
	for i := 0; i < limit; i++ {
		writeChainActivitySeriesLine(&b, series[i])
	}
	return strings.TrimSpace(b.String())
}

func chainActivitySeries(data map[string]interface{}) []map[string]interface{} {
	sm, ok := data["staking_metrics"].(map[string]interface{})
	if !ok {
		return nil
	}
	raw, ok := sm["series"].([]interface{})
	if !ok {
		return nil
	}
	return mapsFromSlice(raw)
}

func writeChainActivityPoint(b *strings.Builder, prefix string, pt map[string]interface{}) {
	writeKV(b, prefix, "date", stringField(pt, "date"))
	writeKV(b, prefix, "validator_active", stringField(pt, "validator_active"))
	writeKV(b, prefix, "total_value_staked_eth", stringField(pt, "total_value_staked_eth"))
	writeKV(b, prefix, "staking_rate", stringField(pt, "staking_rate"))
	writeKV(b, prefix, "eth_supply", stringField(pt, "eth_supply"))
	writeKV(b, prefix, "staking_apr_7d", stringField(pt, "staking_apr_7d"))
	writeKV(b, prefix, "entry_queue_eth", stringField(pt, "entry_queue_eth"))
	writeKV(b, prefix, "exit_queue_eth", stringField(pt, "exit_queue_eth"))
	writeKV(b, prefix, "entry_wait_days", stringField(pt, "entry_wait_days"))
	writeKV(b, prefix, "exit_wait_days", stringField(pt, "exit_wait_days"))
	writeKV(b, prefix, "entry_queue_validator_estimate", stringField(pt, "entry_queue_validator_estimate"))
	writeKV(b, prefix, "exit_queue_validator_estimate", stringField(pt, "exit_queue_validator_estimate"))
	writeKV(b, prefix, "data_status", stringField(pt, "data_status"))
	writeKV(b, prefix, "quality_note", stringField(pt, "quality_note"))
	if missing := stringSliceField(pt, "missing_fields"); len(missing) > 0 {
		writeKV(b, prefix, "missing_fields", strings.Join(missing, ", "))
	}
}

func writeChainActivitySeriesLine(b *strings.Builder, pt map[string]interface{}) {
	date := stringField(pt, "date")
	if date == "" {
		date = "?"
	}
	parts := []string{
		fmt.Sprintf("date=%s", date),
		kvPart("validators", stringField(pt, "validator_active")),
		kvPart("staked_eth", stringField(pt, "total_value_staked_eth")),
		kvPart("rate", stringField(pt, "staking_rate")),
		kvPart("eth_supply", stringField(pt, "eth_supply")),
		kvPart("apr_7d", stringField(pt, "staking_apr_7d")),
		kvPart("entry_q_eth", stringField(pt, "entry_queue_eth")),
		kvPart("exit_q_eth", stringField(pt, "exit_queue_eth")),
		kvPart("entry_wait_d", stringField(pt, "entry_wait_days")),
		kvPart("exit_wait_d", stringField(pt, "exit_wait_days")),
	}
	line := "  - " + joinNonEmptyParts(parts)
	if status := stringField(pt, "data_status"); status != "" && status != "ok" {
		line += fmt.Sprintf(" status=%s", status)
	}
	if note := stringField(pt, "quality_note"); note != "" {
		line += fmt.Sprintf(" note=%s", truncateField(note, 48))
	}
	b.WriteString(line)
	b.WriteByte('\n')
}

func writePrettyBoolFlag(b *strings.Builder, data map[string]interface{}, key string) {
	v, ok := data[key]
	if !ok || v == nil {
		return
	}
	switch t := v.(type) {
	case bool:
		if t {
			writeKV(b, "", key, "true")
		}
	case string:
		if strings.EqualFold(strings.TrimSpace(t), "true") {
			writeKV(b, "", key, "true")
		}
	default:
		if s := formatFloatish(v); strings.EqualFold(s, "true") || s == "1" {
			writeKV(b, "", key, s)
		}
	}
}

func kvPart(label, val string) string {
	if val == "" {
		return ""
	}
	return label + "=" + val
}

func joinNonEmptyParts(parts []string) string {
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p != "" {
			out = append(out, p)
		}
	}
	return strings.Join(out, " ")
}
