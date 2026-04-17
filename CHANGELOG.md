# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [0.5.1] - 2026-04-17

### Changed
- **README**: add `info` / `news` to the command table; document the `--max-output-bytes` global flag; refresh `--verbose` / `--debug` descriptions for Intel backend transport.
- **README**: collapse stale "Intel MCP" / "Intel Migration Notes" sections into a concise "Intel (`info`, `news`)" pointer — behaviour, flags, and environment variables are now maintained in `gate-cli info -h` / `gate-cli news -h` and the `intel:` block of `config.yaml`.

## [0.5.0] - 2026-04-17

### Fixed
- **QC 收尾**：`internal/intelcmd/doc.go`；`toolconfig` deniedHeaders 字母序；`toolargs` `expandUserPath` 更清晰的 home 错误；`mcpclient` `callWithRetry` 重试语义 godoc；`migration` `atomicWritePreservePerm` 注释 Windows `Rename`；`toolrender` `ApplyOutputLimitWithData` 测量字节语义注释；scanner `rawLower` 命名。
- **`internal/toolargs`**：`stringSlice` 与 `stringArray` 一致，归一化后为「全空」时合并 **`[]string{}`** 到 MCP 参数（CR-806 / CR-405）；`TestMergeFromCommand_StringSliceExplicitEmptyPreserved`。
- **`internal/migration` scanner（CR-207）**：每个配置文件最多 **一次** JSON `Unmarshal`；合法 JSON 不再对全文做 `ToLower`，非 JSON 仍走子串检测。
- **Intel MCP HTTP**：默认 `User-Agent` 与交易 REST 同形态（`internal/useragent`，`intel/{backend}` + `jsonrpc` 后缀）；`GATE_INTEL_EXTRA_HEADERS` 若已设 `User-Agent` 则不覆盖。
- **`tools/list`（CR-309）**：缓存与返回值对 `Tool.InputSchema` 做 JSON 深拷贝，避免与解码结构或调用方原地修改共享指针。
- **Intel MCP errors**: response body over the HTTP read limit uses `errors.Is` messaging that separates **transport** `GATE_INTEL_MAX_RESPONSE_BYTES` from **`--max-output-bytes`** (printed output only) (CR-705).
- **Intel config**: non-empty bearer tokens must be at least 8 characters so trivial values like `"123"` fail fast at resolve time (QC / `toolconfig.Resolve`).
- **`tools/list` cache**: unit tests cover `shouldInvalidateListCacheOnListError` and recovery after a malformed list result invalidates the snapshot (CR-805).
- **`--args-file`**: relative paths must stay within the working directory (`filepath.IsLocal` after `Rel`); absolute paths unchanged (CR-1012).
- **Intel schema cache**: temp write uses `0o600`, then restores prior file permission when rewriting (CR-409); `Rename` failure removes `.tmp`.
- **CR-811**: Intel leaf aliases share `internal/intelcmd` (`NewLeafAliasCommand`, `LoadToolSchemasFromCache`, `AddFallbackArgFlags`, `MergeToolBaselineInto`, `ReservedMCPJSONFallbackFlags`); `cmd/info` and `cmd/news` only wire backend-specific baselines.
- **Info `info_coin_get_coin_info` (CR-1002)**: `--symbol` is defined in the frozen baseline input schema; removed the `AfterAliasBuilt` special-case in `cmd/info/aliases.go`.
- **`internal/toolrender` (CR-505)**: `ApplyOutputLimitWithData` returns display JSON for the truncated placeholder so Pretty mode avoids an extra `json.Marshal` of `data`.
- **`internal/migration` (CR-211)**: `readFromReaderLimited` centralizes bounded reads (`LimitReader(max+1)`); `scanner` and `backupFile` reuse it.
- **Root**: format compat notice line uses `io.WriteString` (CR-1005).
- Removed unused `newsObjAny` and dead `pruneGateMarkers` helper so `staticcheck` U1000 stays clean on Intel packages.
- **Intel MCP**: HTTP response body over the configured read limit now wraps a stable sentinel error for `errors.Is` (size-limit diagnostics).
- **Migration**: `backupFile` reads via `Open`+`Stat` on the same descriptor and caps read size (reduced TOCTOU vs `ReadFile`+`Stat`).
- **CLI**: default `--format` compatibility notice prints only on an interactive stderr TTY (or when `GATE_CLI_FORMAT_NOTICE_FORCE` is set for tests); avoids noisy stderr in pipelines.
- **toolschema**: `IsBackendInvoked` documented as deprecated (argv sniffing); schema refresh pointers in `info`/`news` package comments.

### Added
- **`scripts/test-changed-go.sh`**: run `go test` / `go vet` only on packages that contain **changed** `.go` files (from `git diff` / `git diff --cached`, or `base...HEAD`), avoiding default `go test ./...` during local iteration.
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

## [0.4.1]

### Fixed
- **Homebrew tap**: move generated formula into `Formula/` directory so `brew tap` can discover it (`2d7106d`)

### Changed
- **README**: update installation sections to reflect v0.4.0 distribution channels (`5b2795e`)

## [0.4.0]

### Added
- **One-line install scripts**
  - `install.sh` for Unix (macOS / Linux) — checksum-verified download, `$HOME/.local/bin` default, PATH hint
  - `install.ps1` for Windows — parity with Unix installer
  - Homebrew tap wiring in goreleaser (`.goreleaser.yaml`) for `brew install gate/tap/gate-cli`
- **Installation distribution design spec & implementation plan** under `docs/plans/`

### Fixed
- Run checksum verification inside a TMP subshell so partial downloads cannot pollute the target directory
- Pass `HOMEBREW_TAP_TOKEN` through to goreleaser in the release workflow
- Harden install scripts: strict checksum guard, CRLF normalization for Windows, PATH deduplication

### Changed
- Refresh README installation instructions to cover Homebrew, one-line installer, and manual download paths

## [0.3.2]

### Added
- **`--version` flag and `version` subcommand**: unified version reporting for both top-level flag and subcommand invocations
- **Install distribution design spec** committed under `docs/plans/`

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
