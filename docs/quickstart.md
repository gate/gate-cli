# gate-cli Quick Start

## Installation

Build from source (requires Go 1.21+):

```bash
git clone https://github.com/gate/gate-cli.git
cd gate-cli
go build -o gate-cli .
sudo mv gate-cli /usr/local/bin/   # optional: install system-wide
```

Verify the install:

```bash
gate-cli --help
```

---

## Configuration

### Option A — Interactive setup (recommended)

```bash
gate-cli config init
```

This writes `~/.gate-cli/config.yaml`. You will be prompted for your API key and secret. Generate keys at **gate.com → Account → API Management**.

### Option B — Environment variables

```bash
export GATE_API_KEY=your-api-key
export GATE_API_SECRET=your-api-secret
```

### Option C — One-off flag

```bash
gate-cli spot account list --api-key your-key --api-secret your-secret
```

### Credential priority

```
--api-key / --api-secret flag
  > GATE_API_KEY / GATE_API_SECRET env vars
    > config file profile
```

### View current config

```bash
gate-cli config list              # api_key and api_secret are masked by default
gate-cli config list --show-secrets
```

---

## Public market data (no authentication required)

These commands work immediately — no API key needed.

```bash
# Spot
gate-cli spot market ticker --pair BTC_USDT
gate-cli spot market tickers
gate-cli spot market orderbook --pair BTC_USDT
gate-cli spot market trades   --pair BTC_USDT --limit 10
gate-cli spot market candlesticks --pair BTC_USDT --interval 1h --limit 48

# Futures (USDT-settled by default)
gate-cli futures market ticker --contract BTC_USDT
gate-cli futures market funding-rate --contract BTC_USDT
gate-cli futures market candlesticks --contract BTC_USDT --interval 1h
```

---

## Account

```bash
gate-cli spot account list                    # all spot balances
gate-cli spot account get --currency USDT     # single currency

gate-cli futures account get                  # futures account summary
gate-cli futures position list                # open futures positions
gate-cli futures position get --contract BTC_USDT
```

---

## Spot trading

### Limit orders

```bash
# Buy 0.001 BTC at $80,000
gate-cli spot order buy  --pair BTC_USDT --amount 0.001 --price 80000

# Sell 0.001 BTC at $82,000
gate-cli spot order sell --pair BTC_USDT --amount 0.001 --price 82000
```

### Market orders

```bash
# Market buy: specify how much quote currency (USDT) to spend
gate-cli spot order buy  --pair BTC_USDT --quote 10

# Market sell: specify how much base currency (BTC) to sell
gate-cli spot order sell --pair BTC_USDT --amount 0.001
```

> **Note:** For market buy, `--quote` is the USDT amount to spend, not the BTC amount to receive.

### Order management

```bash
gate-cli spot order list   --pair BTC_USDT
gate-cli spot order get    --pair BTC_USDT --id 123456789
gate-cli spot order cancel --pair BTC_USDT --id 123456789
gate-cli spot order cancel --pair BTC_USDT --all          # cancel all open orders
```

---

## Futures trading

`--settle` defaults to `usdt`. You can set a persistent default in the config file (`default_settle: usdt`).

### Open a position

```bash
# Limit long: buy 10 contracts at $80,000
gate-cli futures order long  --contract BTC_USDT --size 10 --price 80000

# Market short: sell 10 contracts at market price
gate-cli futures order short --contract BTC_USDT --size 10
```

### Adjust an existing position

`add` and `remove` automatically detect the current position direction (long or short) and apply the correct sign.

```bash
gate-cli futures order add    --contract BTC_USDT --size 5   # add 5 contracts in current direction
gate-cli futures order remove --contract BTC_USDT --size 5   # reduce position by 5 contracts
```

### Close a position

```bash
gate-cli futures order close --contract BTC_USDT             # close entire position
gate-cli futures order close --contract BTC_USDT --size 5    # partial close: 5 contracts
gate-cli futures order close --contract BTC_USDT --side short  # dual-position mode: close short side
```

### Order management

```bash
gate-cli futures order list   --contract BTC_USDT
gate-cli futures order get    --id 123456789
gate-cli futures order cancel --id 123456789
gate-cli futures order cancel --contract BTC_USDT --all
```

---

## Output formats

### Table (default, human-friendly)

```bash
gate-cli spot market ticker --pair BTC_USDT
```

```
Pair       Last      Change %  High 24h   Low 24h   Volume
--------   -------   --------  --------   -------   ------
BTC_USDT   83241.5   +2.34%    84100.0    81200.0   1523.41
```

### JSON (for scripts and agents)

```bash
gate-cli spot market ticker --pair BTC_USDT --format json
gate-cli futures position list --format json | jq '.[].contract'
```

---

## Multiple profiles

Useful when managing multiple API keys (e.g., main account and sub-account).

```bash
gate-cli config set api-key    your-sub-key    --profile sub
gate-cli config set api-secret your-sub-secret --profile sub

gate-cli spot account list --profile sub
```

---

## Debugging

```bash
gate-cli spot market ticker --pair BTC_USDT --debug
# Prints full HTTP request and response to stderr
```

---

## Tips for scripting

```bash
# Extract a field with jq
gate-cli spot market ticker --pair BTC_USDT --format json | jq -r '.last'

# Wait for a fill, then act
while true; do
  status=$(gate-cli spot order get --pair BTC_USDT --id 123 --format json | jq -r '.status')
  [ "$status" = "closed" ] && break
  sleep 5
done

# Use BTC-settled futures
gate-cli futures market ticker --contract BTC_USD --settle btc
```
