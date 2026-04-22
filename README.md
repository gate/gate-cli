# gate-cli

A command-line interface for the [Gate](https://gate.com) API. Covers spot, futures, delivery, options, margin, unified account, earn, wallet, and 15+ more modules. Exchange API commands are grouped under `gate-cli cex …` (for example `gate-cli cex spot market ticker --pair BTC_USDT`); profile and credentials use `gate-cli config …` at the top level. Designed for developers, quants, and AI agents. For a full walkthrough, see the [English Quick Start](docs/quickstart.md) or [中文快速上手](docs/quickstart_zh.md).

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
- **Futures position modes** — three orthogonal command groups expose every gateapi-go position flow:
  - `position update-*` → **one-way (single)** mode — `UpdatePosition{Margin,Leverage,CrossMode,RiskLimit}` + `GetPosition`
  - `position update-dual-*` → **dual (hedge)** mode — `UpdateDualModePosition*` + `GetDualModePosition`
  - `position update-contract-leverage` → **contract** mode — `UpdateContractPositionLeverage`
- **Order helpers** — `add`, `remove`, `close` automatically detect position direction for single/dual mode via the `dual_comp` API
- **Output formats** — `--format pretty` (default for humans), `--format json` for scripts and agents, and `--format table` only where a command supports tabular list output
- **Multiple profiles** — manage several API keys in one config file
- **Credential priority** — `--api-key` flag > env var > config file

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

# JSON output for scripting
gate-cli cex spot market ticker --pair BTC_USDT --format json | jq '.last'
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
| config | `gate-cli config` | CLI configuration |
| info | `gate-cli info` | Market and intelligence info commands |
| news | `gate-cli news` | News and market intelligence commands |

## Global flags

| Flag | Default | Description |
|------|---------|-------------|
| `--format` | `pretty` | Output format: `pretty`, `json`, or `table` (only on tabular commands) |
| `--profile` | `default` | Config profile to use |
| `--api-key` | — | API key (overrides env and config file) |
| `--api-secret` | — | API secret (overrides env and config file) |
| `--max-output-bytes` | `0` | Cap printed bytes for `info` / `news` results (`0` = unlimited; env `GATE_MAX_OUTPUT_BYTES`) |
| `--verbose` | `false` | Print low-level Intel backend transport lines to stderr (`info` / `news`), prefixed `[verbose]`; stdout JSON unchanged |
| `--debug` | `false` | HTTP debug for Gate API clients; with `info` / `news`, backend transport uses `[debug]` on stderr (wins over `--verbose` when both are set) |

## Intel (`info`, `news`)

Behavior, flags, and environment variables: `gate-cli info -h`, `gate-cli news -h`. Optional defaults go under `intel:` in the same `config.yaml` as `profiles`; trading `GATE_API_KEY` / `--api-key` are not used as the Intel bearer (bearer is optional if your backend allows it).
