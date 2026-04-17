# gate-cli

## 品牌 & 模块
- 品牌名称用 "Gate" 或 "gate.com"，**禁用 "gate.io"**（API URL 域名 api.gateio.ws 除外）
- Go module: `github.com/gate/gate-cli`
- Gate Go SDK: `github.com/gate/gateapi-go/v7`（非旧版 `github.com/gateio/gateapi-go/v6`）

## 常用命令
- 构建: `go build -o gate-cli .`
- 单元测试: `go test ./...`（**仅** CI 发版 / 全仓 merge 前）
- **只跑改动的包**: `./scripts/test-changed-go.sh`（见 `scripts/test-changed-go.sh`）
- Intel / MCP / migrate: `./scripts/test-intel-scope.sh -count=1` 与 `./scripts/test-intel-scope.sh vet`
- Integration 测试: `go test -tags integration ./internal/integration/... -v`
- 本地 smoke test（公共 API 无需 key）: `./gate-cli spot market ticker --pair BTC_USDT`

## Integration 测试
- 配置文件: `testdata/integration.yaml`（gitignored，**不会提交**）
- 模板: `testdata/integration.yaml.example`（复制后填入 testnet key）
- Testnet 域名: `https://api-testnet.gateapi.io`
- 无 api_key / api_secret 时测试强制 Fatal（不会 skip，必须配置才能通过）
- build tag `integration` 隔离，`go test ./...` 不会触发

## SDK v7 调用方式
- **不是**方法链，而是 Opts 结构体 + `github.com/antihax/optional`
  - ✅ `c.SpotAPI.ListTickers(ctx, &gateapi.ListTickersOpts{CurrencyPair: optional.NewString(pair)})`
  - ❌ `c.SpotAPI.ListTickers(ctx).CurrencyPair(pair).Execute()`
- 错误类型：`GateAPIError`（有 label/message）或 `GenericOpenAPIError`
- 状态码从 `httpResp.StatusCode` 取，TraceID 从 `httpResp.Header.Get("x-gate-trace-id")` 取
- Futures account: `ListFuturesAccounts(ctx, settle)` 不是 `GetFuturesAccount`

## SDK v7 字段类型（常见陷阱）
- `FuturesOrder.Size` → `string`（非 int64）
- `FuturesCandlestick.T` → `float64`（时间戳）
- `FuturesTrade.Size` → `string`
- `FuturesTrade.CreateTimeMs` → `float64`

## MCP / `tool`（信息与研究能力，规划中）
- 通过 MCP Streamable HTTP 对接 news / info / docs；**规范主命令** `gate-cli tool`（`list` / `call` / `describe`）；与交易 API **鉴权隔离**（勿用 `GATE_API_KEY` 作 MCP Bearer）。
- 见 [`specs/README.md`](specs/README.md)、[`specs/open-items-and-dependencies.md`](specs/open-items-and-dependencies.md)、[`docs/plans/2026-04-10-gate-cli-tool-mcp-review.md`](docs/plans/2026-04-10-gate-cli-tool-mcp-review.md)。
- **Cursor**：见 [`.cursor/rules/`](.cursor/rules/)（**`gate-cli-cli-layer-conventions.mdc`** + Intel 规则；`specs/cli/mcp-wire-appendix.md` **v0.4**）。

## 架构约定
- 共享 CLI helper（GetPrinter/GetClient）放在 `internal/cmdutil/`，**不要**放 `cmd/root.go`（会循环 import）
- 错误输出走 stderr，正常数据走 stdout（agent 管道安全）
- `--settle` 默认 `usdt`，在 `cmd/futures/market.go` 的 `getSettle()` 中处理

## tablewriter v1.1.3 注意
- `NewWriter(w)` 返回单值（无 error）
- Header/Append 接受 `...interface{}`
- 需关闭 AutoFormat 防止 header 被转大写：
  `t.Configure(func(cfg *tablewriter.Config) { cfg.Header.Formatting.AutoFormat = tw.Off })`
