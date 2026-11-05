# 公共行情 CLI 回归

这套用例接入现有 Go 单元测试和 `tests/agentic` 的真实 CLI 子进程入口；
没有另造 SDK/HTTP 调用实现，也没有修改公共/私有分类、OAuth 或网关 scope。

## 用例与断言

| 验证功能 | 正常/兼容输入 | 异常与边界 | 必须检查的业务结果 |
| --- | --- | --- | --- |
| K线周期 | 默认 1m；全部 YAML 枚举；M1/M15/H1/H4/D1 别名 | 空字符串、M5、M30、1M、未知值 | SDK 的真实 kline_type query；错误零请求；输出时间戳和 OHLC 原值 |
| 保险基金金额 | 永续/交割；数字、负数、字符串、零、大整数和小数、指数文本；JSON/table | null/缺失/空/对象/数组/bool 余额、坏 JSON、错误顶层、读/关闭失败、HTTP 400/503 | 精确字符串余额、非零失败、无成功 stdout、保留 TraceID/解析原因、原响应关闭一次；其他路由不被适配 |
| 响应体上限 | 超过原 2MiB 的合法列表；恰好 8MiB | 8MiB+1；独立 stdout 上限 | SDK 完整解析；超限非零失败；max-output-bytes=0 输出全部，正数沿用原 truncated 占位行为 |
| 保险基金时间范围 | 有效 from/to、相等边界 | 缺任意参数、零、负数、倒序、int64 溢出 | 无效输入零 HTTP；有效请求保留 currency/business/limit/from/to；解析返回余额与时间 |
| 帮助与路由 | 合法父级/叶子帮助、别名、全局参数和参数值 | 未知/退役命令带 --help/-h；正常未知命令；--help=false 和 -- 分隔符；Cobra 动态 __complete | 未知路径非零、无误导 Usage；合法帮助成功；既有补全仍可用；预检不修改真实 flags 的值/Changed 状态 |

单元入口：

```sh
# 从 CLI 根目录运行；保持项目 Go 1.24，不改 go.mod。
GOMAXPROCS=2 go test -p 2 ./cmd/cex/spot ./cmd/cex/tradfi ./internal/client ./internal/cmdutil ./cmd ./cmd/cex ./internal/intelcmd ./internal/output -count=1
```

隔离 E2E：

```sh
# TestMain 构建真实 agentic_e2e CLI。仅 API peer 是 httptest；不读取用户配置或真实凭据。
GOMAXPROCS=2 go test -p 2 -tags agentic_e2e ./tests/agentic -run 'TestSpotInsuranceRangeCLI|TestTradfiKlineCLI|TestInsuranceBalanceCLI|TestInsuranceErrorsCLI|TestUnknownCommandHelpCLI|TestCurrenciesLargeResponseCLI' -count=1 -v
```

真实环境另用既有 `integration` 标签启用，显式指定测试输入；缺少地址将失败，不把跳过算成功：

```sh
# 仅 7 次公开 GET；全量币种列表一次，其他查询 limit=3；每次子进程使用临时 HOME 且清除宿主 GATE 凭据。
GOMAXPROCS=2 go test -p 2 -tags 'agentic_e2e integration' ./tests/agentic -run '^TestMarketPublicLive$' -count=1 -v -args -market-live-base https://api-testnet.gateapi.io
```

真实用例核对 K 线数量/时间/OHLC、保险基金字符串余额、币种列表和现货保险基金数组。合法空历史仍可成功；不保证测试账户在指定历史范围内有记录。不会授权、写入或修改资产，不证明私有 OAuth、生产镜像或上游可用性已全面验收。


## 参数及输出回归入口

- internal/client/business_error_test.go：真实 SDK Generic 错误及值/指针/包裹形式、原 status/trace、大小写、缺失/重复/非法字段、尾文档、8MiB边界和不修改原Body。
- internal/output/business_error_test.go：2xx参数label优先、既有401/403/404/429/503分类、未知写最终不可重试和重复收敛。
- internal/cmdutil/strict_help_test.go：完整topic、合法alias/无必填业务flags、零业务调用、真实flag不变、安装幂等、context及动态completion。
- alpha_business_error_test.go 和扩充的 TestUnknownCommandHelpCLI：真实CLI→Cobra→SDK→HTTP→stderr/exit，只mock HTTP peer；无效help含JSON/table与--help/-h，合法help零API；未知写只在loopback内验证，不能回放真实写。

```sh
# 正式单元与受影响历史用例。
GOMAXPROCS=2 go test -p 2 ./internal/client ./internal/output ./internal/cmdutil ./cmd ./internal/cmdhint ./internal/intelcmd -count=1
# 正式隔离E2E，沿用该目录TestMain/runCLI与自动清理。
GOMAXPROCS=2 go test -p 2 -tags agentic_e2e ./tests/agentic -run 'TestAlpha|TestBusinessErrorWriteCLI|TestUnknownCommandHelpCLI|TestAPIErrorFormats|TestLostWriteResponseFormats|TestStockUnusableWriteResponse|TestCurrenciesLargeResponseCLI|TestInsuranceErrorsCLI' -count=1
# 单独真实集成：显式选择测试环境，只执行3条公开GET；未指定环境将失败。
GOMAXPROCS=2 go test -p 2 -tags 'agentic_e2e integration' ./tests/agentic -run '^TestAlphaBusinessErrorLive$' -count=1 -v -args -market-live-base https://api-testnet.gateapi.io
```


## 全命令组未知子路径

command_groups_test.go复用真实TestMain/runCLI，以独立fixtures/command_groups.json覆盖100个可见组（含root/default completion、83个CEX子组、config/Info/News）。每组覆盖json/pretty/table下未知子命令exit1、stdout空和明确unknown command，以及裸组正确帮助exit0；API peer计数必须为0。另覆盖未注册的组内help、help=false、flag值/分隔符、根级help、alias、动态补全和真实配置叶子位置参数落盘。cmd/command_groups_test.go将真实命令树与fixture比较，未来增删组须同步审查契约。

```sh
# 统一路由、格式、真实全树清单及既有帮助边界单元。
go test ./internal/cmdutil ./cmd ./internal/intelcmd ./internal/cmdhint ./internal/output -count=1
# 新正式E2E，写行为仅发生于隔离配置目录。
go test -tags agentic_e2e ./tests/agentic -run '^TestCommandGroup' -count=1
```
