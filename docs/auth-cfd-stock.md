# Authorization, CFD and stock

Use `gate-cli cex` for exchange commands, `config` for API key profiles, and `info` / `news` for market intelligence. Intel credentials are separate from exchange credentials. The account's eligibility and actual permissions determine whether a private operation is available.

## Login and credentials

```sh
# Check local state before opening a new authorization flow.
gate-cli cex auth status --format json
# Select or create an Agentic subaccount and approve permissions in the browser.
gate-cli cex auth login
# Request only account access and stock trading.
gate-cli cex auth login --scope base,stock
# Use a browser on another machine and paste the returned authorization code.
gate-cli cex auth login --manual --scope base,cfd
```

| Command under `cex auth` | Behavior |
| --- | --- |
| `login` | Requests `base spot_trade futures_trade earn cfd stock` by default; the browser determines the grant. |
| `login --scope <scopes>` | Requests a comma-separated or quoted space-separated subset of those six scopes; empty/unknown values fail. |
| `login --manual` | Prints an authorization URL for copying into a browser, then accepts the authorization code. |
| `login --force` | Starts a new registration and authorization; existing local credentials remain until success. |
| `status --format json` | Reports local `status`, `issuer`, `resource`, `scopes`, `expires_at` when present, and `remote_checked: false`. |
| `refresh` | Refreshes stored authorization when possible; otherwise login is required. |
| `logout` | Clears local tokens and attempts remote revocation. If revocation is unconfirmed, disconnect on the Gate Connections page. |

`status` may be `not_logged_in`, `authenticated`, `expired (refreshable)` or `expired`. `authenticated` means locally unexpired; a remote revocation or insufficient grant can still make API calls fail. Expired refreshable credentials can refresh on the next private call. Requesting explicit scopes starts authorization even if an existing token is locally valid.

Set API keys with `gate-cli config init`. Within the selected profile, command flags override environment variables, which override the file. A complete API key/secret pair takes priority over OAuth; clear unintended key settings before choosing OAuth. Do not put real secrets in command examples, shell history or shared logs.

OAuth credentials are bound to both the issuer and complete API resource. `GATE_BASE_URL` is the API origin without `/api/v4`; changing it does not select a different OAuth issuer. Changing resource or using credentials without resource binding requires authorization again. Use HTTPS for public access.

## Permissions

These scopes describe the supported authorization contract; browser consent and service permissions remain authoritative. `base` must not be treated as universally read-only.

| Operation | OAuth scope |
| --- | --- |
| Public market queries | None |
| Private account, order and position queries described here | `base` |
| CFD order writes, position updates/closes and account transfers | `cfd` |
| Stock order writes and position closes | `stock` |
| Stock account transfers | `base` |
| Spot order writes | `spot_trade` |
| Futures, delivery and options order writes | `futures_trade` |
| AI Hub recommendation, running and detail queries | `base` |
| AI Hub spot/margin/infinite grid, spot martingale and smart-position creation/stop | `spot_trade` |
| AI Hub futures grid, contract martingale and futures-combo creation/stop | `futures_trade` |
| Earn subscription/redemption and supported borrowing/repayment operations | `earn` |

CFD and stock trading grants are independent. An API key uses the key's own permissions rather than OAuth scope names.

## CFD commands

All paths below start with `gate-cli cex tradfi`. CFD identifiers and stock identifiers are distinct. Keep order, position and log IDs as exact positive int64 values.

| Command | Parameters and purpose |
| --- | --- |
| `market categories`, `market symbols` | Public category and symbol lists. |
| `market symbol`, `market ticker` | Public details or ticker; required `--symbol`. |
| `market kline` | Required `--symbol`; optional `--type` (default `1m`), `--limit` (default 100). |
| `market commissions` | Authenticated cost query; optional `--symbols` comma-separated list and `--category-code`. |
| `account info`, `account assets` | Private MT5 account information or asset balances. |
| `account transactions` | Optional Unix-second `--begin`, `--end`, `--type`, `--page` (default 1), `--page-size` (default 20). |
| `account transfer` | Required `--change`, `--type deposit\|withdraw`; `--asset` defaults to USDT. |
| `order list` | Query open CFD orders. |
| `order history` | Optional Unix-second `--begin`, `--end`, `--symbol`, `--side` (`1` sell, `2` buy). |
| `order create` | Required `--symbol`, `--side`, `--volume`; optional `--price`, `--price-type`, `--tp`, `--sl`, `--leverage`. |
| `order update` | Required `--id`, `--price`; optional `--tp`, `--sl`. |
| `order cancel` | Required `--id`. |
| `order log` | Required `--log-id`; this is a log ID, not an order ID. |
| `position list` | Query open CFD positions. |
| `position history` | Optional Unix-second `--begin`, `--end`, `--symbol`, `--direction long\|short`. |
| `position update` | Required `--id`; optional take-profit `--tp` and stop-loss `--sl`. |
| `position close` | Required `--id` and `--close-type`; `1` partial also requires `--volume`; `2` full must omit `--volume`. |

Kline types are `1m`, `15m`, `1h`, `4h`, `1d`, `7d`, `30d`; aliases `M1`, `M15`, `H1`, `H4`, `D1` are accepted. `M5` and `M30` are unsupported.

CFD order creation uses `--price-type market|trigger`, with `market` as default and `--price 0` for market pricing. This differs from stock `market|limit` orders. Optional integer `--leverage` is omitted when absent or zero; the symbol's service limits determine permitted values.

```sh
# Public market data, followed by an authenticated cost query.
gate-cli cex tradfi market ticker --symbol XAUUSD --format json
gate-cli cex tradfi market kline --symbol XAUUSD --type 1h --limit 100 --format json
gate-cli cex tradfi market commissions --symbols XAUUSD --format json
```

## Stock commands

All paths below start with `gate-cli cex stock`. `--exchange` accepts `us`, `hk`, `kr`, `jp` where offered. Paged stock queries default to `--page 1 --page-size 10`; page must be positive and page size must be 1–500. Lists do not automatically fetch subsequent pages.

| Command | Parameters and purpose |
| --- | --- |
| `market exchanges`, `market fee-rate` | Public exchanges and fee information. |
| `market symbols` | Optional `--symbols`, `--exchange`, `--page`, `--page-size`, `--with-desc-i18n`. |
| `market symbol` | Required comma-separated `--symbols`; optional exchange and pagination. |
| `market orderbook` | Required single `--symbol`. |
| `account assets` | Optional PnL settings described below. |
| `account transactions` | Optional `--begin`, `--end` (int64 Unix seconds), `--type` and pagination, or an exact `--ref-id`. |
| `account transfer` | Required positive decimal `--change`, `--type deposit\|withdraw`, `--ref-id`; only `--asset USDT` is accepted. |
| `order list` | Open orders; optional `--symbol`. |
| `order history` | Optional `--symbol`, `--order-ids` (at most 20 positive int64 IDs), `--side`, `--begin`, `--end` and pagination. History times are int32 Unix seconds. |
| `order create` | Required `--symbol`, `--side` (`1` sell, `2` buy), positive decimal `--volume`, `--client-order-id`; pricing combinations below. |
| `order update` | Required positive int64 `--id`, positive decimal `--volume` and `--price`; both new values are required. |
| `order cancel` | Required positive int64 `--id`. |
| `order cancel-all` | Cancels all open stock orders; no symbol filter. |
| `position list` | Optional `--symbol`, `--exchange` and PnL settings. |
| `position close` | Required `--symbol`, `--close-type`; `1` partial requires positive decimal `--close-volume`; `2` full must omit it. |

Stock prices, volumes and transfer amounts are decimal strings, up to 128 bytes. Do not round them through binary floating point. Nonnegative time ranges must end at or after their start; stock order-history timestamps cannot exceed 2147483647.

`account assets` and `position list` accept `--pnl-calc-type 1|2` (average/diluted cost) and `--pnl-calc-price 1|2` (intraday/extended-hours price). Omitted values use service defaults; explicit zero is invalid.

An exact `account transactions --ref-id` cannot be combined with `--begin`, `--end`, `--type`, `--page` or `--page-size`. Supply a stable business reference when transferring and reconcile that same reference after an uncertain result; do not invent a new reference simply to retry.

### Stock order combinations

| `--price-type` | `--trading-session` | `--price` | `--time-in-force` |
| --- | --- | --- | --- |
| `market` (default) | `regular` (default) | Must be omitted, including explicit `0`. | `day` (default and only supported value) |
| `limit` | Explicit `all` | Required positive decimal. | `day` |

Do not use a CFD position ID for a stock close: stock closes identify the symbol. A stock `--client-order-id` is required on creation; keep it associated with the operation when reconciling.

```sh
# Public discovery and a bounded history query.
gate-cli cex stock market symbols --exchange us --page 1 --page-size 10 --format json
gate-cli cex stock order history --page 1 --page-size 10 --format json
```

The following example submits an order. Replace the symbol, price, quantity and unique client reference with the intended values and confirm the account before execution.

```sh
# Limit stock order: limit pricing requires the all-session setting.
gate-cli cex stock order create --symbol AAPL --side 2 --volume 1 \
  --client-order-id unique-reference --price-type limit --trading-session all \
  --price 100 --time-in-force day --format json
```

## Results and recovery

Use `--format json` in scripts and preserve stdout, stderr and exit status separately. A transport response or accepted order is not proof of a fill or completed transfer. Query the original order, position or transaction afterward. Missing routes return `API_NOT_FOUND`; this is not an empty account result.

Write commands are not automatically retried. A timeout, lost response or malformed result can leave the outcome unknown. Reconcile against the original business reference or returned ID before any further write. If reconciliation is unavailable, retain the unknown outcome rather than claiming failure or completion.
