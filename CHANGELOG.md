# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased] - v0.3.1

### Fixed
- Removed unused `newsObjAny` and dead `pruneGateMarkers` helper so `staticcheck` U1000 stays clean on Intel packages.
- **Intel MCP**: HTTP response body over the configured read limit now wraps a stable sentinel error for `errors.Is` (size-limit diagnostics).
- **Migration**: `backupFile` reads via `Open`+`Stat` on the same descriptor and caps read size (reduced TOCTOU vs `ReadFile`+`Stat`); `pruneGateMarkers` recurses into marker values before deleting keys when that helper is used.
- **CLI**: default `--format` compatibility notice prints only on an interactive stderr TTY (or when `GATE_CLI_FORMAT_NOTICE_FORCE` is set for tests); avoids noisy stderr in pipelines.
- **toolschema**: `IsBackendInvoked` documented as deprecated (argv sniffing); schema refresh pointers in `info`/`news` package comments.

### Added
- **`scripts/test-intel-scope.sh`**: run `go test` / `go vet` only on Intel/MCP/migrate-related packages plus `./cmd` (avoids scanning the full `./cmd/...` trading subtree during iteration).
- **Structured User-Agent header** with environment auto-detection
  - Format: `gate-cli/{version}/{command}/{agent}/{extra} {sdkUA}`
  - Auto-detects 9 environments: Claude Code, Cursor (Agent + CLI), Qoder, Antigravity, Trae, OpenCode, Codex (Desktop + CLI), Windsurf, JetBrains
  - Falls back to `TERM_PROGRAM` (e.g. iTerm, VSCode) or `terminal`
  - Supports explicit override via `GATE_CLI_AGENT` / `GATE_CLI_AGENT_VERSION` env vars
  - New package: `internal/useragent` with full unit tests
- **15 new modules** to align with gate-local-mcp feature coverage (~190 new commands):
  - `margin` — margin accounts, funding, cross-margin loans, uni lending, auto-repay, leverage (20 commands)
  - `unified` — unified account mode, borrowing, risk units, portfolio margin, collateral config (22 commands)
  - `sub-account` — sub-account CRUD, lock/unlock, API key management (11 commands)
  - `earn` — dual investment, staking, fixed-term lending, auto-invest plans, uni simple earn (37 commands)
  - `flash-swap` — instant token swaps, multi-currency many-to-one / one-to-many (11 commands)
  - `mcl` — multi-collateral borrowing, repayment, collateral management (12 commands)
  - `cross-ex` — cross-exchange trading, positions, orders, convert (31 commands)
  - `p2p` — merchant ads, transactions, chat, payment methods (17 commands)
  - `rebate` — partner/broker/agency commissions and transaction history (12 commands)
  - `withdrawal` — create withdrawal, push order (UID transfer), cancel (3 commands)
  - `activity` — platform activities and promotions (3 commands)
  - `coupon` — user coupons and details (2 commands)
  - `launch` — launch pool projects, pledge, redeem, records (5 commands)
  - `square` — AI search, live replay (2 commands)
  - `welfare` — user identity, beginner tasks (2 commands)
- Integration tests for all new modules
- Unit tests for all 15 new modules (command structure and flag validation)

### Changed
- **Intel (`info` / `news`)**: default root `--format` is `pretty` (was `table` in older snapshots). Scripts should pass `--format json` explicitly for stable machine output. Optional stderr notice can be suppressed with `GATE_CLI_SUPPRESS_FORMAT_NOTICE=1`.
- **Intel**: the `--refresh-schema` flag was removed; force a one-off schema refresh with environment variable `GATE_INTEL_REFRESH_SCHEMA=1` (see README Intel migration notes).
- **SDK upgraded** from `gateapi-go/v7` v7.2.40 to v7.2.57
  - `InlineObject` / `InlineObject1` replaced with named types (`UpdateDualCompPositionCrossModeRequest`, `AmendOptionsOrderRequest`)
  - `InlineResponse*` types replaced with semantic names (`TrailOrderResponse`, `CreateTrailOrderResponse`, etc.)
  - `GetLeverage` changed from Opts pattern to required params; new `--pos-margin-mode` and `--dual-side` flags added
  - Price trigger order ID type widened from `int32` to `int64`
  - `Currency2` renamed to `AlphaCurrency`, `Ticker2` to `TradFiTicker`
  - Earn: `SwapETH2`, `RateListETH2`, `ListStructuredProducts/Orders`, `PlaceStructuredOrder` removed upstream; replaced by fixed-term and auto-invest APIs
- **Go version** bumped from 1.21 to 1.23 (go.mod and CI workflow)
- **Client struct** extended with 17 new API service fields
- **README** rewritten with full module listing and categorized features

## [0.2.2]

### Fixed
- `wallet transfer`: auto-infer settle currency from currency symbol for futures transfers
- `futures position update-margin`: require `--dual-side` flag in dual-position mode

## [0.2.1]

### Fixed
- `delivery`: remove `--settle` flag, enforce USDT-only settlement

### Changed
- Updated README and quickstart guides for v0.2.0

## [0.2.0]

### Added
- `futures`: expand to full SDK coverage (market, order, position, trail, price-trigger)
- `spot`: expand to full SDK coverage (batch orders, countdown cancel, cross-liquidate)
- `delivery`: full API coverage (contracts, orders, positions, price-triggers)
- `options`: full API coverage (underlyings, contracts, orders, positions, MMP)
- `wallet`: full API coverage (balances, deposits, withdrawals, transfers, small balance)
- `account`: full API coverage (detail, rate limits, STP groups, debit fee)
- `alpha`: alpha token market data, account, orders
- `tradfi`: MT5 account, symbols, positions, orders, transactions

### Changed
- Refactored futures position commands to use `dual_comp` APIs for unified single/dual mode handling
- Grouped subcommands into category groups for options, delivery, wallet

## [0.1.4]

### Fixed
- Set `time_in_force=ioc` for all market orders (spot and futures)

## [0.1.3]

### Fixed
- `ParseGateError`: correctly extract trace ID from response header

## [0.1.2]

### Fixed
- Lower minimum Go version from 1.25.1 to 1.21

## [0.1.1]

### Added
- Custom User-Agent header for all API requests (`gate-cli/{version}`)

### Fixed
- `config set`: respect `default_profile` when `--profile` is not specified

## [0.1.0]

### Added
- Initial release
- Core modules: spot, futures with market data, orders, positions
- Config management with multi-profile support (file/env/flag priority)
- Dual output modes: table (human) and JSON (scripting)
- Transparent dual-position mode support for futures
- Integration test framework with testnet support
- GoReleaser pipeline for multi-platform builds (linux/darwin/windows, amd64/arm64)
