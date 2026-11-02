# Fixed Earn 的 SDK 查询筛选参数

当前 `cex earn fixed lends` 与 `cex earn fixed history` 均补齐 SDK v7.2.167 已支持的两个可选 query 参数：

| CLI 参数 | SDK 字段 | HTTP query | 行为 |
| --- | --- | --- | --- |
| --sub-business | SubBusiness / optional.Int32 | sub_business | 未传则省略；显式 0 会发送；数值范围按 SDK int32 |
| --business-filter | BusinessFilter / optional.String | business_filter | JSON 对象数组字符串；原文传给 SDK，不重编码或修改其中的业务字段 |

```sh
# 查询；不是申购或赎回。需要已有授权，最终权限以网关判断为准。
gate-cli cex earn fixed lends --page 1 --limit 10 --sub-business 0 --business-filter '[{"business":1,"sub_business":0}]' --format json

gate-cli cex earn fixed history --type 3 --page 1 --limit 10 --sub-business 0 --business-filter '[{"business":2,"sub_business":0}]' --format json
```

不传两个新参数时保留原 query；显式 `--business-filter '[]'` 会发送空数组。`business-filter` 依 SDK 描述使用 JSON 数组，不使用旧 Skill 文档 history 表格中的单独 `"1"`/`"2"`。CLI 仅校验数组/对象结构，不新增 SDK 未声明的业务枚举或对象字段限制；合法业务组合由 API 判断。

输入格式不合法、超过 int32 或 JSON 长度大于 65536 字节时，本地失败且不发请求。64KiB 沿用现有 CLI JSON 输入上限值，是内置输入保护，不是新增配置；也不代表上游网关接受同样长的 URL。

单元与 CLI E2E 均接入现有 Go 测试体系：

```sh
# 单元/历史回归；从 CLI 根目录运行，Go 版本保持 1.24。
GOMAXPROCS=2 GOFLAGS=-p=1 go test ./cmd/cex/earn ./cmd ./cmd/cex -count=1

# 真实 CLI 子进程 -> 原 SDK -> HTTP -> JSON/退出码；复用已有 token/HMAC 夹具。
GOMAXPROCS=2 GOFLAGS=-p=1 go test -tags agentic_e2e ./tests/agentic -run '^TestFixedFiltersCLI' -count=1 -v

# 读取/校验与 mock HTTP 的并发检查。
GOMAXPROCS=2 GOFLAGS=-p=1 go test -race ./cmd/cex/earn -run '^TestFixedFilter' -count=1
```

## 版本与验证边界

通过 `gate-cli version --details --format json` 查看源码或发行制品的版本。发布制品版本由对应GitHub tag注入；本地Mock通过不代表上游权限、筛选语义或交易已验证。
