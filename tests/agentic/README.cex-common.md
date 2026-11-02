# CEX与CLI通用兼容回归

正式新增单元分布于原cmd、futures、cmdutil、config、configcmd、agentcmd、version、terminalio及oauth2测试包，随`go test ./...`运行；agent命令另跑`go test -tags agent ./cmd ./internal/agentcmd`。不改Go/SDK，不需要真实Key或资金操作。

正式进程用例统一沿本目录TestMain/childEnv/runCLI，执行：

```sh
# 标准与agent子制品都必须运行；外层tag由TestMain显式传给实际CLI。
GOMAXPROCS=2 GOMEMLIMIT=768MiB GOFLAGS=-p=1 \
  go test -tags agentic_e2e ./tests/agentic -run '^TestCEXCompat' -count=1
GOMAXPROCS=2 GOMEMLIMIT=768MiB GOFLAGS=-p=1 \
  go test -tags agent,agentic_e2e ./tests/agentic -run '^TestCEXCompat' -count=1
```

HTTP只mock Gate最终响应，Cobra/SDK/签名/输出/退出/配置AST/锁真实执行；Unix隐藏输入还执行标准库PTY脚本，检验正常保存、粘贴/UTF8/CRLF、Ctrl-C/TERM/EOF/超限、零写与终端恢复。Python3为该原生夹具的外部测试依赖；缺失即失败，不skip。Windows文件/管道用例保留平台正确的warn/错误预期；交叉编译不能冒充Windows控制台实际运行。

沿原`TestRecoveryCanRestartAfterSecondInterruption`：原真实Begin/Dispatch/恢复落盘链后overlay注入中断，断言RT不复活、Result摘要正确、再次Begin/Finish成功；新增OAuth单元与race覆盖持久化失败、写前/写后窗口及未知来源拒绝。原nonce/来源/陈旧快照断言不削弱。


旧新制品差分另复用tests/e2e目录的正式Python unittest：

```sh
# 两制品使用同完整Go版本、SDK、Version、CGO和tags；旧制品用记录的干净基线。
python3 tests/e2e/cex_common_compatibility.py --baseline /path/old --candidate /path/new -v
python3 tests/e2e/cex_common_compatibility.py --agent --baseline /path/old-agent --candidate /path/new-agent -v
# 原有21操作/可选参数等历史wire断言同样保留并运行。
GATE_CLI_BINARY=/path/new PYTHONDONTWRITEBYTECODE=1 \
  python3 -m unittest discover -s tests/e2e -p '*_cli.py'
```

差分对比stdout/stderr/exit、配置和原HTTP body/鉴权路径；Timestamp/Sign独立验签，不整体忽略。仅夹具HOME路径归一化。Agent目录只允许独立推导的两个新叶子增量，Info/News域与其默认域输出原样；不输出默认值/当前凭据。禁止用基线已知失败或非本机编译成功宣称真实服务/所有平台已通过。

上述命令验证本地CLI与Mock边界；跨服务联合测试须另行准备依赖，不能作为这些本地测试的隐含前置条件或通过结论。
