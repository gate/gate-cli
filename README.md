# gate-cli

A command-line interface for the [Gate](https://gate.com) API. Supports spot and futures trading, account queries, and public market data. Designed for developers, quants, and AI agents.

## Quick Start

- [English Quick Start](docs/quickstart.md)
- [中文快速上手](docs/quickstart_zh.md)

## Features

- **Spot** — market data, account balances, limit/market orders
- **Futures** — market data, account, positions, long/short/add/remove/close orders
- **Dual-position mode** — `add`, `remove`, `close` automatically detect position direction; single and dual (hedge) mode are handled transparently
- **Two output modes** — human-friendly table (default) or `--format json` for scripts and agents
- **Multiple profiles** — manage several API keys in one config file
- **Credential priority** — `--api-key` flag > env var > config file

## Installation

```bash
git clone https://github.com/gate/gate-cli.git
cd gate-cli
go build -o gate-cli .
```

## Configuration

```bash
gate-cli config init          # interactive setup → ~/.gate-cli/config.yaml
```

Or use environment variables:

```bash
export GATE_API_KEY=your-key
export GATE_API_SECRET=your-secret
```

## Usage examples

```bash
# Public market data — no API key required
gate-cli spot market ticker --pair BTC_USDT
gate-cli futures market funding-rate --contract BTC_USDT

# Account
gate-cli spot account list
gate-cli futures position list

# Spot orders
gate-cli spot order buy  --pair BTC_USDT --amount 0.001 --price 80000
gate-cli spot order buy  --pair BTC_USDT --quote 10          # market buy: spend 10 USDT
gate-cli spot order sell --pair BTC_USDT --amount 0.001

# Futures orders
gate-cli futures order long   --contract BTC_USDT --size 10 --price 80000
gate-cli futures order add    --contract BTC_USDT --size 5   # add to current position
gate-cli futures order close  --contract BTC_USDT            # close entire position

# JSON output for scripting
gate-cli spot market ticker --pair BTC_USDT --format json | jq '.last'
```

See the [Quick Start guide](docs/quickstart.md) for a full walkthrough.

## Global flags

| Flag | Default | Description |
|------|---------|-------------|
| `--format` | `table` | Output format: `table` or `json` |
| `--profile` | `default` | Config profile to use |
| `--api-key` | — | API key (overrides env and config file) |
| `--api-secret` | — | API secret (overrides env and config file) |
| `--debug` | `false` | Print raw HTTP request/response |

## Documentation

| Document | Description |
|----------|-------------|
| [docs/quickstart.md](docs/quickstart.md) | English quick start guide |
| [docs/quickstart_zh.md](docs/quickstart_zh.md) | 中文快速上手 |
| [docs/integration-test-plan.md](docs/integration-test-plan.md) | Integration test plan |
| [docs/plans/2026-03-08-gate-cli-design.md](docs/plans/2026-03-08-gate-cli-design.md) | Design document |
