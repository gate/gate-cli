# CEX command reference

Run
`gate-cli cex auth login` for OAuth, or use the existing API key configuration.
Read-only public endpoints need no credentials. Private commands require account
and API permissions. Write commands submit immediately: verify the account, market and
amount before running them. A transport failure can mean an unknown write outcome;
reconcile the existing order or settings before trying again.

This source uses public SDK `v7.2.178`. Published binaries may expose an earlier
command set; inspect the installed command's `--help` instead of assuming support
from this guide. Account eligibility, scope grants and deployed backend support
are separate requirements.

## Command index

All paths below start with `gate-cli cex`. Use `--help` on a leaf for every flag.
JSON output follows the response structure of each command; there is no universal success envelope. Use `--format json` for automation.

| Command | API operation | Access |
| --- | --- | --- |
| `spot pov create` | `POST /spot/pov_orders` | Private write |
| `spot pov list` | `GET /spot/pov_orders` | Private read |
| `spot pov get` | `GET /spot/pov_orders/{order_id}` | Private read |
| `spot pov cancel` | `DELETE /spot/pov_orders/{order_id}` | Private write |
| `spot pov cancel-all` | `DELETE /spot/pov_orders` | Private write |
| `spot market rpi-orderbook` | `GET /spot/rpi_order_book` | Public read |
| `futures market contracts-all` | `GET /futures/{settle}/contracts_all` | Public read |
| `futures market rpi-orderbook` | `GET /futures/{settle}/rpi_order_book` | Public read |
| `futures market adl-risk-states` | `GET /futures/{settle}/adl_risk_states` | Public read |
| `futures chase create` | `POST /futures/{settle}/autoorder/v1/chase/create` | Private write |
| `futures chase get` | `GET /futures/{settle}/autoorder/v1/chase/detail` | Private read |
| `futures chase list` | `GET /futures/{settle}/autoorder/v1/chase/list` | Private read |
| `futures chase stop` | `POST /futures/{settle}/autoorder/v1/chase/stop` | Private write |
| `futures chase stop-all` | `POST /futures/{settle}/autoorder/v1/chase/stop_all` | Private write |
| `unified config delta-neutral-get` | `GET /unified/delta_neutral` | Private read |
| `unified config delta-neutral-set` | `POST /unified/delta_neutral` | Private write |
| `unified loan quick-estimate` | `GET /unified/estimated_quick_repayment` | Private read |
| `unified loan quick-repay` | `POST /unified/quick_repayment` | Private write |
| `unified config leverage-set-all` | `POST /unified/leverage/user_setting` | Private write |
| `tradfi market commissions` | `GET /tradfi/symbols/commissions` | Private read |
| `tradfi order log` | `GET /tradfi/orders/log/{log_id}` | Private read |

`tradfi` is the CFD command group. API paths use the usual `/api/v4` prefix.
Futures commands honor the existing `--settle` / profile setting. Spot POV list
supports pages 1–100 and limits 1–1000. Futures RPI depth is at most 50; spot RPI
depth is at most 100. Lists fetch a single bounded page; there is no automatic
pagination. `pov cancel-all` without `--pair` cancels across all pairs;
`chase stop-all` without filters stops all chase orders for the settlement currency.

For all-contract metadata, start with `--limit 100 --offset 0`. The CLI permits up to 1000; respect any smaller service limit. ADL responses may exceed the default stdout budget: use an
explicit bounded `--max-output-bytes`, and never accept a truncation marker as a
complete result.

## POV and chase orders

```sh
# Participation rate is 5, 10, 20 or 40; amount is base currency.
gate-cli cex spot pov create --pair BTC_USDT --side buy --amount 0.001 \
  --participation-rate 10 --ttl 1h --limit-price 60000

# A signed contract amount selects buy/sell. Keep monetary values as JSON strings.
gate-cli cex futures chase create --settle usdt \
  --json '{"contract":"BTC_USDT","amount":"-1","price_limit":"0","reduce_only":true}'
```

`chase create --json` accepts one request object:

| Field | Meaning |
| --- | --- |
| `contract` | Required contract name. |
| `amount` | Required nonzero signed decimal contract quantity; positive buys, negative sells. |
| `price_limit` | Required decimal chase limit; `"0"` means no price limit. |
| `offset_limit` | Positive chase distance; cannot accompany a nonzero `price_limit`. |
| `reduce_only` | Boolean; only reduce an existing position. |
| `text` | Custom order tag. |
| `is_dual_mode` | Boolean; dual-position mode. |
| `price_type` | `1` best bid/ask, `2` distance from best bid/ask; omit for server default. |
| `price_gap_type` | Required with type 2: `1` absolute gap, `2` percentage. |
| `price_gap_value` | Positive decimal string required with type 2. |
| `pos_margin_mode` | `isolated` or `cross`. |
| `position_mode` | `single`, `dual` or `dual_plus`. |
| `settle` | Optional; must match the effective settlement currency. Prefer `--settle`. |

Unknown fields, duplicate keys, null field values and trailing JSON are rejected.
`chase list --finished` requires both `--from` and `--to` Unix timestamps.
`chase stop` requires exactly one of `--id` or `--text`.
Default lists explicitly select active orders with `is_finished=false`. Confirm the service's timestamp unit before a time-filtered query. Stopping a parent is not proof that every child order
or resulting position has terminated; query them before reporting completion.

## Margin accounts and attached protection

`spot order buy`, `sell`, `get`, `list`, `cancel`, `amend`, `all-open` and
`my-trades` accept `--account spot|margin|unified|cross_margin`. Omit it to keep the
existing server account selection. The `my-trades` account filter is deprecated: the server may return records across accounts. Batch JSON commands
already accept account fields per item; no shared flag overrides those values.

`buy` and `sell` also accept `--auto-borrow` for an explicit `margin`, `unified` or
`cross_margin` account. `--auto-repay` is supported for explicit `unified` and
`cross_margin` orders. Isolated margin repayment uses its existing account-level
setting; it cannot be enabled through the order's `--auto-repay` flag.

```sh
# Limit buy with separate attached take-profit and stop-loss instructions.
gate-cli cex spot order buy --pair BTC_USDT --account unified \
  --amount 0.001 --price 60000 --auto-borrow --auto-repay \
  --stop-profit-json '{"trigger_price":"66000","order_price":"65900"}' \
  --stop-loss-json '{"trigger_price":"57000","order_price":"56900"}'

# Cancel stop loss and explicitly set the authorized parent price; preserve take profit.
gate-cli cex spot order amend --pair BTC_USDT --id 123 --account unified \
  --price 60000 --stop-loss-json '{}'
```

Protection objects use decimal strings: `trigger_price` is required for a nonempty
object; `order_price` is optional and follows server execution rules. Creation
requires a limit order: buy take-profit triggers are above its limit price and
stop-loss triggers below; sell orders reverse those comparisons. On amend, the
server validates against the order's authoritative side and price.
The deployed API requires an amount or price on amendment, including protection
changes. Do not silently resend an old parent value; obtain authority for the
explicit value and verify the entire order afterward. Parent and protective
orders must each satisfy minimum-notional rules.

| Amendment input | Effect |
| --- | --- |
| Flag omitted or JSON `null` | Keep the existing protection unchanged. |
| Literal `{}` | Cancel that protection. |
| Valid nonempty object | Set that protection. |
| Null/empty members or unknown fields | Reject locally; send no request. |

These are attached protection fields, without a claim of native OCO/OTO behavior.

## Options closes and account settings

```sh
# Full close: zero size plus explicit close. IOC with price 0 requests a market order.
gate-cli cex options order create --contract BTC_USDT-20261225-60000-C \
  --size 0 --close --price 0 --tif ioc

# Partial long-position reduction: negative contracts, with a server-side reduce-only guard.
gate-cli cex options order create --contract BTC_USDT-20261225-60000-C \
  --size -1 --reduce-only --price 0 --tif ioc

# Preview debt repayment before selecting the liability and funding currencies.
gate-cli cex unified loan quick-estimate
gate-cli cex unified loan quick-repay --debt-currencies USDT --available-currencies BTC,USDC

# Explicit false is required to disable the mode; omission does not imply consent.
gate-cli cex unified config delta-neutral-set --enabled=false
gate-cli cex unified config leverage-set-all --leverage 3
```

`options position close` continues to query close history. It does not submit a
close order. Full closes require `--close --size 0`; partial closes use signed
nonzero size and `--reduce-only` so they cannot open reverse exposure.

Quick repayment has currency selectors but no amount/spending-cap flag. Read the
estimate and confirm the eligible funding/debt set before submitting; do not
claim that currency selection enforces an amount cap. Verify actual debt and
funding afterward. Delta-neutral changes remain subject to account eligibility;
use `delta-neutral-get` before and after an explicitly authorized change.

`leverage-set-all` returns a nonzero exit code if the API reports any failed
currency, including HTTP 200 responses. Successful currencies are not rolled back;
inspect current settings before retrying. Per-currency `leverage-set` remains
available. Server leverage limits can cap the requested multiplier.
An empty failure list means no currency failure was reported, not that any
setting changed. Outstanding loans can prevent a change. Read the actual
per-currency settings before and after the operation.

## CFD commissions and order logs

```sh
# Authenticated cost query; this is not an affiliate commission report.
gate-cli cex tradfi market commissions --symbols XAUUSD

# Use a known valid log ID from the order-placement workflow, not the order ID.
gate-cli cex tradfi order log --log-id 123456789
```

Both commands require private CFD access. Log IDs remain exact positive int64
values. A returned log is not by itself proof of a fill; reconcile its order,
state and resulting position. Obtain the actual log ID rather than substituting
an unrelated order ID or guessing an ID from nearby records.

## Results and recovery

Keep stdout, stderr and process exit status separate. Preserve decimal strings
and exact IDs, use a finite process deadline and a bounded output budget. No write
is automatically retried by these commands.

Chase services may return flat results or `{code,message,data}` with HTTP 200.
The CLI unwraps valid success data and fails on nonzero business codes or missing
result fields, preserving the business error and trace. Require a nonempty create
ID, an identified `order` for get/stop, or an `orders` array for list/stop-all.
Empty list/stop-all results are `{"orders":[]}`; `{}` is not success evidence.

A successful submission is not a fill, completed repayment or applied setting.
Re-read the relevant business state. On timeout, lost output, malformed response
or partial failure, reconcile the original operation before any further write;
never repeat it just to recover a response. If reconciliation is unavailable,
report the unknown outcome.

## Optional order parameters

`options order create --mmp` sets the order-level MMP marker. It does not configure
MMP thresholds or combine MMP configuration and order creation into one operation.
The flag can accompany the existing `--close` or `--reduce-only` fields; the service
validates eligibility and the requested combination. Omission or `--mmp=false`
keeps the old request without an `mmp` field.

`tradfi order create --leverage 100` forwards the SDK integer field. Permitted
values depend on the symbol and are validated by the service. Omission or value
`0` omits the field, in the request; it does not request zero leverage.
The CLI does not choose a multiplier, change the account or retry a rejected order.

## Triggered and trailing orders

`futures price-trigger create --tif` accepts `gtc` or `ioc`. Omission leaves the field unset for the service default; market execution (`--price 0`) requires `ioc`. This differs from the spot trigger flag `--time-in-force`, which defaults to `ioc`.

Trailing `--price-offset` accepts an absolute price distance or a percentage with `%`: use `'2%'` for a 2% callback, not `0.02`. `--amount` is signed: positive buys, negative sells; the sign alone does not make the order reduce-only.
Trailing creation also requires explicit `--price-type`: `1` (latest), `2` (index), or `3` (mark). Missing or invalid values fail before HTTP; no reference price is chosen automatically.

## Delivery trigger direction

`cex delivery price-trigger create` now requires `--trigger-rule`: `1` selects
`>=`, and `2` selects `<=`. Existing scripts must explicitly supply their intended
rule; omission fails before HTTP instead of submitting an unspecified direction. The CLI
does not choose a direction from the trigger price or retry with another rule.
Representable Int32 values are passed to the API; the service validates the rule,
price relationship and trading eligibility. Other fields and output modes retain their existing behavior.

## Order references and batch results

Ordinary spot `order buy` / `sell` and futures `order long` / `short` / `add` / `remove` accept `--text`. The value must start with `t-`, followed by at most 28 ASCII letters, digits, `_`, `-` or `.`. This reference helps reconcile an order; it is not an idempotency guarantee.

```sh
# Query a pending spot order by its reference; completed orders need a numeric ID.
gate-cli cex spot order get --pair BTC_USDT --id t-reference --format json
# Query a futures order by reference, subject to service history limits.
gate-cli cex futures order get-by-text --text t-reference --format json
```

Cancelled, unfilled futures orders have a 60-second text lookup window. A failed lookup does not establish that an order was never submitted.

Spot and futures batch create/amend/cancel commands keep per-item results on stdout. Any reported item failure returns exit 1 with an error summary on stderr. Spot batches require nonempty arrays and reject a missing or extra result count with `INVALID_RESPONSE`. Inspect successful IDs and failed items separately; do not resubmit a whole partially successful batch. An output truncation placeholder is not the complete result.

## Earn filters and auto-investment

| Command under `cex earn` | Parameters |
| --- | --- |
| `fixed products` | Optional `--status` (int32) and `--show-status` (string); explicitly supplied zero/empty values are sent. |
| `fixed lends`, `fixed history` | Optional `--sub-business` (int32) and `--business-filter` (one JSON array-of-objects string, at most 65,536 bytes); `0` and `[]` are sent when explicit. |
| `auto-invest min-amount` | `--json` must include integer `asset_type`: `1` cryptocurrency, `2` TradFi stocks. |

```sh
# A minimum-investment query; supply the intended assets and product type.
gate-cli cex earn auto-invest min-amount --json '{"money":"USDT","asset_type":1,"items":[{"asset":"BTC","ratio":"70"},{"asset":"ETH","ratio":"30"}]}' --format json
```

## Market queries and response fields

- `tradfi market kline --type` defaults to `1m`; accepted values are `1m`, `15m`, `1h`, `4h`, `1d`, `7d`, `30d`. Aliases `M1`, `M15`, `H1`, `H4`, `D1` map to those values. `M5` and `M30` are unsupported.
- `spot market insurance` requires positive Unix-second `--from` and `--to`, with `from <= to`.
- Futures and delivery `market insurance` return exact string `b` balances, including responses that originally contain numeric balances.
- `alpha market currencies` and `alpha market tickers` use Alpha currency IDs, not ordinary tickers or contract addresses. Read the returned `currency` from `alpha market currencies` without `--currency`, paging as needed, then use that exact ID.
- `options account get` returns `position_notional_limit` as a string; preserve it without floating-point conversion.
- `futures trail list` returns `code`, `message`, `data.orders` and `timestamp`. Read orders from `data.orders`. Trail order IDs and related time fields are strings; list table output preserves the complete ID.
- `futures price-trigger get` treats HTTP 205 as a business error and preserves its `label` and `message`.

## Local diagnostics

`version --details --format json` reports version, revision, Go version, platform and compiled agent command availability. An unavailable revision is `unknown`.

`config check --format json` checks local file type, read access and permissions. It does not check YAML syntax, API permissions or Windows ACLs. Pass returns exit 0; warning/failure returns exit 1. It does not change the file or display credentials.

`config init --hide-input` hides all interactive fields on a terminal, including the profile. Piped input uses the ordinary input behavior.
