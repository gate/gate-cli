# Integration Test Plan

Integration tests make live API calls against the Gate testnet to verify that core trading workflows behave correctly end-to-end.

## Environment Setup

| Item | Details |
|------|---------|
| Config file | `testdata/integration.yaml` (gitignored, never committed) |
| Template | `testdata/integration.yaml.example` |
| Testnet base URL | `https://api-testnet.gateapi.io` |
| Build tag | `integration` (excluded from `go test ./...`) |

If the config file is missing or `api_key` / `api_secret` are empty, the test calls `t.Fatal` — there is no skip path.

**Run command:**

```bash
go test -tags integration ./internal/integration/... -v
```

---

## Spot Tests (`spot_test.go`)

### TestSpotAccountList

Verifies that the spot account query returns at least one currency balance.

| Step | Action | Assertion |
|------|--------|-----------|
| 1 | `ListSpotAccounts` | HTTP 200, non-empty account list |

---

### TestSpotOrderList

Verifies the open spot order list query.

| Step | Action | Assertion |
|------|--------|-----------|
| 1 | `ListOrders(BTC_USDT, open)` | HTTP 200, response well-formed (empty list is acceptable) |

---

### TestSpotMarketTicker

Verifies the public ticker endpoint (no authentication required).

| Step | Action | Assertion |
|------|--------|-----------|
| 1 | `ListTickers` | HTTP 200, non-empty ticker list |

---

### TestSpotOrderCreateAndCancel

Verifies the full spot limit-order lifecycle: place → cancel.

**Precondition:** account holds ≥5 USDT or ≥0.001 BTC; otherwise the test is skipped.
**Amount control:** orders are placed far outside the market and will never fill.

| Step | Action | Parameters | Assertion |
|------|--------|-----------|-----------|
| 1 | Query spot account balances | — | Determine available currency |
| 2 | Place limit order | USDT path: buy 0.001 BTC @ $5,000<br>BTC path: sell 0.001 BTC @ $999,000 | HTTP 201, status=open |
| 3 | Cancel the order | Order ID from step 2 | HTTP 200, status=cancelled, finish_as=cancelled |

**Cleanup:** `t.Cleanup` cancels the order as a safety net if the test fails before the cancel step.

---

## Futures Tests (`futures_test.go`)

Default settle currency: `usdt`. Default test contract: `BTC_USDT`.

### TestFuturesAccountGet

Verifies the futures account query including the dual-mode flag.

| Step | Action | Assertion |
|------|--------|-----------|
| 1 | `ListFuturesAccounts(usdt)` | HTTP 200, currency equals "usdt" (case-insensitive) |

Logs total balance, available balance, and dual_mode flag for manual inspection.

---

### TestFuturesPositionList

Verifies the futures position list query.

| Step | Action | Assertion |
|------|--------|-----------|
| 1 | `ListPositions(usdt)` | HTTP 200, well-formed response |

---

### TestFuturesOrderList

Verifies the futures open order list query.

| Step | Action | Assertion |
|------|--------|-----------|
| 1 | `ListFuturesOrders(usdt, open)` | HTTP 200, well-formed response |

---

### TestFuturesOrderCreateAndCancel

Verifies the futures limit-order lifecycle: place → cancel.

**Amount control:** limit price $1,000 (far below any realistic BTC price), 5 contracts ≈ 5 USDT notional, will never fill.

| Step | Action | Parameters | Assertion |
|------|--------|-----------|-----------|
| 1 | Place limit long | BTC_USDT, size=5, price=1000, tif=gtc | HTTP 201, status=open |
| 2 | Cancel the order | Order ID from step 1 | HTTP 200, status=finished, finish_as=cancelled |

**Cleanup:** `t.Cleanup` cancels the order as a safety net.

---

### TestFuturesGetPosition

Verifies that `client.GetFuturesPosition` returns correct position data in both single and dual mode.

**Precondition:** account has at least one non-zero position; otherwise skipped.

| Step | Action | Assertion |
|------|--------|-----------|
| 1 | `ListPositions` to find a contract with open positions | — |
| 2 | `client.GetFuturesPosition(settle, contract)` | HTTP 200, non-empty slice |

- Single mode: returns 1 entry, `mode=single`
- Dual mode: returns 2 entries, `mode=dual_long` + `mode=dual_short`

---

### TestFuturesLeverageUpdate

Verifies that leverage modification routes correctly to the right endpoint in single/dual mode and cross/isolated margin mode.

**Precondition:** account has at least one non-zero position (`UpdatePositionLeverage` returns an empty array when no position exists); otherwise skipped.

`client.UpdateFuturesPositionLeverage` detects the account mode automatically:
- Dual mode → `UpdateDualModePositionLeverage`
- Single mode → `UpdatePositionLeverage`

| Step | Action | Assertion |
|------|--------|-----------|
| 1 | Record current `cross_leverage_limit` | — |
| 2 | Update to a different value (10 ↔ 20) | HTTP 200, returned value matches target |
| 3 | Restore original value | HTTP 200, returned value matches original |

---

### TestFuturesPositionLifecycle

End-to-end test of the full position lifecycle: open → add → reduce → close.

**Amount control:** all orders use market execution. Peak exposure is 10 contracts ≈ 10 USDT notional (1 contract = 1 USD).

| Step | Action | Parameters | Assertion |
|------|--------|-----------|-----------|
| 1 | **Open** — market long | size=+5, price=0, tif=ioc | HTTP 201, long position size > 0 |
| 2 | Verify position | `client.GetFuturesPosition` | Long position present; log mode/leverage/entry |
| 3 | **Add** — market long | size=+5, price=0, tif=ioc | HTTP 201 |
| 4 | **Reduce** — market sell, reduce-only | size=-5, price=0, tif=ioc | HTTP 201 |
| 5 | **Close** — market sell remaining, reduce-only | size=-(opened−reduced), price=0, tif=ioc | HTTP 201 |
| 6 | Verify final position | `client.GetFuturesPosition` | Long size back to pre-test level |

**Cleanup:** `t.Cleanup` tracks the `opened` counter; if the test fails mid-way it issues a reduce-only sell for any remaining contracts without touching pre-existing positions.

---

## Dual-Position Mode Compatibility

When an account has dual-position mode enabled (`InDualMode=true`), several Gate API endpoints behave differently from single mode:

| Operation | Single Mode | Dual Mode |
|-----------|-------------|-----------|
| Get position | `GetPosition` → `Position` | `GetDualModePosition` → `[]Position` |
| Update leverage | `UpdatePositionLeverage` → `Position` | `UpdateDualModePositionLeverage` → `[]Position` |
| Full close long | `Close=true, Size=0` | `AutoSize="close_long", Size=0` |
| Full close short | `Close=true, Size=0` | `AutoSize="close_short", Size=0` |

The CLI detects the mode via `client.IsDualMode(settle)` (lazily cached after the first call) and routes to the correct endpoint transparently. Users do not need to be aware of which mode the account is in.
