# 2026-10-17 Session Handoff — 孤儿 lifecycle operation 自动回收 + 带外关闭 CLI

Task book: `plans/archive/2026-10/2026-10-17-lifecycle-orphan-reclaim.md`
（完成，原 `ready`，归档）。

## Changed

- **5.1 仓库层**：新增 `apps/api/app/Models/Extensions/lifecycle_repository_orphan.go`
  —— `PostgresLifecycleRepository.ReclaimOrphanedLifecycleOperations(ctx, limit)`
  （默认 16，上限 64）+ 同文件的 `PostgresStore` 薄封装。候选 = 开放
  （`completed_at IS NULL`）且无任何持有未过期 lease 的 running/waiting step，
  且（存在曾被 claim 而过期的开放 step）或（`updated_at` 早于 45s 死亡宽限）。
  命中后在单事务内：`FOR UPDATE` 锁行复核 → 开放 step（planned/running/waiting）
  置 `failed`、`error_code=lifecycle.orphaned`、lease 清空 → revision CAS 关闭
  operation（`state=failed`、`terminal_result=cancelled`）→ 同事务写
  `audit_events(action='extension.lifecycle_orphan_reclaim')`。CAS 落空静默跳过。
  `Support/Audit` 新增常量 `ActionExtensionLifecycleOrphanReclaim`。
- **5.2 bootstrap**：`bootstrap/plugin_runtime_coordinator.go` 的
  `ensurePluginRuntimeCoordinatorGenesis` 每轮 `ErrLifecycleOperationInProgress`
  重试前先经窄接口 `pluginRuntimeLifecycleOrphanReclaimer`（可选断言，
  只生产 PostgresStore 实现）尝试回收；失败仅经新 `OnOrphanReclaim` 回调
  记 warn（生产装配 wired 到 config.Logger），绝不改变重试节奏与 30s 预算。
  注意：runtime 短路语义——dev 库已有 publication 时 genesis 直接短路到
  authoritative 校验，不会触发该分支；它是"空/待播种库"的守门路径。
- **5.3 coordinator**：`plugin_runtime_coordinator.go` `reconcileOnce` 的 fence
  pending 分支：检测到开放 op 后先试回收（同款窄接口），再 re-list，仍非空才
  返回 `errPluginRuntimeLifecyclePending`。回收失败保持 pending 语义——绝不
  升级成 OnError（否则启动边界会把"还在等 lifecycle"误判成协调器终止）。
  频率由 5s poll 自然限流。
- **5.4 带外 CLI**：`cmd/sforum/recovery_lifecycle_close.go`
  —— `sforum extension recovery lifecycle-close --operation-id <id> --reason
  "<原因>" [--force] [--json] [--database-url]。单事务：锁行 → 必须
  `completed_at IS NULL` → 拒绝活跃 lease（除非 `--force`）→ 开放 step 置
  failed（`lifecycle.cli_closed`）→ revision CAS 关闭（cancelled）→
  `extension.cli_recovery` 审计（metadata `mode=lifecycle_close`）。
  注册进 `newExtensionCommand()` 的 `newExtensionRecoveryCommand()` 分组，
  与既有 `extension list/disable/...` 平级共存。
- **架构腾位（门禁要求）**：`Models/Extensions` flat 目录正好 95/95，新增文件
  前先把平台 IO 原语 `stable_file_unix.go`/`stable_file_other.go`
  （仅 `public_frontend.go` 一处消费）抽到新包
  `apps/api/app/Support/StableFile`（`OpenReadFile`/`OpenRootReadFile`，
  语义逐字节不变），目录降到 93；加回收文件后 94，基线同步下调 95→94
  （`tests/architecture-boundaries-baseline.json`，棘轮规则：必须下调）。
- **测试（全部新增，未改旧断言）**：
  `lifecycle_repository_orphan_integration_test.go`（5 用例：活 lease 不回收、
  过期回收全链路断言、未 claim 的 45s 宽限、已关闭不回收、4 并发起跑唯一
  回收 + 断言回收终态满足 `RecoverLifecycleOperation` 入口）；
  `plugin_runtime_coordinator_run_test.go` 增 2 用例（回收清掉 → 当轮收敛 +
  OnReady；回收后仍开放 → pending 不 apply，用包一层的选择性
  `pluginRuntimeCoordinatorReclaimingRepository` 保证既有 double 行为不变）；
  `bootstrap/plugin_runtime_coordinator_test.go` 增 2 用例（回收清掉 → 有界
  预算内 ensure 成功；回收失败 → 重试节奏与触顶语义完全不变）；
  `cmd/sforum/recovery_lifecycle_close_test.go`（参数校验不触库 + schema 隔离
  fixture：直接关闭/JSON 输出/活跃 lease 拒绝与 --force/已关闭与不存在拒绝）。
  fixture 叠加 goose `202607140001`（lifecycle 账本）与 `202607140002`
  （step lease 列）。

## Decisions

- 回收保留 `state='failed'` + `terminal_result='cancelled'`（与任务书第 6 节
  手工 SQL 一致）：`cancelled` 使 `RecoverLifecycleOperation` 的重试入口成立，
  这是闭环衔接点（有集成测试直接断言）。
- 回收 NEVER 触碰任何持有未过期 lease 的 running/**waiting** step：任务书文字
  只写 running，但 `LifecycleStepWaiting` 也是"run 阶段的受保护状态"
  （`lifecycle_coordinator_runtime.go`），所以候选排除与 step 判活都覆盖
  waiting——红线方向从紧不从松。
- 平台 IO 原语抽出 `Support/StableFile` 是**所有权变更**（通用基元离开
  extensions 领域），不是"为腾位而搬家"；得益于两个文件只有一处消费方。

## Evidence

- `go build ./...` 绿；`SFORUM_TEST_DATABASE_URL` 下
  `go test ./app/Models/Extensions/... ./bootstrap/... ./cmd/sforum/...`
  全绿（59.9s / 29.8s / 11.9s）。
- `node tests/validate-architecture-boundaries.mjs` 绿；
  `node tests/validate-docs.mjs` 绿（双语 CLI 文档已收录
  `extension recovery lifecycle-close`）。
- 事故 op 115（`sforum.oauth2-provider` disable 卡在 draining）：**在实施过程中
  已由手工止血 SQL 关闭**（2026-10-17 03:13:22，`state=failed`,
  `terminal_result=cancelled`，无 lifecycle_close 审计行——与手工 SQL 不写审计
  一致），air 于 03:16 重启后 `/api/v1/ready` 恢复 `ready=true`。
- 运行时端到端（新代码路径）：向 dev 库种入模拟孤儿 op **238**
  （`orphan.probe.reclaim`, draining, 过期 lease 的 running step）——live
  coordinator 在一个 poll 窗口（5s）内自动回收：03:29:39 关闭（revision+1、
  terminal cancelled、error_code lifecycle.orphaned、step lease 清空）、
  `extension.lifecycle_orphan_reclaim` 审计 1 行、`open_operations` 归零、
  ready 恒为 true。op 238/step 607 与其审计行按账本不可变性保留为证据。

## Next

- admin UI 针对 lifecycle 操作历史的展示（任务书明确排除，未做）。
- 若要演练 bootstrap 守门路径（5.2），需要一个**无任何 publication** 的
  隔离库（genesis 才会真正走到 in-progress 分支）；runtime 短路使其在正常
  实例上不可达，这是按设计。

## Open Questions

- 无。
