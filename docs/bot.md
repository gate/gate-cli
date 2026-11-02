# AI Hub portfolio strategies

Use explicit `--format json` in scripts. Configure the selected APIv4 profile and authenticate with `cex auth login` or an API Key using the existing authentication flow. HTTP success alone does not prove a successful strategy operation: business failures exit nonzero and are printed to stderr.

## Create and refresh

The examples describe request structure. Replace amounts/assets with reviewed values and use isolated test assets for verification.

```sh
# Smart position: two to ten asset weights; period and threshold are mutually exclusive.
gate-cli --format json cex bot smart-position create --json '{"strategy_type":"smart_position","create_params":{"begin_money_amount":"100","begin_money_currency":"USDT","asset_info":[{"asset":"BTC","rate":"0.5"},{"asset":"ETH","rate":"0.5"}],"period":"1h","close_flag":0}}'

# Futures combo: two to ten markets, buy/sell sides, weights and leverage.
gate-cli --format json cex bot futures-combo create --json '{"strategy_type":"futures_combo","create_params":{"begin_money_amount":"100","begin_money_currency":"USDT","asset_info":[{"market":"BTC_USDT","side":"buy","rate":"0.5"},{"market":"ETH_USDT","side":"sell","rate":"0.5"}],"period":"1h","leverage":2}}'

# Reuse the actual returned identity; exclusion and refresh context are distinct fields.
gate-cli --format json cex bot recommend --scene refresh --exclude-recommendation-id '<returned-recommendation-id>'
```

Recommendations support spot_grid, margin_grid, infinite_grid, futures_grid, spot_martingale and smart_position. Contract martingale and futures combo are manual-only. The API owns asset availability, weights, limits and financial validation.

For contract martingale, provide `create_params.stop_loss_price` and `stop_loss_price_type` together, or omit both. The price type is `last`, `index` or `mark`; `invest_amount` is the total budget; `max_orders` counts additional entries and excludes the first entry.

## Stop and verify

```sh
# Limit stopping requires a string price and positive integer expiration in seconds.
gate-cli --format json cex bot stop --strategy-id '<real-id>' --strategy-type smart_position --stop-mode limit --stop-options '{"price":"100.000000000000000001","expiration":60,"slip":"0.01"}'

# A combo uses its product-defined whole-portfolio exit, without custom stop options.
gate-cli --format json cex bot stop --strategy-id '<real-id>' --strategy-type futures_combo

# Query the original strategy after any partial or uncertain write result.
gate-cli --format json cex bot detail --strategy-id '<real-id>' --strategy-type futures_combo
```

Smart position requires `market`, `limit` or `no_close` stopping. Only `limit` accepts `stop_options`. `no_close` stops the strategy without closing its assets. Other strategy types reject custom stop modes/options.

Inspect the actual ID/status, `operation_status`, `recovery_required`, `positions_status` and `stop_progress`. `starting` is not proof of running; `stopping`/`finished` does not prove funds returned or account release. Missing progress means unverified; explicit false means that step is not complete. Never automatically replay a create/stop after an unknown result. The CLI sends each write once and does not provide server-side idempotency.

JSON amounts, rates and prices remain strings. Unknown fields, null/array bodies and multiple JSON values are rejected instead of silently dropping intent.

## Commands

All paths below start with `gate-cli cex bot`.

| Command | Function |
| --- | --- |
| `recommend` | Find strategy recommendations; optionally filter market, type, direction and investment budget. |
| `grid spot`, `grid margin`, `grid infinite`, `grid futures` | Create the corresponding grid strategy with `--json`. |
| `martingale spot`, `martingale contract` | Create a spot or contract martingale strategy with `--json`. |
| `smart-position create` | Create an asset-weighted portfolio with `--json`. |
| `futures-combo create` | Create a multi-market futures portfolio with `--json`. |
| `running` | List running strategies; optional `--strategy-type`, `--market`, `--page`, `--page-size`. |
| `detail` | Query `--strategy-id` and `--strategy-type`. |
| `stop` | Stop the specified strategy; smart-position stop modes are described above. |

## Authentication and permissions

All Bot commands require authentication. OAuth queries (`recommend`, `running`, `detail`) require `base`. Creation and stop require `spot_trade` for `spot_grid`, `margin_grid`, `infinite_grid`, `spot_martingale` and `smart_position`; they require `futures_trade` for `futures_grid`, `contract_martingale` and `futures_combo`. The final grant and account eligibility determine access. API key requests use the key's permissions. See the [authorization guide](auth-cfd-stock.md#permissions).

The eight portfolio types are `spot_grid`, `margin_grid`, `infinite_grid`, `futures_grid`, `spot_martingale`, `contract_martingale`, `smart_position` and `futures_combo`. Running/detail preserve portfolio assets/markets, detail positions/configuration and stop progress. A returned `total` remains the upstream total, not the current page length.
