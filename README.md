# gate-cli

A command-line interface for the [Gate](https://gate.com) API. Covers spot, futures, delivery, options, margin, unified account, earn, wallet, AI Hub quant strategies, and 15+ more modules.

**Top-level layout:** CEX / trading APIs live under **`gate-cli cex …`** (for example `gate-cli cex spot market ticker --pair BTC_USDT`). Profiles and API credentials use **`gate-cli config …`**. **Intel** (market intelligence) uses **`gate-cli info`** and **`gate-cli news`** (**40** MCP-style tools: 30 + 10). Operational helpers: **`gate-cli doctor`** (local checks), **`gate-cli migrate`** (move legacy MCP provider configs toward CLI-first), **`gate-cli preflight`** (info/news readiness). Shell completion: **`gate-cli completion`**. Designed for developers, quants, and AI agents. For a full walkthrough, see the [English Quick Start](docs/quickstart.md) or [中文快速上手](docs/quickstart_zh.md). Per-release changes are tracked in [CHANGELOG.md](CHANGELOG.md).

## Installation

**macOS / Linux:**
```sh
curl -fsSL https://raw.githubusercontent.com/gate/gate-cli/main/install.sh | sh
```

**macOS — Homebrew:**
```sh
brew install gate/tap/gate-cli
```

**Windows:**
```powershell
irm https://raw.githubusercontent.com/gate/gate-cli/main/install.ps1 | iex
```

## Configuration

```bash
gate-cli config init
```

API keys and secrets for **trading** are stored per profile (for example `gate-cli config set api-key` / `gate-cli config set api-secret`) in `~/.gate-cli/config.yaml`. **Intel** endpoints and optional bearer tokens can use the same file under `intel:` or per-backend environment variables (see [Intel (`info`, `news`)](#intel-info-news) below).

## Features

### Trading
- **Spot** — currencies, pairs, market data, account, orders, price-triggered orders
- **Futures** — contracts, market data, account, positions, orders, price-triggered orders, trailing stop orders
- **Delivery** — delivery contracts, market data, account, positions, orders, price-triggered orders
- **Options** — underlyings, contracts, market data, account, positions, orders, MMP
- **Margin** — margin accounts, funding, cross-margin loans, uni lending, auto-repay, leverage
- **Unified** — unified account mode, borrowing, risk units, portfolio margin, collateral, leverage config
- **Alpha** — alpha token market data, account, orders
- **TradFi** — MT5 account, symbols, positions, orders, transactions
- **Cross-Exchange** — cross-exchange trading, positions, orders, convert, margin
- **AI Hub (Bot)** — Gate's quant strategy engine: AI-recommended strategy discovery, 4 grid types (spot, margin, infinite, futures) and 2 martingale types (spot, contract), running portfolio listing, detail, and stop

### Finance
- **Earn** — dual investment (incl. early-redemption refund, reinvest modify, project recommend), staking, fixed-term lending, auto-invest plans, uni simple earn
- **Asset Swap** — portfolio optimization (valuation, recommended strategies, create/preview/list orders)
- **Flash Swap** — instant token swaps, multi-currency many-to-one / one-to-many
- **Multi-Collateral Loan** — multi-collateral borrowing, repayment, collateral management

### Account & Wallet
- **Wallet** — balances, deposits, withdrawals, transfers (main/sub/cross-chain), small balance conversion
- **Account** — account detail, rate limits, STP groups, debit fee settings
- **Sub-Account** — sub-account CRUD, lock/unlock, API key management
- **Withdrawal** — create withdrawal, push order (UID transfer), cancel

### Ecosystem
- **P2P** — merchant ads, transactions, chat, payment methods
- **Rebate** — partner/broker/agency commissions and transaction history
- **Launch** — launch pool projects/pledge/redeem, Candy Drop V4 activities, HODLer Airdrop V4 activities
- **Activity** — platform activities and promotions
- **Coupon** — user coupons and details
- **Square** — AI search, live replay
- **Welfare** — user identity, beginner tasks

### Architecture
- **Futures position modes** — three orthogonal command groups under `gate-cli cex futures` expose every gateapi-go position flow:
  - `cex futures position update-*` → **one-way (single)** mode — `UpdatePosition{Margin,Leverage,CrossMode,RiskLimit}` + `GetPosition`
  - `cex futures position update-dual-*` → **dual (hedge)** mode — `UpdateDualModePosition*` + `GetDualModePosition`
  - `cex futures position update-contract-leverage` → **contract** mode — `UpdateContractPositionLeverage`
- **Order helpers** — `cex futures order add`, `remove`, `close` automatically detect position direction for single/dual mode via the `dual_comp` API
- **Output formats** — `--format pretty` (default for humans), `--format json` for scripts and agents, and `--format table` only where a command supports tabular list output
- **Multiple profiles** — manage several API keys in one config file
- **Credential priority** — `--api-key` flag > env var > config file

### Intel (Info & News)
- **Tool count** — **40** MCP-backed capabilities in the CLI baseline: **30** under `gate-cli info`, **10** under `gate-cli news` (grouped as `<domain> <tool>` leaves; counts follow the shipped tool list in the binary)
- **Info** — Each tool is `gate-cli info <group> <tool>` with **flat flags** for inputs. Optional JSON object args: `--params` / `--args-json` / `--args-file` when a field has no flag.
- **Info command groups** (the `<group>` segment):
  - **coin** — Coin profiles, multi-criteria search, and ranking boards.
  - **marketsnapshot** — Single-symbol snapshots, batch snapshots, and cross-asset market overview.
  - **markettrend** — OHLC-style klines, historical indicator series, and packaged technical analysis.
  - **onchain** — Address balances and activity, transaction detail, and token-level on-chain metrics.
  - **platformmetrics** — Protocol and CEX analytics: platform directory, DeFi overview, stablecoins, bridges, order-book depth, yield pools, TVL/volume history, reserves, and liquidation heatmaps.
  - **marketdetail** — Live order book, recent trades, and klines for Gate trading symbols (spot/futures/etc.).
  - **macro** — Macro indicators, economic calendar, and condensed macro summaries.
  - **compliance** — Token security and risk screening for a given chain.
- **News** — Same pattern: `gate-cli news <group> <tool>` plus flat flags.
- **News command groups**:
  - **feed** — News and social search (articles, UGC, X, web), web research mode, social sentiment, and exchange announcements.
  - **events** — Latest market-structured events and per-event detail by id.
  - **prediction** — Prediction-market style rankings (volume delta, fastest rising); optional `date_utc`, `venue`, `category`, and `status` flags (see `gate-cli news prediction -h`).
- **Discovery** — `gate-cli info list`, `gate-cli news list` to print tool names; `gate-cli info -h` and `gate-cli news -h` for groups, flags, and env vars

### CLI diagnostics & migration

- **`gate-cli doctor`** — Diagnose CLI version, config, connectivity to Intel backends, and legacy MCP registrations (`--check cli,version,config,connectivity,legacy-mcp` or `all`; `--strict` fails on warnings).
- **`gate-cli migrate`** — Scan and optionally rewrite Codex / Cursor / Claude Desktop configs that still reference legacy Gate MCP entries (`--dry-run`, `--apply`, `--provider`, `--backup-dir`).
- **`gate-cli preflight`** — CLI-first preflight for Gate info/news integrations (toggle MCP fallback with `--fallback-enabled`).

## Usage examples

```bash
# Public market data - no API key required
gate-cli cex spot market ticker --pair BTC_USDT
gate-cli cex futures market funding-rate --contract BTC_USDT
gate-cli cex delivery market contracts
gate-cli cex options market underlyings

# Account & wallet
gate-cli cex account detail
gate-cli cex spot account list
gate-cli cex wallet balance total
gate-cli cex wallet deposit list

# Spot orders
gate-cli cex spot order buy  --pair BTC_USDT --amount 0.001 --price 80000
gate-cli cex spot order buy  --pair BTC_USDT --quote 10          # market buy: spend 10 USDT
gate-cli cex spot order sell --pair BTC_USDT --amount 0.001

# Futures orders
gate-cli cex futures order long   --contract BTC_USDT --size 10 --price 80000
gate-cli cex futures order add    --contract BTC_USDT --size 5   # add to current position
gate-cli cex futures order close  --contract BTC_USDT            # close entire position

# Price-triggered & trailing stop orders
gate-cli cex futures price-trigger create --contract BTC_USDT --trigger-price 90000 --price 0 --size -10
gate-cli cex futures trail create --contract BTC_USDT --amount -10 --price-offset 0.02

# Margin & unified account
gate-cli cex margin uni pairs
gate-cli cex margin account list
gate-cli cex unified mode get
gate-cli cex unified account get

# Earn & staking
gate-cli cex earn dual plans
gate-cli cex earn dual recommend --mode normal --coin BTC     # recommended dual-investment projects
gate-cli cex earn dual refund-preview 12345                    # preview early-redemption
gate-cli cex earn uni currencies
gate-cli cex earn fixed products
gate-cli cex earn auto-invest coins

# Asset swap (portfolio optimization)
gate-cli cex assetswap assets
gate-cli cex assetswap config
gate-cli cex assetswap order list --size 20

# Launch pool / Candy Drop / HODLer Airdrop
gate-cli cex launch projects
gate-cli cex launch candy-drop activities --status active
gate-cli cex launch hodler projects --keyword BTC

# Flash swap
gate-cli cex flash-swap pairs

# Multi-collateral loan
gate-cli cex mcl currencies
gate-cli cex mcl ltv

# Sub-account management
gate-cli cex sub-account list
gate-cli cex sub-account key list --user-id 12345

# AI Hub (quant strategies) — 10 BotAPI methods wrapped under cex bot
gate-cli cex bot recommend --market BTC_USDT --strategy-type spot_grid     # browse AI recommendations
gate-cli cex bot running --strategy-type spot_grid --page 1 --page-size 20 # list running strategies
gate-cli cex bot detail --strategy-id strat-001 --strategy-type spot_grid  # detail by id+type
gate-cli cex bot stop --strategy-id strat-001 --strategy-type spot_grid    # stop a running strategy
# Create flows take a JSON body matching the SDK's *CreateRequest shape; see -h on each leaf:
gate-cli cex bot grid spot       --json '{"strategy_type":"spot_grid","market":"BTC_USDT","create_params":{"money":"100","low_price":"60000","high_price":"70000","grid_num":10,"price_type":0}}'
gate-cli cex bot grid infinite   --json '{"strategy_type":"infinite_grid","market":"BTC_USDT","create_params":{"money":"100","price_floor":"60000","profit_per_grid":"0.005"}}'
gate-cli cex bot martingale spot --json '{"strategy_type":"spot_martingale","market":"BTC_USDT","create_params":{"invest_amount":"100","price_deviation":"0.02","max_orders":5,"take_profit_ratio":"0.01","stop_loss_per_cycle":"0.05"}}'

# JSON output for scripting
gate-cli cex spot market ticker --pair BTC_USDT --format json | jq '.last'

# Intel — 40 MCP tools (30 info + 10 news); list names: gate-cli info list / gate-cli news list
# Below: one minimal example per tool (flat flags; --format json). Arrays use a single JSON token, e.g. --indicators '["rsi"]'.

# Info (30)
# coin — coin profiles, search, rankings
gate-cli info coin get-coin-info --query BTC --format json
# marketsnapshot — per-symbol and batch snapshots
gate-cli info marketsnapshot get-market-snapshot --symbol BTC_USDT --format json
# markettrend — klines, indicators, technical analysis
gate-cli info markettrend get-kline --symbol BTC_USDT --timeframe 1h --format json
gate-cli info markettrend get-indicator-history --symbol BTC_USDT --timeframe 1h --indicators '["rsi"]' --format json
gate-cli info markettrend get-technical-analysis --symbol BTC_USDT --format json
# onchain — addresses, transactions, token metrics
gate-cli info onchain get-address-info --address 0xd8dA6BF26964aF9D7eEd9e03E53415dA322193D --chain eth --format json
gate-cli info onchain get-address-transactions --address 0xd8dA6BF26964aF9D7eEd9e03E53415dA322193D --format json
gate-cli info onchain get-transaction --tx-hash 0x88df016429689c079f1b2ea6911a4055630eac127461fbce8dcb82e83bdb12b4 --format json
gate-cli info onchain get-token-onchain --token USDT --chain eth --format json
# platformmetrics — DeFi/CEX platform and market-structure metrics
gate-cli info platformmetrics get-platform-info --platform-name uniswap --format json
gate-cli info platformmetrics search-platforms --format json
gate-cli info platformmetrics get-defi-overview --format json
gate-cli info platformmetrics get-stablecoin-info --format json
gate-cli info platformmetrics get-bridge-metrics --format json
gate-cli info platformmetrics get-cex-orderbook-depth --symbol BTC_USDT --format json
gate-cli info platformmetrics get-yield-pools --format json
gate-cli info platformmetrics get-platform-history --platform-name uniswap --format json
gate-cli info platformmetrics get-exchange-reserves --format json
gate-cli info platformmetrics get-liquidation-heatmap --symbol BTC_USDT --format json
# marketdetail — live book, trades, klines (Gate symbols)
gate-cli info marketdetail get-orderbook --symbol BTC_USDT --format json
gate-cli info marketdetail get-recent-trades --symbol BTC_USDT --format json
gate-cli info marketdetail get-kline --symbol BTC_USDT --timeframe 1h --format json
# macro — indicators, calendar, summary
gate-cli info macro get-macro-indicator --indicator CPI --format json
gate-cli info macro get-economic-calendar --format json
gate-cli info macro get-macro-summary --format json
# coin — search & rankings (same group as above)
gate-cli info coin search-coins --format json
gate-cli info coin get-coin-rankings --ranking-type popular --format json
# marketsnapshot — batch snapshot & overview
gate-cli info marketsnapshot batch-market-snapshot --symbols '["BTC_USDT"]' --format json
gate-cli info marketsnapshot get-market-overview --format json
# compliance — token security / risk
gate-cli info compliance check-token-security --chain eth --format json

# News (10)
# feed — search, web research, sentiment, announcements (alias: search → search-news)
gate-cli news feed search-news --query bitcoin --format json
gate-cli news feed search-ugc --format json
gate-cli news feed search-x --format json
gate-cli news feed web-search --query bitcoin --format json
gate-cli news feed get-social-sentiment --format json
gate-cli news feed get-exchange-announcements --format json
# events — latest events and detail by id
gate-cli news events get-latest-events --format json
gate-cli news events get-event-detail --event-id example:event-1 --format json
# prediction — prediction-market rankings (optional: --date-utc YYYY-MM-DD, --venue, --category, --status; see -h)
gate-cli news prediction get-volume-delta-ranking --format json
gate-cli news prediction get-fastest-rising-ranking --format json
```

## Modules

| Module | Command | Description |
|--------|---------|-------------|
| spot | `gate-cli cex spot` | Spot trading |
| futures | `gate-cli cex futures` | USDT perpetual contracts |
| delivery | `gate-cli cex delivery` | Delivery (expiry) contracts |
| options | `gate-cli cex options` | Options trading |
| margin | `gate-cli cex margin` | Margin trading & lending |
| unified | `gate-cli cex unified` | Unified account management |
| earn | `gate-cli cex earn` | Earn, staking, dual investment (incl. refund/recommend), auto-invest |
| bot | `gate-cli cex bot` | AI Hub quant strategies — recommend / running / detail / stop + 4 grid types + 2 martingale types |
| assetswap | `gate-cli cex assetswap` | Asset-swap / portfolio optimization |
| flash-swap | `gate-cli cex flash-swap` | Instant token swaps |
| mcl | `gate-cli cex mcl` | Multi-collateral loans |
| cross-ex | `gate-cli cex cross-ex` | Cross-exchange trading |
| wallet | `gate-cli cex wallet` | Wallet & transfers |
| account | `gate-cli cex account` | Account details & settings |
| sub-account | `gate-cli cex sub-account` | Sub-account management |
| withdrawal | `gate-cli cex withdrawal` | Withdrawals |
| alpha | `gate-cli cex alpha` | Alpha token trading |
| tradfi | `gate-cli cex tradfi` | TradFi (MT5) trading |
| p2p | `gate-cli cex p2p` | P2P trading |
| rebate | `gate-cli cex rebate` | Rebate & commissions |
| launch | `gate-cli cex launch` | Launch pool + Candy Drop V4 + HODLer Airdrop V4 |
| activity | `gate-cli cex activity` | Activities & promotions |
| coupon | `gate-cli cex coupon` | Coupons |
| square | `gate-cli cex square` | Gate Square |
| welfare | `gate-cli cex welfare` | Welfare & tasks |
| config | `gate-cli config` | CLI configuration (profiles, API keys, optional `intel:` block) |
| info | `gate-cli info` | **30** MCP tools under groups `coin`, `marketsnapshot`, `markettrend`, `onchain`, `platformmetrics`, `marketdetail`, `macro`, `compliance` (`info list`, `info -h`; see Features) |
| news | `gate-cli news` | **10** MCP tools under `feed`, `events`, and `prediction` (`news list`, `news -h`; see Features) |
| doctor | `gate-cli doctor` | CLI + config + connectivity + legacy MCP diagnostics |
| migrate | `gate-cli migrate` | Migrate provider configs off legacy Gate MCP entries |
| preflight | `gate-cli preflight` | CLI-first preflight for info/news |

## Global flags

| Flag | Default | Description |
|------|---------|-------------|
| `--format` | `pretty` | Output format: `pretty`, `json`, or `table` (only on tabular commands) |
| `--profile` | `default` | Config profile to use |
| `--api-key` | — | Gate API key for **trading** (overrides env and config file; not used as Intel bearer) |
| `--api-secret` | — | Gate API secret for **trading** (overrides env and config file) |
| `--max-output-bytes` | `0` | Cap **printed** bytes for `info` / `news` tool output (`0` = unlimited; env `GATE_MAX_OUTPUT_BYTES`) |
| `--verbose` | `false` | Print low-level Intel backend transport lines to stderr (`info` / `news`), prefixed `[verbose]`; stdout JSON unchanged |
| `--debug` | `false` | HTTP debug for Gate **trading** clients; for `info` / `news`, Intel transport logs use `[debug]` on stderr (wins over `--verbose` when both are set) |

## Intel (`info`, `news`)

**40** MCP tools are wired as CLI leaves (30 `info`, 10 `news`). Command-group summaries (English) live under **Intel (Info & News)** in Features. Defaults can live under `intel:` in `~/.gate-cli/config.yaml` alongside `profiles`. **Do not** use trading `GATE_API_KEY` / `--api-key` as the Intel bearer; use the dedicated bearer env vars or `intel` config when your gateway requires auth.

**Common environment variables** (override file when set; full detail in repo `specs/` if present):

| Variable | Purpose |
|----------|---------|
| `GATE_INTEL_INFO_MCP_URL` / `GATE_INTEL_NEWS_MCP_URL` | JSON-RPC HTTP endpoint for each backend |
| `GATE_INTEL_INFO_BEARER_TOKEN` / `GATE_INTEL_NEWS_BEARER_TOKEN` | Per-backend bearer (optional) |
| `GATE_INTEL_BEARER_TOKEN` | Shared bearer when per-backend tokens are not set |
| `GATE_INTEL_HTTP_TIMEOUT` | HTTP client timeout (Go duration or seconds) |
| `GATE_INTEL_EXTRA_HEADERS` | JSON object of extra request headers (denylisted keys rejected) |
| `GATE_INTEL_MAX_RESPONSE_BYTES` | Max **HTTP response body** read for Intel JSON-RPC (default 16 MiB); distinct from `--max-output-bytes`, which only limits **stdout** |
| `GATE_MAX_OUTPUT_BYTES` | Default for `--max-output-bytes` when the flag is omitted |
| `GATE_INTEL_REFRESH_SCHEMA` | Set to `1` to force a one-off schema refresh (leaf flags / help) |
| `GATE_INTEL_LEAF_HELP` | `full` or `detailed` appends long MCP-spec text to leaf `--help` |

Entry points: `gate-cli info list` / `gate-cli news list`, and `gate-cli info -h` / `gate-cli news -h`. Additional precedence and security notes may live under [`specs/README.md`](specs/README.md) / `specs/intel-config-and-security.md` depending on your checkout.

## Development

From a repository clone: `go build -o gate-cli .`. Prefer `./scripts/test-changed-go.sh` for local iteration; use `go test ./...` before wide merges / releases. Integration tests use build tag `integration` (see `testdata/integration.yaml.example`).