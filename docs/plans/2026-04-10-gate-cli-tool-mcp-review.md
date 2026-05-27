# gate-cli `tool`（MCP 薄客户端）— 评审稿

> **用途**：宣讲与立项评审的仓库内锚点；细节以 [`specs/`](../../specs/) 与 Cursor 计划文档为准。  
> **对齐**：组织内 `mcp-server` 仓库 `cli/leadership-review-brief.md`（聚合 `tool list/call/describe`、Transport 解耦、分阶段交付）。

## 一页结论

- **做什么**：在 `gate-cli` 单二进制中增加 **`gate-cli tool`**，通过 **MCP Streamable HTTP** 调用已部署的 news / info / docs MCP 服务；**不**复制 OpenSearch、聚合业务逻辑。
- **主命令**：`tool list`、`tool call`、`tool describe`（`intel` 仅作可选别名，见 [`specs/README.md`](../../specs/README.md)）。
- **P0 交付**：可配置 URL + Bearer、list/call/describe 闭环、JSON 友好输出、**耗时 + trace 头**（与网关约定后固化字段名）。
- **非目标**：交易策略、与 `GATE_API_KEY` 混用、import 上游 `internal`。

## 架构（口述）

CLI（Cobra）→ **Transport 接口**（P0 仅 MCP HTTP）→ 三套 MCP 服务。未来若增加 **Plain HTTP Invoke**，只新增 Transport 实现，不改子命令语义。

## 分阶段（与评审简报一致）

| 阶段 | 内容 |
|------|------|
| P0 | list / call / describe、env、观测、单测 + 联调 |
| P1 | 429 退避、per-backend 鉴权、错误体与网关对齐、脱敏加强 |
| P2 | 工具列表缓存、doctor/refresh、completion（视服务端是否提供 ETag 等） |

## 会上建议拍板

1. P0 是否 **三 backend 全打通** 或 **先单 backend 竖切**。  
2. Trace / request 对齐：**`x-gate-trace-id`** 还是网关统一字段。  
3. **`tool.yaml` profile** P0 做还是 P1。  
4. 对外叙事：**信息/研究只读** 与交易 CLI 边界（帮助文案与对外文档）。

## 文档索引（本仓库）

| 路径 | 说明 |
|------|------|
| [`specs/README.md`](../../specs/README.md) | 规格总览与命名约定 |
| [`specs/intel-cli-command-map.md`](../../specs/intel-cli-command-map.md) | 子命令与参数映射 |
| [`specs/intel-tool-inventory.md`](../../specs/intel-tool-inventory.md) | 工具名全表（参考真源：`tools/list`） |
| [`specs/intel-config-and-security.md`](../../specs/intel-config-and-security.md) | 环境变量与安全 |
| [`specs/open-items-and-dependencies.md`](../../specs/open-items-and-dependencies.md) | **待对齐项与依赖**（评审后更新） |

## Demo 提示（5 分钟）

```bash
gate-cli tool list --backend news --format json
gate-cli tool describe --backend news --name news_feed_search_news
gate-cli tool call --backend news --name news_feed_get_social_sentiment \
  --args-json '{"coin":"BTC","time_range":"24h"}'
```

（需预先配置 `GATE_INTEL_NEWS_MCP_URL` 与 Bearer；无环境时可改用 mock 集成测试演示。）
