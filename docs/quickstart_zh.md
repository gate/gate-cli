# gate-cli 快速上手

## 安装

从源码构建（需要 Go 1.21+）：

```bash
git clone https://github.com/gate/gate-cli.git
cd gate-cli
go build -o gate-cli .
sudo mv gate-cli /usr/local/bin/   # 可选：安装到系统路径
```

验证安装：

```bash
gate-cli --help
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

```bash
gate-cli spot account list --api-key your-key --api-secret your-secret
```

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
gate-cli spot market ticker --pair BTC_USDT
gate-cli spot market tickers
gate-cli spot market orderbook --pair BTC_USDT
gate-cli spot market trades    --pair BTC_USDT --limit 10
gate-cli spot market candlesticks --pair BTC_USDT --interval 1h --limit 48

# 合约（默认 USDT 结算）
gate-cli futures market ticker --contract BTC_USDT
gate-cli futures market funding-rate --contract BTC_USDT
gate-cli futures market candlesticks --contract BTC_USDT --interval 1h
```

---

## 账户查询

```bash
gate-cli spot account list                    # 所有现货余额
gate-cli spot account get --currency USDT     # 单币种余额

gate-cli futures account get                  # 合约账户概览
gate-cli futures position list                # 当前持仓列表
gate-cli futures position get --contract BTC_USDT
```

---

## 现货交易

### 限价单

```bash
# 以 80,000 USDT 买入 0.001 BTC
gate-cli spot order buy  --pair BTC_USDT --amount 0.001 --price 80000

# 以 82,000 USDT 卖出 0.001 BTC
gate-cli spot order sell --pair BTC_USDT --amount 0.001 --price 82000
```

### 市价单

```bash
# 市价买：--quote 指定花费的计价币数量（如 USDT）
gate-cli spot order buy  --pair BTC_USDT --quote 10

# 市价卖：--amount 指定卖出的标的币数量（如 BTC）
gate-cli spot order sell --pair BTC_USDT --amount 0.001
```

> **注意：** 市价买单使用 `--quote`，代表花费多少 USDT，而非购入多少 BTC。

### 订单管理

```bash
gate-cli spot order list   --pair BTC_USDT
gate-cli spot order get    --pair BTC_USDT --id 123456789
gate-cli spot order cancel --pair BTC_USDT --id 123456789
gate-cli spot order cancel --pair BTC_USDT --all   # 撤销所有挂单
```

---

## 合约交易

`--settle` 默认为 `usdt`，可在配置文件中设置 `default_settle: usdt` 永久生效。

### 开仓

```bash
# 限价做多：以 80,000 USDT 买入 10 张合约
gate-cli futures order long  --contract BTC_USDT --size 10 --price 80000

# 市价做空：以市价卖出 10 张合约
gate-cli futures order short --contract BTC_USDT --size 10
```

### 调整仓位

`add` 和 `remove` 会自动查询当前持仓方向（多/空），按正确方向下单，无需手动指定正负号。

```bash
gate-cli futures order add    --contract BTC_USDT --size 5   # 按当前方向加仓 5 张
gate-cli futures order remove --contract BTC_USDT --size 5   # 减仓 5 张
```

### 平仓

```bash
gate-cli futures order close --contract BTC_USDT              # 全部平仓
gate-cli futures order close --contract BTC_USDT --size 5     # 部分平仓：平 5 张
gate-cli futures order close --contract BTC_USDT --side short # 双仓模式：平空头
```

### 订单管理

```bash
gate-cli futures order list   --contract BTC_USDT
gate-cli futures order get    --id 123456789
gate-cli futures order cancel --id 123456789
gate-cli futures order cancel --contract BTC_USDT --all
```

---

## 输出格式

### 表格（默认，适合人工阅读）

```bash
gate-cli spot market ticker --pair BTC_USDT
```

```
Pair       Last      Change %  High 24h   Low 24h   Volume
--------   -------   --------  --------   -------   ------
BTC_USDT   83241.5   +2.34%    84100.0    81200.0   1523.41
```

### JSON（适合脚本和 AI Agent）

```bash
gate-cli spot market ticker --pair BTC_USDT --format json
gate-cli futures position list --format json | jq '.[].contract'
```

---

## 多账号 Profile

适合同时管理主账号和子账号等多套 API Key 的场景。

```bash
gate-cli config set api-key    your-sub-key    --profile sub
gate-cli config set api-secret your-sub-secret --profile sub

gate-cli spot account list --profile sub
```

---

## 调试

```bash
gate-cli spot market ticker --pair BTC_USDT --debug
# 将完整的 HTTP 请求和响应输出到 stderr
```

---

## 脚本使用技巧

```bash
# 用 jq 提取特定字段
gate-cli spot market ticker --pair BTC_USDT --format json | jq -r '.last'

# 轮询订单状态，成交后执行后续操作
while true; do
  status=$(gate-cli spot order get --pair BTC_USDT --id 123 --format json | jq -r '.status')
  [ "$status" = "closed" ] && break
  sleep 5
done

# 使用 BTC 结算的合约
gate-cli futures market ticker --contract BTC_USD --settle btc
```
