# 定投最小金额查询的 asset_type

`cex earn auto-invest min-amount` / `POST /earn/autoinvest/min_invest_amount` 的 JSON 请求体要求显式整数 `asset_type`：1 为加密货币，2 为 TradFi 股票。缺失、null、0、其他枚举、错误类型及 int32 溢出均本地失败，不默认选择资产类别。`money`、`items[].asset`、`items[].ratio` 继续沿用 API 契约；CLI 不转换金额或比例的字符串精度。

```sh
# 查询加密货币定投的最小金额；业务响应和最终访问权限由 API 判断。
gate-cli cex earn auto-invest min-amount --format json --json '{"money":"USDT","asset_type":1,"items":[{"asset":"BTC","ratio":"70"},{"asset":"ETH","ratio":"30"}]}'
```

在项目根目录运行正式测试；无需真实凭据、Docker 或业务服务：

```sh
# 新增单元和既有 Earn/CEX/根入口回归。
GOWORK=off GOMAXPROCS=2 GOFLAGS=-p=1 go test ./cmd/cex/earn ./cmd ./cmd/cex -count=1 -timeout=2m
# 构建真正 CLI 子进程，执行 CLI -> SDK -> transport -> HTTP -> 输出/退出码。
GOWORK=off GOMAXPROCS=2 GOFLAGS=-p=1 go test -tags agentic_e2e ./tests/agentic -run '^TestAutoInvestAssetTypeCLI' -count=1 -timeout=3m
# 目标解析测试的 race 检查。
GOWORK=off GOMAXPROCS=2 GOFLAGS=-p=1 go test -race ./cmd/cex/earn -run '^TestAutoInvestAssetTypeParsing$' -count=1 -timeout=2m
```

| 正式用例 | 前置与步骤 | 预期/有效断言 |
| --- | --- | --- |
| TestAutoInvestAssetTypeParsing | 实际 SDK DTO 解析 1/2、USDT、两个项目及比例；再 JSON 序列化 | asset_type 原值、money、items 数量和比例保留，最终 JSON 含 asset_type |
| TestAutoInvestAssetTypeParsing/reject | 缺失、null、坏 JSON、空对象、数组根、0/3/-1、int32 溢出、字符串/布尔/小数 | 返回错误，不接受隐式资产类别 |
| TestAutoInvestAssetTypeCLI | 临时 HOME 和合成 OAuth/HMAC；两种 asset_type 启动真实 CLI；loopback peer 返回高精度金额 | 恰好一次 POST 到正确路径；body 的 asset_type/money/items 比例和鉴权头保留；金额完整输出、退出成功 |
| TestAutoInvestAssetTypeCLIRejection | 同一子进程入口提交缺失/null/0/3/-1/溢出/字符串/布尔/小数 | 非零退出、成功 stdout 为空、stderr 含 asset_type、HTTP 次数为零 |
| TestAutoInvestAssetTypeCLIRejection/HTTP | 有效请求，peer 依次返回 400/401/403/429/500 | 恰好一次请求、非零退出、成功 stdout 为空，无自动重放 |

仅外部 API peer、凭据和业务结果为合成数据；不 mock SDK 方法、DTO 编码、transport、打印或退出码。HMAC 用例核对头存在，不证明服务端验签；OAuth 用例不证明真实网关 scope 或授权服务器可用。asset_type=2 的 peer 用例用于验证参数传递，合成币种不作为 TradFi 业务合法性的证据。



## Fixed product 状态筛选

`gate-cli cex earn fixed products --status 2 --show-status 2 --format json` 使用 `GET /earn/fixed-term/product` 的可选整数status和字符串show_status；不指定时不发送，显式0/空字符串仍发送。CLI help展示两项flag。运行同一正式入口：

```sh
# 两个功能和受影响历史筛选，真实CLI子进程/SDK→本地HTTP业务peer。
GOWORK=off GOMAXPROCS=2 GOFLAGS=-p=2 go test -tags agentic_e2e ./tests/agentic -run '^Test(AutoInvestAssetTypeCLI|FixedProductStatusCLI|FixedFiltersCLI)' -count=1 -timeout=4m
```

fixed_product_status_test.go覆盖：省略、单独/组合、0/空串、int32边界、字符串编码、旧参数保留、help、非法数值零HTTP、HTTP错误非零退出。fixed products原为公开接口，两种已配置凭据模式下均断言不发送OAuth/KEY/SIGN；auto-invest沿其原catalog契约，不借本次修改认证规则。金额返回为字符串不丢精度。这里只mock外部HTTP响应和凭据，不证明真实上游业务筛选语义。
