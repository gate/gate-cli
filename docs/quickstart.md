# gate-cli Quick Start

## Installation

### macOS / Linux — shell script

```sh
curl -fsSL https://raw.githubusercontent.com/gate/gate-cli/main/install.sh | sh
```

### macOS — Homebrew

```sh
brew install gate/tap/gate-cli
```

### Windows — PowerShell

```powershell
irm https://raw.githubusercontent.com/gate/gate-cli/main/install.ps1 | iex
```

### Pin to a specific version

```sh
# Unix
curl -fsSL https://raw.githubusercontent.com/gate/gate-cli/main/install.sh | sh -s -- --version v0.3.2

# Windows
$env:GATE_CLI_VERSION="v0.3.2"; irm https://raw.githubusercontent.com/gate/gate-cli/main/install.ps1 | iex
```

### Build from source (requires Go 1.21+)

```bash
git clone https://github.com/gate/gate-cli.git
cd gate-cli
go build -o gate-cli .
sudo mv gate-cli /usr/local/bin/
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

## Delivery futures

Delivery futures follow the same pattern as perpetual futures. Only USDT settlement is supported.

```bash
# Market data (public)
gate-cli delivery market contracts
gate-cli delivery market ticker   --contract BTC_USDT_20260327
gate-cli delivery market orderbook --contract BTC_USDT_20260327

# Account & positions
gate-cli delivery account get
gate-cli delivery position list

# Orders
gate-cli delivery order long  --contract BTC_USDT_20260327 --size 5 --price 80000
gate-cli delivery order close --contract BTC_USDT_20260327
gate-cli delivery order list  --contract BTC_USDT_20260327
```

---

## Options

```bash
# Market data (public)
gate-cli options market underlyings
gate-cli options market contracts --underlying BTC_USDT
gate-cli options market tickers   --underlying BTC_USDT

# Account & positions
gate-cli options account list
gate-cli options position list

# Orders
gate-cli options order create --contract BTC_USDT-20260327-80000-C --size 1 --price 500
gate-cli options order list
gate-cli options order cancel --order-id 123456789

# Market Maker Protection
gate-cli options mmp get   --underlying BTC_USDT
gate-cli options mmp set   --underlying BTC_USDT --window 5000 --freeze-period 30000 --qty-limit 100 --delta-limit 50
gate-cli options mmp reset --underlying BTC_USDT
```

---

## Wallet

```bash
# Balances
gate-cli wallet balance total                         # total balance across all accounts
gate-cli wallet balance small                         # list dust balances
gate-cli wallet balance sa --sa-uid 12345             # sub-account balance

# Deposits & withdrawals
gate-cli wallet deposit address --currency USDT --chain TRX
gate-cli wallet deposit list    --currency USDT --limit 20
gate-cli wallet withdraw list   --currency USDT --limit 20
gate-cli wallet withdraw status                       # supported currencies and chain info

# Transfers
gate-cli wallet transfer create --currency USDT --amount 100 --from spot --to futures
gate-cli wallet transfer sa     --currency USDT --amount 100 --sa-uid 12345 --direction to
```

---

## Account

```bash
gate-cli account detail                       # UID, email, tier, KYC status
gate-cli account rate-limit                   # API rate limit info
gate-cli account main-keys                    # list main account API keys

# STP (Self-Trade Prevention) groups
gate-cli account stp list
gate-cli account stp create --name my-group
gate-cli account stp users  --id 1
```

---

## Price-triggered orders

Place an order automatically when the market reaches a trigger price.

```bash
# Spot
gate-cli spot price-trigger list
gate-cli spot price-trigger create \
  --market BTC_USDT --trigger-price 90000 --side sell \
  --price 90500 --amount 0.001
gate-cli spot price-trigger cancel     --id 123456
gate-cli spot price-trigger cancel-all --market BTC_USDT

# Futures
gate-cli futures price-trigger list
gate-cli futures price-trigger create \
  --contract BTC_USDT --trigger-price 90000 --price 0 --size -10
gate-cli futures price-trigger get    --id 456
gate-cli futures price-trigger update --id 456 --trigger-price 91000
gate-cli futures price-trigger cancel --id 456
```

---

## Trailing stop orders (futures)

Trail the market by a ratio or price distance; order triggers automatically when the market reverses.

```bash
gate-cli futures trail create \
  --contract BTC_USDT --amount -10 --price-offset 0.02   # trail short by 2%

gate-cli futures trail list
gate-cli futures trail get    --id 789
gate-cli futures trail update --id 789 --price-offset 0.015
gate-cli futures trail log    --id 789                    # change history
gate-cli futures trail stop   --id 789
gate-cli futures trail stop-all --contract BTC_USDT
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
