# Integration Test Plan

集成测试覆盖 Gate Testnet 真实 API 调用，验证 CLI 核心交易流程的正确性。

## 环境要求

| 项目 | 说明 |
|------|------|
| 配置文件 | `testdata/integration.yaml`（gitignored，不提交） |
| 模板 | `testdata/integration.yaml.example` |
| Testnet 域名 | `https://api-testnet.gateapi.io` |
| Build tag | `integration`（`go test ./...` 不会触发） |

配置文件缺失或 `api_key` / `api_secret` 为空时，测试强制 `t.Fatal`，不会 skip。

**运行命令：**

```bash
go test -tags integration ./internal/integration/... -v
```

---

## 现货测试（`spot_test.go`）

### TestSpotAccountList

**目的：** 验证现货账户查询接口可用，有返回数据。

| 步骤 | 操作 | 断言 |
|------|------|------|
| 1 | `ListSpotAccounts` | HTTP 200，返回至少 1 条货币余额 |

---

### TestSpotOrderList

**目的：** 验证现货挂单列表查询。

| 步骤 | 操作 | 断言 |
|------|------|------|
| 1 | `ListOrders(BTC_USDT, open)` | HTTP 200，正常返回（空列表也可） |

---

### TestSpotMarketTicker

**目的：** 验证公共行情接口（无需鉴权）。

| 步骤 | 操作 | 断言 |
|------|------|------|
| 1 | `ListTickers` | HTTP 200，返回非空 ticker 列表 |

---

### TestSpotOrderCreateAndCancel

**目的：** 验证现货限价下单与撤单完整流程。

**前置条件：** 账户有余额（≥5 USDT 或 ≥0.001 BTC），否则 skip。
**金额控制：** 挂单挂在远离市场的价格，不会成交，不产生实际资金占用。

| 步骤 | 操作 | 参数 | 断言 |
|------|------|------|------|
| 1 | 查询现货账户余额 | — | 确定可用币种 |
| 2 | 限价下单 | USDT 路径：买 0.001 BTC @ $5,000<br>BTC 路径：卖 0.001 BTC @ $999,000 | HTTP 201，status=open |
| 3 | 撤销订单 | 步骤 2 的订单 ID | HTTP 200，status=cancelled，finish_as=cancelled |

**清理机制：** `t.Cleanup` 兜底撤单，防止测试中途失败时挂单残留。

---

## 合约测试（`futures_test.go`）

默认结算货币：`usdt`，默认测试合约：`BTC_USDT`。

### TestFuturesAccountGet

**目的：** 验证合约账户基础信息查询，包括双仓模式标志。

| 步骤 | 操作 | 断言 |
|------|------|------|
| 1 | `ListFuturesAccounts(usdt)` | HTTP 200，Currency 大小写不敏感等于 "usdt" |

日志输出 total、available、dual_mode 供人工核查。

---

### TestFuturesPositionList

**目的：** 验证合约持仓列表查询。

| 步骤 | 操作 | 断言 |
|------|------|------|
| 1 | `ListPositions(usdt)` | HTTP 200，正常返回 |

---

### TestFuturesOrderList

**目的：** 验证合约挂单列表查询。

| 步骤 | 操作 | 断言 |
|------|------|------|
| 1 | `ListFuturesOrders(usdt, open)` | HTTP 200，正常返回 |

---

### TestFuturesOrderCreateAndCancel

**目的：** 验证合约限价下单与撤单流程。

**金额控制：** 限价 $1,000（远低于 BTC 市价），5 合约 ≈ 5 USDT 名义价值，不会成交。

| 步骤 | 操作 | 参数 | 断言 |
|------|------|------|------|
| 1 | 限价开多 | BTC_USDT，size=5，price=1000，tif=gtc | HTTP 201，status=open |
| 2 | 撤销订单 | 步骤 1 的订单 ID | HTTP 200，status=finished，finish_as=cancelled |

**清理机制：** `t.Cleanup` 兜底撤单。

---

### TestFuturesGetPosition

**目的：** 验证 `client.GetFuturesPosition` 在单仓/双仓模式下均能正确返回持仓数据。

**前置条件：** 账户有任意非零持仓，否则 skip。

| 步骤 | 操作 | 断言 |
|------|------|------|
| 1 | `ListPositions` 找到有持仓的合约 | — |
| 2 | `client.GetFuturesPosition(settle, contract)` | HTTP 200，返回非空列表 |

- 单仓模式：返回 1 条，`mode=single`
- 双仓模式：返回 2 条，`mode=dual_long` + `mode=dual_short`

---

### TestFuturesLeverageUpdate

**目的：** 验证杠杆修改在单仓/双仓、跨保证金/逐仓模式下均能正确路由并生效。

**前置条件：** 账户有任意非零持仓（`UpdatePositionLeverage` 在无持仓时返回空数组），否则 skip。

**实现说明：**
`client.UpdateFuturesPositionLeverage` 自动检测账户模式：
- 双仓模式 → `UpdateDualModePositionLeverage`
- 单仓模式 → `UpdatePositionLeverage`

| 步骤 | 操作 | 断言 |
|------|------|------|
| 1 | 记录当前 `cross_leverage_limit` | — |
| 2 | 修改为不同值（10↔20） | HTTP 200，返回值与目标一致 |
| 3 | 恢复原始值 | HTTP 200，返回值与原始一致 |

---

### TestFuturesPositionLifecycle

**目的：** 端到端验证合约仓位完整生命周期：开仓 → 加仓 → 减仓 → 平仓。

**金额控制：** 全程使用市价单，峰值持仓 10 合约 ≈ 10 USDT 名义价值（1 合约 = 1 USD）。

| 步骤 | 操作 | 参数 | 断言 |
|------|------|------|------|
| 1 | **开仓**：市价买入 | size=+5，price=0，tif=ioc | HTTP 201，持仓 size > 0 |
| 2 | 查询持仓 | `client.GetFuturesPosition` | 存在多头仓位，记录 mode/leverage/entry |
| 3 | **加仓**：市价再买入 | size=+5，price=0，tif=ioc | HTTP 201 |
| 4 | **减仓**：市价卖出（reduce_only） | size=-5，price=0，tif=ioc | HTTP 201 |
| 5 | **平仓**：市价卖出剩余（reduce_only） | size=-(已开-已减)，price=0，tif=ioc | HTTP 201 |
| 6 | 查询最终持仓 | `client.GetFuturesPosition` | 多头 size 回到开仓前水平 |

**清理机制：** `t.Cleanup` 追踪 `opened` 计数器，若测试中途失败则以 reduce_only 卖出剩余合约，不会影响账户既有仓位。

---

## 双仓模式适配说明

账户启用双仓模式（`InDualMode=true`）时，部分 Gate API 行为与单仓不同：

| 场景 | 单仓（Single Mode） | 双仓（Dual Mode） |
|------|---------------------|-------------------|
| 获取持仓 | `GetPosition` → `Position` | `GetDualModePosition` → `[]Position` |
| 修改杠杆 | `UpdatePositionLeverage` → `Position` | `UpdateDualModePositionLeverage` → `[]Position` |
| 全平多头 | `Close=true, Size=0` | `AutoSize="close_long", Size=0` |
| 全平空头 | `Close=true, Size=0` | `AutoSize="close_short", Size=0` |

CLI 通过 `client.IsDualMode(settle)` 自动检测并路由，用户无需关心底层接口差异。
