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

Replace `<release-tag>` with a published tag that provides the required commands.

```sh
# Unix
curl -fsSL https://raw.githubusercontent.com/gate/gate-cli/main/install.sh | sh -s -- --version '<release-tag>'

# Windows
$env:GATE_CLI_VERSION="<release-tag>"; irm https://raw.githubusercontent.com/gate/gate-cli/main/install.ps1 | iex
```

### Build from source (requires Go 1.24+)

```bash
git clone https://github.com/gate/gate-cli.git
cd gate-cli
go build -o gate-cli .
sudo install -m 755 gate-cli /usr/local/bin/gate-cli
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

Flags are visible in process arguments and may enter shell history. Prefer
interactive setup for secrets; the placeholders below only show flag syntax.

```bash
gate-cli cex spot account list --api-key your-key --api-secret your-secret
```

### API server address

API base URLs accept both HTTP and HTTPS, including non-loopback test gateways.
HTTPS remains the default. HTTP traffic is unencrypted; use HTTPS for public and production access.

```bash
# Use the API origin; the CLI appends /api/v4.
export GATE_BASE_URL="https://api.gateio.ws"
```

OAuth credentials remain bound to the complete API resource, including its scheme,
host, port and path. Changing the destination requires authorization for that resource.
Changing the API origin does not change the OAuth issuer. Use HTTPS for public access.

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
gate-cli cex spot market ticker --pair BTC_USDT
gate-cli cex spot market tickers
gate-cli cex spot market orderbook --pair BTC_USDT
gate-cli cex spot market trades   --pair BTC_USDT --limit 10
gate-cli cex spot market candlesticks --pair BTC_USDT --interval 1h --limit 48

# Futures (USDT-settled by default)
gate-cli cex futures market ticker --contract BTC_USDT
gate-cli cex futures market funding-rate --contract BTC_USDT
gate-cli cex futures market candlesticks --contract BTC_USDT --interval 1h

# Additional market data; obey the deployed service's smaller page/depth limits.
gate-cli cex spot market rpi-orderbook --pair BTC_USDT --limit 10
gate-cli cex futures market rpi-orderbook --contract BTC_USDT --limit 10
gate-cli cex futures market contracts-all --limit 100 --offset 0
gate-cli cex futures market adl-risk-states --format json --max-output-bytes 1048576
```

---

## Account

```bash
gate-cli cex spot account list                    # all spot balances
gate-cli cex spot account get --currency USDT     # single currency

gate-cli cex futures account get                  # futures account summary
gate-cli cex futures position list                # open futures positions
gate-cli cex futures position get --contract BTC_USDT
```

---

## Spot trading

Writes execute immediately. The examples require authority for the selected
account, instrument and amount; inspect balances, precision and minimum notional
first. These documents describe this source tree, so check the installed leaf's
`--help` when using a public release.

### Limit orders

```bash
# Buy 0.001 BTC at $80,000
gate-cli cex spot order buy  --pair BTC_USDT --amount 0.001 --price 80000

# Sell 0.001 BTC at $82,000
gate-cli cex spot order sell --pair BTC_USDT --amount 0.001 --price 82000
```

### Market orders

```bash
# Market buy: specify how much quote currency (USDT) to spend
gate-cli cex spot order buy  --pair BTC_USDT --quote 10

# Market sell: specify how much base currency (BTC) to sell
gate-cli cex spot order sell --pair BTC_USDT --amount 0.001
```

> **Note:** For market buy, `--quote` is the USDT amount to spend, not the BTC amount to receive.

### Order management

```bash
gate-cli cex spot order list   --pair BTC_USDT
gate-cli cex spot order get    --pair BTC_USDT --id 123456789
gate-cli cex spot order cancel --pair BTC_USDT --id 123456789
gate-cli cex spot order cancel --pair BTC_USDT --all          # cancel all open orders
```

### Account selection and attached protection

Use `--account spot|margin|unified|cross_margin` on supported order reads/writes.
Automatic borrowing needs an explicit eligible account; order-level auto-repay
supports unified/cross-margin. Both need borrowing/repayment authority.

```bash
# Authorized unified-account limit buy with separate take profit and stop loss.
gate-cli cex spot order buy --pair BTC_USDT --account unified --amount 0.001 --price 60000 \
  --stop-profit-json '{"trigger_price":"66000","order_price":"65900"}' \
  --stop-loss-json '{"trigger_price":"57000","order_price":"56900"}'

# Requires authority to set the parent price as well as clear its stop loss.
gate-cli cex spot order amend --pair BTC_USDT --account unified --id 123456789 \
  --price 60000 --stop-loss-json '{}'
```

On amendment, omission or JSON `null` preserves a protection; only `{}` clears it.
The API also requires a parent amount or price. Do not silently add an old value.
Check both protections afterward and minimum notional for all affected orders.
See the [extension guide](cex-gap-commands.md) for automatic borrowing and native POV.

---

## Futures trading

`--settle` defaults to `usdt`. You can set a persistent default in the config file (`default_settle: usdt`).

### Open a position

```bash
# Limit long: buy 10 contracts at $80,000
gate-cli cex futures order long  --contract BTC_USDT --size 10 --price 80000

# Market short: sell 10 contracts at market price
gate-cli cex futures order short --contract BTC_USDT --size 10
```

### Adjust an existing position

`add` and `remove` automatically detect the current position direction (long or short) and apply the correct sign.

```bash
gate-cli cex futures order add    --contract BTC_USDT --size 5   # add 5 contracts in current direction
gate-cli cex futures order remove --contract BTC_USDT --size 5   # reduce position by 5 contracts
```

### Close a position

```bash
gate-cli cex futures order close --contract BTC_USDT             # close entire position
gate-cli cex futures order close --contract BTC_USDT --size 5    # partial close: 5 contracts
gate-cli cex futures order close --contract BTC_USDT --side short  # dual-position mode: close short side
```

### Order management

```bash
gate-cli cex futures order list   --contract BTC_USDT
gate-cli cex futures order get    --id 123456789
gate-cli cex futures order cancel --id 123456789
gate-cli cex futures order cancel --contract BTC_USDT --all
```

---

## Delivery futures

Delivery supports USDT settlement. Use signed sizes with `order create`; the perpetual `long/close` helpers do not apply. Discover an active contract first; names and IDs below are illustrative. `position close` reads history.

```bash
# Market data (public)
gate-cli cex delivery market contracts
gate-cli cex delivery market tickers   --contract BTC_USDT_20261225
gate-cli cex delivery market order-book --contract BTC_USDT_20261225

# Account & positions
gate-cli cex delivery account get
gate-cli cex delivery position list

# Orders
gate-cli cex delivery order create  --contract BTC_USDT_20261225 --size 5 --price 80000
gate-cli cex delivery order cancel --id 123456789
gate-cli cex delivery order list --contract BTC_USDT_20261225 --status open
```

---

## Options

Discover an active contract first; the contract names and order IDs below are
illustrative and must be replaced with the intended live objects.

```bash
# Market data (public)
gate-cli cex options market underlyings
gate-cli cex options market contracts --underlying BTC_USDT
gate-cli cex options market tickers   --underlying BTC_USDT

# Account & positions
gate-cli cex options account get
gate-cli cex options position list

# Orders
gate-cli cex options order create --contract BTC_USDT-20261225-60000-C --size 1 --price 500
gate-cli cex options order list --underlying BTC_USDT --status open
gate-cli cex options order cancel --id 123456789

# Partial long reduction and full close, subject to position and liquidity checks.
gate-cli cex options order create --contract BTC_USDT-20261225-60000-C --size -1 --reduce-only --price 0 --tif ioc
gate-cli cex options order create --contract BTC_USDT-20261225-60000-C --size 0 --close --price 0 --tif ioc

# Market Maker Protection
gate-cli cex options mmp get   --underlying BTC_USDT
gate-cli cex options mmp set   --underlying BTC_USDT --window 5000 --frozen-period 30000 --qty-limit 100 --delta-limit 50
gate-cli cex options mmp reset --underlying BTC_USDT
```

Full close uses `--close --size 0`; partial reduction uses a signed nonzero size
with `--reduce-only`. Price 0 requires IOC. `options position close` reads close
history and does not place a closing order.

---

## Wallet

```bash
# Balances
gate-cli cex wallet balance total                         # total balance across all accounts
gate-cli cex wallet balance small                         # list dust balances
gate-cli cex wallet balance sub --sub-uid 12345             # sub-account balance

# Deposits & withdrawals
gate-cli cex wallet deposit address --currency USDT
gate-cli cex wallet deposit list    --currency USDT --limit 20
gate-cli cex wallet deposit withdrawals   --currency USDT --limit 20
gate-cli cex wallet market withdraw-status                       # supported currencies and chain info

# Transfers
gate-cli cex wallet transfer create --currency USDT --amount 100 --from spot --to futures
gate-cli cex wallet transfer to-sub --currency USDT --amount 100 --sub-account 12345 --direction to
```

---

## Account

```bash
gate-cli cex account detail                       # UID, email, tier, KYC status
gate-cli cex account rate-limit                   # API rate limit info
gate-cli cex account keys                    # list main account API keys

# STP (Self-Trade Prevention) groups
gate-cli cex account stp list
gate-cli cex account stp create --name my-group
gate-cli cex account stp users  --id 1
```

---

## Price-triggered orders

Place an order automatically when the market reaches a trigger price.

```bash
# Spot
gate-cli cex spot price-trigger list
gate-cli cex spot price-trigger create \
  --market BTC_USDT --trigger-price 90000 --side sell \
  --price 90500 --amount 0.001
gate-cli cex spot price-trigger cancel     --id 123456
gate-cli cex spot price-trigger cancel-all --market BTC_USDT

# Futures
gate-cli cex futures price-trigger list
gate-cli cex futures price-trigger create \
  --contract BTC_USDT --trigger-price 90000 --price 0 --size -10 --tif ioc
gate-cli cex futures price-trigger get    --id 456
gate-cli cex futures price-trigger update --id 456 --body-order-id 789 --trigger-price 91000
gate-cli cex futures price-trigger cancel --id 456
```

Spot triggered limit orders default `--time-in-force` to `ioc`: when triggered, execute immediately and cancel any unfilled remainder. Explicit `gtc` remains available under the SDK contract, subject to server support; the CLI never silently rewrites or retries it as another policy. Other values fail before a request is sent.
Spot `order batch-create`, `batch-amend` and `batch-cancel` preserve the SDK per-item result array. Any failed item returns a nonzero exit status and a success/failure count on stderr. Reconcile per-item outcomes; do not automatically retry the entire batch.


---

## Trailing stop orders (futures)

Trail the market by a ratio or price distance; order triggers automatically when the market reverses.
Creation requires an explicit `--price-type`: `1` for latest price, `2` for index price, or `3` for mark price. Returned Trail IDs and related time fields are strings.
For a nonzero activation price, `--is-gte=true` activates when the selected reference price is greater than or equal to that price; `false` uses less than or equal. Creation defaults to `false`. Updating without this flag preserves the direction; use `--is-gte=false` explicitly to change it to the lower-price condition. An activation price above the current market is not a guarantee that the condition will wait.

Historical lists require `--finished --start-at --end-at`, with positive Unix seconds and start strictly before end. Do not pass milliseconds. Time filters are rejected on active lists because the service does not apply them. `--limit` supports 1–100; omitted or 0 keeps the service default. No additional maximum time-span rule is imposed by the CLI.

```bash
gate-cli cex futures trail create \
  --contract BTC_USDT --amount -10 --price-type 1 --price-offset '2%'   # sell order with a 2% callback

gate-cli cex futures trail list
# Query a specific historical time window in Unix seconds.
gate-cli cex futures trail list --finished --start-at 1793534400 --end-at 1793538000 --limit 20
gate-cli cex futures trail get    --id 789
gate-cli cex futures trail update --id 789 --price-offset '1.5%'
# Explicit false changes direction; omitting the flag leaves it unchanged.
gate-cli cex futures trail update --id 789 --is-gte=false
gate-cli cex futures trail log    --id 789                    # change history
gate-cli cex futures trail stop   --id 789
gate-cli cex futures trail stop-all --contract BTC_USDT
```

When a futures/delivery account or order-list query returns `USER_NOT_FOUND`, the CLI reports `ACCOUNT_NOT_READY` with `retryable=false`, preserving the upstream reason. Check whether the product account is enabled or initialized through its official workflow; do not repeat the same request, change flags, or transfer funds automatically. A matching label on an unrecognized operation remains `UNKNOWN`; it is not an empty account result. Authentication, permission and service failures retain their own error types and exit codes.

---

## Output formats

### Pretty (default) and explicit table output

The default format is `pretty`. Use `--format table` explicitly on commands that support tabular output; non-tabular results use the shared JSON renderer. Errors remain on stderr with a non-zero exit code.

```bash
gate-cli cex spot market ticker --pair BTC_USDT --format table
```

```
Pair       Last      Change %  High 24h   Low 24h   Volume
--------   -------   --------  --------   -------   ------
BTC_USDT   83241.5   +2.34%    84100.0    81200.0   1523.41
```

### JSON (for scripts and agents)

```bash
gate-cli cex spot market ticker --pair BTC_USDT --format json
gate-cli cex futures position list --format json | jq '.[].contract'
```

---

## Intel (`info` & `news`)

Market intelligence and news are shipped as **50** MCP-style CLI tools (**32** `info`, **18** `news`). Invoke a capability as `gate-cli info <group> <tool>` or `gate-cli news <group> <tool>` with **flat flags** for arguments (add `--format json` for scripts and agents).

**`info` groups:** `coin`, `marketsnapshot`, `markettrend`, `onchain`, `platformmetrics`, `marketdetail`, `macro`, `compliance`.

**`news` groups:** `feed` (news, UGC, X, web research, sentiment and announcements), `events` (event lists/details, market-move explanations and saved reports), and `prediction` (rankings, event search, snapshots and live books). `search-ugc` requires `--query` or `--coin`; `search-events` requires at least one of `--query`, `--coin`, `--category`. Inspect partial-result status and metadata. Use each command's `--help` or `gate-cli news mcp-spec` for parameters.

**`info` help text:** each leaf `-h` starts with an English **`description`** (`[Read]` routing) from the spec embedded in the binary; set `GATE_INTEL_LEAF_HELP=full` for per-field notes. Same document: `gate-cli info mcp-spec --format json`.

List tool ids: `gate-cli info list`, `gate-cli news list`. Flags and env vars: `gate-cli info -h`, `gate-cli news -h`.
Deterministic shortcut flows: `gate-cli info +coin-overview --symbol BTC` and `gate-cli news +brief --coin BTC`.

Optional Intel defaults go under `intel:` in `~/.gate-cli/config.yaml`. Gate trading `--api-key` / `GATE_API_KEY` are **not** used as the Intel bearer; follow your Intel/MCP backend if a bearer or base URL is required. See the [README Intel configuration](../README.md#intel-info-news) for environment variables, URLs and timeouts.

```bash
gate-cli info coin get-coin-info --query BTC --format json
gate-cli info marketsnapshot get-market-snapshot --symbol BTC_USDT --format json
gate-cli info marketsnapshot get-institutional-metrics --asset BTC --channel all --limit 30 --format json
gate-cli info platformmetrics get-stablecoin-info --scope full --sections '["issuance_flow","usage_structure"]' --start-date 2026-04-01 --end-date 2026-05-01 --format json
gate-cli info platformmetrics get-chain-activity --metric-group staking --lookback 30d --format json
gate-cli news feed search-news --query bitcoin --format json   # alias: search → search-news
gate-cli news feed search-x --query bitcoin --time-range 7d --lang en --format json
gate-cli news events get-market-move-report --symbol TAIKO --report-id _1782973063855 --format json
gate-cli news events list-market-move-reports --symbol ETH --start-time "2026-07-09 22:00:00" --end-time "2026-07-10 03:15:00" --limit 10 --format json # filters updated_at in UTC0; sorts by event_time DESC; --limit 0 defaults to 20
```

One minimal example per tool (all 50) is in the repository **README.md**.

### GateAI agent mode

The `agent-*` commands require a binary built with `-tags agent`; standard builds
do not register them, and setting the environment variable alone is insufficient.
Build from a trusted source checkout, then enable agent defaults:

```bash
# From the source checkout; use this binary for the following examples.
go build -tags agent -o gate-cli .
export PATH="$PWD:$PATH"
export GATE_CLI_AGENT=1
gate-cli agent-resolve --query "BTC news brief" --format json
gate-cli agent-leaves --format json          # 31 curated intents + 50 MCP tools
gate-cli agent-validate --format json        # Check command availability
```

For standard builds, use `info list`, `news list` and the selected leaf's `--help`. Scripts should set JSON format and an explicit output bound.

---

## Multiple profiles

Useful when managing multiple API keys (e.g., main account and sub-account).

```bash
gate-cli config set api-key    your-sub-key    --profile sub
gate-cli config set api-secret your-sub-secret --profile sub

gate-cli cex spot account list --profile sub
```

---

## Debugging

```bash
gate-cli cex spot market ticker --pair BTC_USDT --debug
# CEX transport logs method, status and elapsed time; query/headers/body are omitted
```

For **`info`** / **`news`**, root `--debug` and `--verbose` print Intel MCP transport lines on **stderr** (stdout JSON shape unchanged). Cap printed bytes with `--max-output-bytes` or `GATE_MAX_OUTPUT_BYTES`. See the **Global flags** table in the repository `README.md`.

---

## Tips for scripting

```bash
# Extract a field with jq
gate-cli cex spot market ticker --pair BTC_USDT --format json | jq -r '.last'

# Read completion evidence; status alone does not prove a full fill.
gate-cli cex spot order get --pair BTC_USDT --id 123 --format json | jq '{status, finish_as, left}'

# Use BTC-settled futures
gate-cli cex futures market ticker --contract BTC_USD --settle btc
```

Keep stdout, stderr and exit status separate. Bound process duration, output,
pagination and polling; never automatically replay a write after a timeout or
lost response. A chase HTTP 200 business rejection is a failed CLI call; an empty
active chase list is `{"orders":[]}`. See [result handling](cex-gap-commands.md#results-and-recovery).

## Agentic subaccount, CFD and stock

```sh
# Inspect local authorization before starting a browser login.
gate-cli cex auth status --format json
# Request account access and stock trading; confirm the grant in the browser.
gate-cli cex auth login --scope base,stock
# Public stock and CFD market queries need no login.
gate-cli cex stock market symbols --exchange us --page 1 --page-size 10
gate-cli cex tradfi market kline --symbol XAUUSD --type 1h --limit 100
# Private queries require base permission.
gate-cli cex stock account assets --format json
gate-cli cex tradfi market commissions --symbols XAUUSD --format json
```

Without `--scope`, login requests `base spot_trade futures_trade earn cfd stock`; browser consent determines the grant. `--manual` lets you copy the URL and paste the code; `--force` starts new authorization. `auth refresh` refreshes credentials; `auth logout` clears local credentials and attempts remote revocation. A revocation error requires disconnecting the connection on Gate.

`auth status` is a local check (`remote_checked: false`), and cannot prove remote access. Credentials are bound to the API resource; authorize again when it changes. A complete API key/secret pair takes priority over OAuth; Intel credentials are separate. This source uses public Gate Go SDK `v7.2.179`.

Private CFD and stock queries use `base`. CFD trading and transfers use `cfd`; stock order changes and position closes use `stock`; stock account transfers use `base`. See the [complete authorization, CFD and stock guide](auth-cfd-stock.md) for the command tables, stock order/session combinations, close parameters, reference IDs and pagination. The [CEX reference](cex-gap-commands.md) covers POV/chase, Earn filters, order references and partial failures.

## Unavailable commands

Sub-account management, LaunchPool, HODLer and CandyDrop commands are unavailable, as are P2P, CrossEx, welfare, activity and coupon groups. Authorizing an Agentic subaccount through `cex auth` remains available. Use the Gate website to manage products without a CLI command.

## Explicit CEX and CLI diagnostics

`version` and `--version` retain their existing output. Use `version --details --format json` for the revision, Go version, platform and compiled agent command capability. A missing revision is reported as `unknown`; runtime environment variables do not change the compiled capability.

`config check --format json` explicitly checks local configuration file type, read access and permissions without reading credential contents or changing files. It does not verify YAML syntax, API authorization or Windows ACLs. Exit 0 means pass; warnings/failures exit 1.

`config init --hide-input` hides all fields, including the profile, on a real terminal. Piped input uses ordinary input behavior. Ctrl-C/TERM/input failure restore the terminal before any configuration write; a force-killed process cannot guarantee restoration.

Ordinary spot buy/sell and futures long/short/add/remove accept optional `--text t-reference` (ASCII alphanumeric, `_`, `-`, `.`, at most 28 bytes after `t-`). It is a correlation reference, not a server-side idempotency guarantee. Spot can query pending orders with `cex spot order get --id t-reference --pair BTC_USDT`; use numeric IDs for completed orders. Futures adds `cex futures order get-by-text --text t-reference`; server historical/status limits apply (cancelled unfilled orders have a 60-second text lookup limit). A failed lookup does not prove a write was never submitted, and must not trigger automatic resubmission.

Futures batch create/amend/cancel keep the original returned result array. Any decoded failed item now produces exit 1 and an error summary on stderr; all-success results retain exit 0. Inspect successful IDs and failed items separately; do not resubmit the entire batch. Use explicit `--max-output-bytes 0` if the complete result is required, since the existing cap may replace an oversized JSON payload with its placeholder.

Agent builds additionally support `agent-index --domain cex --details --format json` and an optional canonical `--command 'cex spot order buy'`. The report describes registered parameters, unconditional required flags and explicitly known read/write effects. `unknown` is not a safety classification; conditional requirements are not claimed complete. It does not execute or authorize the described command. Standard builds retain their existing command availability.

For upgrades, select an explicitly verified release version using the existing installation method, retain the previous binary, and compare `version --details` before/after installation. Roll back by restoring that binary; do not replay any uncertain trade as part of rollback.
