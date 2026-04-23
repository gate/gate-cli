# gate-cli

## 品牌 & 模块
- 品牌名称用 "Gate" 或 "gate.com"，**禁用 "gate.io"**（API URL 域名 api.gateio.ws 除外）
- Go module: `github.com/gate/gate-cli`
- Gate Go SDK: `github.com/gate/gateapi-go/v7`（非旧版 `github.com/gateio/gateapi-go/v6`）

## 常用命令
- 构建: `go build -o gate-cli .`
- 单元测试: `go test ./...`（**仅** CI 发版 / 全仓 merge 前；日常不要用全量 `./...`）
- **只跑改动的包**: `./scripts/test-changed-go.sh`（或 `./scripts/test-changed-go.sh origin/main`）
- Intel / MCP / migrate: `./scripts/test-intel-scope.sh -count=1` 与 `./scripts/test-intel-scope.sh vet`（见 `scripts/test-intel-scope.sh`）
- Integration 测试: `go test -tags integration ./internal/integration/... -v`
- 本地 smoke test（公共 API 无需 key）: `./gate-cli cex spot market ticker --pair BTC_USDT`

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

## MCP / Intel（`info` / `news` 已发布；`tool` 仍为规格命名）
- **用户 CLI**：`gate-cli info` / `gate-cli news`（38 个 MCP 叶子；`info list` / `news list`，`-h` 查 flag）。用户文档：根目录 [`README.md`](README.md)、[`docs/quickstart.md`](docs/quickstart.md)。
- **规格中的统一主路径（规划）**：`gate-cli tool`（`list` / `call` / `describe`）；与交易 API（`gateapi-go`）**鉴权隔离**：勿用 `GATE_API_KEY` 充当 MCP Bearer。规格与待办：[`specs/README.md`](specs/README.md)（**含「已对读」说明**）、[`specs/open-items-and-dependencies.md`](specs/open-items-and-dependencies.md)；**技术评审只读** [`specs/clidocs/gate-cli-intel-mcp-technical-solution-feishu.md`](specs/clidocs/gate-cli-intel-mcp-technical-solution-feishu.md)；评审摘要：[`docs/plans/2026-04-10-gate-cli-tool-mcp-review.md`](docs/plans/2026-04-10-gate-cli-tool-mcp-review.md)。
- **Cursor**：默认决策见 [`.cursor/rules/`](.cursor/rules/) — **`gate-cli-cli-layer-conventions.mdc`**（`GetPrinter`/`PrintError`、失败 **stderr** `{"error":...}`，与交易子命令一致；MCP 仅替换 `mcpclient`）、**`gate-cli-intel-mcp-specs.mdc`**、**`mcp-intel-curl-endpoints.mdc`**；Wire **`specs/cli/mcp-wire-appendix.md` v0.4**。

## 架构约定
- 共享 CLI helper（GetPrinter/GetClient）放在 `internal/cmdutil/`，**不要**放 `cmd/root.go`（会循环 import）
- 错误输出走 stderr，正常数据走 stdout（agent 管道安全）
- `--settle` 默认 `usdt`，在 `cmd/cex/futures/market.go` 的 `getSettle()` 中处理

## tablewriter v1.1.3 注意
- `NewWriter(w)` 返回单值（无 error）
- Header/Append 接受 `...interface{}`
- 需关闭 AutoFormat 防止 header 被转大写：
  `t.Configure(func(cfg *tablewriter.Config) { cfg.Header.Formatting.AutoFormat = tw.Off })`
