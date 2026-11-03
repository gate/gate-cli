# gate-cli 快速上手

## 安装

### macOS / Linux — shell 脚本

```sh
curl -fsSL https://raw.githubusercontent.com/gate/gate-cli/main/install.sh | sh
```

### macOS — Homebrew

```sh
brew install gate/tap/gate-cli
```

### Windows — PowerShell

```powershell
irm https://raw.githubusercontent.com/gate/gate-cli/main/install.ps1 | iex
```

### 指定版本

将 `<release-tag>` 替换为已公开发布、且支持所需命令的版本标签。

```sh
# Unix
curl -fsSL https://raw.githubusercontent.com/gate/gate-cli/main/install.sh | sh -s -- --version '<release-tag>'

# Windows
$env:GATE_CLI_VERSION="<release-tag>"; irm https://raw.githubusercontent.com/gate/gate-cli/main/install.ps1 | iex
```

### 从源码构建（需要 Go 1.24+）

```bash
git clone https://github.com/gate/gate-cli.git
cd gate-cli
go build -o gate-cli .
sudo install -m 755 gate-cli /usr/local/bin/gate-cli
```

---

## 配置

### 方式一 — 交互式初始化（推荐）

```bash
gate-cli config init
```

会生成 `~/.gate-cli/config.yaml`，按提示输入 API Key 和 Secret 即可。
API Key 在 **gate.com → 账户 → API 管理** 中创建。

### 方式二 — 环境变量

```bash
export GATE_API_KEY=your-api-key
export GATE_API_SECRET=your-api-secret
```

### 方式三 — 命令行临时指定

参数可能出现在进程列表和 shell 历史中。密钥优先通过交互式配置输入；下方占位符仅说明参数语法。

```bash
gate-cli cex spot account list --api-key your-key --api-secret your-secret
```

### API 服务地址

API 地址支持 HTTP 和 HTTPS，包括非 loopback 的测试网关；默认仍为 HTTPS。HTTP 明文传输，仅在相应测试环境使用。`GATE_BASE_URL` 是站点前缀，CLI 自动追加 `/api/v4`。

```bash
# 使用 API 站点地址；不要在末尾添加 /api/v4。
export GATE_BASE_URL="https://api.gateio.ws"
```

OAuth 凭据绑定完整 API resource（协议、主机、端口及路径），切换地址需要重新授权。修改 `GATE_BASE_URL` 不会切换 OAuth 授权服务；对外访问使用 HTTPS。

### 凭证优先级

```
--api-key / --api-secret flag（最高）
  > GATE_API_KEY / GATE_API_SECRET 环境变量
    > 配置文件 profile
```

### 查看当前配置

```bash
gate-cli config list                 # api_key 和 api_secret 默认遮蔽
gate-cli config list --show-secrets  # 显示明文
```

---

## 公共行情数据（无需 API Key）

以下命令无需登录，可直接使用。

```bash
# 现货
gate-cli cex spot market ticker --pair BTC_USDT
gate-cli cex spot market tickers
gate-cli cex spot market orderbook --pair BTC_USDT
gate-cli cex spot market trades    --pair BTC_USDT --limit 10
gate-cli cex spot market candlesticks --pair BTC_USDT --interval 1h --limit 48

# 合约（默认 USDT 结算）
gate-cli cex futures market ticker --contract BTC_USDT
gate-cli cex futures market funding-rate --contract BTC_USDT
gate-cli cex futures market candlesticks --contract BTC_USDT --interval 1h

# 新增行情；分页和深度还需遵守服务端的较小上限。
gate-cli cex spot market rpi-orderbook --pair BTC_USDT --limit 10
gate-cli cex futures market rpi-orderbook --contract BTC_USDT --limit 10
gate-cli cex futures market contracts-all --limit 100 --offset 0
gate-cli cex futures market adl-risk-states --format json --max-output-bytes 1048576
```

---

## 账户查询

```bash
gate-cli cex spot account list                    # 所有现货余额
gate-cli cex spot account get --currency USDT     # 单币种余额

gate-cli cex futures account get                  # 合约账户概览
gate-cli cex futures position list                # 当前持仓列表
gate-cli cex futures position get --contract BTC_USDT
```

---

## 现货交易

写命令会立即执行。示例需先取得对应账户、品种和金额的授权，并检查余额、精度和最小成交额。本文描述当前源码；通过公开安装器安装时，应先用具体命令的 `--help` 检查已安装版本是否支持。

### 限价单

```bash
# 以 80,000 USDT 买入 0.001 BTC
gate-cli cex spot order buy  --pair BTC_USDT --amount 0.001 --price 80000

# 以 82,000 USDT 卖出 0.001 BTC
gate-cli cex spot order sell --pair BTC_USDT --amount 0.001 --price 82000
```

### 市价单

```bash
# 市价买：--quote 指定花费的计价币数量（如 USDT）
gate-cli cex spot order buy  --pair BTC_USDT --quote 10

# 市价卖：--amount 指定卖出的标的币数量（如 BTC）
gate-cli cex spot order sell --pair BTC_USDT --amount 0.001
```

> **注意：** 市价买单使用 `--quote`，代表花费多少 USDT，而非购入多少 BTC。

### 订单管理

```bash
gate-cli cex spot order list   --pair BTC_USDT
gate-cli cex spot order get    --pair BTC_USDT --id 123456789
gate-cli cex spot order cancel --pair BTC_USDT --id 123456789
gate-cli cex spot order cancel --pair BTC_USDT --all   # 撤销所有挂单
```

### 账户选择与附加止盈止损

支持的订单读写可使用 `--account spot|margin|unified|cross_margin`。自动借款需明确选择适用账户；订单级自动还款支持 unified/cross-margin。借还款都需要相应授权。

```bash
# 已授权的统一账户限价买单，附加独立的止盈与止损。
gate-cli cex spot order buy --pair BTC_USDT --account unified --amount 0.001 --price 60000 \
  --stop-profit-json '{"trigger_price":"66000","order_price":"65900"}' \
  --stop-loss-json '{"trigger_price":"57000","order_price":"56900"}'

# 此操作还需要设置父单价格的授权，不能只取得清除止损的授权。
gate-cli cex spot order amend --pair BTC_USDT --account unified --id 123456789 \
  --price 60000 --stop-loss-json '{}'
```

改单时，省略保护字段或传 JSON `null` 表示保留，只有 `{}` 表示清除。接口还要求父单 amount 或 price，不能为了满足要求而悄悄补入旧值。操作后回查两项保护，并分别检查父单与保护单的最小成交额。自动借还款和原生 POV 详见[命令扩展说明](cex-gap-commands.md)。

---

## 合约交易

`--settle` 默认为 `usdt`，可在配置文件中设置 `default_settle: usdt` 永久生效。

### 开仓

```bash
# 限价做多：以 80,000 USDT 买入 10 张合约
gate-cli cex futures order long  --contract BTC_USDT --size 10 --price 80000

# 市价做空：以市价卖出 10 张合约
gate-cli cex futures order short --contract BTC_USDT --size 10
```

### 调整仓位

`add` 和 `remove` 会自动查询当前持仓方向（多/空），按正确方向下单，无需手动指定正负号。

```bash
gate-cli cex futures order add    --contract BTC_USDT --size 5   # 按当前方向加仓 5 张
gate-cli cex futures order remove --contract BTC_USDT --size 5   # 减仓 5 张
```

### 平仓

```bash
gate-cli cex futures order close --contract BTC_USDT              # 全部平仓
gate-cli cex futures order close --contract BTC_USDT --size 5     # 部分平仓：平 5 张
gate-cli cex futures order close --contract BTC_USDT --side short # 双仓模式：平空头
```

### 订单管理

```bash
gate-cli cex futures order list   --contract BTC_USDT
gate-cli cex futures order get    --id 123456789
gate-cli cex futures order cancel --id 123456789
gate-cli cex futures order cancel --contract BTC_USDT --all
```

---

## 交割合约

交割合约仅支持 USDT 结算。使用 `order create` 的有符号数量区分买卖，不使用永续合约的 `long/close` 辅助命令。先查询有效合约，下方合约名和订单 ID 仅为示例；`position close` 是历史查询。

```bash
# 行情（无需 API Key）
gate-cli cex delivery market contracts
gate-cli cex delivery market tickers    --contract BTC_USDT_20261225
gate-cli cex delivery market order-book --contract BTC_USDT_20261225

# 账户与持仓
gate-cli cex delivery account get
gate-cli cex delivery position list

# 下单
gate-cli cex delivery order create  --contract BTC_USDT_20261225 --size 5 --price 80000
gate-cli cex delivery order cancel --id 123456789
gate-cli cex delivery order list --contract BTC_USDT_20261225 --status open
```

---

## 期权

先查询有效合约；下方合约名与订单 ID 仅为示例，需替换为实际操作对象。

```bash
# 行情（无需 API Key）
gate-cli cex options market underlyings
gate-cli cex options market contracts --underlying BTC_USDT
gate-cli cex options market tickers   --underlying BTC_USDT

# 账户与持仓
gate-cli cex options account get
gate-cli cex options position list

# 下单
gate-cli cex options order create --contract BTC_USDT-20261225-60000-C --size 1 --price 500
gate-cli cex options order list --underlying BTC_USDT --status open
gate-cli cex options order cancel --id 123456789

# 减少多头和全部平仓，执行前检查持仓与流动性。
gate-cli cex options order create --contract BTC_USDT-20261225-60000-C --size -1 --reduce-only --price 0 --tif ioc
gate-cli cex options order create --contract BTC_USDT-20261225-60000-C --size 0 --close --price 0 --tif ioc

# 做市商保护（MMP）
gate-cli cex options mmp get   --underlying BTC_USDT
gate-cli cex options mmp set   --underlying BTC_USDT --window 5000 --frozen-period 30000 --qty-limit 100 --delta-limit 50
gate-cli cex options mmp reset --underlying BTC_USDT
```

全部平仓使用 `--close --size 0`；部分减仓使用有符号非零数量与 `--reduce-only`。价格为 0 时必须使用 IOC。`options position close` 是平仓历史查询，不会提交平仓单。

---

## 钱包

```bash
# 余额查询
gate-cli cex wallet balance total                         # 所有账户总资产
gate-cli cex wallet balance small                         # 小额（粉尘）余额
gate-cli cex wallet balance sub --sub-uid 12345             # 子账号余额

# 充提记录
gate-cli cex wallet deposit address --currency USDT
gate-cli cex wallet deposit list    --currency USDT --limit 20
gate-cli cex wallet deposit withdrawals   --currency USDT --limit 20
gate-cli cex wallet market withdraw-status                       # 支持的币种与链信息

# 划转
gate-cli cex wallet transfer create --currency USDT --amount 100 --from spot --to futures
gate-cli cex wallet transfer to-sub --currency USDT --amount 100 --sub-account 12345 --direction to
```

---

## 账户管理

```bash
gate-cli cex account detail                       # UID、邮箱、等级、KYC 状态
gate-cli cex account rate-limit                   # API 频率限制信息
gate-cli cex account keys                    # 主账号 API Key 列表

# 自成交防护（STP）组
gate-cli cex account stp list
gate-cli cex account stp create --name my-group
gate-cli cex account stp users  --id 1
```

---

## 价格触发订单

价格触发订单在市场价格到达指定触发价时自动下单。

```bash
# 现货
gate-cli cex spot price-trigger list
gate-cli cex spot price-trigger create \
  --market BTC_USDT --trigger-price 90000 --side sell \
  --price 90500 --amount 0.001
gate-cli cex spot price-trigger cancel     --id 123456
gate-cli cex spot price-trigger cancel-all --market BTC_USDT

# 合约
gate-cli cex futures price-trigger list
gate-cli cex futures price-trigger create \
  --contract BTC_USDT --trigger-price 90000 --price 0 --size -10 --tif ioc
gate-cli cex futures price-trigger get    --id 456
gate-cli cex futures price-trigger update --id 456 --body-order-id 789 --trigger-price 91000
gate-cli cex futures price-trigger cancel --id 456
```

现货触价限价单的 `--time-in-force` 默认为 `ioc`（触发后立即成交，未成交部分取消）。SDK 声明的 `gtc` 仍可显式传入，是否接受以服务端为准；CLI 不会自动改写或重试为另一策略。其他值会在发送请求前被拒绝。
现货 `order batch-create`、`batch-amend`、`batch-cancel` 保留 SDK 逐项结果；部分或全部失败时返回非零退出码，stderr 显示成功/失败数量。请按逐项结果处理，勿自动重试整批。


---

## 跟踪委托（合约）

跟踪委托以固定比例或价格距离追踪市场，当行情反转时自动触发。
创建必须显式提供 `--price-type`：1 最新价、2 指数价、3 标记价。激活价非零时，`--is-gte=true` 表示所选参考价大于等于激活价时激活，`false` 表示小于等于；创建默认 false。更新时不传此参数保持原方向，显式 `--is-gte=false` 才修改为向下条件。激活价高于当前行情并不保证条件尚未满足。

历史查询必须同时提供 `--finished --start-at --end-at`，时间为正整数 Unix 秒且开始早于结束，不能传毫秒。活动列表不接受时间过滤，避免产生筛选已生效的误解。`--limit` 为 1–100，省略或 0 沿用服务端默认值；CLI 不额外规定最大时间跨度。

```bash
gate-cli cex futures trail create \
  --contract BTC_USDT --amount -10 --price-type 1 --price-offset '2%'   # 卖出委托，回调幅度 2%

gate-cli cex futures trail list
# 按明确的 Unix 秒时间窗查询历史委托。
gate-cli cex futures trail list --finished --start-at 1793534400 --end-at 1793538000 --limit 20
gate-cli cex futures trail get    --id 789
gate-cli cex futures trail update --id 789 --price-offset '1.5%'
# 显式 false 修改方向；不传该参数则保持原方向。
gate-cli cex futures trail update --id 789 --is-gte=false
gate-cli cex futures trail log    --id 789                    # 变更记录
gate-cli cex futures trail stop   --id 789
gate-cli cex futures trail stop-all --contract BTC_USDT
```

合约/交割账户和订单列表查询返回 `USER_NOT_FOUND` 时，CLI 输出 `ACCOUNT_NOT_READY`、`retryable=false` 并保留后端原因。应通过官方账户流程核实业务是否开通或初始化，不要重复请求、修改 flags 或自动转账。其它不能确认前置原因的操作保留 `UNKNOWN`，不冒充空账户；认证、权限及服务故障保持各自错误类型和退出码。

---

## 输出格式

### Pretty（默认）与显式表格输出

默认格式为 `pretty`。支持表格的命令需显式传 `--format table`；Launch DTO 沿用统一 JSON 展示。失败信息写 stderr，退出码非零。

```bash
gate-cli cex spot market ticker --pair BTC_USDT --format table
```

```
Pair       Last      Change %  High 24h   Low 24h   Volume
--------   -------   --------  --------   -------   ------
BTC_USDT   83241.5   +2.34%    84100.0    81200.0   1523.41
```

### JSON（适合脚本和 AI Agent）

```bash
gate-cli cex spot market ticker --pair BTC_USDT --format json
gate-cli cex futures position list --format json | jq '.[].contract'
```

---

## Intel（`info` / `news`）

市场情报与资讯以 **50** 个 MCP 风格 CLI 工具提供（**32** 个 `info`，**18** 个 `news`）。调用方式为 `gate-cli info <分组> <子命令>` 或 `gate-cli news <分组> <子命令>`，参数以**平铺 flag** 为主（脚本/Agent 建议加 `--format json`）。

**`info` 分组：** `coin`、`marketsnapshot`、`markettrend`、`onchain`、`platformmetrics`、`marketdetail`、`macro`、`compliance`。

**`news` 分组：** `feed`（新闻、社区内容、X、网页研究、情绪、公告）、`events`（事件列表、详情、行情异动解释及历史报告）和 `prediction`（预测市场排名、事件搜索、事件快照及实时订单簿）。`search-ugc` 需要 `--query` 或 `--coin`；`search-events` 需要 `--query`、`--coin`、`--category` 中至少一个条件。结果可能部分可用，需检查状态和元数据。参数见各命令 `--help` 或 `gate-cli news mcp-spec`。

**命令帮助：** 使用叶子命令 `--help` 查看参数；设置 `GATE_INTEL_LEAF_HELP=full` 可显示字段详细说明。`gate-cli info mcp-spec` 提供离线输入说明。

列出工具名：`gate-cli info list`、`gate-cli news list`。参数与环境变量：`gate-cli info -h`、`gate-cli news -h`。
固定 shortcut：`gate-cli info +coin-overview --symbol BTC`、`gate-cli news +brief --coin BTC`。

可选 Intel 默认值写在 `~/.gate-cli/config.yaml` 的 `intel:` 下。现货/合约用的 `--api-key` / `GATE_API_KEY`**不会**作为 Intel 的 Bearer；若后端需要鉴权，请按 Intel/MCP 环境单独配置。环境变量、URL、超时等见 [README 的 Intel 配置](../README.md#intel-info-news)。

```bash
gate-cli info coin get-coin-info --query BTC --format json
gate-cli info marketsnapshot get-market-snapshot --symbol BTC_USDT --format json
gate-cli info marketsnapshot get-institutional-metrics --asset BTC --channel all --limit 30 --format json
gate-cli info platformmetrics get-stablecoin-info --scope full --sections '["issuance_flow","usage_structure"]' --start-date 2026-04-01 --end-date 2026-05-01 --format json
gate-cli info platformmetrics get-chain-activity --metric-group staking --lookback 30d --format json
gate-cli news feed search-news --query bitcoin --format json   # 别名：search → search-news
gate-cli news feed search-x --query bitcoin --time-range 7d --lang en --format json
gate-cli news events get-market-move-report --symbol TAIKO --report-id _1782973063855 --format json
gate-cli news events list-market-move-reports --symbol ETH --start-time "2026-07-09 22:00:00" --end-time "2026-07-10 03:15:00" --limit 10 --format json # 按 UTC0 updated_at 过滤、event_time DESC 排序；--limit 0 默认按 20 处理
```

全部 **50** 个工具各一条最小示例见仓库根目录 **README.md**。

### GateAI Agent 模式

`agent-*` 发现命令需要使用 `-tags agent` 构建；普通构建不会注册这些命令，仅设置环境变量不能启用。先在可信源码目录构建，再启用 Agent 默认行为：

```bash
# 在源码目录执行；后续示例使用该构建物。
go build -tags agent -o gate-cli .
export PATH="$PWD:$PATH"
export GATE_CLI_AGENT=1
gate-cli agent-resolve --query "BTC 新闻简报" --format json
gate-cli agent-leaves --format json          # 31 条 curated + 50 个 MCP 工具
gate-cli agent-validate --format json        # 检查命令可用性
```

普通构建使用 `info list`、`news list` 和具体叶子命令的 `--help`；脚本应显式设置 JSON 格式与输出上限。

---

## 多账号 Profile

适合同时管理主账号和子账号等多套 API Key 的场景。

```bash
gate-cli config set api-key    your-sub-key    --profile sub
gate-cli config set api-secret your-sub-secret --profile sub

gate-cli cex spot account list --profile sub
```

---

## 调试

```bash
gate-cli cex spot market ticker --pair BTC_USDT --debug
# CEX 仅在 stderr 打印方法、状态和耗时，不打印 query/header/body
```

对 **`info`** / **`news`**，根级 `--debug` 与 `--verbose` 会在 **stderr** 打印 Intel MCP 传输日志（stdout 上 JSON 结构不变）。可用 `--max-output-bytes` 或环境变量 `GATE_MAX_OUTPUT_BYTES` 限制输出字节。详见仓库根目录 **README.md** 的 **Global flags** 表。

---

## 脚本使用技巧

```bash
# 用 jq 提取特定字段
gate-cli cex spot market ticker --pair BTC_USDT --format json | jq -r '.last'

# 查询完成证据；仅凭 status 不能认定全部成交。
gate-cli cex spot order get --pair BTC_USDT --id 123 --format json | jq '{status, finish_as, left}'

# 使用 BTC 结算的合约
gate-cli cex futures market ticker --contract BTC_USD --settle btc
```

分别处理 stdout、stderr 和退出码，对进程时长、输出、分页及轮询设置上限。超时或丢失响应后不得自动重发写命令。追单 HTTP 200 内的业务拒绝仍应失败退出；活跃追单空列表为 `{"orders":[]}`。详见[结果与恢复](cex-gap-commands.md#results-and-recovery)。

## Agentic 子账号、CFD 与股票

```sh
# 先查看本地授权状态，再决定是否登录。
gate-cli cex auth status --format json
# 申请账户查询和股票交易权限，在浏览器确认授权。
gate-cli cex auth login --scope base,stock
# 股票及 CFD 公开行情无需登录。
gate-cli cex stock market symbols --exchange us --page 1 --page-size 10
gate-cli cex tradfi market kline --symbol XAUUSD --type 1h --limit 100
# 私有查询需要 base 权限。
gate-cli cex stock account assets --format json
gate-cli cex tradfi market commissions --symbols XAUUSD --format json
```

不传 `--scope` 时申请 `base spot_trade futures_trade earn cfd stock`，实际权限以浏览器确认结果为准。`--manual` 用于复制授权 URL 并粘贴授权码；`--force` 重新发起授权；同组 `refresh` 刷新凭据，`logout` 清除本地凭据并尝试远端撤销。撤销失败时需在 Gate 连接管理页面断开连接。

`auth status` 只检查本地状态，`remote_checked: false` 不代表已通过远端鉴权。切换 API resource 需要重新授权；完整 API Key/Secret 优先于 OAuth，Intel 凭据独立。当前源码使用公共 Gate Go SDK `v7.2.176`。

CFD 和股票私有查询使用 `base`；CFD 交易和划转使用 `cfd`，股票订单修改和平仓使用 `stock`，股票账户划转使用 `base`。完整命令、股票订单与交易时段组合、平仓参数、业务引用和分页见[授权、CFD 与股票指南](auth-cfd-stock.md)。POV/追单、Earn 筛选、订单引用及部分失败处理见[CEX 命令参考](cex-gap-commands.md)。

## 不可用命令

子账号管理、LaunchPool、HODLer、CandyDrop、P2P、CrossEx、welfare、activity 和 coupon 命令不可用。`cex auth` 的 Agentic 子账号授权仍可使用；没有 CLI 入口的产品请通过 Gate 网站管理。

## 诊断与脚本结果

- `version --details --format json` 查看版本、构建修订、Go 版本、平台及 Agent 命令能力；修订未知时为 `unknown`。
- `config check --format json` 检查本地配置文件类型、读取权限和文件权限，不读取凭据内容或修改文件；不验证 YAML 语法、API 权限或 Windows ACL。通过返回 0，告警或失败返回 1。
- `config init --hide-input` 在终端隐藏包括 profile 在内的交互输入；管道输入使用普通模式。
- 现货 `order buy/sell` 和合约 `order long/short/add/remove` 支持 `--text t-reference`；`t-` 后最多 28 字节 ASCII 字母、数字、`_`、`-`、`.`。它用于查单关联，不保证服务端幂等。
- 现货未完成订单可用 `spot order get --id t-reference --pair BTC_USDT` 查询，已完成订单使用数字 ID；合约可用 `futures order get-by-text --text t-reference`。已撤销且未成交的合约订单按 text 查询的时间窗口为 60 秒。查询失败不能证明未下单。
- 现货及合约批量创建、修改、撤销保留逐项结果；任一项失败均返回非零退出码，stderr 给出错误摘要。逐项检查成功订单和失败项，不得整批重发。
- `futures trail list` 的列表位于 `data.orders`；`futures price-trigger get` 的 HTTP 205 按业务错误返回，保留 `label` 和 `message`。
- Agent 构建支持 `agent-index --domain cex --details --format json`，可加 `--command 'cex spot order buy'` 查询注册参数及已知读写影响。`unknown` 不能视为安全，条件必填约束仍需查阅命令帮助。

升级时选择明确的版本并保留旧二进制，可通过 `version --details` 对比；恢复旧二进制不会回滚已执行的交易，也不能作为重发未知结果交易的理由。
