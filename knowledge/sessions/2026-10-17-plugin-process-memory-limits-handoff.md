# 2026-10-17 插件子进程内存上限（GOMEMLIMIT + GOGC）Handoff

## Changed

- `Support/Extensions`：所有 Protocol V2 插件子进程固定注入 `GOGC=50` +
  `GOMEMLIMIT=48MiB`（软上限），已部署实例零配置生效。可选覆盖键
  `SFORUM_PLUGIN_GOMEMLIMIT`（宿主自身环境，空值回落默认）；不进入
  `pluginEnvAllowlist` 透传循环，宿主密钥/production 剥离语义原样保留。
- 因 `protocol.go` 注入后超 legacy cap（1011→1038），同变更把 env 构建段
  （allowlist、`buildPluginProcessEnv`、`hostEnvIsProduction`、
  `pluginSettingEnvName`、GC 默认值常量）抽至
  `plugin_command_transport.go`（56→156 行）；`protocol.go` 回落 941 行，
  基线**下调** 1011→941（`tests/architecture-boundaries-baseline.json`）。
- 测试：`protocol_test.go` 新增 3 条（默认值固定、宿主 GOMEMLIMIT 不透传、
  覆盖键生效 + 空值回落），原允许/拒绝矩阵保持。

## Decisions

- `knowledge/decisions/2026-10-17-plugin-process-memory-limits.md`
  （取值理由：48MiB 而非 20MiB 避免 GC 抖动；CPU 换内存权衡；不改契约）。

## Evidence

- `knowledge/reports/2026-10-17-plugin-memory-limits-recheck.md`：
  19 个 dev 插件子进程全带新 env；PSS 平均 ~15.2MiB/进程（较 08-20 基线
  19.3MiB 再降约 20%）；`go test ./app/Support/Extensions/...` 全绿
  （IntegrationTests 除既有 go.mod 失败外全绿，与本变更无关）。
- 架构门禁：本变更相关项清零（基线下调）；剩余 5 条红项为工作区既有问题
  （`Models/Extensions/service.go` 1001 行、`service_package.go`/`manifest.go`/
  `Pages/registry.go` 超 legacy cap、`adminExtensions.ts` 1051 行），非本次引入。

## Next

- 空闲插件进程挂起/懒启动：独立工作流，先写 ADR（触发条件/首调用延迟披露/
  required hook provider 豁免/runtime lease 交互）；复用 stop/start 事务而非
  SIGSTOP。
- 既有失败 `TestGitHubAuthPluginProtocolV2HeadlessHostSession`：由工作区另一批
  未完成变更引起（`apps/api/go.mod` 升 `golang.org/x/sys` + 新增 `x/term` 未
  tidy 进 `sforum-auth-github` 插件 go.mod），待那批工作收口时一并处理。

## Open Questions

- 生产部署后按 60 秒 PSS 中位数复测一次（管理台资源接口已有该口径）。
