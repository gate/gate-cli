package info

import (
	"context"
	"errors"
	"strings"
	"sync"

	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/cmdutil"
	"github.com/gate/gate-cli/internal/intelcmd"
	"github.com/gate/gate-cli/internal/output"
	"github.com/gate/gate-cli/internal/toolrender"
)

func init() {
	buildInfoShortcuts()
}

func buildInfoShortcuts() {
	Cmd.AddCommand(
		newInfoCoinOverviewCmd(),
		newInfoMarketOverviewCmd(),
		newInfoCoinCompareCmd(),
		newInfoTrendAnalysisCmd(),
		newInfoTokenRiskCmd(),
		newInfoAddressTrackerCmd(),
		newInfoTokenOnchainCmd(),
	)
}

func newInfoCoinOverviewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "+coin-overview --symbol <symbol>",
		Aliases: []string{"coin-overview"},
		RunE: func(cmd *cobra.Command, args []string) error {
			symbol, err := requireShortcutSymbol(cmd, "symbol")
			if err != nil {
				return err
			}
			return runInfoShortcut(cmd, "info/+coin-overview", func(ctx context.Context, svc infoService) (map[string]interface{}, error) {
				coin, err := callCoinInfoForShortcut(ctx, svc, symbol)
				if err != nil {
					return nil, err
				}
				out := map[string]interface{}{
					"summary":         map[string]interface{}{"symbol": symbol},
					"basic_info":      coin,
					"market_snapshot": map[string]interface{}{},
					"technical_view":  map[string]interface{}{},
					"risk_flags":      []interface{}{},
				}
				var missing []string
				var mu sync.Mutex
				_ = intelcmd.RunParallel(intelcmd.DefaultShortcutParallelism, []func() error{
					func() error {
						if v, err := callMarketSnapshotForShortcut(ctx, svc, symbol); err == nil {
							mu.Lock()
							out["market_snapshot"] = v
							mu.Unlock()
						} else {
							mu.Lock()
							missing = append(missing, "market_snapshot")
							mu.Unlock()
						}
						return nil
					},
					func() error {
						if v, err := callInfoShortcutTool(ctx, svc, "info_markettrend_get_technical_analysis", map[string]interface{}{"symbol": symbol}); err == nil {
							mu.Lock()
							out["technical_view"] = v
							mu.Unlock()
						} else {
							mu.Lock()
							missing = append(missing, "technical_view")
							mu.Unlock()
						}
						return nil
					},
				})
				if len(missing) > 0 {
					out["partial"] = true
					out["missing_sections"] = missing
				}
				return out, nil
			})
		},
	}
	cmd.Flags().String("symbol", "", "Coin symbol (required)")
	return cmd
}

func newInfoMarketOverviewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "+market-overview",
		Aliases: []string{"market-overview"},
		RunE: func(cmd *cobra.Command, args []string) error {
			rawBench, _ := cmd.Flags().GetString("benchmark")
			benchmarks := splitCSVOrDefault(rawBench, []string{"BTC", "ETH", "SOL"})
			if len(benchmarks) > 5 {
				benchmarks = benchmarks[:5]
			}
			return runInfoShortcut(cmd, "info/+market-overview", func(ctx context.Context, svc infoService) (map[string]interface{}, error) {
				summary, err := callInfoShortcutTool(ctx, svc, "info_marketsnapshot_get_market_overview", map[string]interface{}{})
				if err != nil {
					return nil, err
				}
				out := map[string]interface{}{
					"market_summary":       summary,
					"benchmark_snapshots":  []interface{}{},
					"trend_anchor":         []interface{}{},
					"risk_watchlist":       []interface{}{},
					"requested_benchmarks": benchmarks,
					"partial":              false,
					"missing_sections":     []string{},
				}
				var snapshots []interface{}
				var anchors []interface{}
				var mu sync.Mutex
				snapshotTasks := make([]func() error, 0, len(benchmarks))
				for _, symbol := range benchmarks {
					symbol := symbol
					snapshotTasks = append(snapshotTasks, func() error {
						if v, err := callMarketSnapshotForShortcut(ctx, svc, symbol); err == nil {
							mu.Lock()
							snapshots = append(snapshots, map[string]interface{}{"symbol": symbol, "data": v})
							mu.Unlock()
						}
						return nil
					})
				}
				_ = intelcmd.RunParallel(intelcmd.DefaultShortcutParallelism, snapshotTasks)
				out["benchmark_snapshots"] = snapshots
				anchorLimit := minInt(2, len(benchmarks))
				anchorTasks := make([]func() error, 0, anchorLimit)
				for i := 0; i < anchorLimit; i++ {
					symbol := benchmarks[i]
					anchorTasks = append(anchorTasks, func() error {
						if v, err := callInfoShortcutTool(ctx, svc, "info_markettrend_get_technical_analysis", map[string]interface{}{"symbol": symbol}); err == nil {
							mu.Lock()
							anchors = append(anchors, map[string]interface{}{"symbol": symbol, "data": v})
							mu.Unlock()
						}
						return nil
					})
				}
				_ = intelcmd.RunParallel(intelcmd.DefaultShortcutParallelism, anchorTasks)
				out["trend_anchor"] = anchors
				var missing []string
				if len(snapshots) < len(benchmarks) {
					missing = append(missing, "benchmark_snapshots")
				}
				if len(anchors) < minInt(2, len(benchmarks)) {
					missing = append(missing, "trend_anchor")
				}
				if len(missing) > 0 {
					out["partial"] = true
					out["missing_sections"] = missing
				}
				return out, nil
			})
		},
	}
	cmd.Flags().String("benchmark", "BTC,ETH,SOL", "Comma-separated benchmark symbols")
	return cmd
}

func newInfoCoinCompareCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "+coin-compare --symbols BTC,ETH",
		Aliases: []string{"coin-compare"},
		RunE: func(cmd *cobra.Command, args []string) error {
			raw, _ := cmd.Flags().GetString("symbols")
			if strings.TrimSpace(raw) == "" {
				return intelcmd.FailAfterPrintError(getPrinter(cmd), output.InvalidArgsError("missing required flag: symbols"))
			}
			symbols := splitCSVOrDefault(raw, nil)
			if len(symbols) < 2 || len(symbols) > 5 {
				return intelcmd.FailAfterPrintError(getPrinter(cmd), output.InvalidArgsError("symbols must contain 2 to 5 items"))
			}
			return runInfoShortcut(cmd, "info/+coin-compare", func(ctx context.Context, svc infoService) (map[string]interface{}, error) {
				var matrix []interface{}
				var dropped []string
				for _, symbol := range symbols {
					row := map[string]interface{}{"symbol": symbol}
					okCount := 0
					var rowMu sync.Mutex
					_ = intelcmd.RunParallel(intelcmd.DefaultShortcutParallelism, []func() error{
						func() error {
							if v, err := callCoinInfoForShortcut(ctx, svc, symbol); err == nil {
								rowMu.Lock()
								row["basic_info"] = v
								okCount++
								rowMu.Unlock()
							}
							return nil
						},
						func() error {
							if v, err := callMarketSnapshotForShortcut(ctx, svc, symbol); err == nil {
								rowMu.Lock()
								row["market_snapshot"] = v
								okCount++
								rowMu.Unlock()
							}
							return nil
						},
						func() error {
							if v, err := callInfoShortcutTool(ctx, svc, "info_markettrend_get_technical_analysis", map[string]interface{}{"symbol": symbol}); err == nil {
								rowMu.Lock()
								row["technical_view"] = v
								okCount++
								rowMu.Unlock()
							}
							return nil
						},
					})
					if okCount >= 2 {
						matrix = append(matrix, row)
					} else {
						dropped = append(dropped, symbol)
					}
				}
				return map[string]interface{}{
					"compare_matrix":  matrix,
					"ranking_view":    map[string]interface{}{},
					"technical_diff":  map[string]interface{}{},
					"key_deltas":      map[string]interface{}{},
					"dropped_symbols": dropped,
				}, nil
			})
		},
	}
	cmd.Flags().String("symbols", "", "Comma-separated 2-5 symbols (required)")
	return cmd
}

func newInfoTrendAnalysisCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "+trend-analysis --symbol <symbol>",
		Aliases: []string{"trend-analysis"},
		RunE: func(cmd *cobra.Command, args []string) error {
			symbol, err := requireShortcutSymbol(cmd, "symbol")
			if err != nil {
				return err
			}
			return runInfoShortcut(cmd, "info/+trend-analysis", func(ctx context.Context, svc infoService) (map[string]interface{}, error) {
				ta, err := callInfoShortcutTool(ctx, svc, "info_markettrend_get_technical_analysis", map[string]interface{}{"symbol": symbol})
				if err != nil {
					return nil, err
				}
				out := map[string]interface{}{
					"trend_summary":      ta,
					"indicator_snapshot": ta,
					"price_context":      map[string]interface{}{},
					"watch_levels":       []interface{}{},
				}
				if snapshot, err := callMarketSnapshotForShortcut(ctx, svc, symbol); err == nil {
					out["price_context"] = snapshot
				} else {
					out["price_context_unavailable"] = true
				}
				return out, nil
			})
		},
	}
	cmd.Flags().String("symbol", "", "Coin symbol (required)")
	return cmd
}

func newInfoTokenRiskCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "+token-risk",
		Aliases: []string{"token-risk"},
		RunE: func(cmd *cobra.Command, args []string) error {
			symbol, _ := cmd.Flags().GetString("symbol")
			address, _ := cmd.Flags().GetString("address")
			chain, _ := cmd.Flags().GetString("chain")
			return runInfoShortcut(cmd, "info/+token-risk", func(ctx context.Context, svc infoService) (map[string]interface{}, error) {
				symbol = strings.ToUpper(strings.TrimSpace(symbol))
				address = strings.TrimSpace(address)
				chain = strings.TrimSpace(chain)
				if address == "" && symbol == "" {
					return nil, intelcmd.ShortcutArgsError("either symbol or address is required")
				}
				if address != "" && symbol != "" {
					return nil, intelcmd.ShortcutArgsError("provide either symbol or address, not both")
				}
				if address != "" {
					if chain == "" {
						return nil, intelcmd.ShortcutArgsError("chain is required when address is provided")
					}
					sec, err := callTokenSecurityForShortcut(ctx, svc, address, chain)
					if err != nil {
						return nil, err
					}
					tokenIdentity := map[string]interface{}{}
					partial := false
					missing := []string{}
					if coin, err := callInfoShortcutTool(ctx, svc, "info_coin_get_coin_info", map[string]interface{}{
						"query":      address,
						"query_type": "address",
						"scope":      "basic",
					}); err == nil {
						tokenIdentity = coin
					} else {
						partial = true
						missing = append(missing, "token_identity")
					}
					out := map[string]interface{}{
						"risk_level":     sec,
						"risk_items":     sec,
						"token_identity": tokenIdentity,
						"coverage_note":  "shortcut",
					}
					if partial {
						out["partial"] = true
						out["missing_sections"] = missing
					}
					return out, nil
				}
				coin, err := callCoinInfoForTokenRisk(ctx, svc, symbol)
				if err != nil {
					return nil, err
				}
				resolvedAddress, resolvedChain := resolveTokenAddressAndChain(coin, symbol)
				if resolvedAddress == "" || resolvedChain == "" {
					if coinInfoIndicatesNativeAsset(coin, symbol) {
						return nil, intelcmd.ShortcutArgsError("native coin has no contract address for token security; use +coin-overview or pass --address and --chain for a wrapped token")
					}
					return nil, intelcmd.ShortcutArgsError("failed to resolve canonical contract or chain from symbol; try --address and --chain")
				}
				sec, err := callTokenSecurityForShortcut(ctx, svc, resolvedAddress, resolvedChain)
				if err != nil {
					return nil, err
				}
				return map[string]interface{}{
					"risk_level":     sec,
					"risk_items":     sec,
					"token_identity": coin,
					"coverage_note":  "shortcut",
				}, nil
			})
		},
	}
	cmd.Flags().String("symbol", "", "Token symbol")
	cmd.Flags().String("address", "", "Token address")
	cmd.Flags().String("chain", "", "Chain")
	return cmd
}

func newInfoAddressTrackerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "+address-tracker --address <address> --chain <chain>",
		Aliases: []string{"address-tracker"},
		RunE: func(cmd *cobra.Command, args []string) error {
			address, chain, err := requireShortcutAddressChain(cmd)
			if err != nil {
				return err
			}
			minValue, _ := cmd.Flags().GetFloat64("min-value")
			if minValue < 0 {
				return intelcmd.FailAfterPrintError(getPrinter(cmd), output.InvalidArgsError("min-value must be non-negative"))
			}
			return runInfoShortcut(cmd, "info/+address-tracker", func(ctx context.Context, svc infoService) (map[string]interface{}, error) {
				profile, err := callInfoShortcutTool(ctx, svc, "info_onchain_get_address_info", map[string]interface{}{
					"address": address,
					"chain":   chain,
					"scope":   "with_defi",
				})
				if err != nil {
					return nil, err
				}
				out := map[string]interface{}{
					"entity_guess":          profile,
					"defi_context":          profile,
					"recent_activity":       map[string]interface{}{},
					"fund_flow_graph":       map[string]interface{}{},
					"fund_flow_unavailable": true,
					"watch_items":           []interface{}{},
					"coverage_note":         "partial: trace_fund_flow unavailable",
				}
				txArgs := map[string]interface{}{
					"address": address,
					"chain":   chain,
				}
				if minValue > 0 {
					txArgs["min_value_usd"] = minValue
				}
				if recent, err := callInfoShortcutTool(ctx, svc, "info_onchain_get_address_transactions", txArgs); err == nil {
					out["recent_activity"] = recent
				} else {
					out["recent_activity_unavailable"] = true
				}
				return out, nil
			})
		},
	}
	cmd.Flags().String("address", "", "Wallet address (required)")
	cmd.Flags().String("chain", "", "Chain (required)")
	cmd.Flags().Float64("min-value", 100000, "Minimum transaction value in USD for recent activity")
	cmd.Flags().Int("depth", 3, "Fund-flow depth (reserved; trace-fund-flow not yet available)")
	return cmd
}

func newInfoTokenOnchainCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "+token-onchain",
		Aliases: []string{"token-onchain"},
		RunE: func(cmd *cobra.Command, args []string) error {
			symbol, _ := cmd.Flags().GetString("symbol")
			address, _ := cmd.Flags().GetString("address")
			chain, _ := cmd.Flags().GetString("chain")
			return runInfoShortcut(cmd, "info/+token-onchain", func(ctx context.Context, svc infoService) (map[string]interface{}, error) {
				symbol = strings.ToUpper(strings.TrimSpace(symbol))
				address = strings.TrimSpace(address)
				chain = strings.TrimSpace(chain)
				if address == "" && symbol == "" {
					return nil, intelcmd.ShortcutArgsError("either symbol or address is required")
				}
				if address != "" && chain == "" {
					return nil, intelcmd.ShortcutArgsError("chain is required when address is provided")
				}
				if symbol != "" && address != "" {
					return nil, intelcmd.ShortcutArgsError("provide either symbol or address, not both")
				}
				var coin map[string]interface{}
				token := symbol
				onchainChain := chain
				if symbol != "" {
					if v, err := callCoinInfoForTokenRisk(ctx, svc, symbol); err == nil {
						coin = v
						if resolvedAddress, resolvedChain := resolveTokenAddressAndChain(v, symbol); resolvedAddress != "" {
							token = resolvedAddress
							if onchainChain == "" {
								onchainChain = resolvedChain
							}
						}
					}
				} else {
					token = address
				}
				onchain, err := callTokenOnchainForShortcut(ctx, svc, token, onchainChain)
				if err != nil {
					return nil, err
				}
				out := map[string]interface{}{
					"token_distribution":      onchain,
					"holder_structure":        onchain,
					"transfer_summary":        onchain,
					"activity_snapshot":       onchain,
					"smart_money_view":        map[string]interface{}{},
					"smart_money_unavailable": true,
					"coverage_note":           "partial: get_smart_money unavailable",
				}
				if coin != nil {
					out["coin_context"] = coin
				}
				return out, nil
			})
		},
	}
	cmd.Flags().String("symbol", "", "Token symbol")
	cmd.Flags().String("address", "", "Token contract address")
	cmd.Flags().String("chain", "", "Chain (required with address)")
	return cmd
}

func runInfoShortcut(cmd *cobra.Command, path string, runner func(ctx context.Context, svc infoService) (map[string]interface{}, error)) error {
	p := getPrinter(cmd)
	if p.IsTable() {
		return intelcmd.FailLeafUnsupportedTable(p, "info")
	}
	svc, err := newInfoService(cmd)
	if err != nil {
		return intelcmd.FailIntelClientInit(p, err, "info", "shortcut", "")
	}
	ctx, cancel := intelcmd.WithShortcutBudget(cmd.Context())
	defer cancel()
	out, err := runner(ctx, svc)
	if err != nil {
		ge := intelcmd.GateErrorFromShortcutErr(err, path)
		output.FillAgentErrorConvergence(ge)
		return intelcmd.FailAfterPrintError(p, ge)
	}
	return toolrender.RenderIntelPayload(p, path, out, cmdutil.GetMaxOutputBytes(cmd))
}

func callInfoShortcutTool(ctx context.Context, svc infoService, name string, args map[string]interface{}) (map[string]interface{}, error) {
	return intelcmd.CallShortcutTool(ctx, svc, name, args)
}

func callCoinInfoForShortcut(ctx context.Context, svc infoService, symbol string) (map[string]interface{}, error) {
	return callInfoShortcutTool(ctx, svc, "info_coin_get_coin_info", map[string]interface{}{
		"query":      symbol,
		"query_type": "symbol",
	})
}

func callCoinInfoForTokenRisk(ctx context.Context, svc infoService, symbol string) (map[string]interface{}, error) {
	coin, err := callCoinInfoForShortcut(ctx, svc, symbol)
	if err != nil {
		return nil, err
	}
	if addr, chain := resolveTokenAddressAndChain(coin, symbol); addr != "" && chain != "" {
		return coin, nil
	}
	detailed, err := callInfoShortcutTool(ctx, svc, "info_coin_get_coin_info", map[string]interface{}{
		"query":      symbol,
		"query_type": "symbol",
		"scope":      "detailed",
	})
	if err != nil {
		return coin, nil
	}
	return detailed, nil
}

func callMarketSnapshotForShortcut(ctx context.Context, svc infoService, symbol string) (map[string]interface{}, error) {
	snapshot, err := callInfoShortcutTool(ctx, svc, "info_marketsnapshot_get_market_snapshot", map[string]interface{}{
		"symbol": symbol,
		"scope":  "full",
	})
	if err == nil {
		return snapshot, nil
	}
	var isErr *intelcmd.ShortcutToolIsError
	if !errors.As(err, &isErr) {
		return nil, err
	}
	return callInfoShortcutTool(ctx, svc, "info_marketsnapshot_get_market_snapshot", map[string]interface{}{"symbol": symbol})
}

func callTokenOnchainForShortcut(ctx context.Context, svc infoService, token, chain string) (map[string]interface{}, error) {
	args := map[string]interface{}{
		"token": token,
		"scope": "full",
	}
	if chain != "" {
		args["chain"] = chain
	}
	return callInfoShortcutTool(ctx, svc, "info_onchain_get_token_onchain", args)
}

func callTokenSecurityForShortcut(ctx context.Context, svc infoService, address, chain string) (map[string]interface{}, error) {
	sec, err := callInfoShortcutTool(ctx, svc, "info_compliance_check_token_security", map[string]interface{}{
		"address": address,
		"chain":   chain,
		"scope":   "full",
	})
	if err == nil {
		return sec, nil
	}
	var isErr *intelcmd.ShortcutToolIsError
	if !errors.As(err, &isErr) {
		return nil, err
	}
	return callInfoShortcutTool(ctx, svc, "info_compliance_check_token_security", map[string]interface{}{
		"address": address,
		"chain":   chain,
	})
}

func requireShortcutAddressChain(cmd *cobra.Command) (address, chain string, err error) {
	address, _ = cmd.Flags().GetString("address")
	chain, _ = cmd.Flags().GetString("chain")
	address = strings.TrimSpace(address)
	chain = strings.TrimSpace(chain)
	if address == "" {
		return "", "", intelcmd.FailAfterPrintError(getPrinter(cmd), output.InvalidArgsError("missing required flag: address"))
	}
	if chain == "" {
		return "", "", intelcmd.FailAfterPrintError(getPrinter(cmd), output.InvalidArgsError("missing required flag: chain"))
	}
	return address, chain, nil
}

func requireShortcutSymbol(cmd *cobra.Command, flag string) (string, error) {
	raw, _ := cmd.Flags().GetString(flag)
	symbol := strings.ToUpper(strings.TrimSpace(raw))
	if symbol == "" {
		return "", intelcmd.FailAfterPrintError(getPrinter(cmd), output.InvalidArgsError("missing required flag: "+flag))
	}
	return symbol, nil
}

func splitCSVOrDefault(raw string, fallback []string) []string {
	if strings.TrimSpace(raw) == "" {
		if fallback == nil {
			return nil
		}
		return append([]string(nil), fallback...)
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.ToUpper(strings.TrimSpace(p)); t != "" {
			out = append(out, t)
		}
	}
	return out
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func firstStringByKeys(m map[string]interface{}, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k].(string); ok && strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
