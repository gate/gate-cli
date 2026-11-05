# gate-cli

A command-line interface for the [Gate](https://gate.com) API. Covers spot, futures, delivery, options, margin, unified account, earn, wallet, AI Hub quant strategies, and 15+ more modules.

**Top-level layout:** CEX / trading APIs live under **`gate-cli cex …`** (for example `gate-cli cex spot market ticker --pair BTC_USDT`). Profiles and API credentials use **`gate-cli config …`**. **Intel** (market intelligence) uses **`gate-cli info`** and **`gate-cli news`** (**50** MCP-style tools: 32 + 18) plus deterministic top-level shortcut flows such as `gate-cli info +coin-overview` / `gate-cli news +brief`. Operational helpers: **`gate-cli doctor`** (local checks), **`gate-cli migrate`** (move legacy MCP provider configs toward CLI-first), **`gate-cli preflight`** (info/news readiness). Shell completion: **`gate-cli completion`**. Designed for developers, quants, and automation. For a full walkthrough, see the [English Quick Start](docs/quickstart.md) or [中文快速上手](docs/quickstart_zh.md). Per-release changes are tracked in [CHANGELOG.md](CHANGELOG.md).

See the [documentation index](docs/README.md), [CEX command reference](docs/cex-gap-commands.md), [authorization, CFD and stock guide](docs/auth-cfd-stock.md), and [AI Hub strategy guide](docs/bot.md) for commands and parameters. These documents describe the current source tree; public installers obtain published releases. Check the installed leaf command's `--help` before relying on a capability.

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
- **Spot** — currencies, pairs, RPI books, account selection, automatic borrowing/repayment, attached take profit/stop loss, orders, price-triggered orders and native POV execution
- **Futures** — contracts (including paged all-contract metadata), RPI books, ADL risk states, account, positions, orders, price-triggered orders, trailing stops and native chase orders
- **Delivery** — delivery contracts, market data, account, positions, orders, price-triggered orders
- **Options** — underlyings, contracts, market data, account, positions, orders, reduce-only/full-close parameters and MMP
- **Margin** — margin accounts, funding, cross-margin loans, uni lending, auto-repay, leverage
- **Unified** — account mode, borrowing, risk units, portfolio margin, collateral, Delta-neutral mode, quick-repayment estimates/execution and per-currency/account leverage settings
- **Alpha** — alpha token market data, account, orders
- **TradFi (CFD)** — 20 commands for MT5 accounts, symbols, positions, orders, transactions, authenticated symbol commissions and order-log queries
- **Stock** — 16 commands for exchanges, symbols, order books, accounts, transfers, orders and positions
- **AI Hub (Bot)** — Gate's quant strategy engine: AI-recommended strategy discovery, 4 grid types (spot, margin, infinite, futures), 2 martingale types (spot, contract), smart-position and futures-combo creation, running portfolio listing, detail, and stop

### Finance
- **Earn** — dual investment (incl. early-redemption refund, reinvest modify, project recommend), staking, fixed-term lending, auto-invest plans, uni simple earn
- **Asset Swap** — portfolio optimization (valuation, recommended strategies, create/preview/list orders)
- **Flash Swap** — instant token swaps, multi-currency many-to-one / one-to-many
- **Multi-Collateral Loan** — multi-collateral borrowing, repayment, collateral management

### Account & Wallet
- **Wallet** — balances, deposits, withdrawal records/status, transfers (main/sub/cross-chain), small balance conversion
- **Account** — account detail, rate limits, STP groups, debit fee settings

### Ecosystem
- **Rebate** — partner/broker/agency commissions and transaction history
- **Square** — AI search, live replay

P2P, CrossEx, welfare, activity, coupon, LaunchPool, HODLer Airdrop, CandyDrop and sub-account management commands are unavailable.
The CEX Exchange Announcement API is unavailable; exchange announcements remain accessible through `news`. Existing orders and positions are not modified.
Manage outstanding activity through the Gate website.

### Agentic subaccount authorization

Run `gate-cli cex auth login` and select or create the Agentic subaccount in the browser.
The CLI requests `base spot_trade futures_trade earn cfd stock`; the browser decides
which permissions to grant. Use `--scope base,stock` for an explicit request or
`--manual` for headless login. `cex auth status` reports local state, and
`cex auth logout` clears local credentials and attempts remote revocation; check any reported revocation failure.

`cex auth status --format json` returns the selected issuer, resource, scopes, expiry and `remote_checked: false`. A locally unexpired token can still be revoked remotely. Compare issuer/resource before logging in again; changing the API origin does not change the OAuth issuer. Plain route 404 errors are `API_NOT_FOUND`, not evidence that the account has no records.

Private CFD and stock queries require `base`; CFD trading and transfers require `cfd`, while stock order writes and position closes require `stock`. Stock account transfers require `base`. Spot strategy writes require `spot_trade`, and futures strategy writes require `futures_trade`. The final grant and account eligibility determine access. See the [authorization and product guide](docs/auth-cfd-stock.md).

Credentials are bound to the selected API address; changing it requires authorization again. A complete API key/secret pair takes priority over OAuth. CEX base URLs use the site origin without `/api/v4`; use HTTPS for public access. Changing `GATE_BASE_URL` does not select another OAuth issuer. Credentials without resource binding require login again.

### Command behavior

- **Futures position modes** — `position get` and `update-*` target one-way mode; `get-dual` and `update-dual-*` target hedge mode; `update-contract-leverage` adjusts contract-mode leverage.
- **Order helpers** — futures `order add`, `remove` and `close` detect the current position direction. A mode lookup failure stops the operation.
- **Output** — `--format pretty` is the default; use `--format json` for scripts and `--format table` for commands with tabular output.
- **Profiles** — select API key profiles with `--profile`; flags override environment variables, which override the profile.
- **Agent commands** — `agent-*` commands require a build with `-tags agent`; see the [Quick Start](docs/quickstart.md#gateai-agent-mode).

### CEX execution and result handling

Write commands submit immediately; verify the account, product, amount and existing authorization first. Keep decimal values and IDs exact, and pass arguments separately. Use `--format json` with bounded output for automation.

- Chase calls accept flat or `{code,message,data}` service responses. HTTP 200 business errors fail the CLI; active lists explicitly select `is_finished=false`, and empty list/stop-all results are `{"orders":[]}`.
- Spot protection amendments require an explicitly authorized parent `--price` or `--amount`. Omitted/JSON-null protection preserves it; `{}` clears it. Check minimum notional for the parent and protective orders.
- Quick repayment selects currencies, not a spending cap. Account leverage updates can partly fail; inspect every failure and re-read settings even after HTTP 200.
- Acceptance is not completion. Reconcile orders, child orders, positions or debt after writes; never automatically replay an unknown write outcome.

The [CEX command reference](docs/cex-gap-commands.md) describes parameters, partial failures, order references, Earn filters and recovery. Product access depends on the account, permissions and service availability.

### Intel (Info & News)
- **Tool count** — **50** market-intelligence capabilities: **32** under `gate-cli info`, **18** under `gate-cli news` (grouped as `<domain> <tool>` leaves; counts follow the shipped tool list in the binary)
- **Info** — Each tool is `gate-cli info <group> <tool>` with **flat flags** for inputs. Optional JSON object args: `--params` / `--args-json` / `--args-file` when a field has no flag. `--args-file` accepts relative, absolute, and `~/` paths only when the real file is within the current working directory; links to outside files are rejected.
- **Info command groups** (the `<group>` segment):
  - **coin** — Coin profiles, multi-criteria search, and ranking boards.
  - **marketsnapshot** — Single-symbol snapshots, batch snapshots, cross-asset market overview, and institutional ETF/CME/CFTC channel metrics.
  - **markettrend** — OHLC-style klines, historical indicator series, and packaged technical analysis.
  - **onchain** — Address balances and activity, transaction detail, and token-level on-chain metrics.
  - **platformmetrics** — Protocol and CEX analytics: platform directory, DeFi overview, stablecoins (optional `scope=full` + `sections=issuance_flow|usage_structure` for supply flows and usage structure), bridges, order-book depth, yield pools, TVL/volume history, reserves, liquidation heatmaps, and chain activity (Ethereum staking metrics).
  - **marketdetail** — Live order book, recent trades, and klines for Gate trading symbols (spot/futures/etc.).
  - **macro** — Macro indicators, economic calendar, and condensed macro summaries.
  - **compliance** — Token security and risk screening for a given chain.
- **News** — Same pattern: `gate-cli news <group> <tool>` plus flat flags.
- **News command groups**:
  - **feed** — Platform news index (`search-news`), UGC (`search-ugc`; needs `--query` or `--coin`), X/Twitter (`search-x`; needs `--query` for xAI), open-web synthesis (`web-search`), social sentiment, exchange announcements, 24h mention bursts (`get-mention-burst`; alias `mention-burst`), and 4h hot topics (`get-hot-topics`; alias `hot-topics`, limit 2–4). Both social-insight tools require `--coin`; `--platforms` accepts `all`, `gate_square`, `binance_square`, `twitter`, `telegram`, `youtube`, `reddit`, or `discord` (`all` cannot be combined). Use each command’s `--help` or `gate-cli news mcp-spec` for parameters.
  - **events** — Filtered event list with `event_id`, single-event detail, market-move evidence synthesis (`explain-market-move`), and stored market-move report queries: `get-market-move-report` (aliases `market-move-report`, `get-report`) requires `--symbol`; `--report-id` takes priority over `--event-id`, and omitting both returns the latest report for that symbol. `list-market-move-reports` (aliases `market-move-reports`, `report-list`) filters `updated_at` by required UTC0 `--start-time` / `--end-time` and accepts `--limit 0` for the default 20 (maximum 100).
  - **prediction** — Daily rankings (`get-volume-delta-ranking`, `get-fastest-rising-ranking`) accept a free-form `--category`. `search-events` defaults to `--sort-by recently_listed` and requires at least one of `--query`, `--coin` or `--category`. `get-event-signal` returns an event snapshot; use `get-market-orderbook --venue <venue> --market-id <id>` for a live order book. `predict.fun` uses its numeric market ID. Some results may be partial or unavailable; inspect the returned status and metadata. See `gate-cli news prediction --help`.
- **Discovery** — `gate-cli info list`, `gate-cli news list` to print tool names; `gate-cli info -h` and `gate-cli news -h` for groups, flags, env vars, and top-level `+shortcut` commands
- **Boolean flags** — On `info` / `news` leaves, JSON boolean fields are exposed as flags that accept **`--flag`** (means true), **`--flag=false`**, or the legacy spaced form **`--flag false`**. For `cex` / `config`, use `--flag=false` when specifying false explicitly.

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
gate-cli cex futures price-trigger create --contract BTC_USDT --trigger-price 90000 --price 0 --size -10 --tif ioc
gate-cli cex futures trail create --contract BTC_USDT --amount -10 --price-type 1 --price-offset '2%'

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

# Stocks: public market data and authorized account queries
gate-cli cex stock market symbols --exchange us
gate-cli cex stock account assets

# Flash swap
gate-cli cex flash-swap pairs

# Multi-collateral loan
gate-cli cex mcl currencies
gate-cli cex mcl ltv

# AI Hub strategies; see the AI Hub strategy guide for creation and stop parameters.
gate-cli cex bot recommend --market BTC_USDT --strategy-type spot_grid     # browse AI recommendations
gate-cli cex bot running --strategy-type spot_grid --page 1 --page-size 20 # list running strategies
gate-cli cex bot detail --strategy-id strat-001 --strategy-type spot_grid  # detail by id+type
gate-cli cex bot stop --strategy-id strat-001 --strategy-type spot_grid    # stop a running strategy
# Create commands accept one JSON request object; use --help for required fields:
gate-cli cex bot grid spot       --json '{"strategy_type":"spot_grid","market":"BTC_USDT","create_params":{"money":"100","low_price":"60000","high_price":"70000","grid_num":10,"price_type":0}}'
gate-cli cex bot grid infinite   --json '{"strategy_type":"infinite_grid","market":"BTC_USDT","create_params":{"money":"100","price_floor":"60000","profit_per_grid":"0.005"}}'
gate-cli cex bot martingale spot --json '{"strategy_type":"spot_martingale","market":"BTC_USDT","create_params":{"invest_amount":"100","price_deviation":"0.02","max_orders":5,"take_profit_ratio":"0.01","stop_loss_per_cycle":"0.05"}}'

# JSON output for scripting
gate-cli cex spot market ticker --pair BTC_USDT --format json | jq '.last'

# Intel — 50 MCP tools (32 info + 18 news); list names: gate-cli info list / gate-cli news list
# Below: one minimal example per tool (flat flags; --format json). Arrays use a single JSON token, e.g. --indicators '["rsi"]'.

# Info (32)
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
gate-cli info onchain get-address-info --address 0xd8dA6BF26964aF9D7eEd9e03E53415dA322193D --chain optimism --format pretty
gate-cli info onchain get-address-transactions --address 0xd8dA6BF26964aF9D7eEd9e03E53415dA322193D --format json
gate-cli info onchain get-transaction --tx-hash 0x88df016429689c079f1b2ea6911a4055630eac127461fbce8dcb82e83bdb12b4 --format json
gate-cli info onchain get-token-onchain --token USDT --chain eth --format json
# platformmetrics — DeFi/CEX platform and market-structure metrics
gate-cli info platformmetrics get-platform-info --platform-name uniswap --scope full --include-oi-symbol-detail --oi-symbol-limit 20 --format json
gate-cli info platformmetrics search-platforms --format json
gate-cli info platformmetrics get-defi-overview --format json
gate-cli info platformmetrics get-stablecoin-info --scope full --sections '["issuance_flow","usage_structure"]' --start-date 2026-04-01 --end-date 2026-05-01 --format json
gate-cli info platformmetrics get-bridge-metrics --format json
gate-cli info platformmetrics get-cex-orderbook-depth --symbol BTC_USDT --format json
gate-cli info platformmetrics get-yield-pools --format json
gate-cli info platformmetrics get-platform-history --platform-name uniswap --format json
gate-cli info platformmetrics get-exchange-reserves --scope full --include-history --asset BTC --format json
gate-cli info platformmetrics get-liquidation-heatmap --symbol BTC_USDT --format json
gate-cli info platformmetrics get-chain-activity --metric-group staking --lookback 30d --format json
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
gate-cli info marketsnapshot get-institutional-metrics --asset BTC --channel all --limit 30 --format json
# compliance — token security / risk
gate-cli info compliance check-token-security --chain eth --token USDT --format json

# News (18)
# feed — search, web research, sentiment, announcements, mention bursts, hot topics (alias: search → search-news)
gate-cli news feed search-news --query bitcoin --format json
gate-cli news feed search-ugc --query bitcoin --format json
gate-cli news feed search-x --query bitcoin --coin BTC --time-range 7d --lang en --format json
gate-cli news feed web-search --query bitcoin --format json
gate-cli news feed get-social-sentiment --format json
gate-cli news feed get-exchange-announcements --format json
gate-cli news feed mention-burst --coin BTC --platforms all --format json
gate-cli news feed hot-topics --coin ETH --platforms twitter,reddit --limit 3 --format json
# events — latest events, detail by id, market-move evidence, stored report queries
gate-cli news events get-latest-events --format json
gate-cli news events get-event-detail --event-id example:event-1 --format json
gate-cli news events explain-market-move --query "Why did BTC move?" --coin BTC --format json
gate-cli news events get-report --symbol TAIKO --report-id _1782973063855 --format json
gate-cli news events report-list --symbol ETH --start-time "2026-07-09 22:00:00" --end-time "2026-07-10 03:15:00" --limit 10 --format json # filters updated_at in UTC0; sorts by event_time DESC; --limit 0 defaults to 20
# prediction — rankings, signal-index search, external signal, live order book (see -h)
gate-cli news prediction get-volume-delta-ranking --format json
gate-cli news prediction get-fastest-rising-ranking --format json
gate-cli news prediction search-events --coin BTC --format json
gate-cli news prediction get-event-signal --event-ref polymarket:107711 --format json
gate-cli news prediction get-market-orderbook --venue polymarket --market-id 12345 --format json
```

## Modules

| Module | Command | Description |
|--------|---------|-------------|
| spot | `gate-cli cex spot` | Spot trading |
| futures | `gate-cli cex futures` | Perpetual contracts; USDT settlement by default, BTC where supported |
| delivery | `gate-cli cex delivery` | Delivery (expiry) contracts |
| options | `gate-cli cex options` | Options trading |
| margin | `gate-cli cex margin` | Margin trading & lending |
| unified | `gate-cli cex unified` | Unified account management |
| earn | `gate-cli cex earn` | Earn, staking, dual investment (incl. refund/recommend), auto-invest |
| bot | `gate-cli cex bot` | AI Hub quant strategies — recommend / running / detail / stop + 4 grid types + 2 martingale types + smart-position / futures-combo |
| assetswap | `gate-cli cex assetswap` | Asset-swap / portfolio optimization |
| flash-swap | `gate-cli cex flash-swap` | Instant token swaps |
| mcl | `gate-cli cex mcl` | Multi-collateral loans |
| wallet | `gate-cli cex wallet` | Wallet & transfers |
| account | `gate-cli cex account` | Account details & settings |
| alpha | `gate-cli cex alpha` | Alpha token trading |
| tradfi | `gate-cli cex tradfi` | CFD (MT5) trading |
| stock | `gate-cli cex stock` | Stock market, orders, positions and assets |
| auth | `gate-cli cex auth` | Agentic subaccount login, local status, refresh and logout |
| rebate | `gate-cli cex rebate` | Rebate & commissions |
| square | `gate-cli cex square` | Gate Square |
| config | `gate-cli config` | CLI configuration (profiles, API keys, optional `intel:` block) |
| info | `gate-cli info` | **32** market-intelligence tools under groups `coin`, `marketsnapshot`, `markettrend`, `onchain`, `platformmetrics`, `marketdetail`, `macro`, `compliance` (`info list`, top-level `+shortcut`, `info -h`; see Features) |
| news | `gate-cli news` | **18** MCP tools under `feed`, `events`, and `prediction` (`news list`, `news -h`; see Features) |
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
| `--max-output-bytes` | `0` | Cap **printed** stdout for any command (`0` = unlimited; env `GATE_MAX_OUTPUT_BYTES`) |
| `--verbose` | `false` | Print low-level Intel backend transport lines to stderr (`info` / `news`), prefixed `[verbose]`; stdout JSON unchanged |
| `--debug` | `false` | CEX transport method, status and elapsed time on stderr, without query/headers/body; for `info` / `news`, Intel transport logs use `[debug]` (wins over `--verbose`) |

## Intel (`info`, `news`)

**50** MCP tools are wired as CLI leaves (32 `info`, 18 `news`). Command-group summaries (English) live under **Intel (Info & News)** in Features. Defaults can live under `intel:` in `~/.gate-cli/config.yaml` alongside `profiles`. **Do not** use trading `GATE_API_KEY` / `--api-key` as the Intel bearer; use the dedicated bearer env vars or `intel` config when your gateway requires auth.

**Common environment variables** (override file when set):

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
| `GATE_INTEL_LEAF_HELP` | `full` or `detailed` appends MCP-spec per-field notes to leaf `--help`. English `description` is shown on info/news leaf `-h` from the embedded bundled spec |

Entry points: `gate-cli info list` / `gate-cli news list`, and `gate-cli info -h` / `gate-cli news -h`. Use the embedded leaf help or `mcp-spec` for shipped schemas; no separate schema files are required.

## Source installation

Source builds require Go 1.24 or later and use the public Gate Go SDK `v7.2.179`.

```sh
# Run from a source checkout to build the CLI.
go build -o gate-cli .
```

Use `gate-cli version --details --format json` to inspect the installed version, platform and agent command availability.
