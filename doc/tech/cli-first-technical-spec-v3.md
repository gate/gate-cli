# Gate CLI-First 技术方案 v3.6

> 文档版本：v3.6
> 日期：2026-04-15
> 状态：Draft
> 来源：CEX-CLI规范-MCP至CLI映射 PRD + 技术方案讨论
> 变更记录：
> - v3.1 — 新增跨平台兼容性分析（§13）；修正 OpenClaw 支持结论
> - v3.2 — OAuth2 PKCE 授权流重构（§4.3 自动/手动双模 + stdin 秒切 + 浏览器二进制预检）；
>          OAuth2 server 侧改造清单（§4.6）；apiv4-gateway 请求数据流全景（§4.7）；
>          命令树硬切换（§3.1 所有 CEX 命令下沉至 `cex/`，顶层保留 `dex(future)/info/news`）；
>          三团队交付 Phase 拆分（§12.1-§12.5）；
>          P0/P1 审查修复：tokens.yaml 0600 权限 / DCR 多端口候选 / login 语义 / scope 统一 /
>          refresh 并发互斥锁 / 认证优先级调整 / code TTL 300s / gateway fan-out 降级 /
>          gateway 紧急回滚开关 / RoundTripper 注入模式
> - v3.3 — 补齐认证生命周期闭环：
>          §4.3.5 追加实际颁发 scope < 申请 scope 的校验与告警；
>          新增 §4.5.3 Refresh 失败 / crash recovery / refresh_token 彻底失效处理；
>          新增 §4.8 Logout / Status / Scope 管理（logout revoke 策略、status 输出 schema、
>          `login --scope` / `login --force` 组合、`insufficient_scope` 引导文案）；
>          §3.2 认证优先级追加 `call` 兜底命令的透传说明；
>          §4.4 追加多 profile tokens.yaml 示例；
>          §5.2 明确 Bearer + HMAC headers 同时存在时 Bearer 优先并丢弃 HMAC headers；
>          §5.9 规范 --debug 模式下 token redact 格式为 `pkce_at_****<last6>`
> - v3.6 — 认证准入与登录引导（§3.2 / §3.2.1）：
>          明确"运行时选择"与"登录引导"是两件事——并存时 API Key 胜出（运行时），裸环境下优先引导 OAuth2（引导）；
>          §3.2 优先级链显式标注 API Key 任意来源 > OAuth2 token > unauthenticated；
>          新增 §3.2.1 未认证状态下的命令准入决策矩阵（公开方法裸跑 / 私有方法引导登录 / 单认证静默使用 / 双认证 API Key 优先）；
>          gateway 侧 `apiv4_multi_access.lua` 追加 `auth_required=false` 白名单 pass-through 逻辑（与 §5.2 / §5.6 联动）
> - v3.5 — MCP Tool 缩写处理：
>          §3.1.2 新增 gateapi-mcp-service 服务端 + gate-local-mcp 客户端两套缩写映射表；
>          明确 CLI 命令**不跟随缩写**，推导源头从 MCP tool 名改为 SDK operationId；
>          §3.4 示例表补充"MCP tool 名（含缩写）"列，展示三方对照；
>          §7.1 `call` 兜底命令支持双入口（MCP 缩写名 + SDK operationId snake_case）
> - v3.4 — 对齐 PRD v1.0（CEX-CLI 规范）缺口：
>          §3.1.1 新增 MCP Tool → CLI 机械命名推导规则（PRD §3.3）；
>          §3.3 追加全局参数 `--timeout` / `--verbose` 并明确 `pretty` 为 `table` 别名（PRD §7）；
>          §3.4 新增 MCP Tool → CLI 映射示例表（PRD §5，覆盖 12 个 CEX Tool）；
>          §7.1 补充 `--params` 与扁平 flag 覆盖优先级（flag 覆盖 JSON 同名字段，PRD §6.1）；
>          §7.2 固定 schema 查询形式为 Cobra `--help` 为主 + `gate-cli schema <tool>` 兜底；
>          §8.1.1 新增兼容模式状态机流程图（PRD §4.5）；
>          §8.3 列出 migrate 扫描的 CEX MCP server_name 清单（PRD §4.2）；
>          §8.6 新增 preflight / doctor / migrate 职责对照表与统一退出码（PRD §4.6 + §7）；
>          §1.3 补充"不做 Shortcut 编排 / 不定义 alias 专章"非目标（PRD §1.2）；
>          §3.3 新增"所有业务命令统一支持 --params + 扁平 flag 双模"硬约束（PRD §7.1）；
>          §9.1 补充 Skill 边界不等于单个 module、允许组合多条 CLI 的原则（PRD §1.2 / §2）；
>          §8.7 新增兼容模式验收要点清单，合并 PRD §4.7 + §8 验收条目

## 1. 背景与目标

### 1.1 问题

- gate-local-mcp 有 **396 个 tools**，gateapi-mcp-service 有 **395 个 tools**
- Info MCP 有 **10 个 tools**，News MCP 有 **4 个 tools**
- 全量 MCP 加载预估消耗 **80K-150K tokens/会话**（tool description + schema 注入 LLM 上下文）
- Skill 路由收益被 MCP 全量暴露抵消

### 1.2 目标

- 本地 Agent 的默认执行面从 MCP 切换为 CLI
- Token 消耗降低 **70% 以上**
- CEX、Info、News 相关 skill 全部完成迁移
- **命令行重组**：所有现有 CEX 业务命令**全部下沉至 `cex` 模块**（硬切换，无向后兼容 alias），为后续 `dex` / `info` / `news` 等模块让出顶层命名空间
- **模块化顶层结构**：顶层只保留域模块（`cex` / `dex` / `info` / `news`）和全局工具命令（`call` / `schema` / `doctor` / `version`）

### 1.3 非目标

- 不下线平台侧 MCP 服务
- 不重新定义 Skill 编排（CEX 现有 Skill 的意图识别、编排、结果组织边界保持不变，仅切换底层执行面）
- **不做 Shortcut 编排**（PRD §1.2 对齐，Shortcut / 主 Skill 收敛属 Info/News 源 PRD 范围）
- **不定义 alias 专章**（旧 Skill 兼容层不做独立设计，硬切换 + `migrate` 批量改造 `references/cli.md` 为唯一迁移路径）
- **当前阶段不实现 `dex` 模块**（仅在命令树中占位，规划到后续 Phase）
- 不保留顶层 `spot` / `futures` / `wallet` 等旧命令作为 alias（硬切换，存量 skill 与用户脚本随 Phase 4 同步迁移到 `cex <subcmd>` 形式）

## 2. 整体架构

```
┌─────────────────────────────────────────────────────┐
│                   Skill Layer                        │
│  gate-exchange-spot / gate-info-research / ...       │
│  意图识别 → 编排 → 结果组织                            │
│  执行面: Shell 调用 gate-cli                          │
│  (Claude Code: Bash / Cursor: Terminal / OpenClaw: exec)│
└────────────────────┬────────────────────────────────┘
                     │ shell("gate-cli ...")
┌────────────────────▼────────────────────────────────┐
│                   CLI Layer (gate-cli)                │
│                                                      │
│  ┌────────────┐ ┌────────────┐ ┌────────┐ ┌────────┐│
│  │ cex 模块   │ │ dex 模块   │ │ info   │ │ news   ││
│  │ (重构下沉) │ │ (future)   │ │ (新增) │ │ (新增) ││
│  │ auth/cfg + │ │            │ │        │ │        ││
│  │ spot/      │ │            │ │        │ │        ││
│  │ futures/   │ │            │ │        │ │        ││
│  │ wallet/... │ │            │ │        │ │        ││
│  └─────┬──────┘ └─────┬──────┘ └───┬────┘ └───┬────┘│
│        │              │             │          │     │
│  ┌─────▼──────┐ ┌─────▼──────┐ ┌───▼──────────▼───┐ │
│  │ Gate SDK   │ │ DEX SDK    │ │ 轻量 MCP Client  │ │
│  │ OAuth2 +   │ │ (TBD)      │ │ (按需单 tool)    │ │
│  │ API Key    │ │            │ │ 无需认证         │ │
│  └─────┬──────┘ └─────┬──────┘ └───┬──────────────┘ │
│        │              │             │                │
│  ┌─────▼──────────────▼─────────────▼──────────────┐ │
│  │ call 兜底  /  schema 按需查询  /  doctor         │ │
│  └──────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────┘
                     │
        ┌────────────┼────────────────────┐
        ▼            ▼                    ▼
┌───────────────┐  ┌──────────────┐  ┌──────────────┐
│ apiv4-gateway │  │ api.gatemcp  │  │ api.gatemcp  │
│ (OpenResty)   │  │ /mcp/info    │  │ /mcp/news    │
│ OAuth2 Bearer │  │ 无需认证     │  │ 无需认证     │
│ → check_ep    │  │ (MCP 协议)   │  │ (MCP 协议)   │
│ → X-Gate-*    │  └──────────────┘  └──────────────┘
│ → upstream    │
│   CEX 业务    │
└───────────────┘
        │
        ▼
   api.gateio.ws/api/v4
   (CEX 业务后端)

授权流（仅 login 时一次性）:
  gate-cli ──► api.gatemcp.ai/mcp/oauth/{register,authorize,token,cli-callback}
```

## 3. 命令结构

### 3.1 完整命令树

```
gate-cli
├── cex                              # CEX 域（所有现有 Gate CEX 功能全部下沉于此）
│   ├── auth                         #   OAuth2 认证
│   │   ├── login [--manual]         #     PKCE 授权登录（默认本地回调，headless/远程自动或 --manual 走复制粘贴模式）
│   │   ├── status                   #     查看认证状态
│   │   └── logout                   #     清除 OAuth2 token
│   ├── config                       #   凭证配置
│   │   ├── init                     #     交互式配置 API Key
│   │   ├── set <key> <value>        #     设置 api-key / api-secret / base-url
│   │   └── list                     #     列出 profiles
│   ├── spot                         #   现货交易（原顶层 spot）
│   │   ├── market                   #     行情（ticker / orderbook / trades / candles）
│   │   ├── order                    #     订单（place / cancel / list / get）
│   │   ├── account                  #     账户（balances / fee）
│   │   └── ...
│   ├── futures                      #   合约交易（原顶层 futures）
│   │   ├── market
│   │   ├── order
│   │   ├── position
│   │   └── ...
│   ├── delivery                     #   交割合约（原顶层 delivery）
│   ├── options                      #   期权（原顶层 options）
│   ├── margin                       #   杠杆（原顶层 margin）
│   ├── unified                      #   统一账户（原顶层 unified）
│   ├── wallet                       #   钱包 / 充提（原顶层 wallet）
│   ├── account                      #   账户总览（原顶层 account）
│   ├── sub-account                  #   子账号管理（原顶层 sub_account）
│   ├── earn                         #   理财（原顶层 earn）
│   ├── rebate                       #   返佣（原顶层 rebate）
│   ├── flash-swap                   #   闪兑（原顶层 flash_swap）
│   ├── cross-ex                     #   跨所套利（原顶层 cross_ex）
│   ├── alpha                        #   Alpha 区（原顶层 alpha）
│   ├── tradfi                       #   TradFi（原顶层 tradfi）
│   └── p2p                          #   P2P 交易（原顶层 p2p）
│
├── dex                              # DEX 域（future，当前阶段不实现，仅注册占位命令）
│   └── (auth / wallet / swap / ... — 规划中)
│
├── info                             # Info 模块（无需认证）
│   ├── coin
│   │   ├── search --query <symbol>
│   │   └── get --query <symbol> [--scope]
│   ├── market
│   │   ├── overview
│   │   └── snapshot --symbol <sym> [--timeframe --source]
│   ├── trend
│   │   ├── kline --symbol <sym> --timeframe <tf> [--limit --source]
│   │   ├── indicators --symbol <sym> --indicators <list> [--timeframe]
│   │   └── analysis --symbol <sym>
│   ├── defi
│   │   └── overview [--category]
│   ├── macro
│   │   └── summary
│   ├── onchain
│   │   └── token --token <sym> [--chain --scope]
│   └── compliance
│       └── check --token <sym> --chain <chain>
│
├── news                             # News 模块（无需认证）
│   ├── events
│   │   ├── latest [--coin --time-range --limit]
│   │   └── detail --event-id <id>
│   └── feed
│       ├── search [--coin --sort-by --limit --lang]
│       └── sentiment [--coin]
│
├── call <tool_name> --params        # 兜底调用（跨模块 tool 直调）
├── schema <tool_name>               # 按需 schema 查询
├── doctor                           # 环境检查
├── migrate                          # 迁移辅助（辅助从 v2 旧命令迁移）
└── version
```

> 🚨 **硬切换声明**：v3 之前版本的顶层命令 `spot` / `futures` / `wallet` / `margin` / `options` / `delivery` / `earn` / `rebate` / `sub_account` / `unified` / `flash_swap` / `cross_ex` / `alpha` / `tradfi` / `p2p` / `account` **全部删除**，不再保留作为 alias。旧的顶层 `config` 命令也同步删除（迁移到 `cex config`）。升级后这些命令会返回 `unknown command`，用户必须改用 `gate-cli cex <subcmd>` 形式。
>
> 📋 **迁移辅助**：`gate-cli migrate` 命令（§8）会扫描用户 shell 历史和 skill 配置，批量输出迁移前后的命令对照表，辅助存量脚本和 skill 一次性改造。

### 3.1.1 MCP Tool → CLI 命令机械推导规则

源自 PRD §3.3，所有 CEX MCP Tool 到 CLI 三级命令的映射**完全机械、可脚本生成**，不允许逐 Tool 手写别名。

**算法（输入 `cex_<module>_<snake_rest>`）：**

```
1. 剥离前缀 "cex_"                        → "<module>_<snake_rest>"
2. 按第一个 "_" 分割                       → module, snake_rest
3. 将 snake_rest 中的 "_" 逐个替换为 "-"   → <kebab-action>
4. 拼装为: gate-cli cex <module> <kebab-action>
```

**示例对照：**

| MCP Tool 名 | 推导出的 CLI 命令 |
|------------|------------------|
| `cex_spot_get_spot_accounts` | `gate-cli cex spot get-spot-accounts` |
| `cex_spot_list_candlesticks` | `gate-cli cex spot list-candlesticks` |
| `cex_spot_create_order` | `gate-cli cex spot create-order` |
| `cex_futures_list_positions` | `gate-cli cex futures list-positions` |
| `cex_futures_update_position_leverage` | `gate-cli cex futures update-position-leverage` |
| `cex_wallet_get_total_balance` | `gate-cli cex wallet get-total-balance` |
| `cex_wallet_list_withdrawals` | `gate-cli cex wallet list-withdrawals` |
| `cex_margin_list_loans` | `gate-cli cex margin list-loans` |
| `cex_earn_list_lending_orders` | `gate-cli cex earn list-lending-orders` |
| `cex_sub_account_list_sub_accounts` | `gate-cli cex sub-account list-sub-accounts` |
| `cex_unified_get_unified_accounts` | `gate-cli cex unified get-unified-accounts` |
| `cex_flash_swap_list_flash_swap_orders` | `gate-cli cex flash-swap list-flash-swap-orders` |

> 📌 **多段 module 处理**：部分 module 名本身含 `_`（如 `sub_account` / `flash_swap` / `cross_ex`），推导时需保留 module 段的完整 snake → kebab 转换。实现上先在 `internal/registry/cex_tools.go` 中以白名单枚举所有合法 module 名（§3.1 命令树列出的 16 个），分词时优先最长匹配。

**验收**：§3.4 的 12 行示例表必须通过一个单元测试 `TestToolNameToCLI(t)` 全量回归；新增 Tool 时只需更新 `interface.json`（gateapi-mcp-service 同步），CLI 侧**不需要新增任何代码**，Cobra 命令自动按规则生成。

### 3.1.2 MCP Tool 名缩写与 CLI 反缩写策略

#### 3.1.2.1 现状：两套缩写流水线

gateapi-mcp-service 和 gate-local-mcp 在生成 MCP tool 名时对部分业务词做了激进缩写，目的是压缩 tool 名长度。**这两套缩写表并非完全一致，且是流水线叠加的**：

**① 服务端缩写**（gateapi-mcp-service / `internal/tools/*` 运行时 tool 注册）

来源：`gateapi-mcp-service/CLAUDE.md` §缩写映射

| 原词 | 缩写 | 涉及模块 |
|------|------|---------|
| `futures` | `fx` | 永续合约 |
| `sub_account` | `sa` | 子账号 |
| `dual_mode` / `dual_comp` | `dual` | 合约双模 |
| `delivery` | `dc` | 交割合约 |
| `crossex` | `crx` | 跨所交易 |

**② 客户端缩写**（gate-local-mcp `utils.ts` 的 `NAME_ABBREVIATIONS` 常量；`gateapi-mcp-service/scripts/build_mcp_tools_comparison.py` 保留同名拷贝）

| 原词 | 缩写 | 备注 |
|------|------|------|
| `futures` | `fx` | 与服务端一致 |
| `sub_account` | `sa` | 与服务端一致 |
| `dual_mode` / `dual_comp` | `dual` | 与服务端一致 |
| `flash_swap` | `fc` | **仅客户端** |
| `multi_collateral_loan` | `mcl` | **仅客户端** |
| `trad_fi` | `tradfi` | **仅客户端** |
| `cross_ex` | `crossex` | 规范化后再被服务端 `crossex → crx` **二次缩写** |

**③ 典型名称转换示例**

| Go SDK operationId | snake_case 中间形式 | 最终 MCP tool 名（含缩写） |
|-------------------|---------------------|---------------------------|
| `ListFuturesPositions` | `list_futures_positions` | `cex_fx_list_fx_positions` |
| `GetFuturesOrder` | `get_futures_order` | `cex_fx_get_fx_order` |
| `ListSubAccounts` | `list_sub_accounts` | `cex_sa_list_sas` |
| `ListDeliveryContracts` | `list_delivery_contracts` | `cex_dc_list_dc_contracts` |
| `CreateFlashSwapOrder` | `create_flash_swap_order` | `cex_fc_create_fc_order` |
| `ListMultiCollateralCurrencies` | `list_multi_collateral_currencies` | `cex_mcl_list_mcl_currencies` |
| `ListCrossexPositions` | `list_crossex_positions` | `cex_crx_list_crx_positions` |
| `ListSpotTickers` | `list_spot_tickers` | `cex_spot_list_tickers`（spot 无缩写） |

> ⚠️ **已知歧义与不一致**（本次方案需感知）：
> - `cross_ex` 在客户端被先归一化为 `crossex`，服务端又缩写为 `crx`——两段流水线串联，简单的字符串替换难以反推
> - `cex_sa_list_sas` 中的 `sas` 是"sub_account 复数 + 缩写"的产物，反向拆解并非单纯 `sa → sub_account` 的字符串替换
> - `doc/tech/mcp-tool-to-operationid-mapping.json` 历史版本曾把 `delivery` 记为 `dt`，与当前运行时的 `dc` 不一致（changelog 已记录）——映射文件不可信，以运行时代码为准

#### 3.1.2.2 CLI 的策略：完全不跟随缩写

**gate-cli 命令名使用原始完整单词**，不引入任何缩写：

| MCP 世界（含缩写） | gate-cli 世界（完整单词） |
|-------------------|--------------------------|
| `cex_fx_list_fx_positions` | `gate-cli cex futures list-futures-positions` |
| `cex_sa_list_sas` | `gate-cli cex sub-account list-sub-accounts` |
| `cex_dc_list_dc_contracts` | `gate-cli cex delivery list-delivery-contracts` |
| `cex_fc_create_fc_order` | `gate-cli cex flash-swap create-flash-swap-order` |
| `cex_mcl_list_mcl_currencies` | `gate-cli cex multi-collateral-loan list-multi-collateral-currencies` |
| `cex_crx_list_crx_positions` | `gate-cli cex cross-ex list-crossex-positions` |

**理由：**

1. **CLI 服务于人类交互和 Shell 补全**，缩写显著损害可发现性（`fx` vs `futures` 的辨识度差距大）
2. **两套缩写表存在歧义与不一致**，"从 MCP tool 名反推 CLI 命令"不稳定
3. **未来缩写规则调整**（如服务端追加 margin→mg 之类），CLI 命令名不应同步变化
4. **与 Gate Go SDK 的 PascalCase 方法名对齐**，用户阅读 SDK 文档 / IDE 跳转时心智连续

#### 3.1.2.3 实现：推导源头改为 SDK operationId

§3.1.1 的机械推导算法**不以 MCP tool 名为输入**，而是以 **SDK 原始 operationId**（`github.com/gate/gateapi-go/v7` 中 Go 方法的 PascalCase 名）为输入：

```
    SDK operationId (PascalCase)
           │
           ▼
    PascalCase → snake_case            (ListFuturesPositions → list_futures_positions)
           │
           ▼
    拆分首段作为 module                 (list_futures_positions → futures + list_futures_positions 的非首段)
           │  （module 来自 §3.1 命令树的白名单）
           ▼
    剩余段合并为 kebab-case             (list-futures-positions)
           │
           ▼
    gate-cli cex <module> <kebab>
```

> 📌 **module 从哪里取**：不是从 operationId 内部拆（有歧义），而是从 SDK package 名 / `interface.json` 中显式声明的 `module` 字段取。`interface.json` 为每个 operation 记录三元组 `(operationId, mcp_tool_name, module)`，CLI 启动时以 module 为 Cobra 父命令挂载点，以 kebab-case 的 operationId 为子命令名。

**interface.json 的增强 schema：**

```json
{
  "operations": [
    {
      "operation_id": "ListFuturesPositions",
      "module": "futures",
      "mcp_tool_name_server": "cex_fx_list_fx_positions",
      "mcp_tool_name_local":  "cex_fx_list_fx_positions",
      "sdk_method": "FuturesAPI.ListPositions",
      "http_method": "GET",
      "http_path": "/futures/{settle}/positions",
      "params": [ ... ]
    }
  ]
}
```

`mcp_tool_name_server` / `mcp_tool_name_local` 分别对应两套缩写流水线产物，供 `gate-cli call` 兜底时双向路由查找（详见 §7.1）。CLI 生成命令名时**完全不看 mcp_tool_name_\* 字段**，只看 `operation_id` + `module`。

#### 3.1.2.4 反缩写的范围边界

**CLI 不做任何"缩写 → 原词"的字符串替换**。所有从缩写到原词的映射都通过 `interface.json` 的显式三元组实现（以数据替代规则），原因：

- 字符串替换有歧义（`sas` → ?）
- 需要跨服务端/客户端两套表叠加，容易漏
- 数据驱动的查找表可随 `interface.json` 同步更新，CLI 代码零改动

唯一的例外是 `gate-cli call` 兜底命令需要兼容用户直接输入 MCP tool 名的场景，走 `interface.json` 的双向索引解析，不是字符串运算（§7.1）。

### 3.2 CEX 认证优先级

所有 `cex` 子树下的交易命令（`cex spot` / `cex futures` / `cex wallet` 等）共用以下认证优先级链：

```
--api-key/--api-secret flag               (优先级 1，显式覆盖，最高)
  > cex config profile API Key            (优先级 2，~/.gate-cli/config.yaml 用户显式配置)
  > GATE_API_KEY/GATE_API_SECRET env       (优先级 3，CI/自动化 fallback)
  > OAuth2 token (~/.gate-cli/tokens.yaml) (优先级 4，未显式配置 API Key 时的主认证)
  > unauthenticated                        (优先级 5，仅允许调用 auth_required=false 的公开方法)
```

> ⚠️ **运行时选择 vs 登录引导是两件事**：
> - **运行时选择**（本节优先级链）：当 API Key 与 OAuth2 token **并存**时，一律走 HMAC（API Key 胜出），完全忽略 tokens.yaml。
> - **登录引导**（§3.2.1 规则 #2）：当**两者都不存在**且命令需要认证时，提示用户**优先选择 OAuth2 登录**，同时附 API Key 配置作为备选路径。
>
> 两者不矛盾：登录引导是对"新用户该选哪条认证路径"的推荐（OAuth2 体验更好、无需用户管理密钥），运行时选择是对"已配置的多种凭证按哪个执行"的判定（API Key 是显式长期凭证，并存时尊重其确定性）。
>
> 💡 OAuth2-only 场景：若希望强制走 OAuth2 而忽略 API Key，用独立 profile：`--profile oauth-only` 指向一个不含 `api_key` 字段的 profile。
>
> 💡 API Key 显式覆盖：`gate-cli cex spot market ticker --api-key $KEY --api-secret $SECRET` 永远走 HMAC，无视 tokens.yaml。
>
> 🔁 **`call` 兜底命令的认证透传**：`gate-cli call <tool>` 当 tool_name 前缀为 `cex_*` 时，**完全走上述同一优先级链**，与直接调用 `gate-cli cex <subcmd>` 等价（复用 `cmdutil.GetClient()`）；前缀为 `info_*` / `news_*` 时直接走无认证的轻量 MCP Client。`call` 命令本身不引入额外认证参数，用户通过全局 `--profile` / `--api-key` / `--api-secret` 控制。

#### 3.2.1 未认证状态下的命令准入与登录引导

CLI 启动每个 `cex` 子命令时，按以下决策矩阵处理"是否放行 / 是否提示登录"：

**前置：方法分类**

`interface.json` 为每个 operation 标注 `auth_required` 字段（与 SDK / gateway 后端一致）：
- `auth_required=false`：公开行情、K线、市场深度等只读、无 uid 关联的接口（约占 cex 全量的 30-40%）
- `auth_required=true`：账户、订单、资金、持仓、转账等所有需要 uid 与签名的接口

`cmdutil.GetClient()` 在判定认证源前先读取该字段，结合下面 4 条规则决定行为。

**决策矩阵**

| # | API Key（任意来源） | OAuth2 token | 命令是否需要认证 | CLI 行为 |
|---|---------------------|--------------|------------------|----------|
| 1 | ❌ 无 | ❌ 无 | ❌ 不需要（公开方法） | ✅ **直接放行**，匿名调用 gateway，gateway 走无认证旁路 |
| 2 | ❌ 无 | ❌ 无 | ✅ 需要登录 | ⛔ **拒绝执行**，打印登录引导（OAuth2 优先 + API Key 备选），退出码 `4 (auth_required)` |
| 3a | ❌ 无 | ✅ 有 | 任意 | ✅ 直接走 OAuth2 Bearer，无任何提示 |
| 3b | ✅ 有 | ❌ 无 | 任意 | ✅ 直接走 HMAC，无任何提示 |
| 4 | ✅ 有 | ✅ 有 | 任意 | ✅ **优先 API Key**（HMAC），完全忽略 tokens.yaml，无任何提示。`--verbose` 下打印一行 `auth source: api_key_config (oauth2 token present but ignored)` |

**规则 #2 的引导文案**（exit code 4，输出到 stderr）：

```
✗ 此命令需要认证: cex spot order create

请选择登录方式（推荐 OAuth2）：

  ① OAuth2 授权登录（推荐，无需管理 API Key）
     gate-cli cex auth login

  ② 配置 API Key（适合 CI / 自动化场景）
     gate-cli cex config set api-key <YOUR_API_KEY>
     gate-cli cex config set api-secret <YOUR_API_SECRET>
     # 或一次性传入：
     gate-cli cex spot order create --api-key <KEY> --api-secret <SECRET> ...

完整文档：https://www.gate.com/docs/developers/apiv4/zh_CN/#authentication
```

**规则 #1 的实现要点**：
- 公开方法在 RoundTripper 注入阶段**不附加任何 Authorization / KEY / SIGN header**
- gateway 侧 `apiv4_multi_access.lua` 在 §5.6 / §5.2 的分发逻辑追加：若请求路径命中 `auth_required=false` 白名单且无任何认证 header，**直接 pass-through 到上游业务服务**，不强制 401（与 §5.2 当前"两种 header 都不存在 → 401"的兜底逻辑互斥，需在 gateway 侧同步白名单数据源，详见 §5.2 / §5.6）
- 公开方法白名单数据源：与 CLI 共用 `interface.json` 的 `auth_required` 字段，gateway 启动时加载为 Lua table（重启刷新）

**`source` 字段扩展**（§4.7 verbose JSON）：

| source 取值 | 含义 |
|------------|------|
| `api_key_flag` | 命中规则 #4 / #3b / 显式 flag 覆盖 |
| `api_key_config` | 命中规则 #4 / #3b 通过 config profile |
| `api_key_env` | 命中规则 #3b 通过环境变量 |
| `oauth2` | 命中规则 #3a |
| `unauthenticated` | 命中规则 #1（公开方法匿名调用） |
| `none_blocked` | 命中规则 #2，未实际发起请求，命令退出码 4 |

**验收要点**（追加到 §12.3 子任务 8）：
- 公开方法在裸环境（无 token、无 API Key）能正常返回数据
- 私有方法在裸环境立即退出 4 并打印引导文案，**不发送任何 HTTP 请求**到 gateway
- 同时存在 OAuth2 token 与 config API Key 时，抓包确认请求只有 `KEY/SIGN/Timestamp`，无 `Authorization: Bearer`
- `--verbose` 下 `source` 字段与上述矩阵一致

### 3.3 全局运行参数

| Flag | 默认值 | 说明 |
|------|-------|------|
| `--format` | `table` | 输出格式：`table` / `json`。`pretty` 作为 `table` 的**别名**保留（PRD §7.2 兼容），两者完全等价；`table` 为实现规范形态 |
| `--profile` | `default` | 配置 profile 名称 |
| `--timeout` | `30s` | 单次请求超时（Go duration 格式，如 `5s` / `2m`）。影响 HTTP client、OAuth2 token/refresh、MCP call 所有底层调用 |
| `--verbose` / `-v` | false | 打印结构化执行日志到 stderr：命令解析、认证源命中、URL、耗时、缓存命中状态。不含 HTTP body、不 dump token |
| `--debug` | false | 打印原始 HTTP 请求/响应到 stderr（含 header、body）。**token 按 §5.9 redact 规则自动遮罩**。`--debug` 暗含 `--verbose` |
| `--api-key` | "" | Gate API Key（覆盖所有其他认证来源，见 §3.2） |
| `--api-secret` | "" | Gate API Secret（覆盖所有其他认证来源，见 §3.2） |

> 📌 **`--verbose` 与 `--debug` 分层**：PRD §7 要求 `--verbose`，v3.2/v3.3 曾用 `--debug` 覆盖原始 HTTP dump。v3.4 拆为两层：`--verbose` 是轻量执行轨迹，适合 skill preflight / agent 日志上报；`--debug` 是重 dump，适合工程排障。两者都走 stderr，不污染 stdout 管道。

**`--params` + 扁平 flag 双模（所有业务命令统一约束，PRD §7.1 对齐）：**

所有 `gate-cli cex <module> <action>` / `gate-cli info <...>` / `gate-cli news <...>` / `gate-cli call <tool>` **统一支持两种参数形态**，不区分命令：

1. **扁平 flag**：如 `--pair BTC_USDT --limit 100`，适合简单字段、交互式输入
2. **`--params '<json>'`**：适合复杂嵌套结构（如下单参数、批量操作）、脚本化调用
3. **两者混用**：扁平 flag 严格覆盖 `--params` JSON 中的同名字段（覆盖规则见 §7.1）

**硬性约束**：**任何包含嵌套结构 / 数组 / 对象字段的命令都必须支持 `--params`**，不允许仅暴露扁平 flag 导致复杂参数无法表达。实现上通过 `internal/cmdutil/params.go` 的 `MergeParams()` 工具函数统一注入，每个子命令的 Cobra Run 函数只需调用一次即可拿到合并后的 `map[string]any`，交给 SDK 或 MCP client。

### 3.4 SDK operationId → MCP Tool 名 → CLI 命令 三方对照表（PRD §5 对齐 + §3.1.2 缩写处理）

下表覆盖 PRD §5 要求的 12 个 CEX MCP Tool 示例，**同时展示三套命名体系的对照关系**：

- **SDK operationId**：`gateapi-go/v7` 中 Go 方法的 PascalCase 名，CLI 推导的**唯一可信输入**
- **MCP tool 名（服务端）**：gateapi-mcp-service 注册到 MCP 协议的名字，含缩写（§3.1.2 ①）
- **CLI 命令**：gate-cli 下发给用户的形态，**完全不含缩写**

| # | SDK operationId | MCP Tool 名（含缩写）| CLI 命令（不含缩写）| 典型参数 |
|---|----------------|-------------------|------------------|---------|
| 1 | `ListSpotTickers` | `cex_spot_list_tickers` | `gate-cli cex spot list-spot-tickers` | `--pair BTC_USDT` |
| 2 | `ListSpotAccounts` | `cex_spot_list_spot_accounts` | `gate-cli cex spot list-spot-accounts` | `--currency BTC` |
| 3 | `CreateSpotOrder` | `cex_spot_create_spot_order` | `gate-cli cex spot create-spot-order` | `--params '{"currency_pair":"BTC_USDT","side":"buy","amount":"0.001","price":"60000"}'` |
| 4 | `ListSpotCandlesticks` | `cex_spot_list_spot_candlesticks` | `gate-cli cex spot list-spot-candlesticks` | `--pair BTC_USDT --interval 1h --limit 100` |
| 5 | `ListFuturesPositions` | `cex_fx_list_fx_positions` ⚠️`fx` | `gate-cli cex futures list-futures-positions` | `--settle usdt` |
| 6 | `UpdateFuturesPositionLeverage` | `cex_fx_update_fx_position_leverage` ⚠️`fx` | `gate-cli cex futures update-futures-position-leverage` | `--settle usdt --contract BTC_USDT --leverage 10` |
| 7 | `GetTotalBalance` | `cex_wallet_get_total_balance` | `gate-cli cex wallet get-total-balance` | (无) |
| 8 | `ListWithdrawals` | `cex_wallet_list_withdrawals` | `gate-cli cex wallet list-withdrawals` | `--currency USDT --limit 50` |
| 9 | `ListMarginLoans` | `cex_margin_list_loans` | `gate-cli cex margin list-margin-loans` | `--currency BTC` |
| 10 | `ListLendingOrders` | `cex_earn_list_lending_orders` | `gate-cli cex earn list-lending-orders` | `--currency USDT` |
| 11 | `ListSubAccounts` | `cex_sa_list_sas` ⚠️`sa`+复数 | `gate-cli cex sub-account list-sub-accounts` | (无) |
| 12 | `ListDeliveryContracts` | `cex_dc_list_dc_contracts` ⚠️`dc` | `gate-cli cex delivery list-delivery-contracts` | `--settle usdt` |

**关键观察：**

1. **spot / wallet / margin / earn / unified 等 module 不受缩写影响**，三方命名一致。
2. **futures / sub_account / delivery / flash_swap / multi_collateral_loan / crossex 六个 module 名被缩写**，MCP tool 名 ≠ SDK operationId ≠ CLI 命令，但 CLI 命令始终跟随 SDK 原词。
3. **第 11 行的 `cex_sa_list_sas`** 演示了反推的困难：`sas` = `sub_account` 的 `sa` 缩写 + 复数 `s`，用正则/字符串替换反推会出错，这就是 §3.1.2 强调"不做反缩写字符串运算，走 interface.json 数据查找"的原因。

**验收：**

- 每行均通过机械推导产出（无手写别名），`gate-cli call` 可同时接受 SDK operationId（snake_case 形式 `list_futures_positions`）和 MCP tool 名（含缩写 `cex_fx_list_fx_positions`）两种输入。
- 单元测试 `TestToolNameToCLI_Fixtures` 以本表为 fixture 全量回归，断言三方字段一一匹配。
- 新增 MCP Tool 时只需更新 `interface.json` 的 `operations[]`，无需改 CLI 代码（Cobra 命令由 registry 动态注册）。

## 4. OAuth2 PKCE 认证

### 4.1 OAuth2 基础设施现状

基于源码分析确认：

| 维度 | 结论 | 来源 |
|------|------|------|
| PKCE 支持 | ✅ 完整支持，仅 S256 | `oauth2/pkce/pkce.go` |
| 动态客户端注册 (DCR) | ✅ 开放，无需 initial access token | `oauth2/pkce/handler.go:93-164` |
| Metadata 发现 | ✅ `/.well-known/oauth-authorization-server` | gateapi-mcp-service 代理并缓存（**仅 gate-local-mcp 链路使用；gate-cli 直连 OAuth2 server，不经代理**） |
| Token 格式 | 不透明字符串，前缀 `pkce_at_` / `pkce_rt_` | 非 JWT |
| Refresh Token | ✅ 全量轮换（旧 token 删除） | `oauth2/pkce/token.go:134-225` |
| Token Introspection | 返回 `active, scope, client_id, sub, exp, iat` | `oauth2/pkce/model.go:62-71` |
| user_tier 等扩展字段 | ❌ 不在 introspection 中 | 需从 user-center API 获取 |

### 4.2 OAuth2 端点（生产环境）

| 端点 | URL |
|------|-----|
| Metadata 发现 | `GET https://api.gatemcp.ai/.well-known/oauth-authorization-server` |
| Resource Metadata | `GET https://api.gatemcp.ai/.well-known/oauth-protected-resource` |
| 客户端注册 (DCR) | `POST https://api.gatemcp.ai/mcp/oauth/register` |
| 授权 | `GET https://api.gatemcp.ai/mcp/oauth/authorize` |
| Token 换取/刷新 | `POST https://api.gatemcp.ai/mcp/oauth/token` |
| Token 校验（内部） | `POST {check_endpoint}` with `Gatepay-Access-Token` header |
| CLI 手动回调展示页 🆕 | `GET https://api.gatemcp.ai/mcp/oauth/cli-callback` — 仅展示 `?code=...`，供 headless/远程场景用户复制 |

> 🆕 **归属：OAuth2 项目内实现**（非 gateapi-mcp-service、非 gate-cli）。`cli-callback` 是 OAuth2 server 直接返回的最简 HTML 模板，从 querystring 取出 `code` 以等宽字体展示 + 一键复制按钮。**不读取 cookie、不下发任何 token、不访问数据库**，仅把 `code` / `state` / 可能的 `error` 原样回显。`state` 展示但不参与校验（手动模式依赖 PKCE 绑定）。
>
> 🆕 **gate-cli 与 gateapi-mcp-service 解耦**：CLI 的 OAuth2 授权流直连 OAuth2 server（DCR / authorize / token / cli-callback 四个端点），不经过 gateapi-mcp-service。§4.1 表格中 "gateapi-mcp-service 代理并缓存 metadata" 是 gate-local-mcp 场景的链路，**CLI 场景直接访问 `https://api.gatemcp.ai/.well-known/oauth-authorization-server`**，不复用该代理。

### 4.3 gate-cli cex auth login 流程

支持两种回调模式，**同一个 DCR 客户端** 在注册时申明两类 `redirect_uris`：

1. **本地回调**（自动模式使用）：`http://127.0.0.1:<port>/callback`，5 个候选端口 `18991-18995`
2. **手动复制展示页**（手动模式使用）：由 gate-cli 配置项 `oauth2.display_callback_url` 指定，默认值 `https://api.gatemcp.ai/mcp/oauth/cli-callback`

授权请求时通过 `redirect_uri` 参数选择本次走哪条路径，**任意时刻只发起一条授权链**。

**展示页 URL 配置化**（`~/.gate-cli/config.yaml`）：

```yaml
oauth2:
  # 手动模式回调展示页 URL（DCR 注册 + authorize 请求时作为 redirect_uri）
  # 指向 OAuth2 server 的静态展示页，用户授权后跳转到此页展示 code
  # 默认值：api.gatemcp.ai/mcp/oauth/cli-callback
  display_callback_url: "https://api.gatemcp.ai/mcp/oauth/cli-callback"
  # OAuth2 server base URL（用于 metadata 发现，可覆盖默认值）
  base_url: "https://api.gatemcp.ai"
```

> 💡 **为什么做成配置而非硬编码**：
> - 支持私有化部署：企业用户可将 OAuth2 server 部署到自有域名（如 `https://oauth.corp.example.com/...`），CLI 仅改配置即可对接
> - 支持本地联调：开发期可指向 `http://localhost:8080/mcp/oauth/cli-callback` 测试展示页改动
> - 运行时校验：DCR 注册 + authorize + token 三处的 `redirect_uri` 必须完全一致（精确匹配），配置值在 `login` 启动时读入内存后全流程复用同一个字符串变量,避免拼写漂移
> - 配置缺省时使用默认值;显式配置后优先生效。配置值必须为 HTTPS(localhost 例外),否则 `login` 拒绝启动

#### 4.3.1 公共前置步骤

```
1. gate-cli cex auth login [--manual] [--force]
2. 已登录状态预检（见下方"已登录语义"）
3. GET /.well-known/oauth-authorization-server → 获取端点列表
4. POST /mcp/oauth/register → 动态注册客户端（幂等缓存：若 ~/.gate-cli/tokens.yaml 已有可用 client_id 则跳过）
   {
     client_name: "gate-cli",
     redirect_uris: [
       "http://127.0.0.1:18991/callback",    # 本地回调候选端口 1（默认首选）
       "http://127.0.0.1:18992/callback",    # 候选端口 2
       "http://127.0.0.1:18993/callback",    # 候选端口 3
       "http://127.0.0.1:18994/callback",    # 候选端口 4
       "http://127.0.0.1:18995/callback",    # 候选端口 5
       "<config.oauth2.display_callback_url>"            # 手动复制展示页(来自配置,默认 https://api.gatemcp.ai/mcp/oauth/cli-callback)
     ],
     grant_types: ["authorization_code", "refresh_token"],
     token_endpoint_auth_method: "none",
     scope: "read spot_trade futures_trade earn asset"    # 业务 scope，空格分隔（RFC 6749 §3.3）
   }
   → { client_id: "pkce_xxxxxxxxxxxx" }
5. 生成 PKCE: code_verifier (43-128 chars), code_challenge = BASE64URL(SHA256(verifier))
6. 选择回调模式（见 §4.3.2 决策）→ 跳 §4.3.3（自动）或 §4.3.4（手动）
```

**"已登录"状态语义**（步骤 2）：

| 当前 token 状态 | `login` 无 flag 行为 | `login --force` 行为 |
|----------------|-------------------|-------------------|
| access_token 有效且未过期 | 打印 `已登录 as uid=<uid>, scope=<scope>，如需重新授权请 'login --force' 或先 'logout'`，退出码 0 | 跳过 DCR（复用 client_id），**强制走完整授权流**（§4.3.2 决策） |
| access_token 过期 + refresh_token 有效 | **静默走 refresh 流程**（§4.5），不启动浏览器 | 同上 |
| access_token / refresh_token 都过期 或 tokens.yaml 不存在 | 进入步骤 3（完整授权） | 同 |

> 💡 约定：`login --force` 不清除旧 token，而是在新授权成功后原子替换；若新授权中途失败（例如用户取消），旧 token 保持可用，避免"登录失败反而把自己踢下线"。

**本地回调端口候选机制**（§4.3.3 A1 展开）：

- DCR 注册时一次性申明 5 个候选端口（`18991-18995`），全部精确匹配 `host+port+path`（与 §4.6 改造项 #2 的 exact match 规则兼容,**无需依赖 loopback wildcard**）
- 运行时 CLI 按序 `net.Listen("127.0.0.1:<port>")` 寻找**首个空闲端口**
- 选定端口后,授权 URL 和换 token 请求的 `redirect_uri` 都使用**同一个**候选 URL
- **端口耗尽处理(显式失败,不自动降级)**：若 5 个候选端口全被占用,CLI **立即报错退出**,不自动切换手动模式。错误文案：
  ```
  ❌ 本地回调端口已耗尽 (18991-18995 全部被占用)
     请关闭占用这些端口的程序后重试,或使用手动模式:
       gate-cli cex auth login --manual
  ```
  退出码 **`5`**(环境不满足,与 `doctor` 依赖缺失同义；**不复用 3**,避免与 `cex auth status` 的"未认证"语义冲突)。**理由**：端口冲突通常由用户本机其他长期运行的服务造成,属于环境问题,静默降级会掩盖它；显式报错让用户明确感知并决策(关进程 vs 走手动)。
- 未来若 OAuth2 server 支持 RFC 8252 §7.3 loopback wildcard,可简化为单个注册项,作为后续优化

#### 4.3.2 回调模式选择策略

**核心原则**：真 headless 场景**零等待直接走手动**；有浏览器场景**允许用户秒切**，不把人锁在超时里。

预检测采用**直接探测浏览器二进制**而非间接环境变量（`$DISPLAY` / `$SSH_CONNECTION` 在容器 / tmux / reattach 等场景有假阳假阴）：

```
func detectBrowser() bool {
    switch runtime.GOOS {
    case "darwin":
        return true                              // macOS 总有 `open`
    case "windows":
        return true                              // Windows 默认有浏览器关联
    case "linux":
        // WSL 可调用 Windows 宿主浏览器
        if b, _ := os.ReadFile("/proc/version"); bytes.Contains(
            bytes.ToLower(b), []byte("microsoft")) {
            return true
        }
        // 显式探测常见浏览器启动器
        for _, bin := range []string{
            "xdg-open", "sensible-browser",
            "google-chrome", "chromium", "chromium-browser",
            "firefox", "firefox-esr",
        } {
            if _, err := exec.LookPath(bin); err == nil {
                return true
            }
        }
        // $BROWSER 指向的自定义二进制
        if custom := os.Getenv("BROWSER"); custom != "" {
            if _, err := exec.LookPath(custom); err == nil {
                return true
            }
        }
        return false
    }
    return false
}
```

CLI 按以下优先级选择**初始**模式：

| 优先级 | 判定条件 | 选择 |
|--------|----------|------|
| 1 | 用户显式 `--manual` | 手动模式 |
| 2 | `detectBrowser() == false` | 手动模式（**零等待**） |
| 3 | 5 个候选端口 `18991-18995` 全部 `net.Listen` 失败 | **显式报错退出**(退出码 **5**,环境不满足),不自动降级；提示用户释放端口或改用 `--manual` |
| 4 | 其他(至少一个候选端口空闲) | 自动模式 |

> ⚠️ **不把 `browser.OpenURL` 的返回值作为判据**——它在 WSL / 自定义 `$BROWSER` / xdg-open 异常等场景下常有"假成功 / 假失败"，不可靠。自动模式内部通过"始终打印 URL + stdin 秒切 + 超时兜底"三重机制处理所有边界（见 §4.3.3）。

自动模式内置到手动模式的秒切路径和超时回退（§4.3.3 步骤 A5），覆盖预检测漏判的极端 headless 场景。

#### 4.3.3 自动模式（本地回调 + 双入口 + stdin 秒切 + 超时兜底）

自动模式**同时铺设两条将用户送到浏览器的入口**（auto-open + 终端打印 URL），共享**一条回调链路**（127.0.0.1:18991）。同时在等待期间**并发监听 stdin**，允许用户按 Enter 立即切换手动模式——真 headless 用户（即使预检测漏判）**零等待**。

```
A1. 按序尝试 18991 → 18995 端口启动回调监听（DCR 注册时已预留这 5 个候选 URI）
    - 首个 net.Listen 成功的端口记为 `chosenPort`
    - **全部占用 → 立即报错退出**（exit code **5**，环境不满足："端口已耗尽"），**不自动降级**手动模式（§4.3.2 priority 3）
    - 注册 /callback handler
A2. 构造授权 URL（redirect_uri 必须与 A1 选中的 chosenPort 一致）：
    GET /mcp/oauth/authorize
        ?response_type=code
        &client_id=pkce_xxx
        &redirect_uri=http://127.0.0.1:<chosenPort>/callback
        &code_challenge=...
        &code_challenge_method=S256
        &scope=read+spot_trade+futures_trade+earn+asset
        &state=<随机 32 字节>
A3. 【入口 1】终端打印授权 URL + 提示框：
    ┌─────────────────────────────────────────────────────┐
    │ 请在浏览器中完成授权:                                │
    │   https://api.gatemcp.ai/mcp/oauth/authorize?...    │
    │                                                     │
    │ • 浏览器已尝试自动打开                               │
    │ • 若未打开, 请手动复制上面的链接                     │
    │ • 若本机无浏览器 (SSH / 容器), 按 [Enter] 切手动模式 │
    └─────────────────────────────────────────────────────┘
A4. 【入口 2】best-effort 调用 browser.OpenURL（忽略返回值，不作为控制流判据）
A5. 三路并发等待（select）：
    ├─ callbackCh : /callback 触发 → 正常完成，跳 A6
    ├─ stdinCh    : 用户按 Enter → 立即切换，跳 A7（零等待）
    └─ timeoutCh  : 60 秒兜底 → 自动切换，跳 A7
A6. 正常回调 → http://127.0.0.1:18991/callback?code=abc&state=xxx
    - CLI 校验 state 一致
    - 回调页面 HTML 返回「授权成功，可关闭此页面」
    - 关闭监听器 + 取消 stdin goroutine（进程即将退出，泄漏可接受）
    - 跳 §4.3.5 换 token
A7. 切换到手动模式（秒切 或 超时触发）:
    - 关闭 localhost 监听器（释放端口）
    - 丢弃当前 state / code_verifier，生成新的
    - 跳 §4.3.4 重新发起 authorize（redirect_uri 换成展示页）
```

**stdin 并发监听实现要点**（`auth/oauth2.go`）：

```go
stdinCh := make(chan struct{}, 1)
go func() {
    reader := bufio.NewReader(os.Stdin)
    _, _ = reader.ReadString('\n')
    select { case stdinCh <- struct{}{}: default: }
}()

select {
case result := <-callbackCh:
    // A6
case <-stdinCh:
    // A7: 零等待
case <-time.After(60 * time.Second):
    // A7: 兜底
}
```

> ⚠️ **stdin goroutine 泄漏说明**：Go 标准库无法 cancel 阻塞在 `os.Stdin.Read` 的 goroutine。A6 正常完成后该 goroutine 会泄漏直到进程退出。`cex auth login` 是单次短生命周期命令，换完 token 后 §4.3.5 立即返回主函数退出，泄漏可接受。**不要在长驻进程中复用此模式。**
>
> 💡 超时从 90s 缩短到 **60s**：§4.3.2 已经通过浏览器二进制探测过滤了绝大多数真 headless 场景；即使漏判，用户看到提示框第三行 "按 Enter 切手动模式" 可在 1 秒内主动切换。60s 只兜"用户离开座位"之类的边缘情形。
>
> 💡 入口 1（打印 URL）覆盖 WSL / 自定义 `$BROWSER` / xdg-open 异常等 auto-open 不可靠的场景——用户复制到任意本地浏览器即可，**仍走本地 127.0.0.1 回调，无需换模式**。

#### 4.3.4 手动模式（OOB 复制粘贴）

```
M1. 不启动本地 HTTP 服务器
M2. 构造授权 URL（redirect_uri 换成展示页，取值来自 `config.oauth2.display_callback_url`）：
    GET /mcp/oauth/authorize
        ?response_type=code
        &client_id=pkce_xxx
        &redirect_uri=<config.oauth2.display_callback_url>    # 默认 https://api.gatemcp.ai/mcp/oauth/cli-callback
        &code_challenge=...
        &code_challenge_method=S256
        &scope=read+spot_trade+futures_trade+earn+asset
        &state=<随机 32 字节>
M3. 终端打印提示框（单行 URL 便于鼠标选中 / tmux copy-mode 复制）：
    ┌─────────────────────────────────────────────────────┐
    │ 手动授权模式:                                        │
    │                                                     │
    │ 1. 复制下方链接到任意浏览器打开:                     │
    │    https://api.gatemcp.ai/mcp/oauth/authorize?...   │
    │                                                     │
    │ 2. 完成授权后, 浏览器会跳转到展示页,                 │
    │    将页面上显示的 code 粘贴到下方:                   │
    └─────────────────────────────────────────────────────┘
    请粘贴 code: ▊
M4. 用户登录 + 授权确认
M5. 浏览器跳转到 https://api.gatemcp.ai/mcp/oauth/cli-callback?code=abc&state=xxx
    页面展示 code（等宽字体 + 复制按钮）；若 server 返回错误，则以 `?error=...&error_description=...`
    形式展示，用户可回到 CLI 按 Ctrl+C 中止后重试
M6. 用户将 code 粘贴回 CLI；CLI 通过 term.ReadPassword 隐式读取（不回显、不落 shell history）
    - ⚠️ CLI 不提供 --code <value> 命令行参数方式，避免 code 进入 shell 历史 / 进程列表
    - 输入前后自动 TrimSpace，容忍用户复制时带入的前后空白
    - 空输入或 Ctrl+D 视为中止，返回非 0
M7. 手动模式下 state 无法自动带回 CLI，**不校验 state**，安全性由 PKCE code_verifier 绑定保证
    （符合 RFC 8252 §8.1 native app BCP）
M8. 跳 §4.3.5 换 token
```

#### 4.3.5 换 token & 落盘（两种模式共用）

```
T1. POST /mcp/oauth/token
    { grant_type: "authorization_code",
      code: "abc",
      code_verifier: "zzz",
      client_id: "pkce_xxx",
      redirect_uri: <与授权请求一致的那一个> }
    → { access_token: "pkce_at_...", refresh_token: "pkce_rt_...",
        expires_in: 7200, scope: "read spot_trade futures_trade earn asset" }
T1.5 **实际颁发 scope 校验**：比对响应 `scope` 与请求 `scope`：
    - 完全一致 → 继续 T2
    - 子集（用户或 server 降级同意）→ 打印 warning：
        ⚠️  实际授权范围 "<granted>" 小于申请范围 "<requested>"，
            缺失: <missing>。部分命令（如 cex spot order place）将在调用时被
            gateway 拒绝（403 insufficient_scope）。
            如需补齐: gate-cli cex auth login --force --scope "<requested>"
      继续 T2（token 仍落盘，用户可立即使用已获得的子集 scope）
    - 超集（理论上 server 不应下发）→ 仅记录 debug 日志，按实际 scope 存盘
T2. 存储到 ~/.gate-cli/tokens.yaml（包含 client_id 以便下次复用跳过 DCR；scopes 字段
    始终记录 server 实际颁发值，而非申请值）
```

> 🔒 **安全要点**
> - `code_verifier` 仅存在 CLI 进程内存，授权链结束后立即丢弃。
> - 自动模式**必须**校验 `state`；手动模式**不校验 state**，由 PKCE 绑定保证 code 不可被替换。
> - `redirect_uri` 在换 token 时必须与授权请求完全一致，否则 OAuth2 server 会拒绝（RFC 6749 §4.1.3）。
> - `authorization_code` 推荐 TTL = **300 秒（5 分钟）**，与 §4.6 改造项 #3 对齐。安全由 PKCE `code_verifier` + one-time-use 保证，不依赖短 TTL。

#### 4.3.6 体验矩阵（TL;DR）

覆盖所有常见场景、等待时长与用户操作的汇总表：

| 场景 | 预检测结果 | 初始模式 | 等待 | 用户操作 |
|------|-----------|---------|------|---------|
| macOS 桌面 | 有浏览器 | 自动 | 0s | 浏览器自动弹出 → 点同意 |
| Windows 桌面 | 有浏览器 | 自动 | 0s | 同上 |
| WSL（Ubuntu on Windows） | 有浏览器（`/proc/version` 含 `microsoft`） | 自动 | 0s | Windows 宿主浏览器弹出 → 点同意 |
| Linux 桌面 + Firefox/Chrome | 有浏览器（`LookPath` 命中） | 自动 | 0s | 浏览器自动弹出 → 点同意 |
| Linux 桌面 + auto-open 假失败（自定义 `$BROWSER` / xdg-open 坏了） | 有浏览器 | 自动 | 0s | 从终端复制 URL 到本地浏览器 → 仍走 127.0.0.1 回调 |
| SSH 远程（`LookPath` 全失败） | 无浏览器 | **直接手动** | **0s** | 复制 URL → 本机浏览器授权 → 粘 code |
| Docker 容器 / CI runner | 无浏览器 | **直接手动** | **0s** | 同上 |
| `--manual` 显式 | 不检测 | **直接手动** | **0s** | 同上 |
| 预检测漏判（极端：装了 `xdg-open` 但没装浏览器） | 误判有浏览器 | 自动 → 用户按 Enter 秒切 | **≈ 1-3s** | 读提示框 → 按 Enter → 粘 code |
| 用户离开座位（极少数） | 任意 | 自动 → 60s 超时兜底 | 60s | 回来时已在手动模式提示 |

**关键结论：真 headless 环境等待时长为 0，不存在"等 90 秒才能切换"的糟糕路径。**

#### 4.3.7 决策流图

```
          ┌──────────────────────────┐
          │ gate-cli cex auth login  │
          └────────────┬─────────────┘
                       │
              ┌────────▼────────┐
              │ --manual 标志?  │
              └────┬────────┬───┘
              是   │        │   否
                   │        │
     ┌─────────────┘        ▼
     │          ┌──────────────────────┐
     │          │ detectBrowser() ?    │
     │          │ (exec.LookPath)      │
     │          └────┬─────────────┬───┘
     │           有  │             │ 无
     │               ▼             │
     │     ┌─────────────────┐     │
     │     │ 监听端口成功?   │     │
     │     └───┬──────────┬──┘     │
     │      是 │          │ 否     │
     │         ▼          │        │
     │   ┌───────────┐    │        │
     │   │ 自动模式  │    │        │
     │   │ §4.3.3    │    │        │
     │   └─────┬─────┘    │        │
     │         │          │        │
     │   ┌─────┴─────────┐│        │
     │   │ 三路 select:  ││        │
     │   │ • callback    ││        │
     │   │ • stdin       ││        │
     │   │ • 60s timeout ││        │
     │   └─┬──────┬──────┘│        │
     │     │callback      │        │
     │     ▼      │       │        │
     │  ┌───────┐ │       │        │
     │  │ §4.3.5│ │       │        │
     │  │ token │ │       │        │
     │  └───────┘ │       │        │
     │           stdin/timeout      │
     │            │      │        │
     └─────────┬──┴──────┴────────┘
               ▼
        ┌─────────────┐
        │ 手动模式    │
        │ §4.3.4      │
        └──────┬──────┘
               │
               ▼
         ┌───────────┐
         │ §4.3.5    │
         │ token 落盘│
         └───────────┘
```

### 4.4 Token 存储

```yaml
# ~/.gate-cli/tokens.yaml
cex:
  default:  # profile name
    oauth2:  # 认证协议 namespace（为 DEX 的钱包签名认证预留并列空间）
      client_id: "pkce_xxxxxxxxxxxx"
      access_token: "pkce_at_xxxxxxxxxxxxxxxxxxxxxxxx"
      refresh_token: "pkce_rt_xxxxxxxxxxxxxxxxxxxxxxxx"
      token_type: "Bearer"
      expires_at: "2026-04-11T20:00:00Z"
      scopes: "read spot_trade futures_trade earn asset"
      oauth2_base_url: "https://api.gatemcp.ai"
# 多 profile 示例（同一物理文件，profile 之间相互隔离）
# cex:
#   personal:
#     oauth2:
#       client_id: "pkce_aaa..."
#       access_token: "pkce_at_aaa..."
#       refresh_token: "pkce_rt_aaa..."
#       expires_at: "2026-04-13T20:00:00Z"
#       scopes: "read spot_trade futures_trade earn asset"
#       oauth2_base_url: "https://api.gatemcp.ai"
#   trading-bot:
#     oauth2:
#       client_id: "pkce_bbb..."
#       access_token: "pkce_at_bbb..."
#       refresh_token: "pkce_rt_bbb..."
#       expires_at: "2026-04-13T22:00:00Z"
#       scopes: "read spot_trade"
#       oauth2_base_url: "https://api.gatemcp.ai"
#
# 未来扩展（DEX 使用钱包签名而非 OAuth2）
# dex:
#   default:
#     wallet:
#       address: "0x..."
#       signature: "..."
#       chain_id: 1
```

> 💡 **Profile 隔离语义**：`--profile trading-bot` 仅读写 `cex.trading-bot.*` 子树，互不影响。`logout --all` 会清除所有 profile 的 oauth2 子树；`logout`（默认）只清当前 profile。`tokens.lock` 为全局锁（§4.5.1），跨 profile 并发 refresh 会串行化。

#### 4.4.1 文件权限要求（安全强制）

> 📌 **已评审决议(v3)**：token 采用 **0600 明文落盘**,**不引入** OS keyring / 加密存储方案。评审结论：
> - 威胁模型主要防护"同机其他用户"和"误提交进版本库",0600 + 独立目录已覆盖
> - keyring 跨平台兼容性差(Linux headless / Docker / CI 环境需大量 fallback 代码),增加交付复杂度
> - 参照业界同类工具(aws-cli / gh / kubectl)惯例,明文 0600 是通行做法
> - 后续若有企业用户提出强合规需求,作为 Phase 2+ 独立专项评估

⚠️ **`access_token` / `refresh_token` 是 30 天长期凭证，文件权限必须严格控制**，参照 `~/.ssh/id_rsa` 的处理：

| 对象 | 权限 | 实现 |
|-----|------|------|
| `~/.gate-cli/` 目录 | `0700`（owner rwx） | 创建时 `os.MkdirAll(dir, 0700)` |
| `~/.gate-cli/tokens.yaml` | `0600`（owner rw） | 写入时 `os.OpenFile(path, O_CREATE\|O_WRONLY\|O_TRUNC, 0600)` |
| `~/.gate-cli/tokens.lock` 🆕 | `0600` | 并发 refresh 互斥锁（见 §4.5） |
| `~/.gate-cli/config.yaml`（旧） | `0600` | 迁移时同步收紧 |

**启动时权限校验**（`internal/auth/store.go` 的 `Load()`）：

```go
info, err := os.Stat(path)
if err != nil { return nil, err }
if runtime.GOOS != "windows" && info.Mode().Perm() != 0600 {
    return nil, fmt.Errorf(
        "tokens.yaml 权限不安全 (%o)，应为 0600；已拒绝加载以保护 refresh_token。\n"+
        "修复: chmod 600 %s", info.Mode().Perm(), path)
}
```

> 💡 Windows NTFS 没有 POSIX 权限位，跳过该检查（依赖文件系统 ACL）。

### 4.5 Token 自动刷新

> 📌 **设计前提（已评审决议）**：apiv4-gateway 的 token 缓存**不做主动失效**（见 §5.7 评审决议）——token 撤销/scope 变更后,gateway 仍可能在最长 300 秒内使用旧缓存。因此 **gate-cli 必须保证 access_token 在其服务端有效期到期之前已被主动刷新**,避免出现"本地 token 已过期但请求仍发出、被 gateway 拒绝"的窗口。刷新窗口由 CLI 侧单方面保证,不依赖 gateway 侧的任何感知或反馈。

**基础流程：**

- **主动提前刷新(mandatory)**：每次 API 调用前检查 `access_token.expires_at`,若剩余有效期 **≤ 60 秒** 则**必须**先刷新再发请求。60 秒窗口覆盖：(a) 刷新请求的 RTT、(b) gateway 侧校验 token 的 RTT、(c) 本地时钟误差容差
- 过期(或即将过期)则调用 `POST /mcp/oauth/token` with `grant_type=refresh_token`
- Refresh token 默认有效期 30 天(OAuth2 server 配置)
- 刷新后新 token **原子写回** `tokens.yaml`(§4.5.1 文件锁保护),后续请求使用新 token
- Refresh token 也过期时,提示用户重新 `gate-cli cex auth login`

**60 秒刷新窗口的量化依据**：

刷新窗口必须满足以下不等式:

```
refresh_window ≥ network_rtt + gateway_check_rtt + local_clock_skew + safety_margin
```

| 分项 | 预算 | 依据 |
|-----|------|------|
| `network_rtt` | ≤ 10s | CLI → OAuth2 server `/mcp/oauth/token` refresh 调用的 P99(跨地域、弱网场景) |
| `gateway_check_rtt` | ≤ 5s | apiv4-gateway → check_endpoint 的 P99 < 20ms(§4.6 改造项 #6 硬性指标) + L1/L2 缓存查找 + fan-out 补全 user-center/gatekeeper |
| `local_clock_skew` | ≤ 30s | 未启用 NTP 或 NTP 同步异常的工作站常见偏差上限(参考 Kerberos 默认 5 分钟容差的 1/10 保守值) |
| `safety_margin` | ≥ 15s | 覆盖 refresh 失败重试、文件锁等待、原子写盘等本地开销 |
| **合计下界** | **60s** | |

**关键假设(前置条件,写入 §12 验收标准)**:
- OAuth2 server `check_endpoint` P99 < 20ms(§4.6 改造项 #6)
- OAuth2 server `refresh_token` 端点 P99 < 3s
- 用户工作站本地时钟误差 ≤ 30s(未满足时 CLI `doctor` 应给出警告,Phase 1b+ 增强)

**不使用 30 秒的理由**:原 30s 方案**未计入本地时钟误差**。若本地时钟快 30s(NTP 失同步场景),CLI 以为"还有 31s 到期",实际 token 已在 gateway 侧过期,请求被拒——这是"本地尚未过期但请求失败"的反直觉窗口,必须通过拉大刷新窗口预防。

**运行时降级**:若 refresh 请求本身超时或失败,CLI 进入 `refresh.go` 的重试逻辑(指数退避,最多 3 次)；全部失败才按"refresh_token 过期"处理,提示用户重新 login。详见 §4.5.3。

#### 4.5.1 并发 refresh 互斥（P0 race 修复）

OAuth2 server 对 refresh_token 采用**全量轮换策略**（§4.1）：每次 refresh 成功后旧 refresh_token 立即失效。若用户并发执行多条 CLI 命令（如 `gate-cli cex spot market ticker &` + `gate-cli cex futures position list &`），两个进程同时发现 access_token 快过期并发起 refresh——**第二个必然因 `invalid_grant` 失败**（旧 token 已被删）。

**解决方案：文件锁互斥 + double-check**

```go
// internal/auth/refresh.go
import "github.com/gofrs/flock"

func RefreshIfNeeded(store *TokenStore) (*Token, error) {
    // 1. 先用非独占锁读一下，快速路径
    tok := store.Load()
    if !tok.ExpiringSoon(60 * time.Second) {
        return tok, nil  // 无需 refresh，直接返回
    }

    // 2. 获取独占文件锁（~/.gate-cli/tokens.lock，0600 权限）
    lock := flock.New(filepath.Join(store.Dir(), "tokens.lock"))
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    if ok, err := lock.TryLockContext(ctx, 100*time.Millisecond); !ok || err != nil {
        return nil, fmt.Errorf("tokens.lock 获取失败，可能有其他 gate-cli 进程正在刷新: %w", err)
    }
    defer lock.Unlock()

    // 3. 临界区内重新读 tokens.yaml（double-check）
    //    别的进程可能在我们等锁期间已经 refresh 完成并写回
    fresh := store.Load()
    if !fresh.ExpiringSoon(60 * time.Second) {
        return fresh, nil  // 复用别人的新 token，跳过 refresh
    }

    // 4. 确认确实需要 refresh，发起 OAuth2 refresh_token 请求
    newTok, err := doRefresh(fresh.RefreshToken, fresh.ClientID)
    if err != nil {
        return nil, err
    }

    // 5. 写回 tokens.yaml（原子写：先写到 tmp 文件，再 rename）
    return store.Save(newTok)
}
```

**关键设计点：**

| 点 | 说明 |
|----|------|
| 锁文件独立 | `tokens.lock` 单独管理，不锁 `tokens.yaml` 本身，避免 reader 被阻塞 |
| Double-check | 获取锁后重读 tokens.yaml——若别的进程已刷新完，直接复用新 token，跳过重复 refresh |
| 原子写回 | `tokens.yaml.tmp` → `os.Rename(tmp, tokens.yaml)`，避免半写状态被其他进程读到 |
| 10 秒锁等待超时 | 防止死锁；超时后报清晰错误而非无限等待 |
| 非 Windows 校验权限 | 同 §4.4.1，读取时检查 `0600` |

> 📌 **tokens.lock 为全局锁**：当前实现单一 `~/.gate-cli/tokens.lock`，而非 per-profile 锁。多 profile 并发 refresh 会串行化（锁等待典型 < 200ms），相比 per-profile 锁换来的简单性和无 profile 名称转义问题，性能代价可接受。若未来出现多 profile 高频并发场景再拆分。

#### 4.5.2 并发场景覆盖测试

§12.3 子任务 5 的验收标准追加：

- ✅ 并发 2 个 `gate-cli cex spot market ticker` 进程，均能成功返回数据
- ✅ 并发 5 个进程，首个 refresh，其余 4 个 double-check 后复用，OAuth2 server 只收到 1 次 refresh 请求
- ✅ Kill 正在 refresh 的进程后，flock 自动释放，后续命令可继续

#### 4.5.3 Refresh 失败 / Crash Recovery / refresh_token 彻底失效

Refresh 链路的所有失败场景及 CLI 的恢复策略：

| 场景 | Server 返回 | CLI 行为 | tokens.yaml 变动 |
|------|-----------|---------|-----------------|
| 网络瞬时抖动 | timeout / 5xx | 退避重试 2 次（200ms/1s），仍失败则按 **业务 API 失败**向用户报错（不清 token） | 无 |
| refresh_token 过期（30 天） | 400 `invalid_grant` + `error_description: expired` | 清除本节下方 §4.5.3.1 定义的字段，提示 `请重新运行 gate-cli cex auth login`，退出码 2 | 清除 access_token / refresh_token / expires_at / scopes；**保留** client_id + oauth2_base_url（下次 login 跳过 DCR） |
| refresh_token 被服务端撤销（管理员 revoke） | 400 `invalid_grant` | 同上 | 同上 |
| refresh_token 因并发 race 已被全量轮换删除 | 400 `invalid_grant` | **先重读 tokens.yaml 一次**（可能别的进程刚写完）→ 若读到的 refresh_token ≠ 刚才发起请求用的值，复用新值继续；仍不一致则按 "彻底失效" 处理 | 按情况 |
| client_id 被吊销 | 400 `invalid_client` | 清除全部 oauth2 字段（含 client_id），提示重新 login（会重新 DCR） | 清空整个 `cex.<profile>.oauth2` 子树 |
| 任意 4xx 非 invalid_grant/invalid_client | 按原始错误消息返回 | 不清 token（避免误删） | 无 |

##### 4.5.3.1 "清理 token" 的准确语义

```yaml
# 清理前
cex:
  default:
    oauth2:
      client_id: "pkce_xxxxxxxxxxxx"        # 保留
      access_token: "pkce_at_xxx"           # 清除
      refresh_token: "pkce_rt_xxx"          # 清除
      token_type: "Bearer"                  # 清除
      expires_at: "2026-04-11T20:00:00Z"    # 清除
      scopes: "read spot_trade futures_trade earn asset" # 清除
      oauth2_base_url: "https://api.gatemcp.ai"  # 保留

# 清理后
cex:
  default:
    oauth2:
      client_id: "pkce_xxxxxxxxxxxx"
      oauth2_base_url: "https://api.gatemcp.ai"
```

##### 4.5.3.2 Crash Recovery

**场景**：进程 A 在 §4.5.1 步骤 4 成功收到 server 返回的新 token 但步骤 5 写回磁盘**之前** crash（OOM / kill -9 / 宕机）。此时：

- Server 侧：旧 refresh_token 已失效，新 refresh_token 已颁发但只在崩溃进程的内存
- 本地：tokens.yaml 仍是旧值
- 后续进程 B 读 tokens.yaml → 用旧 refresh_token refresh → server 返回 `invalid_grant`

**恢复路径**：进程 B 命中 `invalid_grant` → 按 §4.5.3.1 清理 oauth2 子树（保留 client_id）→ 提示用户重新 `cex auth login`。重新 login 时由于 client_id 保留，跳过 DCR，用户只需走一次 PKCE 授权流（约 5 秒）。

**为什么不做"写前镜像"**：原子 rename 已保证文件层面一致性，唯一无法恢复的是 server 已颁发但本地未落盘的新 token。这是 OAuth2 refresh 全量轮换协议的固有 race，业界通行处理就是"要求用户重新授权"。复杂的 WAL / 两阶段提交换不回足够大的收益。

##### 4.5.3.3 `invalid_grant` 告警指标

`internal/auth/refresh.go` 上报：

| 指标 | 说明 |
|------|------|
| `cli_auth_refresh_invalid_grant_total` | counter，本地日志即可，不上报（CLI 无 metrics 通道） |
| `auth.log` 追加 `event=refresh_invalid_grant uid=<uid> reason=<desc>` | 便于用户 / SRE 排障 |

### 4.6 OAuth2 Server 侧改造清单（前置依赖）

gate-cli 本身不依赖 gateapi-mcp-service，但 §4.3 的新授权流需要 **OAuth2 项目（`oauth2/pkce/*`）** 做以下改造。这些是 Phase 1a 的**前置依赖**，必须先于 gate-cli 的 `cex auth login` 实现落地。

| # | 改造项 | 模块 / 文件 | 说明 |
|---|--------|-------------|------|
| 1 | **新增 CLI 回调展示页** | OAuth2 server 路由层（新增 `handler/cli_callback.go` 或等价） | `GET /mcp/oauth/cli-callback` → 返回静态 HTML 模板，内嵌 JS 从 `location.search` 解析 `code` / `state` / `error` 并展示。页面**不读 session、不下发 token、不查 DB**，纯展示。需通过 OAuth2 server 现有路由注册，保证 `api.gatemcp.ai` 域名下可访问。 |
| 2 | **DCR 支持多 `redirect_uris`** | `oauth2/pkce/handler.go:93-164`（DCR handler） | 确认 `redirect_uris` 字段按 RFC 7591 接收数组并全部落库；**授权阶段**校验请求中的 `redirect_uri` 参数必须**精确匹配**已注册列表中的某一项（exact match，不允许子串/前缀匹配）。若当前实现已按数组处理，仅需回归测试覆盖"同一 client 用不同 redirect_uri 发起授权"的场景。 |
| 3 | **确认 authorization_code TTL** | `oauth2/pkce/token.go`（code 生成 / 校验） | 核实当前 code 的有效期，**推荐 300 秒（5 分钟）**。理由：手动模式下用户需完成"终端读 URL → 切浏览器（含 SSH 跨机）→ 登录 → 2FA → 授权 → 展示页 → 复制 code → 切回终端 → 粘贴"全链路；首次登录带 2FA 时 60s 明显过紧。300s 是 Google / GitHub / Okta 等主流 OAuth2 server 的通用值，安全性由 PKCE `code_verifier` + one-time-use 保证，不依赖短 TTL。确认当前 TTL 并调整到 300s。 |
| 4 | **授权请求支持 `redirect_uri` 显式传参** | `oauth2/pkce/handler.go`（authorize endpoint） | 当 client 注册了多个 `redirect_uris` 时，授权请求**必须**携带 `redirect_uri` 参数指定本次使用哪一个（RFC 6749 §3.1.2.3）。确认 authorize handler 已正确解析该参数并在换 token 阶段校验一致性（RFC 6749 §4.1.3）。 |
| 5 | **cli-callback 路由白名单** | OAuth2 server 中间件 / 鉴权配置 | 确保 `/mcp/oauth/cli-callback` 被列入**无需认证**的公开路径。该端点本身不做任何权限判断，仅为浏览器渲染展示用。 |
| 6 | **内部 check_endpoint 对 apiv4-gateway 开放** | OAuth2 server 内网路由 + `oauth2/rest/trading_view.go:356-394` `doCheckAccessToken()` + `oauth2/pkce/model.go` AccessTokenCheckResult | apiv4-gateway 每次收到 CLI Bearer 请求都需要向 OAuth2 server 的内部 check_endpoint 换取 `uid` + `scope` + `active`（见 §5.3）。需确认：<br>**(a) 路由与连通性**：确切路径为 `POST /oauth/internal/api/check`，header `Gatepay-Access-Token: <token>`；apiv4-gateway upstream IP 白名单包含该端点；<br>**(b) 响应 schema 字段名与类型保持不变**（**向后兼容硬约束**）：`uid`(int64)、`clientId`(string)、`active`(bool)、`scope`(string，空格分隔)、`expiresIn`(**int64 Unix 时间戳秒，非 RFC 6749 相对秒数**)、`deviceId`(string)、`deviceName`(string)；<br>**(c) ⚠️ `expiresIn` 字段语义不得修改**：虽然字段名易与 RFC 6749 的 `expires_in`（相对秒数）混淆，但 gateapi-mcp-service 和本次 apiv4-gateway 改造均按"Unix 时间戳"语义消费该字段（gateway 侧计算剩余 TTL 使用 `expiresIn - now()`）。若本次改造计划重命名或改语义，必须同步通知 gateapi-mcp-service 与 apiv4-gateway 团队做 breaking change 评估，**默认保持现状**；<br>**(d) 性能**：压测 1k QPS 下 P99 < 20ms（Phase 1b-GW CLI 60s 主动刷新窗口依赖此假设，见 §4.5）；<br>**(e) 建议与 gateapi-mcp-service 共用同一端点**，避免额外部署。该端点**已存在**（gateapi-mcp-service 在用），此项主要为**确认 + 回归测试 + schema 冻结文档化**，非新开发。 |
| 7 | **端点限流配置** | OAuth2 server 入口层（Nginx / Envoy） | 对 §5.10 表格中的 5 个端点配置限流，防止无认证 DCR 滥用和 code 暴力枚举。超限返回 429 + `Retry-After`。与业务逻辑解耦，在入口层实现。 |

> ⚠️ **改造范围限定**：以上 7 项全部在 OAuth2 项目（后端）内完成，**不涉及 gateapi-mcp-service、不涉及 apiv4-gateway、不涉及前端独立页面工程**。gate-cli 与 OAuth2 server 之间是直接 HTTPS 调用关系，链路如下：
> ```
> gate-cli ──直连──▶ api.gatemcp.ai (OAuth2 server)
>                    ├─ /.well-known/oauth-authorization-server
>                    ├─ /mcp/oauth/register
>                    ├─ /mcp/oauth/authorize
>                    ├─ /mcp/oauth/token
>                    └─ /mcp/oauth/cli-callback    🆕
> ```
> gateapi-mcp-service 仅在 gate-local-mcp 的 MCP 链路中作为 metadata 代理存在，**不在 gate-cli 授权链路上**。

> 📋 **交付顺序**：OAuth2 server 改造（§4.6）→ gate-cli `cex auth login` 实现（§4.3）→ apiv4-gateway OAuth2 Bearer 支持（§5）。前者是后者的阻塞依赖。

### 4.7 请求数据流全景

授权流程（§4.3）只是**初次登录**时的一次性动作；日常使用中 CLI 发起的每一次业务 API 调用（如 `gate-cli cex spot market ticker --pair BTC_USDT`）都会走以下链路：

```
┌─────────┐   Bearer pkce_at_xxx   ┌─────────────────┐
│ gate-cli│ ─────────────────────▶ │ apiv4-gateway   │
└─────────┘   (HTTPS /api/v4/...)  │ (OpenResty+Lua) │
                                   └────────┬────────┘
                                            │
                          ┌─────────────────┼──────────────────┐
                          │                 │                  │
              POST check_endpoint    GET user-center     GET gatekeeper
              Header: Gatepay-Access-  /user/{uid}/info   /internal/user/
                      Token: <token>                      {uid}/main_uid
                          │                 │                  │
                          ▼                 ▼                  ▼
                 ┌─────────────────┐ ┌───────────────┐ ┌──────────────┐
                 │ OAuth2 server   │ │ user-center   │ │ gatekeeper   │
                 │ 返回 uid/scope/ │ │ 返回 tier /   │ │ 返回 main_uid│
                 │ active/expires  │ │ verified 等   │ │              │
                 └─────────────────┘ └───────────────┘ └──────────────┘
                          │                 │                  │
                          └─────────────────┼──────────────────┘
                                            ▼
                                ┌─────────────────────┐
                                │ 组装 X-Gate-*       │
                                │ headers             │
                                │ (uid, tier, main_uid│
                                │  verified, ...)     │
                                └──────────┬──────────┘
                                           │
                                           ▼
                                ┌──────────────────────┐
                                │ proxy_pass 后端业务  │
                                │ 服务（spot/futures/  │
                                │  wallet/account ...） │
                                └──────────────────────┘
```

**关键点：**

1. **apiv4-gateway 是 token 验证的强制入口**。CLI 不直连 OAuth2 server 做业务 API 调用——授权阶段（§4.3）CLI 直连 OAuth2 server 拿 token；业务阶段所有请求走 apiv4-gateway，由 gateway 代为验证 token 并换取 uid。CLI 本身**不知道也不需要知道** uid。

2. **check_endpoint 是 gateway 向 OAuth2 server 的内部调用**。它**不是** OAuth2 server 对外的 `/mcp/oauth/token` 或 RFC 7662 `/introspect`，而是 gateapi-mcp-service 已在使用的内部快速校验端点（`oauth2/pkce/model.go:62-71`）。apiv4-gateway 与 gateapi-mcp-service **共用同一 check_endpoint**，避免重复实现（见 §4.6 改造项 #6）。

3. **uid 是 gateway 派发权限和拼装 X-Gate-* headers 的唯一身份锚点**。check_endpoint 返回的 `uid` 是后续所有下游权限判断的起点：
   - `scope` → §5.8 scope → API 路径段白名单
   - `uid` → user-center 换 `user_tier` / `verified` / `language` / `country`
   - `uid` → gatekeeper 换 `main_uid`（子账号 → 主账号映射）
   - 组装完毕后通过 `proxy_pass` 的 header 注入，下游业务服务**完全不感知** OAuth2 的存在，继续按原 `X-Gate-Uid` / `X-Gate-Main-Uid` 等 header 工作。

4. **缓存是性能关键**（详见 §5.7）：
   - check_endpoint 调用结果 L1 mlcache 120s + L2 shared_dict 300s；
   - user-center / gatekeeper 结果独立缓存（复用现有 user_info_cache 模块）；
   - CLI 高频调用（如行情轮询）命中 L1 缓存时，**单次请求额外开销 < 1ms**，与原 HMAC 路径持平。

5. **HMAC 与 OAuth2 并行**：gateway 按 `Authorization: Bearer` header 是否存在分发（§5.2），**不破坏**现有 HMAC 调用方（SDK 用户、老脚本、webhook 签名回调等）。Phase 1b 完成后两种认证方式在同一 gateway 实例上共存。

> 📌 **Info / News 链路不在此图范围**：`gate-cli info *` 和 `gate-cli news *` 走 MCP 协议直连 `api.gatemcp.ai/mcp/info` 和 `api.gatemcp.ai/mcp/news`（见 §6.1），**无需认证、不经 apiv4-gateway、不消耗 OAuth2 token、不触发 check_endpoint**。Info/News 后端自有路由，与 CEX 业务链路完全解耦。本节的所有性能分析和改造都仅适用于 CEX 路径。

### 4.8 Logout / Status / Scope 管理（认证生命周期闭环）

除 `login` 之外的认证管理命令，以及 `insufficient_scope` / `login --scope` 等用户引导路径。

#### 4.8.1 `cex auth logout`

```
gate-cli cex auth logout [--profile <name>] [--all] [--keep-client] [--purge]
```

**行为矩阵：**

| Flag | 行为 |
|------|------|
| （默认） | 清除当前 profile 的 access_token / refresh_token / expires_at / scopes，**保留 client_id 与 oauth2_base_url**（与 §4.5.3.1 一致，下次 login 复用 DCR） |
| `--keep-client` | 同默认，显式化语义 |
| `--all` | 清除 tokens.yaml 中**所有 profile** 的 oauth2 子树；client_id 全部保留 |
| `--purge` | 额外清除 client_id，下次 login 会重新 DCR。适用于"怀疑 client 被泄露"场景 |

**server 端 revoke：**

当前 OAuth2 server（`oauth2/pkce/*`）**未暴露 RFC 7009 `/revoke` 端点**，logout 为纯本地动作。Phase 2+ 的增强项：

- OAuth2 server 侧新增 `POST /mcp/oauth/revoke` 接受 `token` + `token_type_hint`（access_token / refresh_token）
- CLI `logout` 先 best-effort 调用 revoke（失败不阻断），再清本地文件
- 在 §4.6 改造清单标记为 **P2 后置项**，不阻塞 Phase 1

logout 退出码：0=成功；1=tokens.yaml 不存在或已为空（幂等，仅 warning）。

#### 4.8.2 `cex auth status`

只读查询，不触发任何 refresh。输出 schema（`--format json` 时）：

```json
{
  "profile": "default",
  "authenticated": true,
  "uid": 12345678,
  "client_id": "pkce_xxxxxxxxxxxx",
  "oauth2_base_url": "https://api.gatemcp.ai",
  "scopes": ["read", "spot_trade", "futures_trade", "earn", "asset"],
  "access_token_expires_at": "2026-04-13T20:00:00Z",
  "access_token_expires_in_seconds": 3540,
  "refresh_token_expires_at": "2026-05-11T12:00:00Z",
  "needs_refresh": false,
  "source": "oauth2"
}
```

**字段说明：**

| 字段 | 来源 |
|------|------|
| `uid` | 通过 OAuth2 server 的 introspection 端点（§4.1）查询得到；查询失败时字段省略并显示 `uid_unavailable: true` |
| `refresh_token_expires_at` | 若 server 返回了 refresh_expires_in 则精确，否则按 `login 时间 + 30 天` 估算并标 `refresh_token_expires_at_estimated: true` |
| `source` | `api_key_flag` / `api_key_config` / `api_key_env` / `oauth2` / `unauthenticated` / `none_blocked`——反映 §3.2 / §3.2.1 决策矩阵的**实际命中源** |
| `needs_refresh` | `access_token_expires_in_seconds < 30` |

**table 模式输出**（默认）以人类可读格式展示核心字段，不展示 token 明文；`access_token` 始终 redact 为 `pkce_at_****<last6>`，`refresh_token` 仅显示 `present: yes/no`。

未登录状态（tokens.yaml 不存在或 oauth2 子树空）：

```
profile:        default
authenticated:  no
source:         api_key_config  # 或其他 fallback
hint:           运行 `gate-cli cex auth login` 完成 OAuth2 授权
```

退出码：0=已认证；3=未认证（便于脚本判断）。

#### 4.8.3 `cex auth login --scope` & `--force` 组合

补充 §4.3 `login` 命令的完整 flag 矩阵：

| Flag | 说明 |
|------|------|
| `--manual` | 强制手动复制粘贴模式（§4.3.4） |
| `--force` | 忽略现有有效 token，强制重新授权（见 §4.3.1 "已登录语义"） |
| `--scope "<space-separated>"` | 覆盖默认 scope（`read spot_trade futures_trade earn asset`）。授权后实际 scope 以 server 响应为准（§4.3.5 T1.5） |
| `--profile <name>` | 指定目标 profile（默认 `default`） |

**典型组合：**

- 首次 login：`gate-cli cex auth login`
- 补齐 scope：`gate-cli cex auth login --force --scope "read spot_trade futures_trade earn asset"`（`--force` 必需，否则命中"已登录"快路径直接返回）
- headless CI 初始化：`gate-cli cex auth login --manual --profile ci`
- 切换账号：`gate-cli cex auth logout && gate-cli cex auth login`

#### 4.8.4 `insufficient_scope` 错误引导

当 gateway 返回 403 + `error: insufficient_scope`（§5.4.1 / §5.8），CLI 的 `http.RoundTripper` 在业务命令返回前统一拦截并输出：

```
✗ 权限不足: 当前 access_token 的 scope 为 "read"，该命令需要 "spot_trade"。

  补齐 scope:
    gate-cli cex auth login --force --scope "read spot_trade futures_trade earn asset"

  或切换到 API Key（若已配置）:
    gate-cli cex spot order place --api-key $KEY --api-secret $SECRET ...
```

退出码：4（专用码，区别于认证失败 2 与普通业务错误 1），便于 skill 层捕获后自动触发补齐提示。

## 5. apiv4-gateway OAuth2 改造

### 5.1 现状

| 维度 | 现状 |
|------|------|
| 技术栈 | OpenResty（Nginx + Lua） |
| 当前认证 | HMAC-SHA512（KEY + SIGN + Timestamp headers） |
| Handler 注册 | `gatekeeper_access.lua` 按 `$gate_auth` 值分发 |
| 已有 OAuth2 | `tradingview_auth.lua` — Bearer token → 外部校验，但仅用于 TradingView 路由 |

#### 5.1.1 现有代码锚点（Phase 1b-GW 改造依据）

本节列出 apiv4-gateway 中与本次改造直接相关的现有文件与行号，作为新增/修改的锚点，避免重复实现：

| 文件 | 行号 | 现有职责 | 本次复用方式 |
|------|------|---------|-------------|
| `app.d/apiv4_access.lua` | L237 | `access.validate_access(auth.auth_http)` 入口 | **改造点**：替换为 `apiv4_multi_access.validate_access()`，不改其他逻辑 |
| `app.d/access.lua` | L20-100 | 认证分发器 `validate_access()` | 复用其 header 设置框架 |
| `app.d/access.lua` | L82-91 | 设置 `X-Gate-User-Id` 等下游 header | **直接复用**：OAuth2 路径拿到 uid 后调用同一段逻辑注入 headers |
| `app.d/gatekeeper/auth.lua` | L216-350 | HMAC 校验 `auth_http()` | 保持原样，作为 §5.2 分发的 fallback 分支 |
| `app.d/gatekeeper/auth.lua` | L187-206, L252-293 | HMAC 权限映射表 | §5.8 scope→API 映射参考此结构 |
| `app.d/tradingview_auth.lua` | L298-304 | `extractToken(headers)` — 从 `Authorization: Bearer` 提取 token | **直接复用**：`oauth2_auth.lua` 通过 `require` 引用，不重写 |
| `app.d/tradingview_auth.lua` | L356-373 | `auth_token_request()` — POST 到 oauth2 服务校验 token | **参考实现**：复制其 HTTP client 超时配置（3s）和错误处理模板 |
| `app.d/key_cache.lua` | L10-14 | `mlcache` 初始化（L1 内存 + IPC） | **直接复用**：新增 `oauth2_token` 命名空间，不另起 cache 模块 |
| `app.d/config.lua` | L59-64 | 既有缓存配置（L1 1000 条/300s，L2 Redis 1800s） | 在同文件追加 OAuth2 相关字段（见 §5.6） |
| `app.d/exception.lua` | — | 统一异常响应 | 新增 `INVALID_TOKEN` / `INSUFFICIENT_SCOPE` 两个错误码分支 |

> 📌 **复用原则**：Phase 1b-GW 的新增文件（§5.5）原则上只写"OAuth2 专属逻辑"（check_endpoint 调用、scope 校验），所有 Bearer 提取、header 注入、缓存实例化、异常响应都走现有工具函数。预期新增总代码量 ≤ 260 行。

### 5.2 改造方案：多认证模式分发

```
Client Request
     │
     ▼
apiv4.routes: set $gate_auth apiv4_multi
     │
     ▼
apiv4_multi_access.lua (新增)
  ├─ Authorization: Bearer header 存在？
  │   ├─ YES → oauth2_auth.lua 验证流程
  │   │        （即使 KEY/SIGN 也同时存在，Bearer 优先，
  │   │          HMAC headers 被忽略并记日志 warn_mixed_auth）
  │   └─ NO  → KEY + SIGN headers 存在？
  │       ├─ YES → 原 auth.auth_http() HMAC 流程
  │       └─ NO  → 401 Unauthorized
```

> 📌 **Bearer 优先 + HMAC 丢弃**：CLI 侧 §12.3 子任务 8 的 RoundTripper 注入会在发请求前清空 HMAC headers，正常不应出现两种认证并存。但 gateway 仍按 **Bearer 优先** 策略防御式处理，避免老 SDK + 代理注入等边界场景下造成"KEY 绕过 OAuth2 scope 校验"的语义混淆。每次命中该分支上报 counter `oauth2_auth_mixed_header_total` 并在 5 分钟内 >10 时告警。

### 5.3 Token 验证方式

使用与 gateapi-mcp-service 相同的内部 check_endpoint（非标准 RFC 7662 introspect）。

**实际端点与实现位置**（基于 oauth2 仓库代码调查）：

| 项 | 值 |
|---|---|
| HTTP 方法 | POST |
| 路径 | `/oauth/internal/api/check` |
| 入参 header | `Gatepay-Access-Token: <token>` |
| Go 实现 | `oauth2/rest/trading_view.go:356-394` `doCheckAccessToken()` |
| 返回结构体 | `AccessTokenCheckResult`（oauth2/pkce/model.go） |
| 网络要求 | 内网路由，apiv4-gateway upstream IP 白名单 |

**返回 schema**：

```
Response 200:
{
  "code": 200,
  "data": {
    "uid": 12345678,
    "client_id": "pkce_xxxxxxxxxxxx",
    "active": true,
    "scope": "read spot_trade futures_trade earn asset",
    "expiresIn": 1718000000,     // ⚠️ Unix 时间戳（秒），非相对秒数
    "deviceId": "cli-abc123",
    "deviceName": "gate-cli/v0.3.0 darwin"
  }
}
```

> ⚠️ **字段语义澄清**：`expiresIn` 在 Go 源码中为 Unix 时间戳（绝对时间），**不是** RFC 6749 中 `expires_in` 的相对秒数。gateway 侧计算剩余 TTL 必须用 `expiresIn - now()`。若与 §4.6 OAuth2 server 改造清单中字段命名冲突，以 OAuth2 server 源码为准，并在 §4.6 标注此字段需保持向后兼容（**不要**改成相对秒数，否则 gateapi-mcp-service 会 break）。
>
> 🔗 **同源校验**：apiv4-gateway 与 gateapi-mcp-service 共享同一 `/oauth/internal/api/check` 端点（见 §4.6 改造项 #6），Phase 1b-GW 不需要 OAuth2 server 新增接口，只需开放内网白名单。

### 5.4 用户信息补全

Token check 仅返回 `uid`，但 apiv4-gateway 下游服务需要完整 `X-Gate-*` headers。
需要额外并发调用 user-center 和 gatekeeper 获取：

```
Token check → uid
并发:
├── GET user-center/user/{uid}/info → user_tier, verified, language, country
└── GET gatekeeper/internal/user/{uid}/main_uid → main_uid
组装 X-Gate-* headers → proxy_pass 到后端
```

#### 5.4.1 Fan-out 降级策略（P1 故障模型补全）

三路调用（check_endpoint / user-center / gatekeeper）中任一路失败时的处理规则，按严重程度区分：

| 失败路径 | 是否阻塞请求 | 降级行为 | 原因 |
|---------|-------------|---------|------|
| **check_endpoint 失败** | **是** | 返回 401 Unauthorized | 没有 uid 就无法确定请求者身份，**必须拒绝**，不能降级 |
| check_endpoint 返回 `active: false` | 是 | 返回 401 + `error: token_expired` | token 已失效，CLI 会触发 refresh |
| check_endpoint 返回 scope 不足 | 是 | 返回 403 + `error: insufficient_scope` | 按 §5.8 白名单校验 |
| **user-center 失败** | 否 | 优先使用 L2 shared_dict 中上次成功的缓存值（即使已 stale 超 TTL）；若完全无缓存 → 传 `X-Gate-Tier: unknown` + `X-Gate-Verified: false`，由后端按非认证用户默认等级处理 | 用户 tier 信息非安全关键字段，短暂降级不会放大权限 |
| **gatekeeper 失败** | 否 | 同上降级：stale 缓存优先 → 无缓存时传 `X-Gate-Main-Uid: <uid>`（假定当前 uid 即为主账号） | 子账号映射在 main 账号直接访问场景下退化为 noop |

**实现伪代码（`oauth2_auth.lua` 内部）**：

```lua
local uid, scope, err = check_endpoint(token)
if err or not uid then
    return ngx.exit(401)  -- 硬拒绝，不降级
end

-- 并发补全
local info, tier_err = user_center.get_info(uid)  -- with L2 stale fallback
local main_uid, main_err = gatekeeper.get_main_uid(uid)  -- with L2 stale fallback

-- 降级值
if tier_err and info == nil then
    info = { tier = "unknown", verified = false, language = "", country = "" }
    log.warn("user-center unavailable, degraded to unknown tier", uid, tier_err)
end
if main_err and main_uid == nil then
    main_uid = uid  -- 假定自己是主账号
    log.warn("gatekeeper unavailable, degraded to self as main_uid", uid, main_err)
end

-- 注入 headers
ngx.req.set_header("X-Gate-Uid", uid)
ngx.req.set_header("X-Gate-Main-Uid", main_uid)
ngx.req.set_header("X-Gate-Tier", info.tier)
-- ...
```

**监控指标**（§5 可观测性）：
- `oauth2_auth_user_center_degraded_total`（counter，降级次数）
- `oauth2_auth_gatekeeper_degraded_total`（counter，降级次数）
- 告警阈值：任一降级计数器在 5 分钟内 > 100 → 触发 oncall

§12.4 子任务 7 的验收标准追加：mock user-center 超时/500，验证降级路径正确触发、stale cache 优先、headers 正确注入。

### 5.5 新增文件

| 文件 | 内容 | 行数 |
|------|------|------|
| `app.d/oauth2_auth.lua` | Token 验证 + 用户信息补全 + 缓存 + scope 校验 | ~230 行 |
| `app.d/apiv4_multi_access.lua` | 多认证分发（Bearer → OAuth2, 否则 → HMAC） | ~30 行 |
| `app.d/oauth2_client.lua` | check_endpoint HTTP 客户端（封装 resty.http + 超时 + 错误映射） | ~80 行 |
| `app.d/scope_validator.lua` | scope → API 路径前缀匹配器（加载 §5.8 映射表） | ~60 行 |

**`oauth2_auth.lua` 模块职责与依赖图**：

```
oauth2_auth.lua
  ├── require "tradingview_auth" → extractToken()          -- L298-304 复用
  ├── require "key_cache"         → mlcache 实例            -- L10-14 复用
  ├── require "oauth2_client"     → check_endpoint POST
  ├── require "scope_validator"   → validate(scope, uri)
  ├── require "user_center"       → get_info(uid)（§5.4 fan-out，已有或新建）
  ├── require "gatekeeper_client" → get_main_uid(uid)（同上）
  └── require "access"            → set_downstream_headers()  -- L82-91 复用
```

**主流程伪代码**（`oauth2_auth.lua`）：

```lua
local M = {}

function M.validate_access()
    local token = tradingview_auth.extractToken(ngx.req.get_headers())
    if not token then
        return exception.raise("INVALID_TOKEN", "missing bearer token")
    end

    local key = "oauth2_token:" .. ngx.sha1_bin(token)  -- 简化示意
    local data, err = key_cache:get(key, nil, fetch_from_check_endpoint, token)
    if err or not data then
        return exception.raise("INVALID_TOKEN", err or "check failed")
    end

    -- scope 校验
    local ok, deny_reason = scope_validator.validate(data.scope, ngx.var.uri, ngx.req.get_method())
    if not ok then
        metrics.inc("oauth2_auth_scope_denied_total", { scope = data.scope, path = ngx.var.uri })
        return exception.raise("INSUFFICIENT_SCOPE", deny_reason)
    end

    -- fan-out 补全用户信息（§5.4）
    local info   = user_center.get_info_with_fallback(data.uid)
    local mainuid = gatekeeper_client.get_main_uid_with_fallback(data.uid)

    -- 复用 access.lua L82-91 的 header 注入
    access.set_downstream_headers(data.uid, mainuid, info)
end

function fetch_from_check_endpoint(token)
    local resp, err = oauth2_client.check(token)  -- POST /oauth/internal/api/check
    if err then return nil, err end
    if not resp.active then return nil, "token_inactive" end
    return {
        uid = resp.uid,
        scope = resp.scope,
        expires_at = resp.expiresIn,      -- Unix ts，见 §5.3 ⚠️
        client_id = resp.clientId,
    }
end

return M
```

### 5.6 修改文件

| 文件 | 修改内容 |
|------|---------|
| `app.d/gatekeeper_access.lua` | 注册 `apiv4_multi` handler (~5 行) |
| `app.d/config.lua` | 新增 `oauth2_enabled`（🆕 全局开关）、`oauth2_check_endpoint`, `user_center_url`, `gatekeeper_internal_url`, `oauth2_token_cache_ttl` |
| `nginx.conf` | 新增 `lua_shared_dict oauth2_cache_dict 10m` |
| `conf.d/apiv4.routes` | 批量替换 `set $gate_auth apiv4` → `apiv4_multi` |

#### 5.6.1 OAuth2 紧急回滚开关

Phase 1b-GW 是全量 apiv4 路由的高风险改造。若 check_endpoint 生产故障或发现 P0 bug，需要**秒级回滚到纯 HMAC**，不能依赖 git revert + reload 这种分钟级流程。

**设计：`oauth2_enabled` 配置驱动开关**

```lua
-- app.d/apiv4_multi_access.lua 开头
local config = require "config"

if not config.oauth2_enabled then
    -- 全局回滚模式：强制走 HMAC 路径
    -- Bearer 请求直接 401（强制客户端回退），不进 oauth2_auth.lua
    local auth_header = ngx.req.get_headers()["Authorization"]
    if auth_header and auth_header:find("^Bearer ") then
        ngx.status = 401
        ngx.header["WWW-Authenticate"] = 'Basic, error="oauth2_disabled"'
        ngx.say('{"error":"oauth2 temporarily disabled, fallback to HMAC"}')
        return ngx.exit(401)
    end
    -- HMAC 请求透传到原流程
    return auth.auth_http()
end

-- 正常 OAuth2 + HMAC 双路径（略）
```

**切换方式：**

| 切换路径 | 生效时间 | 适用场景 |
|---------|---------|---------|
| 修改 `config.lua` + `nginx -s reload` | ~1 秒 | 常规切换 |
| 动态配置中心（nacos / consul）下发 `oauth2_enabled=false` | 秒级 | **紧急回滚**，后续对接（当前 Phase 1b 只做静态配置） |

**切换后行为：**
- ✅ 所有 HMAC 调用方（SDK、老脚本、webhook）**完全不受影响**
- ✅ Bearer 调用方（CLI 新用户）收到 401 + `error="oauth2_disabled"` → CLI 可据此提示"OAuth2 服务临时不可用，请使用 API Key 或稍后重试"
- ⚠️ 不会自动降级 CLI 到 HMAC，因为 CLI 侧没有 HMAC 凭证（用户登录 OAuth2 后通常不保留 API Key）

§12.4 子任务 6 的验收标准追加：
- ✅ `oauth2_enabled=false` + reload 后 30 秒内 OAuth2 验证停用
- ✅ HMAC 流量吞吐无影响（对比基线 ± 5%）
- ✅ Bearer 请求返回规范的 `WWW-Authenticate` header，便于 CLI 识别

### 5.7 缓存策略

**缓存基础设施复用**：基于 `app.d/key_cache.lua:10-14` 现有 mlcache 实例，新增 `oauth2_token` 命名空间，不另起 cache 模块。L2 使用 `lua_shared_dict oauth2_cache_dict 10m`（nginx.conf 新增）。

**缓存键与值设计**：

| 层 | key | value | TTL | 说明 |
|---|-----|-------|-----|------|
| L1 | `oauth2_token:{sha256(token)}` | `{uid, scope, expires_at, client_id}` | `min(120, expires_at - now())` | 进程内 LRU，上限取 token 自身剩余寿命 |
| L2 | `oauth2_token:{sha256(token)}` | 同上（cjson 序列化） | `min(300, expires_at - now())` | `lua_shared_dict`，worker 间共享 |
| L1/L2 | `oauth2_neg:{sha256(token)}` | `"invalid"` | 10s | 负缓存，防无效 token 打穿 check_endpoint |

**Key 设计规则**：
- `SHA256(token)` 作为 key 主体，避免 token 明文落进 shared_dict（dump 时泄露风险）
- **不加 client_id 前缀**：同一 token 只可能属于一个 client_id，冲突不存在
- **加环境前缀**（`oauth2_token:` vs `oauth2_neg:`）便于运维按前缀批量清理

**TTL 上限对齐 token 寿命**：

```lua
-- 伪代码
local now = ngx.time()
local remaining = token_data.expires_at - now  -- expires_at 来自 check_endpoint 的 expiresIn
if remaining <= 0 then
    return nil, "token_expired"
end
local l1_ttl = math.min(120, remaining)
local l2_ttl = math.min(300, remaining)
```

> ✅ **缓存失效策略（已评审决议：纯被动 TTL，不做主动失效）**
>
> 本次 Phase 1b-GW **不实现**任何主动缓存失效机制。token 撤销 / scope 变更后的行为明确如下：
>
> 1. **缓存有效期内旧 token 仍可用**：gateway L1/L2 缓存 TTL 到期前,使用旧 uid/scope 继续放行请求。最长影响窗口 = 300 秒(L2 TTL)
> 2. **缓存过期后自动重新校验**：L1/L2 都未命中 → gateway 再次调用 `POST /oauth/internal/api/check`,自动拉到最新的 token 状态(active / scope / expires)
> 3. **无 Pub/Sub、无 webhook、无主动删缓存接口**：避免 gateway 与 OAuth2 server 之间额外的耦合通道,降低 Phase 1b 交付复杂度
> 4. **负缓存 `neg_ttl = 10s`**：无效 token 的短窗口,防止打穿 check_endpoint,**非**安全用途
>
> **CLI 侧补偿(强约束)**：gate-cli **必须**在 access_token 到期前至少 60 秒主动调用 refresh 接口换新 token(见 §4.5 更新),避免"CLI 本地 token 已过期但 gateway 缓存中的映射仍有效、请求发出后却被拒"的反直觉窗口。refresh 窗口由 CLI 单方面保证,与 gateway 缓存策略完全解耦。
>
> **用户侧告知**：`logout` 命令的输出追加一行提示：
> ```
> ℹ️  本地 token 已清除。注意:网关最长在 5 分钟内仍可能接受此 token,
>    如需立即失效请联系运维人员。
> ```
> 告警文案在 §4.8 logout 小节同步。

### 5.8 Scope → API 权限映射

> 📌 **权威依据**:本节 scope 清单与映射规则**严格对齐** OAuth2 项目的 scope 语义重构设计,不再作为 CLI 侧独立草案:
> - 需求文档:`oauth2/doc/oAuth2授权Scope调整需求.pdf`
> - 技术方案:`oauth2/doc/tech/oauth2-scope-semantic-refactor-design.md`(1657 行,v1.0,2026-04-13)
>
> 任何 scope 新增/重命名/归属调整,必须先在 OAuth2 项目评审通过,本节随之更新,**不得在 apiv4-gateway 侧单方面变更**。

#### 5.8.1 权威 Scope 清单(5 个平行 scope)

| Scope | 中文名 | 含义 | 强制性 | 风险等级 |
|-------|-------|------|-------|---------|
| `read` | 读取 | 所有只读权限:公开行情 + 私有查询(资产、持仓、交易历史、充值地址等) | **必选**(OAuth2 server 强制注入,用户无法取消) | 低 |
| `spot_trade` | 现货交易 | 现货、杠杆、Alpha 等非衍生品的下单/撤单/改单 | 可选 | 中 |
| `futures_trade` | 合约交易 | 永续合约、交割合约、期权、TradFi、跨所合约的下单/撤单/改单 | 可选 | 高 |
| `earn` | 理财与借贷 | 理财申购/赎回、LaunchPool 认购、所有借贷操作(margin/mcl/unified loan) | 可选 | 中 |
| `asset` | 资产与账户管理 | 划转、提现、子账号管理、统一账户配置、法币、P2P、闪兑 | 可选 | 高 |

**层级关系**:5 个 scope 完全平行,无父子继承。`read` 由 OAuth2 server 在 authorize 阶段强制注入,不论用户是否勾选。

> ⚠️ **scope 命名变更**(相对 gate-cli v2 草案):
> - `market` + 原有"私有查询"→ 合并为 `read`
> - `trade` 按品种拆分为 `spot_trade` / `futures_trade`
> - `wallet` + `account` → 合并重命名为 `asset`
> - `earn` 独立新增,承接所有借贷/理财/LaunchPool 操作
>
> 兼容性:老 token(含 `read spot_trade futures_trade earn asset profile`)由 **gateapi-mcp-service 双注册表**兼容,apiv4-gateway 新增 OAuth2 路径无需处理老 scope(仅 Phase 2 旧 HMAC 路径涉及,本次不涉及)。

#### 5.8.2 Scope → API 路径映射表

**数据源**:`oauth2-scope-semantic-refactor-design.md §4.2`(行 214-263)的 415 条工具级映射,汇总到路径前缀级别供 apiv4-gateway 使用。`scope_validator.lua` 启动时加载下表为路由 trie,按段匹配。

| Scope | API 路径前缀 | HTTP 方法 | 典型工具来源(§4.2) |
|-------|-------------|----------|-------------------|
| `read` | `/api/v4/spot/*`(查询类)、`/api/v4/futures/*/*`(查询类)、`/api/v4/delivery/*/*`(查询类)、`/api/v4/options/*`(查询类)、`/api/v4/margin/*`(查询类,含 funding_book)、`/api/v4/wallet/*`(查询类:余额/充提记录/地址)、`/api/v4/account/*`(查询类)、`/api/v4/sub_accounts`(查询类)、`/api/v4/unified/accounts`(查询)、`/api/v4/earn/*`(查询)、`/api/v4/launch/*`(查询)、`/api/v4/rebate/*`(**全部**,只查询)、`/api/v4/flash_swap/currencies`、`/api/v4/flash_swap/orders`(查询)、`/api/v4/p2p/*`(查询类)、`/api/v4/tradfi/*`(查询类)、`/api/v4/cross-exchange/*`(查询类)、`/api/v4/mcl/*`(查询类) | **GET** | §4.2 行 216/217/219/220/231/238/241 |
| `spot_trade` | `/api/v4/spot/orders`、`/api/v4/spot/batch_orders`、`/api/v4/spot/cancel_batch_orders`、`/api/v4/spot/cross_liquidate_orders`、`/api/v4/spot/countdown_cancel_all`、`/api/v4/spot/amend_batch_orders`、`/api/v4/margin/*`(**交易类,不含借贷**)、`/api/v4/margin/auto_repay` | **POST / PUT / PATCH / DELETE** | §4.2 行 218(现货)、224(杠杆扣除借贷) |
| `futures_trade` | `/api/v4/futures/*/orders`、`/api/v4/futures/*/batch_orders`、`/api/v4/futures/*/price_orders`、`/api/v4/futures/*/countdown_cancel_all`、`/api/v4/futures/*/positions/*/margin`、`/api/v4/futures/*/positions/*/leverage`、`/api/v4/futures/*/dual_comp/*`、`/api/v4/options/orders`、`/api/v4/options/countdown_cancel_all`、`/api/v4/options/mmp`、`/api/v4/delivery/*/orders`、`/api/v4/delivery/*/price_orders`、`/api/v4/delivery/*/positions/*/margin`、`/api/v4/tradfi/orders`、`/api/v4/tradfi/price_orders`、`/api/v4/cross-exchange/orders`、`/api/v4/cross-exchange/price_orders` | **POST / PUT / PATCH / DELETE** | §4.2 行 221(永续)、227(期权)、230(交割)、252(TradFi)、255(跨所) |
| `earn` | `/api/v4/earn/*`(**操作类**:申购/赎回/改单/续期)、`/api/v4/margin/uni/loans`、`/api/v4/margin/cross/loans`、`/api/v4/mcl/*`(借贷类)、`/api/v4/unified/loans`、`/api/v4/unified/portfolio_margin_calculator`、`/api/v4/launch/orders`(LaunchPool 认购) | **POST / DELETE / PUT** | §4.2 行 248(理财)、224 决策 2(杠杆借贷)、250(MCL)、235 决策 4(统一借贷)、263(LaunchPool) |
| `asset` | `/api/v4/wallet/transfers`、`/api/v4/wallet/sub_account_transfers`、`/api/v4/wallet/sub_account_to_sub_account`、`/api/v4/wallet/withdrawals`、`/api/v4/wallet/small_balance*`、`/api/v4/wallet/push`、`/api/v4/sub_accounts`(**操作类**:创建/更新 key/冻结)、`/api/v4/unified/unified_mode`、`/api/v4/unified/unified_collateral`、`/api/v4/unified/transferable`、`/api/v4/account/stp_groups`、`/api/v4/account/debit_fee`、`/api/v4/flash_swap/orders`(**创建/预览**)、`/api/v4/cross-exchange/transfers`、`/api/v4/p2p/ads/*`、`/api/v4/p2p/orders/*`、`/api/v4/pay/*` | **POST / PUT / DELETE**(flash_swap preview 为 POST) | §4.2 行 232(钱包划转)、234(统一账户配置)、237(子账号)、240(账户配置)、245 决策 7(闪兑全线)、254(跨所划转)、258(P2P)、259(Pay) |

#### 5.8.3 匹配规则

1. **路径段匹配**:请求 URI 按 `/` 分段,与 scope 前缀 trie 逐段匹配;`*` 代表单段通配(如 `/api/v4/futures/usdt/orders` 匹配 `/api/v4/futures/*/orders`)
2. **方法敏感**:同一路径前缀在不同 HTTP 方法下映射到不同 scope(如 `/api/v4/earn/dual/orders` GET→`read`,POST→`earn`)
3. **多 scope 命中取并集**:请求 token scope 与该路径所需 scope 有任一交集即放行(OR 匹配,等价于 `toolScopeMap` 的 `HasAnyScope` 语义)
4. **read 必选放行**:由于 OAuth2 server 强制注入 `read`,所有 GET 查询请求**永远可通过**(除非 token 本身无效);scope_validator 对 GET 请求可快路径放行(命中 `read` 立即通过)
5. **fail-closed**:未在映射表中的路径 → 403 `insufficient_scope` + 上报 `oauth2_auth_scope_unmapped_total{path=...}` 指标,由运维跟进归类
6. **不做 scope 翻译**:apiv4-gateway 从 check_endpoint 拿到什么 scope 字符串就用什么,**不翻译老 scope 到新 scope**(与 OAuth2 server `CheckToken` 纯透传契约对齐,见 refactor design §7.12)

#### 5.8.4 实现约束(scope_validator.lua)

```lua
-- 加载映射表(启动时从 config.oauth2.scope_map 读取)
local SCOPE_MAP = {
    -- { pattern = "/api/v4/spot/orders",      methods = {POST=true,DELETE=true}, scopes = {"spot_trade"} },
    -- { pattern = "/api/v4/futures/*/orders", methods = {POST=true,DELETE=true}, scopes = {"futures_trade"} },
    -- { pattern = "/api/v4/earn/*",           methods = {GET=true},               scopes = {"read"} },
    -- { pattern = "/api/v4/earn/*",           methods = {POST=true,DELETE=true}, scopes = {"earn"} },
    -- ...
}

function M.validate(token_scopes, uri, method)
    -- token_scopes 是字符串(空格分隔),先切成 set
    local granted = parse_scope_set(token_scopes)

    -- 快路径:GET 请求且 token 含 read → 直接放行(read 必选)
    if method == "GET" and granted["read"] then
        return true
    end

    -- 精确路径 + 方法匹配
    local required = lookup_trie(SCOPE_MAP, uri, method)
    if not required then
        metrics.inc("oauth2_auth_scope_unmapped_total", { path = uri, method = method })
        return false, "path_unmapped"
    end

    -- OR 匹配:token scope 与 required scope 有任一交集即放行
    for _, s in ipairs(required) do
        if granted[s] then return true end
    end

    metrics.inc("oauth2_auth_scope_denied_total", { required = table.concat(required, ","), path = uri })
    return false, "insufficient_scope"
end
```

**关键约束**:
- **不做前缀继承**:`earn` 不继承 `read`,由 OAuth2 server 强制注入 `read` 保证 GET 可通
- **不做 scope 翻译**:老 token 的 `market/trade/wallet/account` 在本次 OAuth2 Bearer 路径下**不兼容**;老 client 继续用 HMAC 路径,不走 §5 改造
- **映射表热加载**:scope map 存在 nginx shared_dict,reload 时从 config 重新加载,无需重启 worker
- **兼容 OAuth2 server 字段冻结**:`check_endpoint` 返回的 `scope` 字段为空格分隔字符串(与 §5.3 和 §4.6 改造项 #6 的 schema 冻结对齐)

### 5.9 可观测性与监控

新增 OAuth2 验证链路的指标与日志，纳入 apiv4-gateway 现有 Prometheus / VictoriaMetrics 采集：

**核心指标**（`oauth2_auth.lua` 内部上报）：

| 指标名 | 类型 | 标签 | 告警阈值 |
|-------|------|------|---------|
| `oauth2_auth_check_endpoint_total` | counter | `result={success,expired,invalid,network_error}` | `network_error` 5 分钟 > 50 触发 oncall |
| `oauth2_auth_check_endpoint_duration_seconds` | histogram | - | P99 > 50ms 预警 |
| `oauth2_auth_cache_hit_ratio` | gauge | `layer={L1,L2}` | L1 命中率 < 70% 预警（缓存效率低） |
| `oauth2_auth_user_center_degraded_total` | counter | - | 5 分钟 > 100 触发 oncall |
| `oauth2_auth_gatekeeper_degraded_total` | counter | - | 同上 |
| `oauth2_auth_scope_denied_total` | counter | `scope=<scope>`,`path=<api_prefix>` | - |
| `oauth2_auth_disabled_bypass_total` | counter | - | `oauth2_enabled=false` 期间 Bearer 请求数 |

**关键日志**（结构化 JSON，接入现有日志收集）：
- `level=warn` + `event=check_endpoint_failed` + `uid` + `trace_id` → 链路追踪排障
- `level=warn` + `event=user_center_degraded` + `uid` → 降级事件
- `level=info` + `event=oauth2_disabled_mode` → 回滚开关切换

**CLI 侧日志**（`internal/auth/*`）：

| 日志路径 | 内容 |
|---------|------|
| `~/.gate-cli/logs/auth.log`（0600） | `login` / `refresh` / `logout` 事件，仅记录时间、uid、scope，**绝不记录 token 明文** |
| 通过 `--debug` 显式开启 | 原始 HTTP 请求/响应（token 按统一规则 redact，见下方） |

**Token redact 规则（统一规范）**：所有日志 / 错误消息 / stderr / `status` 输出 / `--debug` dump 必须走 `redact.Token(tok)` 工具函数，规则：

| 输入长度 | 输出 |
|---------|------|
| ≥ 12 字符 | `<前缀>****<后 6 位>`，如 `pkce_at_****abc123` |
| < 12 字符 | `****`（整体遮罩） |
| 空 | `<none>` |

适用对象：`access_token` / `refresh_token` / `code` / `code_verifier` / `client_secret`（若将来引入）。`client_id` 可明文输出（非秘密）。

日志路径遵循 XDG：优先 `$XDG_STATE_HOME/gate-cli/`，否则 `~/.gate-cli/logs/`。

### 5.10 限流与 DoS 防护（OAuth2 server 侧）

§4.6 改造清单追加限流要求：

| 端点 | 限流策略 | 理由 |
|-----|---------|------|
| `POST /mcp/oauth/register` | 按源 IP，10 次/分钟 | DCR 无认证，防注册滥用 |
| `POST /mcp/oauth/token` (grant_type=authorization_code) | 按 client_id，30 次/分钟 | 防 code 暴力枚举 |
| `POST /mcp/oauth/token` (grant_type=refresh_token) | 按 client_id，60 次/分钟 | 正常用户不会这么频繁 |
| `GET /mcp/oauth/authorize` | 按源 IP + client_id，30 次/分钟 | 防恶意跳转 |
| `GET /mcp/oauth/cli-callback` | 按源 IP，100 次/分钟 | 展示页只读静态 HTML，限流较宽 |

超限行为：返回 429 + `Retry-After` header。建议在 OAuth2 server 的入口层（Nginx / Envoy）实现，不占用业务逻辑资源。

## 6. Info / News 模块实现

### 6.1 后端调用方式

采用内嵌轻量 MCP Client，按需调用单个 tool（非全量加载 schema）：

- Info: 连接 `https://api.gatemcp.ai/mcp/info`
- News: 连接 `https://api.gatemcp.ai/mcp/news`
- 每次仅发送 `tools/call` 请求，不预加载 `tools/list`

选择理由：
1. 完全兼容现有 skill 的 tool 参数和返回格式，skill 改造零成本
2. 不需要逆向 Info/News 后端 REST API
3. 14 个 tools 数量少，MCP 协议开销可忽略

### 6.2 Info MCP Tools 全量清单

| CLI 命令 | MCP Tool | 参数 |
|---------|---------|------|
| `info coin search` | `info_coin_search_coins` | `query` |
| `info coin get` | `info_coin_get_coin_info` | `query`, `scope` |
| `info market overview` | `info_marketsnapshot_get_market_overview` | (无) |
| `info market snapshot` | `info_marketsnapshot_get_market_snapshot` | `symbol`, `timeframe`, `source` (必须 "spot") |
| `info trend kline` | `info_markettrend_get_kline` | `symbol`, `timeframe`, `limit`, `source` (必须 "spot") |
| `info trend indicators` | `info_markettrend_get_indicator_history` | `symbol`, `indicators`, `timeframe` |
| `info trend analysis` | `info_markettrend_get_technical_analysis` | `symbol` |
| `info defi overview` | `info_platformmetrics_get_defi_overview` | `category` |
| `info macro summary` | `info_macro_get_macro_summary` | (无) |
| `info onchain token` | `info_onchain_get_token_onchain` | `token`, `chain`, `scope` |
| `info compliance check` | `info_compliance_check_token_security` | `token`, `chain` |

### 6.3 News MCP Tools 全量清单

| CLI 命令 | MCP Tool | 参数 |
|---------|---------|------|
| `news events latest` | `news_events_get_latest_events` | `coin`, `time_range`, `limit` |
| `news events detail` | `news_events_get_event_detail` | `event_id` |
| `news feed search` | `news_feed_search_news` | `coin`, `sort_by`, `limit`, `lang` |
| `news feed sentiment` | `news_feed_get_social_sentiment` | `coin` |

## 7. call 兜底 + schema 按需

### 7.1 call 命令

```bash
gate-cli call <tool_name> --params '<json>'
```

内部路由：
| tool_name 前缀 | 后端 | 调用方式 |
|---------------|------|---------|
| `cex_*` | api.gateio.ws/api/v4 | Gate SDK（需认证） |
| `info_*` | api.gatemcp.ai/mcp/info | MCP Client |
| `news_*` | api.gatemcp.ai/mcp/news | MCP Client |

CEX 的 `call` 兜底通过嵌入 `interface.json`（来自 gateapi-mcp-service）实现 tool_name → SDK method 映射。

**双入口兼容（§3.1.2 缩写处理的对外出口）：**

用户从 MCP 生态迁移过来，手里可能直接拿着含缩写的 MCP tool 名；同时 CLI 内部用 SDK operationId 做命令生成。`gate-cli call` 接受两种形式，内部归一：

```bash
# 形态 A：MCP tool 名（含缩写，与 gateapi-mcp-service / gate-local-mcp 一字不差）
gate-cli call cex_fx_list_fx_positions --params '{"settle":"usdt"}'
gate-cli call cex_sa_list_sas
gate-cli call cex_dc_list_dc_contracts --settle usdt

# 形态 B：SDK operationId 的 snake_case（去掉 cex_ 前缀、完整单词）
gate-cli call list_futures_positions --params '{"settle":"usdt"}'
gate-cli call list_sub_accounts
gate-cli call list_delivery_contracts --settle usdt

# 两者等价，解析层走 interface.json 的双向索引（mcp_tool_name_server / mcp_tool_name_local / operation_id → 同一条 operation）
```

**解析顺序**：
1. 精确匹配 `mcp_tool_name_server`（含服务端 5 条缩写）
2. 精确匹配 `mcp_tool_name_local`（含客户端 8 条缩写）
3. 精确匹配 `operation_id` 的 snake_case 形式
4. 均未命中 → 报错 `unknown tool: <name>`，附带最近似候选（Levenshtein 距离 ≤ 3 的前 5 项）

**不做的事**：CLI 不做任何"缩写 ↔ 原词"的正则或字符串运算。所有匹配都是 `interface.json` 表查。这避免了 §3.1.2 提到的 `sas → sub_accounts` 歧义。

**`--params` 与扁平 flag 覆盖优先级（PRD §6.1 对齐）：**

同一个调用允许混用 `--params` JSON 与扁平 flag 两种参数形态。当同名字段并存时，**扁平 flag 严格覆盖 `--params` JSON 中的同名字段**：

```bash
# 示例：--params 指定 pair=BTC_USDT，但扁平 flag --pair=ETH_USDT 生效
gate-cli call cex_spot_list_tickers \
  --params '{"pair":"BTC_USDT","limit":10}' \
  --pair ETH_USDT
# 等价于 --params '{"pair":"ETH_USDT","limit":10}'
```

覆盖规则：

| 场景 | 结果 |
|------|------|
| 字段仅在 `--params` 中 | 生效（如上例的 `limit`） |
| 字段仅在扁平 flag 中 | 生效 |
| 字段两处均有 | **扁平 flag 覆盖**，`--params` 同名字段被丢弃 |
| 字段两处类型不一致（如 flag 是 string，JSON 是 int） | 扁平 flag 按 Cobra 声明类型解析，覆盖后以 flag 类型为准；`--verbose` 下打印 warning |

实现位置：`internal/cmdutil/params.go` 的 `MergeParams(flagValues, jsonParams) map[string]any`，顺序：先解析 `--params` → 再用非零 flag 覆盖。

### 7.2 schema 命令

**规范形态（PRD §6.2 要求二选一固定）：**

v3.4 明确固定为 **Cobra `--help` 为主 + `gate-cli schema <tool>` 兜底** 的双入口：

```bash
# 主形态（推荐）：每个子命令原生 --help，零实现成本
gate-cli cex spot list-tickers --help
gate-cli info coin search --help

# 兜底形态：只知 MCP Tool 名、不知 kebab 命令时使用
gate-cli schema cex_spot_list_tickers
gate-cli schema info_coin_search_coins
```

两种形态返回等价内容（参数列表 + 类型 + 描述 + 示例），仅入口不同：

| 入口 | 适用场景 | 数据来源 |
|------|---------|---------|
| `<cmd> --help` | 已知命令，交互式查询 | Cobra 命令元数据（编译期嵌入） |
| `gate-cli schema <tool>` | 只知 MCP Tool 名，或脚本化批量查询 | `interface.json`（CEX） + MCP `tools/list` 按需拉取（Info/News） |

**兜底形态的路由逻辑**：`gate-cli schema <tool>` 内部先按 §3.1.1 规则推导出 CLI 命令路径，再 delegate 到 Cobra 的 `--help` 渲染，确保输出一致。

**实现要点：**
- CEX schema 通过嵌入的 `interface.json` 生成 Cobra flag 描述（编译期代码生成）。
- Info/News 的 schema 首次调用时 lazy fetch MCP `tools/list` 缓存到 `~/.gate-cli/cache/schemas/<tool>.json`，TTL 24h。
- `--help` 和 `schema` 均不需要认证（读取本地元数据 + 可选 MCP 公开端点）。

## 8. 兼容模式与迁移治理

### 8.1 状态矩阵

| gate-cli 安装 | CEX MCP 存在 | Info/News MCP 存在 | 执行面 | 处理 |
|-------------|------------|-----------------|--------|------|
| ✅ | ❌ | ❌ | CLI | 目标态 |
| ✅ | ✅ | ✅ | CLI | 提示 `gate-cli migrate` |
| ❌ | ✅ | ✅ | MCP fallback | 提示安装 gate-cli |
| ❌ | ❌ | ❌ | 终止 | 提示安装 gate-cli |

#### 8.1.1 兼容模式状态机（PRD §4.5 对齐）

当用户触发任意 CEX 相关 Skill 时，Skill 的 preflight 节点按下列状态机决策执行路径：

```
              ┌─────────────────────────┐
              │ 用户触发 CEX 相关 Skill │
              └────────────┬────────────┘
                           │
                  ┌────────▼────────┐
                  │ Skill Preflight │
                  │    Warning      │
                  └────────┬────────┘
                           │
              ┌────────────▼────────────┐
              │ 本地是否存在 gate-cli ? │
              └──────┬───────────┬──────┘
                 是  │           │  否
                     │           │
        ┌────────────▼──┐    ┌───▼────────────────────┐
        │ 环境 / 版本   │    │ 本地是否已有 CEX 相关  │
        │ 是否满足 ?    │    │ Gate MCP ?             │
        └───┬───────┬───┘    └────┬───────────────┬───┘
         是 │    否 │            是│             否│
            │       │              │              │
            │       ▼              ▼              ▼
            │  ┌────────────┐  ┌────────────┐  ┌────────────┐
            │  │ 提示运行   │  │ 走 MCP     │  │ 终止执行   │
            │  │ gate-cli   │  │ fallback + │  │ + 提示安装 │
            │  │ doctor     │  │ 提示安装   │  │ gate-cli   │
            │  └────────────┘  │ gate-cli   │  └────────────┘
            │                  └────────────┘
            │
            ▼
   ┌───────────────────────────────┐
   │ 是否检测到旧 CEX MCP 残留 ?   │
   └──────┬─────────────────┬──────┘
       是 │              否 │
          ▼                 ▼
   ┌──────────────┐   ┌──────────────┐
   │ 走 CLI 主路径│   │ 走 CLI 主路径│
   │ + 提示运行   │   │   (目标态)   │
   │ gate-cli     │   └──────────────┘
   │ migrate      │
   └──────────────┘
```

**状态枚举**（供 skill 侧调用 `gate-cli doctor --format json` 后分支判断，与 PRD §4.5 节点文案对齐）：

| 状态 | 含义 | Skill 行为 |
|------|------|-----------|
| `cli_ok_clean` | CLI 已安装且无旧 MCP 残留 | 直接走 CLI 主路径 |
| `cli_ok_migrate_pending` | CLI 已安装但存在旧 CEX MCP 条目 | 走 CLI 主路径 + 一次性提示 `gate-cli migrate` |
| `cli_env_unsatisfied` | CLI 安装但版本/依赖/权限不满足 | 提示运行 `gate-cli doctor` |
| `mcp_fallback` | CLI 未安装但旧 CEX MCP 仍可用 | 走 MCP fallback + 显著提示安装 `gate-cli` |
| `unavailable` | CLI 与 MCP 均不可用 | 终止执行 + 提示安装 `gate-cli`，**严禁静默失败** |

### 8.2 doctor 检查项

```
gate-cli doctor

✅ gate-cli version: 0.5.0
✅ CEX 认证: API Key 已配置 (profile: default)
✅ CEX OAuth2: access_token 有效 (expires: 2026-04-12T10:00:00Z)
✅ 连通性: api.gateio.ws — 正常
✅ 连通性: api.gatemcp.ai/mcp/info — 正常
✅ 连通性: api.gatemcp.ai/mcp/news — 正常
⚠️ 旧 MCP 残留: gate-cex-pub (在 ~/.claude.json 中)
   → 运行 gate-cli migrate 清理
```

### 8.3 migrate 行为

1. 扫描 `~/.claude.json`、`~/.cursor/mcp.json` 中的 Gate MCP 配置
2. 备份原配置到 `<原路径>.bak-v2-<timestamp>`
3. 注释掉**下述清单**中列出的 MCP server 条目（**不删除**，保留用户显式回滚能力）
4. 输出迁移报告

**migrate 扫描的 Gate MCP server_name 清单（PRD §4.2 对齐）：**

| 类别 | server_name | 备注 |
|------|------------|------|
| CEX 公共行情 | `gate-cex-pub` | PRD §4.2 明确列出 |
| CEX 交易 | `gate-cex-ex` | PRD §4.2 明确列出 |
| CEX 本地全量 | `gate-local-mcp` | 396 tools 的本地 MCP |
| CEX 平台代理 | `gateapi-mcp-service` | 仅当作为本地 MCP server 注册时 |
| Info | `gate-info` | 10 tools |
| News | `gate-news` | 4 tools |
| 历史别名 | `gate-mcp`, `gate-cex`, `gate-cex-spot`, `gate-cex-futures` | 各历史版本残留名，按前缀 `gate-cex*` + 精确名匹配 |

**匹配规则：**

- **精确匹配**：上表中所有非 `gate-cex*` 的 server_name 按完整字符串匹配。
- **前缀匹配**：`gate-cex*`（含 `gate-cex-pub` / `gate-cex-ex` / `gate-cex-spot` 等）按前缀命中，覆盖未来可能新增的子类。
- **白名单兜底**：匹配清单以 `internal/migrate/mcp_servers.go` 的 `KnownCEXServers` / `KnownInfoServers` / `KnownNewsServers` 常量管理，便于后续新增。

**交互策略：**

- 默认 **dry-run**：输出将被注释的条目列表，用户二次确认（输入 `yes`）后真正写盘。
- `--yes` flag：跳过确认（供 CI / 自动化脚本）。
- `--server <name>`：仅处理指定 server_name，其余保留（供用户精细控制）。
- 遇到清单外的 `gate-*` 前缀 MCP server：不改动，输出 warning 提示"未识别的 Gate 类 MCP server: xxx，如需迁移请手动处理"。

### 8.4 旧 config 命令删除（硬切换）

顶层 `gate-cli config` 命令**直接删除**，不保留转发。首次运行 gate-cli 时，若检测到 `~/.gate-cli/config.yaml` 存在且非空，启动时打印一次性迁移提示：

```
⚠️  检测到旧版配置文件 ~/.gate-cli/config.yaml
    旧的顶层 `gate-cli config` 命令已在 v3 中删除，请使用 `gate-cli cex config` 管理凭证。
    运行 `gate-cli migrate` 可自动迁移配置文件格式。
```

`gate-cli migrate` 会原地读取旧 config 并按新 schema 重写到同一路径，不破坏用户现有 profile 名称。

### 8.5 v2 → v3 数据迁移路径

两个文件的职责分离 + 迁移步骤：

| 文件 | v2 | v3 | 迁移动作 |
|-----|----|----|---------|
| `~/.gate-cli/config.yaml` | 存 API Key / Secret / Base URL（顶层 `config` 命令管理） | 仅存 API Key 等 HMAC 凭证（由 `cex config` 管理，顶层 `config` 删除） | schema 字段向前兼容，无需改动，仅权限收紧到 0600 |
| `~/.gate-cli/tokens.yaml` 🆕 | 不存在 | 存 OAuth2 token（见 §4.4 嵌套 schema） | 新建，0600 权限 |

**迁移命令行为**（`gate-cli migrate`）：

```
1. 检测 ~/.gate-cli/config.yaml 存在？
   - 是 → 备份到 ~/.gate-cli/config.yaml.bak-v2
   - 否 → 跳过
2. 检测权限
   - 非 0600 → chmod 0600 ~/.gate-cli/config.yaml
   - 父目录非 0700 → chmod 0700 ~/.gate-cli/
3. schema 检查
   - v2 schema（已兼容 v3）→ 无需改动
   - 含任何废弃字段 → 清理并告知
4. MCP 配置清理（原 §8.3 行为）
   - 扫描 ~/.claude.json / ~/.cursor/mcp.json
   - 注释掉 CEX/Info/News MCP server 条目
5. Skill 扫描（可选 --scan-skills）
   - 扫描指定目录所有 references/cli.md
   - 输出 gate-cli <module> → gate-cli cex <module> 的 diff
6. 输出迁移报告
```

**schema 兼容性保证**：v3 的 `config.yaml` 结构字段**严格是 v2 的超集**，不删字段、不改字段名，只新增可选字段。用户不运行 migrate 也能正常使用 v3（仅权限提示 + 顶层命令 unknown），不会丢失凭证。

### 8.6 preflight / doctor / migrate 职责对照（PRD §4.6 对齐）

三个命令 / 机制共同构成迁移治理体系，职责与输出严格互补，不得重叠：

| 维度 | Skill Preflight | `gate-cli doctor` | `gate-cli migrate` |
|------|----------------|-------------------|---------------------|
| **触发时机** | 每次 Skill 执行前（隐式） | 用户主动运行 / doctor 报错时 | 用户主动运行，一次性 |
| **运行位置** | Skill 侧（SKILL.md 的顶部检查块） | CLI 子命令 | CLI 子命令 |
| **主要职责** | 决定本次执行走 CLI / MCP fallback / 终止 | 体检：版本 / 认证 / 网络 / 配置权限 | 改写配置 + 注释旧 MCP + 扫描 skill |
| **是否写盘** | 否（只读） | 否（只读） | **是**（带备份） |
| **输出形态** | 文案提示（中文） | 分类健康检查报告 | dry-run diff → 确认后写盘 |
| **状态枚举** | §8.1.1 五种 | `ok` / `warn` / `fail` + 具体检查项 | `migrated` / `skipped` / `backup_path` |
| **依赖关系** | 依赖 doctor 的状态（可调用 `doctor --format json`） | 独立 | 独立；可被 doctor 提示触发 |

**统一退出码（所有 CLI 命令遵循，PRD §7 + §8 对齐）：**

| 退出码 | 语义 | 使用场景 |
|-------|------|---------|
| `0` | 成功 | 命令正常完成 |
| `1` | 通用错误 / 业务失败 | API 返回错误、参数校验失败、网络错误、migrate 部分失败 |
| `2` | 认证失败 | OAuth2 token 无效、refresh_token 过期、`login` 中途取消（§4.5.3） |
| `3` | 未认证 | `cex auth status` 判定未登录（§4.8.2），便于脚本判断 |
| `4` | 权限不足 | `insufficient_scope`（§4.8.4），scope 不足无法调用 |
| `5` | 环境不满足 | `doctor` 发现依赖缺失 / 版本过低 / 权限不安全（`tokens.yaml` 非 0600 等）；`cex auth login` DCR 本地回调端口（18991-18995）全部被占用 |
| `6` | 配置冲突 | `migrate` dry-run 检测到用户已有冲突自定义配置，需手动介入 |
| `130` | 用户中断 | Ctrl+C / SIGINT（Go runtime 默认） |

> 📌 **退出码使用约束**：业务命令统一只用 0/1/2/4/130；`cex auth status` 额外用 3；`doctor` 与 `cex auth login`（DCR 端口耗尽场景）共用 5；`migrate` 额外用 6。其他退出码保留。Skill 侧可据此做机械化分支判断，无需解析 stderr 文案。

**三者协同示例**：

```
用户触发 "gate-exchange-spot" skill
  ↓
Skill preflight: 检测到 gate-cli 存在但状态不明
  ↓
  调用 `gate-cli doctor --format json`  → 退出码 5，报告 tokens.yaml 权限不对
  ↓
  Skill 打印："环境检查未通过，请运行 gate-cli doctor 查看详情"，终止本次执行
  ↓
用户手动运行 `gate-cli doctor`
  ↓
  doctor 提示 "检测到旧 CEX MCP 残留 gate-cex-pub，建议运行 gate-cli migrate"
  ↓
用户运行 `gate-cli migrate`
  ↓
  migrate dry-run → 确认 → 写盘 → 退出码 0
  ↓
用户重新触发 skill → preflight 通过 → 走 CLI 主路径
```

### 8.7 兼容模式验收要点（PRD §4.7 + §8 对齐）

将 PRD §4.7 的四条要点与 PRD §8 的验收条目合并为本方案的统一验收清单，测试与上线前逐项走查：

| # | 验收项 | 对应章节 | 测试方式 |
|---|-------|---------|---------|
| 1 | CLI 已安装且可用时，Skill **必须**优先走 CLI，不得优先走 MCP | §8.1.1 状态 `cli_ok_clean` / `cli_ok_migrate_pending` | Skill preflight 单元测试 + 端到端回归 |
| 2 | CLI 未安装但旧 CEX MCP 仍存在时，允许 MCP fallback，**且须有明确用户提示** | §8.1.1 状态 `mcp_fallback` | Skill 卸载 gate-cli 后运行，检查提示文案 |
| 3 | CLI 与 MCP 均不存在时，**禁止无提示硬失败** | §8.1.1 状态 `unavailable` | 在干净环境运行，必须看到安装提示，退出码非 0 |
| 4 | `doctor` 与 `migrate` 为**独立命令**，不内嵌在单个业务 Skill 实现里 | §3.1 命令树 + §8.2 / §8.3 | `gate-cli doctor` / `gate-cli migrate` 可独立执行 |
| 5 | §3.4 的 12 个 MCP Tool 均存在可推导的 `gate-cli cex ...` 对应命令 | §3.1.1 + §3.4 | `TestToolNameToCLI_Fixtures` 单测 |
| 6 | `gate-cli call <exact_mcp_tool_name>` 可覆盖上述全部 Tool | §7.1 | mock `interface.json` 的端到端测试 |
| 7 | `schema` / `--help` 可单独获取任一命令入参，**无需预加载全部 MCP Schema** | §7.2 | 启动时 profiling 验证无 schema 批量加载 |
| 8 | migrate 扫描的 GateMCP 条目包含 CEX（`gate-cex-pub`/`gate-cex-ex` 等） | §8.3 | 构造 mock `~/.claude.json` 验证命中 |
| 9 | 未重新定义 Skill 编排，未引入 Shortcut / alias 专章 | §1.3 非目标 | 文档复审（非代码验收） |
| 10 | 所有 CLI 命令遵循 §8.6 统一退出码表 | §8.6 | `exit_code_test.go` 覆盖 0/1/2/3/4/5/6/130 |

## 9. Skill 迁移

### 9.1 迁移原则

**Skill 边界不等于单个 module（PRD §1.2 / §2 对齐）**：

一个 Skill 为完成用户意图，**允许按业务需要组合多条 CLI 命令或多次 `call` 兜底**，不要求与 `gate-cli cex <module>` 的模块划分一一对应。例如：

- `gate-exchange-trading` 可同时调用 `cex spot create-order` + `cex futures list-positions` + `cex wallet get-total-balance`
- `gate-info-coinanalysis` 可串联 `info coin get` + `info trend analysis` + `info onchain token` + `news feed sentiment`
- 某个 MCP Tool 尚未生成 kebab 子命令的过渡期，Skill 可临时走 `gate-cli call <tool> --params '...'`，后续不需改动 Skill

Skill 负责意图识别与结果组织，CLI 负责单次动作的参数与输出契约，两层不重叠。

**命令行迁移机械规则**：每个 Skill 的 `references/mcp.md` 替换为 `references/cli.md`：

```markdown
# 迁移前（MCP 调用）
调用 info_marketsnapshot_get_market_snapshot(symbol="BTC", timeframe="1d", source="spot")

# 迁移后（CLI 调用）
Bash: gate-cli info market snapshot --symbol BTC --timeframe 1d --source spot --format json
```

### 9.2 Skill Preflight 检查

每个 Skill 的 SKILL.md 顶部添加运行时检查：

```
1. which gate-cli → 确认 CLI 已安装
2. gate-cli version → 版本 >= 0.5.0
3. 需认证操作：gate-cli cex auth status → 确认认证状态
4. CLI 不可用：检查 MCP fallback
5. 都不可用：提示安装 gate-cli
```

### 9.3 迁移优先级

⚠️ **所有 CEX skill 的命令示例必须从 `gate-cli <module>` 改为 `gate-cli cex <module>` 形式**（硬切换，无 alias）。这是 Phase 4 的强制改造项。

| 阶段 | Skills | 命令依赖 | 改造内容 | 工作量 |
|------|--------|---------|---------|--------|
| P0 | gate-exchange-spot, futures, assets (3) | cex spot / cex futures / cex wallet | 批量替换 `gate-cli spot` → `gate-cli cex spot` 等 | 低 |
| P1 | gate-info-* (11), gate-news-* (4) | info / news（顶层不变） | 仅命令输出格式微调 | 中 |
| P2 | 剩余 CEX skills（earn, unified, margin 等 ~12） | cex earn / cex unified / cex margin | 同 P0 批量替换 | 低 |
| P3 | gate-exchange-trading, marketanalysis 等 (~10) | cex <sub> + call | 同 P0 + call 兜底处理 | 低 |

**批量改造建议**：
- 使用 `sed -i 's|gate-cli spot |gate-cli cex spot |g'` 等脚本对每个 skill 的 `references/cli.md` 做模块级替换
- `gate-cli migrate --scan-skills` 子命令（§8）可扫描指定目录下所有 skill md 文件，输出迁移前后 diff

### 9.4 Skill 中的 CLI 调用方式（跨平台）

Skill 的 `references/cli.md` 中，CLI 调用指令统一写为 shell 命令，不绑定特定平台 tool 名称：

```markdown
## 执行

运行以下命令获取市场快照：
gate-cli info market snapshot --symbol BTC --timeframe 1d --source spot --format json

各平台 Agent 使用对应的 shell 执行工具调用上述命令：
- Claude Code: Bash tool
- Cursor: Terminal / run_command
- OpenClaw: exec tool
```

Skill 不需要为不同平台编写不同版本——命令本身是平台无关的，只有调用 shell 的工具名称不同。

## 10. Token 节省量化

### 当前消耗

| MCP Server | Tool 数量 | Token/tool | 总消耗 |
|------------|----------|-----------|--------|
| gate-local-mcp (CEX) | 396 | ~200 | ~79,200 |
| gate-info | 10 | ~250 | ~2,500 |
| gate-news | 4 | ~250 | ~1,000 |
| **合计** | **410** | | **~82,700** |

### CLI-first 后

| 场景 | 消耗 |
|------|------|
| 固定加载成本 | **0**（无 tool schema 注入） |
| 单次 CLI 调用（命令 + 输出） | ~300-1,500 |

### 节省计算

```
假设单次会话 5 次 skill 执行：
  原始: 82,700 (固定) + 5 × 500 (执行) = 85,200 tokens
  现在: 0 (固定) + 5 × 1,500 (执行) = 7,500 tokens
  节省: 91.2%

保守场景（1 次 skill 执行）：
  原始: 82,700 + 500 = 83,200
  现在: 0 + 1,500 = 1,500
  节省: 98.2%
```

## 11. 项目结构变更

```
gate-cli/
├── cmd/
│   ├── root.go                  # 根命令（仅挂载顶层模块 + call/schema/doctor/migrate/version）
│   ├── cex/                     # 🔧 CEX 域（重构：所有现有 CEX 子命令迁入此目录）
│   │   ├── cex.go              #   cex 根命令（Cobra Command）
│   │   ├── auth/                #   🆕 OAuth2 认证
│   │   │   ├── auth.go         #     auth 子命令根
│   │   │   ├── login.go        #     login [--manual]
│   │   │   ├── status.go       #     status
│   │   │   └── logout.go       #     logout
│   │   ├── config/              #   🆕 凭证配置（原顶层 config 迁入）
│   │   │   ├── config.go
│   │   │   ├── init.go
│   │   │   ├── set.go
│   │   │   └── list.go
│   │   ├── spot/                #   ⬇️ 原顶层 cmd/spot/ 整体迁入
│   │   ├── futures/             #   ⬇️ 原顶层 cmd/futures/
│   │   ├── delivery/            #   ⬇️ 原顶层 cmd/delivery/
│   │   ├── options/             #   ⬇️ 原顶层 cmd/options/
│   │   ├── margin/              #   ⬇️ 原顶层 cmd/margin/
│   │   ├── unified/             #   ⬇️ 原顶层 cmd/unified/
│   │   ├── wallet/              #   ⬇️ 原顶层 cmd/wallet/
│   │   ├── account/             #   ⬇️ 原顶层 cmd/account/
│   │   ├── subaccount/          #   ⬇️ 原顶层 cmd/sub_account/
│   │   ├── earn/                #   ⬇️ 原顶层 cmd/earn/
│   │   ├── rebate/              #   ⬇️ 原顶层 cmd/rebate/
│   │   ├── flashswap/           #   ⬇️ 原顶层 cmd/flash_swap/
│   │   ├── crossex/             #   ⬇️ 原顶层 cmd/cross_ex/
│   │   ├── alpha/               #   ⬇️ 原顶层 cmd/alpha/
│   │   ├── tradfi/              #   ⬇️ 原顶层 cmd/tradfi/
│   │   └── p2p/                 #   ⬇️ 原顶层 cmd/p2p/
│   ├── dex/                     # 🆕 DEX 域占位（future，当前仅注册空 root 命令 + "not implemented" 提示）
│   │   └── dex.go
│   ├── info/                    # 🆕 Info 模块
│   │   ├── info.go
│   │   ├── coin.go
│   │   ├── market.go
│   │   ├── trend.go
│   │   ├── defi.go
│   │   ├── macro.go
│   │   ├── onchain.go
│   │   └── compliance.go
│   ├── news/                    # 🆕 News 模块
│   │   ├── news.go
│   │   ├── events.go
│   │   └── feed.go
│   ├── call/                    # 🆕 call 兜底
│   │   └── call.go
│   ├── schema/                  # 🆕 schema 查询
│   │   └── schema.go
│   ├── doctor/                  # 🆕 环境检查
│   │   └── doctor.go
│   └── migrate/                 # 🆕 迁移辅助（扫描 shell 历史 + skill 配置，输出旧→新命令对照）
│       └── migrate.go
├── internal/
│   ├── auth/                    # 🆕 OAuth2 + Token 管理
│   │   ├── oauth2.go           #   PKCE flow（公共前置 + 自动模式 + 手动模式）
│   │   ├── browser.go          #   detectBrowser() 跨平台浏览器探测（§4.3.2）
│   │   ├── store.go            #   token 持久化 ~/.gate-cli/tokens.yaml
│   │   └── refresh.go          #   自动刷新（30s 过期窗口）
│   ├── mcpclient/               # 🆕 轻量 MCP 客户端
│   │   └── client.go           #   tools/call 单 tool 调用
│   ├── registry/                # 🆕 Tool 注册表
│   │   ├── registry.go         #   tool_name → handler 映射（call 兜底用）
│   │   └── cex_tools.go        #   嵌入 interface.json
│   ├── cmdutil/                 # 🔧 共享 CLI helper（GetPrinter/GetClient 等，原位置不变）
│   ├── client/client.go         # 修改: 支持 OAuth2 Bearer token 注入
│   └── config/config.go         # 修改: 读取 ~/.gate-cli/tokens.yaml 的 client_id / token
```

> ⚠️ **cmd/ 目录重构是 Phase 1a-CLI 的前置动作**（见 §12.3 子任务 0）：所有现有 `cmd/<module>/` 迁移到 `cmd/cex/<module>/`，import 路径从 `github.com/gate/gate-cli/cmd/spot` 等改为 `github.com/gate/gate-cli/cmd/cex/spot`。每个原顶层命令文件保持内部逻辑不变，只需调整 `func init()` 里的 `rootCmd.AddCommand(...)` 改为挂到 `cexCmd` 而非 `rootCmd`。Cobra 命令注册模式天然支持这种下沉，单元测试和业务逻辑无需触碰。

## 12. 交付计划

OAuth2 认证链路涉及 **3 个团队** 协同交付，按照 §4.7 数据流（授权阶段 CLI→OAuth2 server；业务阶段 CLI→apiv4-gateway→OAuth2 check_endpoint）拆分职责。其中 **OAuth2 server 改造（Phase 1a-OAUTH）是 Phase 1a-CLI 和 Phase 1b-GW 的共同阻塞依赖**。

### 12.1 阶段总览

| 阶段 | 团队 | 内容 | 前置依赖 | 可并行 |
|------|-----|------|---------|--------|
| **Phase 1a-OAUTH** | OAuth2 server | §4.6 改造清单共 6 项（见 §12.2） | 无（立即启动） | 与 Phase 2 并行 |
| **Phase 1a-CLI** | gate-cli | §3.1 cmd/ 目录重构（所有现有命令下沉到 `cmd/cex/`，硬切换）+ §4.3 `cex auth login` + `cex config` + 业务调用带 Bearer + `dex` 占位 | Phase 1a-OAUTH | 与 Phase 2 并行 |
| **Phase 1b-GW** | apiv4-gateway | §5 OAuth2 Bearer 支持 + check_endpoint 调用 + user-center/gatekeeper 并发补全 + 双认证分发 | Phase 1a-OAUTH（check_endpoint 就绪）；**Phase 1a-CLI 可并行开发**，联调需 CLI 先出可发起 Bearer 请求的调试版本 | - |
| **Phase 2** | gate-cli | `info` + `news` 命令模块 + 轻量 MCP client | 无 | 与 Phase 1a-* 并行 |
| **Phase 3** | gate-cli | `call` + `schema` + `doctor` + `migrate` | Phase 1a-CLI + Phase 1b-GW + Phase 2 | - |
| **Phase 4** | 全员 | Skill 迁移（40+ skills） | Phase 1+2+3 | - |

### 12.2 Phase 1a-OAUTH（OAuth2 server 团队）

对应 §4.6 改造清单 6 项，拆解为可独立验收的子任务：

| 子任务 | 文件 / 模块 | 验收标准 |
|-------|------------|---------|
| 1. 新增 `/mcp/oauth/cli-callback` 展示页 | 新增 `handler/cli_callback.go` + 静态 HTML 模板 | 浏览器访问 `https://api.gatemcp.ai/mcp/oauth/cli-callback?code=abc&state=xxx` 能看到 code 展示 + 复制按钮；`?error=...` 分支正确渲染；**不读 cookie、不查 DB**（日志审计验证） |
| 2. DCR 支持多 `redirect_uris` + exact match 校验 | `oauth2/pkce/handler.go:93-164` | 单元测试：同一 client 注册 2 个 redirect_uri，分别发起 authorize 均成功；换 token 时 redirect_uri 不匹配返回 `invalid_grant` |
| 3. authorize 端点处理 `redirect_uri` 参数 | `oauth2/pkce/handler.go` authorize handler | 集成测试：请求不带 `redirect_uri` 且 client 注册了多个 → 返回 `invalid_request`；带正确 `redirect_uri` → 正常颁发 code |
| 4. authorization_code TTL 核实与调整 | `oauth2/pkce/token.go` | 读现有代码报告当前 TTL；若 > 60s 则评估缩短方案并 PR；回填 spec §4.3.5 |
| 5. `/mcp/oauth/cli-callback` 路由白名单 | 中间件 / 路由配置 | 未登录状态直接访问该路径返回 200 HTML，不触发登录跳转 |
| 6. check_endpoint 对 apiv4-gateway 开放 | 内网路由 + `oauth2/rest/trading_view.go:356-394` | apiv4-gateway upstream IP 白名单包含 `POST /oauth/internal/api/check`；压测 1k QPS 下 P99 **< 20ms**（硬性指标，CLI 60s 刷新窗口依赖）；响应 schema 严格包含 `uid`(int64)/`clientId`/`active`/`scope`/`expiresIn`(**Unix 时间戳秒,非相对秒数**)/`deviceId`/`deviceName`；schema 冻结文档化,任何字段重命名必须经 apiv4-gateway + gateapi-mcp-service 双方评审 |
| 7. 端点限流配置 | Nginx / Envoy 入口层 | 按 §5.10 表格的 5 个端点 + QPS 阈值配置；超限返回 429 + `Retry-After`；回归测试：正常 CLI 流量不被误限 |

**交付物**：
- 6 项子任务全部合并主干并部署到 staging
- gate-cli 团队可访问的 staging 环境（需提供 `api.gatemcp.ai-staging` 或等价域名）
- 回归测试：`tradingview_auth` 和 `gateapi-mcp-service` 的 OAuth2 链路不受影响

**预估工期**：3 项为确认/回归（#4, #6, #7），4 项为新开发/小改（#1, #2, #3, #5）。建议 **1 周** 完成并 staging 可用。

### 12.3 Phase 1a-CLI（gate-cli 团队）

依赖 Phase 1a-OAUTH staging 可用后启动联调，但**编码阶段可与 1a-OAUTH 并行**（按 spec §4.3 mock OAuth2 server 响应开发）。

| 子任务 | 文件 / 模块 | 验收标准 |
|-------|------------|---------|
| **0. cmd/ 目录重构（硬切换，前置动作）** | 所有 `cmd/<module>/` → `cmd/cex/<module>/` + 新增 `cmd/cex/cex.go` 根命令 + 删除顶层 `cmd/config/` | (a) `go build ./...` 通过；(b) `gate-cli cex spot market ticker --pair BTC_USDT` 等命令正常执行；(c) 顶层 `gate-cli spot ...` 返回 `unknown command`；(d) 现有单元测试全部通过（import 路径批量改造，业务逻辑不动）；(e) `gate-cli help` 输出只显示 cex/dex/info/news/call/schema/doctor/migrate/version |
| 1. `auth/oauth2.go` PKCE flow 核心 | 新增 | 单测覆盖 verifier/challenge 生成、state 校验、token 解析 |
| 2. `detectBrowser()` 浏览器二进制探测 | `auth/browser.go` 新增 | 跨平台单测（darwin/windows/linux/WSL）；`LookPath` mock 覆盖命中/未命中分支 |
| 3. 自动模式（本地回调 + stdin 秒切 + 超时兜底） | `auth/oauth2.go` | 集成测试：mock OAuth2 server + mock callback，验证三路 select 行为 |
| 4. 手动模式（URL 打印 + `term.ReadPassword` 粘贴 code） | `auth/oauth2.go` | 手工 QA：SSH 场景 + `--manual` 显式 + 自动模式漏判 Enter 秒切 |
| 5. Token 存储 `~/.gate-cli/tokens.yaml` + 自动刷新 | `auth/store.go` + `auth/refresh.go` | 30 秒过期窗口触发 refresh；refresh 失败提示重新 login |
| 6. `cex auth login/status/logout` 命令 | `cmd/cex/auth/*.go` | 端到端：login → status 显示 token 信息 → logout 清空 |
| 7. `cex config` 命令 + 旧 config 启动迁移提示 | `cmd/cex/config/*.go` + `cmd/root.go` 启动 hook | 旧 `~/.gate-cli/config.yaml` 存在时启动打印一次性迁移提示（§8.4），`gate-cli migrate` 可原地重写为新 schema |
| 8. 业务调用透传 Bearer | `internal/client/client.go` + `internal/cmdutil/client.go` | **通过 `http.RoundTripper` 装饰器注入**，不侵入 SDK 内部：① `cmdutil.GetClient()` 在认证优先级判定到 OAuth2 分支时，构造 `&bearerTransport{base: http.DefaultTransport, tokenStore: store}`；② RoundTrip 方法中 `req.Header.Set("Authorization", "Bearer "+token)` 并**清空** SDK 自动加的 HMAC headers（`KEY`, `SIGN`, `Timestamp`）避免 gateway 认证分发混淆；③ 把该 transport 赋给 `gateapi.Configuration.HTTPClient = &http.Client{Transport: bt}`；④ RoundTrip 内部每次调用前先 `RefreshIfNeeded()`（§4.5.1），触发并发安全的 refresh。验收：mock apiv4-gateway 收到的请求只有 `Authorization: Bearer`，无 HMAC headers；并发 5 个命令 refresh 互斥生效。 |
| 9. `cmd/dex/dex.go` 占位命令 | 新增 | `gate-cli dex` 返回 "dex module is planned but not yet implemented, see roadmap"，退出码 0 |

**联调里程碑**：
- M0（最优先）：**子任务 0 目录重构必须在子任务 1-9 之前完成**。因为后续所有 `cmd/cex/auth/*.go` / `cmd/cex/config/*.go` 的路径都依赖新的 `cmd/cex/` 目录存在。建议 W1 前 2 天内独立开 PR 合并。
- M1（可独立开发）：子任务 1-5 用 mock OAuth2 server 完成
- M2（需 Phase 1a-OAUTH staging）：子任务 6-7 端到端走通 `login → token → 调用 staging`
- M3（需 Phase 1b-GW staging）：子任务 8 业务请求在 staging 环境成功返回数据；子任务 9 占位命令上线

**预估工期**：**1.5 周 + 目录重构 2 天 ≈ 2 周**（与 Phase 1a-OAUTH 重叠 1 周 + 联调 0.5 周 + 子任务 0 重构 2 天）。

> 🚨 **子任务 0 的风险点**：Cobra 命令 `init()` 注册是静态的（`rootCmd.AddCommand(spotCmd)` 写死在 `cmd/spot/spot.go` 的 init），重构时需把所有原 `rootCmd.AddCommand(...)` 改为 `cexCmd.AddCommand(...)`，并在 `cmd/cex/cex.go` 里补一次 `rootCmd.AddCommand(cexCmd)`。如果某些命令依赖 `cmdutil.GetClient()` 等 helper，路径不变，只是调用方 import 路径改了。建议用一个清单列表逐模块迁移，每迁一个立刻跑 `go build ./... && go test ./...` 验证。

### 12.4 Phase 1b-GW（apiv4-gateway 团队）

依赖 Phase 1a-OAUTH 的 check_endpoint 就绪即可启动；**与 Phase 1a-CLI 编码完全并行**，联调需 CLI 出可发起 Bearer 请求的调试版本。

对应 §5 改造，拆解为：

| 子任务 | 文件 | 验收标准 |
|-------|------|---------|
| 1. 新增 `app.d/oauth2_auth.lua`（token 验证 + 用户补全 + 缓存 + scope 校验） | ~230 行 | 单测：mock check_endpoint 返回各种状态（active/expired/invalid/network_error）；L1+L2 缓存命中率 |
| 2. 新增 `app.d/apiv4_multi_access.lua`（Bearer vs HMAC 分发） | ~30 行 | Bearer 存在 → OAuth2；KEY+SIGN 存在 → HMAC；都不存在 → 401 |
| 3. 修改 `app.d/gatekeeper_access.lua`（注册 `apiv4_multi` handler） | ~5 行 | 保持现有 handler 兼容 |
| 4. 修改 `app.d/config.lua`（新增 4 个配置项） | `oauth2_check_endpoint` / `user_center_url` / `gatekeeper_internal_url` / `oauth2_token_cache_ttl` | 配置热加载验证 |
| 5. 修改 `nginx.conf`（`lua_shared_dict oauth2_cache_dict 10m`） | `nginx.conf` | reload 后 shared_dict 可用 |
| 6. 修改 `conf.d/apiv4.routes`（批量 `$gate_auth apiv4 → apiv4_multi`） | 全量 apiv4 路由 | 老 HMAC 调用方不受影响（回归测试套件全绿） |
| 7. user-center + gatekeeper 并发调用 + 结果缓存 | `oauth2_auth.lua` 内部 | 单次请求 P99 < 20ms（缓存命中 < 1ms） |
| 8. scope → API 路径段校验 | `oauth2_auth.lua` 内部 | scope=`read` 访问 `POST /api/v4/spot/orders` 返回 403(缺 `spot_trade`);scope=`read spot_trade` 访问同一路径放行;GET 查询请求仅凭 `read` 即可放行 |

**联调里程碑**：
- M1（可独立开发）：子任务 1-6 用 mock check_endpoint 走通
- M2（需 Phase 1a-OAUTH staging）：子任务 1 对接真实 check_endpoint
- M3（需 Phase 1a-CLI M2 版本）：端到端 CLI → gateway → OAuth2 → 业务后端

**预估工期**：**1.5 周**（编码 1 周 + 联调 + 性能压测 0.5 周）。**必须在 CLI 发版前完成**，否则 CLI 登录后的业务命令会全部失败。

### 12.5 Phase 1 甘特图

```
周次:       W1          W2          W3          W4          W5
         ├────────────┼────────────┼────────────┼────────────┼────────
OAuth2:  ████████████ staging
         │Phase 1a-OAUTH (§12.2, 6 项改造)
         │
CLI:     ████████████████████████
         │Phase 1a-CLI 编码(mock)  │联调
         │         (§12.3, M1)    │(M2→M3)
         │
GW:          ████████████████████████
             │Phase 1b-GW 编码(mock)  │联调
             │         (§12.4, M1)   │(M2→M3)
             │
Info/News:████████████████████
         │Phase 2 (§6, 与 Phase 1 并行)
         │
Phase 3:                              ████████████
                                      │call/schema/doctor/migrate
                                      │
Phase 4:                                          ████████████████
                                                  │Skill 迁移 (40+)
```

**关键里程碑：**
- **W1 末**：Phase 1a-OAUTH staging 就绪 → 解锁 CLI M2 / GW M2
- **W3 末**：CLI M3 + GW M3 端到端联调通过 → Phase 1 完结
- **W4 起**：Phase 3 启动（依赖 Phase 1 完结）
- **W5 起**：Phase 4 Skill 迁移启动

## 13. 跨平台兼容性

### 13.1 目标平台

CLI-first 方案需在以下三个 AI Agent 平台上运行：

| 平台 | Shell 执行工具 | 工具来源 |
|------|-------------|---------|
| **Claude Code** | `Bash` tool | 内置 |
| **Cursor** | Terminal / `run_command` | 内置（沙箱模式，需用户审批） |
| **OpenClaw** | `exec` tool | 内置（`bash-tools.exec.ts`） |

### 13.2 各平台 Shell 执行能力（基于源码分析）

#### Claude Code

- **Bash tool** 原生支持，无沙箱限制
- 可自由执行任意 shell 命令
- 输出（stdout/stderr）直接回传到 LLM 上下文
- CLI-first 方案的**主要设计目标平台**

#### Cursor

- 支持终端命令执行，但运行在**沙箱模式**
- 每次 shell 命令可能需要用户点击"允许"审批
- 不影响功能正确性，但影响交互流畅度
- **应对策略**：Skill preflight 提示用户开启 auto-approve（如适用）

#### OpenClaw

- **`exec` tool** 提供完整的 shell 命令执行能力
- 源码位置：`openclaw/src/agents/bash-tools.exec.ts`
- 支持特性：
  - 任意命令执行（`command` 参数）
  - 工作目录指定（`workdir`）
  - 环境变量覆盖（`env`）
  - 超时控制（`timeout`）
  - PTY 伪终端模式（`pty`）
  - 后台执行（`background` + `process` tool 管理）
  - 多执行宿主（gateway / sandbox / node）
- 安全分层：`allowlist`（默认）/ `deny` / `full` 模式
- **注意**：默认 `allowlist` 模式下，`gate-cli` 需要被加入安全列表，或用户将安全模式设为 `full`
- mcporter **不是**通用工具执行机制，仅用于 QMD 内存后端优化

### 13.3 兼容性结论

| 能力 | Claude Code | Cursor | OpenClaw |
|------|:-----------:|:------:|:--------:|
| 运行 `gate-cli` | ✅ | ⚠️ 需审批 | ✅ |
| 获取 JSON 输出 | ✅ | ✅ | ✅ |
| OAuth2 浏览器授权 | ✅ | ✅ | ✅ |
| 后台进程管理 | ❌ | ❌ | ✅ |
| CLI 安装 (`install.sh`) | ✅ | ⚠️ | ✅ |

**结论：CLI-first 方案在三个平台上均可行，无需维护 MCP fallback 双执行面。**

### 13.4 各平台 Token 节省预估

| 平台 | 当前消耗 | CLI-first 后 | 节省 |
|------|---------|-------------|------|
| Claude Code | ~82K tokens | ~0（CLI 无 schema 注入） | **~95%** |
| Cursor | ~82K tokens | ~0（CLI 无 schema 注入） | **~95%** |
| OpenClaw | ~82K tokens | ~0（CLI 无 schema 注入） | **~95%** |

注：上述为 MCP tool schema 注入的节省。Skill 文件（SKILL.md + references/cli.md）仍会注入
LLM 上下文，预估 ~5-10K tokens，整体节省约 **85-90%**。

### 13.5 OpenClaw exec 安全配置建议

OpenClaw 的 `exec` tool 默认使用 `allowlist` 安全模式。gate-cli 需要被加入允许列表：

```yaml
# OpenClaw 配置建议
tools:
  exec:
    security: allowlist
    safeBins:
      - name: gate-cli
        # 允许所有子命令
```

或者在 Skill preflight 中检测安全模式，提示用户调整配置。

### 13.6 Skill 迁移的平台无关性

迁移后的 Skill（`references/cli.md`）**不绑定任何特定平台**：

```markdown
# 示例：gate-info-coinanalysis/references/cli.md

## 获取币种信息
gate-cli info coin get --query BTC --scope full --format json

## 获取市场快照
gate-cli info market snapshot --symbol BTC --timeframe 1d --source spot --format json

## 获取技术分析
gate-cli info trend analysis --symbol BTC --format json
```

各平台的 AI Agent 使用各自的 shell 执行工具（Bash / Terminal / exec）调用上述命令。
命令本身是平台无关的标准 shell 命令。

### 13.7 migrate 命令的平台覆盖

`gate-cli migrate` 需要扫描所有平台的 MCP 配置：

| 平台 | 配置文件路径 | 格式 |
|------|-----------|------|
| Claude Code | `~/.claude.json` (`mcpServers` key) | JSON |
| Cursor | `~/.cursor/mcp.json` | JSON |
| OpenClaw | 插件配置（无统一 MCP 配置文件） | N/A |

OpenClaw 不使用传统 MCP server 配置，因此 `migrate` 对 OpenClaw 的处理是：
检测是否存在旧的 mcporter MCP 路由配置，提示用户移除。
