# gate-cli

A command-line interface for the [Gate](https://gate.com) API. Covers spot, futures, delivery, options, margin, unified account, earn, wallet, and 15+ more modules. Designed for developers, quants, and AI agents. For a full walkthrough, see the [English Quick Start](docs/quickstart.md) or [中文快速上手](docs/quickstart_zh.md).

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
- **Earn** — dual investment, staking, fixed-term lending, auto-invest plans, uni simple earn
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
- **Launch** — launch pool projects, pledge, redeem, records
- **Activity** — platform activities and promotions
- **Coupon** — user coupons and details
- **Square** — AI search, live replay
- **Welfare** — user identity, beginner tasks

### Architecture
- **Dual-position mode** — `add`, `remove`, `close` automatically detect position direction; single and dual (hedge) mode handled transparently via the `dual_comp` API
- **Output formats** — `--format pretty` (default for humans), `--format json` for scripts and agents, and `--format table` only where a command supports tabular list output
- **Multiple profiles** — manage several API keys in one config file
- **Credential priority** — `--api-key` flag > env var > config file

## Usage examples

```bash
# Public market data - no API key required
gate-cli spot market ticker --pair BTC_USDT
gate-cli futures market funding-rate --contract BTC_USDT
gate-cli delivery market contracts
gate-cli options market underlyings

# Account & wallet
gate-cli account detail
gate-cli spot account list
gate-cli wallet balance total
gate-cli wallet deposit list

# Spot orders
gate-cli spot order buy  --pair BTC_USDT --amount 0.001 --price 80000
gate-cli spot order buy  --pair BTC_USDT --quote 10          # market buy: spend 10 USDT
gate-cli spot order sell --pair BTC_USDT --amount 0.001

# Futures orders
gate-cli futures order long   --contract BTC_USDT --size 10 --price 80000
gate-cli futures order add    --contract BTC_USDT --size 5   # add to current position
gate-cli futures order close  --contract BTC_USDT            # close entire position

# Price-triggered & trailing stop orders
gate-cli futures price-trigger create --contract BTC_USDT --trigger-price 90000 --price 0 --size -10
gate-cli futures trail create --contract BTC_USDT --amount -10 --price-offset 0.02

# Margin & unified account
gate-cli margin uni pairs
gate-cli margin account list
gate-cli unified mode get
gate-cli unified account get

# Earn & staking
gate-cli earn dual plans
gate-cli earn uni currencies
gate-cli earn fixed products
gate-cli earn auto-invest coins

# Flash swap
gate-cli flash-swap pairs

# Multi-collateral loan
gate-cli mcl currencies
gate-cli mcl ltv

# Sub-account management
gate-cli sub-account list
gate-cli sub-account key list --user-id 12345

# JSON output for scripting
gate-cli spot market ticker --pair BTC_USDT --format json | jq '.last'
```

## Modules

| Module | Command | Description |
|--------|---------|-------------|
| spot | `gate-cli spot` | Spot trading |
| futures | `gate-cli futures` | USDT perpetual contracts |
| delivery | `gate-cli delivery` | Delivery (expiry) contracts |
| options | `gate-cli options` | Options trading |
| margin | `gate-cli margin` | Margin trading & lending |
| unified | `gate-cli unified` | Unified account management |
| earn | `gate-cli earn` | Earn, staking, dual investment, auto-invest |
| flash-swap | `gate-cli flash-swap` | Instant token swaps |
| mcl | `gate-cli mcl` | Multi-collateral loans |
| cross-ex | `gate-cli cross-ex` | Cross-exchange trading |
| wallet | `gate-cli wallet` | Wallet & transfers |
| account | `gate-cli account` | Account details & settings |
| sub-account | `gate-cli sub-account` | Sub-account management |
| withdrawal | `gate-cli withdrawal` | Withdrawals |
| alpha | `gate-cli alpha` | Alpha token trading |
| tradfi | `gate-cli tradfi` | TradFi (MT5) trading |
| p2p | `gate-cli p2p` | P2P trading |
| rebate | `gate-cli rebate` | Rebate & commissions |
| launch | `gate-cli launch` | Launch pool |
| activity | `gate-cli activity` | Activities & promotions |
| coupon | `gate-cli coupon` | Coupons |
| square | `gate-cli square` | Gate Square |
| welfare | `gate-cli welfare` | Welfare & tasks |
| config | `gate-cli config` | CLI configuration |

## Global flags

| Flag | Default | Description |
|------|---------|-------------|
| `--format` | `pretty` | Output format: `pretty`, `json`, or `table` (only on tabular commands) |
| `--profile` | `default` | Config profile to use |
| `--api-key` | — | API key (overrides env and config file) |
| `--api-secret` | — | API secret (overrides env and config file) |
| `--verbose` | `false` | Print Intel MCP transport lines to stderr (`info` / `news`), prefixed `[verbose]`; stdout JSON unchanged |
| `--debug` | `false` | Print HTTP debug for Gate API clients; with Intel commands, MCP transport lines use `[debug]` (wins if both flags are set) |