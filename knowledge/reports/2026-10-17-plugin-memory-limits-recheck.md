# 插件子进程内存上限复核（GOMEMLIMIT + GOGC 注入实测）

日期：2026-10-17
关联决策：`knowledge/decisions/2026-10-17-plugin-process-memory-limits.md`

## 实测环境

dev 实例（air 热重载后自动以新 env 重启全部插件子进程），Linux，
10 个启用插件 + dev 探针共 19 个 `backend/plugin` 子进程常驻。

## 注入验证

子进程 `/proc/<pid>/environ` 全部确认：

```
GODEBUG=disablethp=1
GOGC=50
GOMEMLIMIT=48MiB
```

## PSS 对比

| 口径 | 2026-08-20 治理后（记录值） | 本轮注入后（实测） |
| --- | --- | --- |
| 单插件 PSS（SMTP / 常规内置） | ~19.3 MiB | 平均 ~15.2 MiB |
| 插件总占用 | RSS 合计 ~260 MiB（重复计共享页） | PSS 合计 288 MiB / 19 进程 |

说明：RSS 口径把 Go runtime 静态段/共享库按进程重复计数（RSS 25–30MB/进程），
PSS 才是真实物理内存账；本次首次用 PSS 聚合取证，平均 15.2MiB 相比
08-20 治理后的 19.3MiB 再降约 20%（软上限压掉 GC 滞后水位）。

## 回归验证

- `go test ./app/Support/Extensions/ -count=1` 全绿（env 注入矩阵
  `TestBuildPluginProcessEnv*` 5 条）。
- `go test ./app/Support/Extensions/IntegrationTests/ -count=1`
  （真实子进程链路）除既有失败
  `TestGitHubAuthPluginProtocolV2HeadlessHostSession` 外全绿；
  该失败由工作区另一批未完成的 go.mod 变更引起（`golang.org/x/sys` 升级未
  tidy 进 `sforum-auth-github` 插件），与本变更无关（复现命令见决策记录）。
- 插件调用无超时回归：runtime lease/心跳测试全绿；dev 实例重启后插件
  运行时正常恢复。

## 架构门禁

`tests/validate-architecture-boundaries.mjs`：`protocol.go` 因注入超 cap
（1011→1038），同变更内把 env 构建段（allowlist、builder、GC 默认值）抽至
`plugin_command_transport.go`（53→156 行），`protocol.go` 回落 941 行并
**下调基线 1011→941**。门禁剩余 5 条红项均为工作区既有问题
（`Models/Extensions/service*.go` 等），非本次引入。

## 结论

默认零配置（不碰 .env）即可让已部署实例生效；如需放宽个别大负载插件，
在宿主环境设 `SFORUM_PLUGIN_GOMEMLIMIT`（如 `96MiB`）。
