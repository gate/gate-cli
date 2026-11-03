# Changelog

## [v0.9.1] - 2026-11-03

### Fixed

- `cex futures trail stop-all` preserves cancelled orders in `data.orders`, together with the response `message` and `timestamp`. A nonzero `code` is reported as an API error.

## [v0.9.0] - 2026-11-03

### Added

- `cex auth login`, `status`, `refresh` and `logout`: browser or manual authorization, explicit scope selection, resource-bound credentials and local status fields `issuer`, `resource`, `scopes`, `expires_at` and `remote_checked`.
- `cex stock`: market, account, transfer, order and position commands with exact int64 identifiers. CFD queries and transfers expose their existing `cex tradfi` parameters, including symbol commissions and order logs.
- `cex bot smart-position create` and `futures-combo create`; smart-position stop modes `market`, `limit` and `no_close`, plus `--stop-options`; recommendation queries accept `--exclude-recommendation-id`.
- Spot POV create/list/get/cancel/cancel-all, spot/futures RPI order books, futures `contracts-all` and `adl-risk-states`, and futures chase create/get/list/stop/stop-all commands.
- Unified Delta-neutral get/set, quick-repayment estimate/execute and account leverage updates.
- `version --details --format json` exposes the public fields `version`, `revision`, `go`, `os`, `arch` and `agent_commands`. `config check` reports local configuration checks; `config init --hide-input` hides terminal input.
- Spot `order buy`/`sell` and futures `order long`/`short`/`add`/`remove` accept optional `--text`; futures `order get-by-text --text` retrieves the matching order.
- Options `order create --mmp` and CFD `order create --leverage`; Agent `agent-index --domain cex --details` describes registered command parameters and effects.

### Changed

- `config list` hides all custom header values and endpoint userinfo, query and fragment components by default, including aliases. Intel debug errors hide credential-bearing URLs; `info/news mcp-spec` and leaf help expose public command contracts.

- Spot order account selection, automatic borrowing/repayment and attached `stop_profit`/`stop_loss` parameters; options full-close and reduce-only parameters retain explicit values.
- CEX authorization distinguishes `base`, `spot_trade`, `futures_trade`, `earn`, `cfd` and `stock` operations. CFD and stock trading grants remain independent; actual access depends on the granted permissions and account eligibility.
- Futures chase responses accept flat or `code`/`message`/`data` envelopes; active queries explicitly select `is_finished=false`. Empty list and stop-all responses retain `orders: []`.
- `cex earn auto-invest min-amount` requires integer `asset_type`: `1` for cryptocurrency or `2` for TradFi stocks. Fixed-product queries accept `--status` and `--show-status`; fixed lends/history accept `--sub-business` and `--business-filter`, preserving explicit zero/empty values.
- `cex options account get` returns `position_notional_limit` as a string. Futures/delivery insurance responses preserve exact `b` balances as strings.
- `cex tradfi market kline --type` defaults to `1m`; supports the documented timeframes and aliases. Spot insurance queries require positive, ordered `--from`/`--to` timestamps.
- Delivery price-trigger creation requires explicit `--trigger-rule`; futures price-trigger creation accepts optional `--tif`. Spot price-trigger creation accepts explicit `--time-in-force`, defaulting to `ioc`.
- Spot and futures batch order operations preserve per-item results on stdout and fail on partial or malformed results. Empty arrays, ambiguous duplicate keys and malformed attached-protection objects are rejected before writes.
- `info`/`news --args-file` must resolve inside the current working directory, including absolute, home-relative and symbolic-link paths. Unknown command paths and unexpected positional arguments return a nonzero status; spaced native boolean values preserve explicit false.
- CEX responses above 8 MiB or undecodable successful JSON fail explicitly. The independent `--max-output-bytes` option continues to control printed output.

### Fixed

- `cex futures trail create` requires explicit integer `--price-type`: `1` latest, `2` index or `3` mark. Trail identifiers and declared time fields remain strings.
- Trail list returns `data.orders`; update and stop return `data.order`; log returns `data.change_log` with string `updated_at` and explicitly requests page one when pagination is omitted. Declared nonzero business codes fail even for HTTP 200.
- Futures price-trigger detail HTTP 205 preserves the backend error. Unsupported API-order TP/SL amendment returns `UNSUPPORTED_OPERATION`, `retryable: false` and an action to stop retrying.
- Futures long orders reject negative size; add/close preserve signed-position behavior and validate dual-position `--side`. Writes stop when position mode cannot be determined.
- Missing CEX routes return `API_NOT_FOUND`; table-format errors remain visible, including missing-credential `AUTH_ERROR` with exit code 4. Manual authorization accepts CRLF input.

### Removed

- `cex launch` and its LaunchPool, HODLer Airdrop and CandyDrop query/participation/record commands.
- `cex sub-account` administration and API-key commands; wallet balance and transfer commands remain available.
- P2P, CrossEx, activity, coupon and welfare command groups and CEX Exchange Announcement API access. Separate `news` commands remain available.

## [v0.8.0]

### Fixed

- **Intel MCP argument isolation**: `info` / `news` and `invoke` keep CLI-only flags (`--format`, `--profile`, `--debug`, `--verbose`, `--max-output-bytes`, `--api-key`, `--api-secret`, `--help`) separate from tool arguments. Explicit keys in `--params`, `--args-json` and `--args-file` are preserved.

## [v0.7.9]

### Added

- `news feed get-mention-burst` / `get-hot-topics`, with `mention-burst` / `hot-topics` aliases; market-move report get/list commands under `news events`, including `market-move-report`, `get-report`, `market-move-reports` and `report-list` aliases. Report queries use `--symbol`.
- `info platformmetrics get-chain-activity` for Ethereum staking activity. Required `--metric-group staking`; optional `--chain`, `--start-date`, `--end-date` and `--lookback` (`30d`, `90d`, `1y`, default `30d`). Responses include `staking_metrics.series[]` fields `eth_supply`, `staking_apr_7d`, `entry_wait_days` and `exit_wait_days`.
- Info shortcuts `+coin-overview`, `+market-overview`, `+coin-compare`, `+trend-analysis`, `+token-risk`, `+address-tracker` and `+token-onchain`; News shortcuts `+brief`, `+event-explain` and `+community-scan`. Partial on-chain results expose unavailable-section hints.
- `agent-leaves`, `agent-search --query`, `agent-index`, `agent-resolve --query` and `agent-validate` for command discovery. Search/index accept `--domain`; result metadata includes `match_source`, `is_shortcut`, `resolved_leaves`, `count_curated`, `scope` and `recommended_flow`.
- In Agent mode, discovery defaults to Info/News when `--domain` is omitted. `info list` / `news list` return compact `{name, path}` entries in Agent JSON mode; describe output includes `{name, description, has_input_schema}`.
- Agent responses expose `{data, meta}`, freshness hints and `meta.freshness_summary`; missing timestamps produce `freshness_status_cli=unknown`. Preflight, doctor and migrate failures remain stderr-only and provide next-action hints.
- `cex earn uni +redeem-records` / `+lends`, `cex spot market +ticker` and `cex alpha market +tickers` shortcuts.

### Changed

- In Agent mode (`GATE_CLI_AGENT=1` or `GATE_AI_AGENT=1`), `--max-output-bytes` defaults to 65536 when `GATE_MAX_OUTPUT_BYTES` is unset. Parent `--help` is blocked by default with command-discovery guidance; `info` / `news` list and describe help remain available.
- `info markettrend get-kline` (`size` / `limit`) and `info marketdetail get-kline` (`limit`) default to 200 bars, capped at 500.
- News queries validate symbols, time windows (`24h`, `4h`), platforms, limits and report time ranges. ISO 8601 and `YYYY-MM-DD HH:MM:SS` are accepted; timezone-less times use UTC0.
- `info macro get-economic-calendar` accepts omitted dates. Coin rankings accept `time_range` only for top gainers/losers and `listing_*` only for new listings. DeFi overview accepts free-form `category`.
- Coin rankings add `market_pulse_hot`; yield-pool queries add `--scope basic|full`.
- Stablecoin queries expose `scope`, `sections`, date, symbol, chain and limit selectors, including `issuance_flow` and `usage_structure`; default `limit` is 10, maximum 400. Institutional metrics queries support asset, channel and date filters. Indicator history no longer accepts obsolete `source` / `quote` parameters.
- Platform-info queries support `include_oi_symbol_detail` / `oi_symbol_limit`. Exchange-reserve queries support `scope`, `include_history`, `history_window` and the documented asset values.
- News prediction/search queries validate required fields, venue/category/status, sorting, depth and limits. `get-event-signal` deprecates `include_orderbook_summary`; search-event status is omitted when unspecified.

### Fixed

- `+token-risk --symbol` resolves symbol queries, address lists, and multi-chain priorities. Native assets without a contract return a clear error. `get-coin-info` shortcuts use explicit `query` / `query_type`; address shortcuts use `query_type=address`.
- `get-coin-info` no longer forces a scope. Token-risk queries can request detailed results; market overview and token-security shortcuts accept basic results when full results are unavailable.
- Errors and help show public CLI paths. Unknown flags and missing required arguments produce visible structured errors; misplaced commands provide corrected paths.
- `agent-resolve --domain` filters results and removes duplicate command paths. Discovery output describes unavailable address-risk functionality without advertising it as runnable.
- News/Info argument checks cover search, market snapshots, on-chain queries, economic data and platform metrics. Batch market snapshots require `symbols` with at most 20 entries; security checks require exactly one token or address.
- `get-platform-history` accepts either `platform_name` or `exchange_slug`. Empty structured responses no longer hide nonempty result data.
- Doctor/migrate failures preserve the first failing check, stderr-only failure output and nonzero status. Agent error metadata includes `retryable` and `suggested_next_action`.

## [v0.7.3] - 2026-05-13

### Fixed

- Info/News boolean flags accept bare flags, `--flag=value` and legacy `--flag true|false`, preserving explicit false without consuming the next flag.
- Info/News, preflight, doctor and migrate flag errors remain visible on stderr.
- News search and prediction queries validate their required arguments, enums, ranges and venue-specific filters. `search-x` and `web-search` retain documented defaults; prediction `limit` has minimum 1.

### Added

- `news prediction get-market-orderbook`, `search-events` and `get-event-signal`.

### Changed

- News ranking `category` accepts a free-form string. Search-event `status` is omitted when unspecified; order-book `market_id` follows venue semantics. `include_orderbook_summary` is deprecated for event signals.

## [v0.7.2] - 2026-05-12

### Added

- `news events explain-market-move` requires `--query` and `--coin`; optional `--time-range`, `--mode` and `--lang`.

### Changed

- `info platformmetrics get-platform-info` adds `include_oi_symbol_detail` and `oi_symbol_limit`.
- `info platformmetrics get-exchange-reserves` replaces `period` with `scope`, `include_history` and `history_window`, and documents accepted `asset` values.

### Fixed

- `info platformmetrics get-platform-history` accepts either `platform_name` or `exchange_slug`; requests missing both return `INVALID_ARGUMENTS`.

## [v0.7.0] - 2026-05-06

### Added

- `cex bot` recommendation, running-strategy, detail and stop commands; spot, margin, infinite and futures grid creation; spot and contract martingale creation. Recommendation filters include market, strategy type, direction, investment amount, scene, refresh recommendation ID, limits, maximum drawdown and backtest APR.
- `cex cross-ex account book --statement-type`; spot account-book responses add `code` alongside the deprecated `type` field.

### Changed

- `cex p2p chat list --txid 0` returns the latest order with chat for the current user.
- Cross-exchange account-book output uses `statement_type` and the `Statement Type` heading.
- P2P advertisement status accepts `1=listed`, `3=delisted`, `4=closed`.
- Contract martingale creation does not support `stop_loss_price`; omit it from `--json`.
- P2P transaction confirm-payment, confirm-receipt and cancel requests use `txid` instead of `trade_id`. CLI flag names retain their meaning.

### Removed

- `cex p2p ads update-status --trade-type`; scripts must omit this flag.

## [v0.6.8]

### Changed

- News prediction volume-delta and fastest-rising rankings accept optional free-form `category`.

## [v0.6.7] - 2026-04-30

### Added

- `news prediction get-volume-delta-ranking` and `get-fastest-rising-ranking`.

### Fixed

- Info/News calls reject mismatched response IDs and invalid initialization results. Describe queries continue to return consistent results after temporary tool-list failures.

## [v0.6.5] - 2026-04-22

### Added

- `cex assetswap` assets, config, evaluate, order create/preview/list/get commands. Evaluation supports `--max-value`, `--cursor`, `--size`; order lists support time, status, offset, size, sorting and ordering filters.
- `cex launch hodler` projects, order, order-records and airdrop-records; project queries support status, keyword, join and pagination, and records support keyword, time range and pagination.
- `cex launch candy-drop` activities, rules, register, progress, participations and airdrops; registration accepts currency and optional activity ID.
- `cex earn dual refund-preview`, `refund`, `modify-reinvest` and `recommend`. Refund uses `--order-id` and the preview's `--req-id`; reinvest supports status and duration; recommendations support mode, coin, type and history IDs.
- One-way futures position get, update-margin, update-leverage, update-cross-mode and update-risk-limit; market `risk-limit-table <table-id>`.
- `cex wallet sub --page --limit` pagination.

### Changed

- Hedge-mode position commands are renamed; scripts must use the corresponding dual-mode names:

| Previous command | Dual-mode command |
| --- | --- |
| `cex futures position get` | `cex futures position get-dual` |
| `cex futures position update-margin` | `cex futures position update-dual-margin` |
| `cex futures position update-leverage` | `cex futures position update-dual-leverage` |
| `cex futures position update-cross-mode` | `cex futures position update-dual-cross-mode` |
| `cex futures position update-risk-limit` | `cex futures position update-dual-risk-limit` |

## [0.6.4] - 2026-04-22

### Added

- `info platformmetrics get-cex-orderbook-depth`: required `symbol`; optional `market_type` (`perp`, `spot`, `perps`, `futures`, `future`), `exchange`, `data_scope` (`exchange`, `market`) and `limit` (default 20, maximum 100).

### Changed

- Platform search `sort_by` adds `volume_perps_7d`, `volume_perps_30d` and `volume_perps_qtd`.
- DeFi overview `category` adds `dexs` alongside `dex` / `dexes`.
- Platform-history queries add `exchange_slug` and `granularity` (`day`, `week`, `month`, `quarter`, default `day`).

### Fixed

- `doctor` / `preflight` report `version_ok` against the minimum migration-compatible CLI version 0.6.0.

## [0.6.1] - 2026-04-20

### Added

- `info mcp-spec` / `news mcp-spec` provide offline public input contracts in JSON or pretty format; table format is unsupported.
- `GATE_INTEL_LEAF_HELP=full|detailed` enables detailed field descriptions in leaf help, including enums, defaults, bounds and patterns.

### Changed

- Info/News flags expose documented field types and constraints, including query, scope, sorting, time, platform, pagination and handle filters.

### Fixed

- Info/News argument failures return HTTP 400 with `INVALID_ARGUMENTS`; service failures retain HTTP 502 with `INTEL_RESULT_ERROR`.
- Errors retain concise, redacted messages. Empty structured responses no longer hide available text results.

## [0.6.0] - 2026-04-20

### Changed

- Trading and account commands move under `gate-cli cex`: use `gate-cli cex spot`, `cex futures`, `cex wallet`, `cex earn`, `cex margin`, `cex unified`, and the corresponding domain groups. Top-level config, Info/News, preflight, doctor and migrate commands retain their names.

## [0.5.2] - 2026-04-18

### Removed

- Unsupported Info contracts for smart-money, entity-profile, fund-flow tracing, address-risk checks and regulatory-update search.

## [0.5.0] - 2026-04-17

### Added

- Margin, unified, sub-account, earn, flash-swap, multi-collateral lending, cross-exchange, P2P, rebate, withdrawal, activity, coupon, launch, square and welfare command groups.
- Request `User-Agent` includes CLI version, command and detected environment. `GATE_CLI_AGENT` / `GATE_CLI_AGENT_VERSION` allow explicit environment identification.
- Leverage commands expose `--pos-margin-mode` / `--dual-side`; price-trigger order identifiers accept 64-bit values.

### Changed

- Info/News default output is `pretty`; scripts can request `--format json`. `GATE_CLI_SUPPRESS_FORMAT_NOTICE=1` suppresses the compatibility notice.
- `--refresh-schema` is replaced by `GATE_INTEL_REFRESH_SCHEMA=1` for a one-off schema refresh.
- ETH2 swap/rate and structured-product/order interfaces are retired in favor of fixed-term and auto-invest interfaces.

### Fixed

- Explicit empty list arguments remain empty. Nonempty Intel bearer tokens shorter than eight characters are rejected.
- Relative `--args-file` paths stay within the working directory. Schema updates preserve file permissions.
- HTTP response-size errors distinguish `GATE_INTEL_MAX_RESPONSE_BYTES` from the printed-output limit `--max-output-bytes`.
- Format notices are emitted only on interactive stderr, keeping pipelines quiet. Coin-info queries accept `--symbol`.

## [0.4.1]

### Fixed

- `brew tap gate/tap` discovers the `gate-cli` formula correctly.

## [0.4.0]

### Added

- Unix and Windows installation commands with checksum verification; Unix defaults to `$HOME/.local/bin` and provides a PATH hint.
- `brew install gate/tap/gate-cli`.

### Fixed

- Installation rejects checksum failures and handles Windows CRLF input without duplicating PATH entries.

## [0.3.2]

### Added

- `--version` and `version` report the same CLI version.

## [0.2.2]

### Fixed

- `wallet transfer` infers futures settlement currency from the currency symbol.
- Dual-position `futures position update-margin` requires `--dual-side`.

## [0.2.1]

### Changed

- Delivery commands remove `--settle` and use USDT settlement only.

## [0.2.0]

### Added

- Futures market/order/position/Trail/price-trigger commands; spot batch orders, countdown cancellation and cross-liquidation; delivery contracts/orders/positions/price-trigger commands; options underlyings/contracts/orders/positions/MMP; wallet balances/deposits/withdrawals/transfers/small balances; account details/rate limits/STP/debit fees; Alpha and TradFi market/account/trading commands.

### Changed

- Futures position commands support single- and dual-position modes.

## [0.1.4]

### Fixed

- Spot and futures market orders use `time_in_force=ioc`.

## [0.1.3]

### Fixed

- Error responses retain the request trace ID.

## [0.1.1]

### Added

- API requests identify the CLI through `User-Agent`.

### Fixed

- `config set` uses `default_profile` when `--profile` is omitted.

## [0.1.0]

### Added

- Spot/futures market-data, order and position commands; multi-profile configuration; table and JSON output; futures dual-position support.
