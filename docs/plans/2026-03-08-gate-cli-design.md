# gate-cli Design Document

Date: 2026-03-08

## Overview

gate-cli is a command-line tool that wraps the Gate API, designed for developers, quants, AI agents, and general traders. It simplifies Gate API usage via intuitive CLI commands while remaining agent-friendly through structured JSON output.

## Target Users

- **Developers / Quants** — scripting, automation, API testing
- **AI Agents** — structured output for programmatic consumption
- **General Traders** — human-friendly table output

## Technology Stack

- **Language**: Go
- **CLI Framework**: [Cobra](https://github.com/spf13/cobra)
- **API SDK**: Gate official Go SDK (`github.com/gate/gateapi-go/v7`)
- **Table output**: `github.com/olekukonko/tablewriter`
- **Release**: `goreleaser` + GitHub Actions (darwin/linux/windows multi-arch)

## Architecture

```
gate-cli/
├── cmd/
│   ├── root.go          # Root command, global flags (--format, --profile, --debug)
│   ├── config/
│   │   └── config.go    # config init/set/list subcommands
│   ├── spot/
│   │   ├── spot.go      # spot command group
│   │   ├── account.go   # spot account subcommands
│   │   ├── order.go     # spot order subcommands
│   │   └── market.go    # spot market subcommands
│   └── futures/
│       ├── futures.go   # futures command group
│       ├── account.go   # futures account/position subcommands
│       ├── order.go     # futures order subcommands
│       └── market.go    # futures market subcommands
├── internal/
│   ├── client/          # Gate API client initialization and auth
│   ├── config/          # Config file read/write, profile management
│   └── output/          # Table/JSON rendering, unified output interface
├── main.go
├── .gate-cli.yaml       # Example config file
└── .goreleaser.yaml     # Multi-platform build config
```

## Configuration

### Config File: `~/.gate-cli/config.yaml`

```yaml
default_profile: main
default_settle: usdt

profiles:
  main:
    api_key: "your-api-key"
    api_secret: "your-api-secret"
    base_url: "https://api.gateio.ws"  # optional, has default
  sub:
    api_key: "sub-api-key"
    api_secret: "sub-api-secret"
```

### Priority (highest to lowest)

```
CLI flag (--api-key)
  > Environment variable (GATE_API_KEY / GATE_API_SECRET)
    > Config file profile
      > Default profile
```

### Config Commands

```bash
gate-cli config init                              # Interactive setup
gate-cli config set api-key <key>                 # Set key for current profile
gate-cli config set api-key <key> --profile sub   # Set key for named profile
gate-cli config list                              # List all profiles
```

## Output Format

### Global Flags

```bash
--format json      # Output JSON (default: table)
--profile main     # Use named profile (default: "default")
--debug            # Print raw HTTP request/response
```

### Default (TTY table)

```
Currency    Available       Locked
--------    ---------       ------
BTC         0.12300000      0.00000000
USDT        1250.50000000   500.00000000
```

### JSON output (`--format json`)

- Lists return arrays, single items return objects
- Field names match Gate API original field names (no translation) for unambiguous agent parsing
- Output goes to stdout; errors go to stderr

### Error Format

All errors (both JSON and table mode) include:
- HTTP status code
- Gate `label` + `message` (when Gate standard error format is parseable)
- `x-gate-trace-id` response header value
- Request info: method, URL, body

**JSON error (Gate standard):**
```json
{
  "error": {
    "status": 400,
    "label": "INVALID_PARAM_VALUE",
    "message": "Invalid currency pair",
    "trace_id": "abc123xyz",
    "request": {
      "method": "POST",
      "url": "https://api.gateio.ws/api/v4/spot/orders",
      "body": "{\"currency_pair\":\"BTC_USDT\",...}"
    }
  }
}
```

**JSON error (generic HTTP):**
```json
{
  "error": {
    "status": 502,
    "message": "Bad Gateway",
    "trace_id": "abc123xyz",
    "request": {
      "method": "POST",
      "url": "https://api.gateio.ws/api/v4/spot/orders",
      "body": "{\"currency_pair\":\"BTC_USDT\",...}"
    }
  }
}
```

**Table mode stderr:**
```
Error [400 INVALID_PARAM_VALUE]: Invalid currency pair
Trace ID: abc123xyz
Request: POST https://api.gateio.ws/api/v4/spot/orders
```

## Command Reference

### Spot Market (public, no auth)

```bash
gate-cli spot market ticker --pair BTC_USDT
gate-cli spot market tickers
gate-cli spot market orderbook --pair BTC_USDT [--depth 20]
gate-cli spot market trades --pair BTC_USDT [--limit 20]
gate-cli spot market candlesticks --pair BTC_USDT [--interval 1h] [--limit 100]
```

### Spot Account

```bash
gate-cli spot account list
gate-cli spot account get --currency BTC
```

### Spot Orders

```bash
gate-cli spot order buy  --pair BTC_USDT --amount 0.01 --price 80000
gate-cli spot order buy  --pair BTC_USDT --amount 0.01              # market order
gate-cli spot order sell --pair BTC_USDT --amount 0.01 --price 82000
gate-cli spot order sell --pair BTC_USDT --amount 0.01              # market order

gate-cli spot order get --id 12345678 --pair BTC_USDT
gate-cli spot order list --pair BTC_USDT [--status open|closed|cancelled]
gate-cli spot order cancel --id 12345678 --pair BTC_USDT
gate-cli spot order cancel --all --pair BTC_USDT
```

### Futures Market (public, no auth)

```bash
gate-cli futures market ticker --contract BTC_USDT [--settle usdt]
gate-cli futures market tickers [--settle usdt]
gate-cli futures market orderbook --contract BTC_USDT [--settle usdt]
gate-cli futures market trades --contract BTC_USDT [--settle usdt]
gate-cli futures market candlesticks --contract BTC_USDT [--interval 1h] [--settle usdt]
gate-cli futures market funding-rate --contract BTC_USDT [--settle usdt]
```

### Futures Account & Positions

```bash
gate-cli futures account get [--settle usdt]
gate-cli futures position list [--settle usdt]
gate-cli futures position get --contract BTC_USDT [--settle usdt]
```

### Futures Orders

`long`/`short`/`add`/`remove`/`close` automatically handle Gate API `size` sign conversion internally.

```bash
# Open position
gate-cli futures order long  --contract BTC_USDT --size 10 --price 80000
gate-cli futures order short --contract BTC_USDT --size 10 --price 80000
gate-cli futures order long  --contract BTC_USDT --size 10              # market
gate-cli futures order short --contract BTC_USDT --size 10              # market

# Adjust position
gate-cli futures order add    --contract BTC_USDT --size 5 --price 80000
gate-cli futures order remove --contract BTC_USDT --size 5 --price 79000

# Close position
gate-cli futures order close --contract BTC_USDT                        # close all
gate-cli futures order close --contract BTC_USDT --size 5               # partial close

# Manage orders
gate-cli futures order get --id 12345678 [--settle usdt]
gate-cli futures order list --contract BTC_USDT [--status open|finished] [--settle usdt]
gate-cli futures order cancel --id 12345678 [--settle usdt]
gate-cli futures order cancel --all --contract BTC_USDT [--settle usdt]
```

## Default Values

| Parameter    | Default  | Override via        |
|--------------|----------|---------------------|
| `--settle`   | `usdt`   | config file         |
| `--limit`    | `20`     | flag                |
| `--interval` | `1h`     | flag                |
| `--format`   | `table`  | flag / env          |
| `--profile`  | `default`| flag / env          |

## Future Considerations (Out of Scope for v1)

- Further command simplification (e.g., merging complex scenarios)
- Options / delivery contract support
- Sub-account management commands
- Wallet / transfer commands
- Interactive TUI mode
