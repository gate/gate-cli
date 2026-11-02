# CLI process regression tests

The public suite builds the real CLI and uses its normal parsing, SDK, HTTP transport, authorization client, output and exit-code paths. Only remote HTTP services and credentials are synthetic. It requires Go and local loopback networking.

```sh
# Complete self-contained process suite; no private repositories or gateway runtime.
make test-cli-e2e
# Unit and Agent builds use the same maintained packages.
make test-cli-unit
make test-cli-agent
```

The suite covers normal responses, invalid inputs, missing credentials, resource isolation, dependency failures, partial results and safe output. It does not certify live account permissions, remote authorization policy or trading success. Temporary profiles and owned subprocesses are cleaned up by the test framework.

## 跨模块协议回归

`cmd/github_sync_regression_test.go` 检查完整命令树的名称/别名所有权、基础/CEX/Intel入口及退役边界，使用真实继承flag验证Intel参数合并：CLI鉴权和输出flag不透传、显式JSON保留、坏JSON拒绝。

`github_sync_test.go` 在同一临时HOME中运行真实CLI进程、Info/News的MCP初始化/发现/调用、CEX的HMAC/正式SDK和根配置读写。只替换远端HTTP peer，覆盖成功、Intel业务错误、CEX HTTP200业务拒绝、非法JSON零RPC、日志第一页和字符串时间、配置保留OAuth/Intel状态及失败写入原子性。所有密钥/token均为合成值，不连接真实账户。

```sh
go test ./cmd -run '^TestGitHubSync' -count=1
go test -tags agentic_e2e ./tests/agentic -run '^TestGitHubSync' -count=1
go test -tags agentic_e2e,agent ./tests/agentic -run '^TestGitHubSync' -count=1
```

这些用例自动纳入既有 `make test-cli`、Go全量及标准/Agent本地E2E入口，无需新配置。内部服务联合回归单独维护，不能用公共远端mock替代其断言。


## Intel 和 Python 回归覆盖

统一 `make test-cli` 现在同时执行基础/CEX 与 Intel 的历史单元包，以及标准/Agent 本地 E2E。`TestHistoricalPythonCLI` 复用同一个真实 CLI 二进制，运行原有 Intel 文件输入、CEX 可选参数和 Bot 流程 Python 套件（26 个顶层测试，另有内部子用例），再运行 4 个公开测试工具自身的单元/契约测试；后者不能当作真实 CLI E2E 的替代。需要本机已有 `python3`，缺少依赖时测试明确失败。

`tests/contracts/intel.json` 冻结 32 Info + 18 News 的独立命令路径、工具名、有效输入、JSON 回包及 pretty 字段。`TestIntelAllLeafContracts` 对全部 50 个公开叶子检查 flat flags、JSON/pretty 输出、全局 flag 隔离和解析失败零 RPC。链上交易与质押活动使用各自业务响应/专用 pretty 预期，不能假设所有工具都输出通用 Result 模板。

`intel_boundaries.json` 的 41 个独立边界输入同时用于正式校验单元与真实 CLI E2E：含上下限、空值、批量大小、20/21 个 Unicode 字符、相等/反向/带时区时间与闰日。另有 18 组显式 JSON/文件/flat 优先级场景、20 组 HTTP/协议/断连/超时场景，以及不支持 table、基础/CEX 单次错误输出回归。

所有测试只替换远端 MCP/API peer，CLI、Cobra、配置、鉴权、SDK、渲染及退出码保持真实；HTTP 超时验证请求取消，任何无效参数要求零 RPC，调用失败验证不重放。根解析错误按现有契约分别检查错误对象及可选 Hint/Agent 诊断后缀，不接受重复错误对象。来源为 wire name、完整/短 CLI 路径及 `--name=` 的 schema 绑定也有单元覆盖。

```sh
make test-cli
```
