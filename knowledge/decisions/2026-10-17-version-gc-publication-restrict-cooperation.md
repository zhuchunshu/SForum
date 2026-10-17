# Version GC 与发布账本 RESTRICT 的协同（方案 A 修正）

- Date: 2026-10-17
- Status: Accepted
- Task book: `plans/2026-10-17-extension-version-gc.md`（阶段 2）
- Related: 迁移 `202607160027_plugin_runtime_publications.sql`、
  `knowledge/modules/extensions.md`（Trust And Lifecycle Boundaries）、
  `sessions/2026-10-17-extension-version-gc-handoff.md`

## Context

扩展版本行（`extension_versions`）被
`plugin_runtime_publication_members(extension_version_id, extension_id,
extension_version, package_digest)` 以复合 FK **ON DELETE RESTRICT** 引用。
任务书方案 A 据此设想：GC 同轮删除过期发布修订（先 members 后
publications），让"曾上线"的版本行在保留窗口过后可以被回收。

实现阶段核对运行时行为时发现方案 A 与数据库防线直接冲突：迁移
`202607160027` 不止装了 FK，还给两张表装了 **append-only fail-closed
触发器**——`plugin_runtime_publication_immutable` 与
`plugin_runtime_publication_member_immutable`（`BEFORE UPDATE OR DELETE`
一律 `RAISE EXCEPTION 'plugin runtime desired publications are append-only'`）、
两张表的 `no_truncate` 触发器，Down 迁移也在存在任何发布历史时拒绝执行。
发布账本是多节点收敛的持久 desired 集与运行时证据（回滚、修复、审计、
`plugin_runtime_applied_members` 全部挂在它上面）；删除历史发布修订会同时
摧毁收敛语义与证据链。

任务书的冲突处理规则是"以运行时行为为准，不伪造验收"，因此方案 A 的
"删除过期发布修订"部分废止。

## Decision

1. **GC 永不删除发布历史。** `plugin_runtime_publications` /
   `plugin_runtime_publication_members` 不进入 GC 的删除范围。被发布历史
   引用的版本行判据为永久钉住，dry-run 以 `blockedReason:
   publication_history` 报告，执行层在锁内复核同一判据，FK RESTRICT 保留为
   最后一道 fail-closed 背线（有集成测试 `TestVersionGCFKRestrictBackstop`
   直接钉住 SQL 层行为）。
2. **可回收集合的定义（最终语义）。** 一个版本行可回收当且仅当：
   非 active、非 staged；无开放 `extension_lifecycle_operations` 以其
   `(extension_id, extension_version, package_digest)` 为目标；无任何发布
   引用；超出保留窗口（默认每扩展保留 active + 1 个最近历史版本，
   `--retain N` / `--retain-days D` 可调）；磁盘目录存在。
3. **诚实的回收面结论。** 每一次插件 enable/upgrade/disable 都会产生一条
   全量发布修订，因此**曾被启用的插件版本行被永久钉住**——版本 GC 对这类
   版本不释放磁盘。GC 的实际回收面是：主题历史版本（发布账本只覆盖插件，
   `enforce_plugin_runtime_member_type` 触发器拒绝非 plugin 成员）与从未
   进入过发布的插件版本行（上传后未启用、staged 候选被替换等）。
4. **lifecycle 历史行清理条款取消。** 任务书阶段 2 的
   "`extension_lifecycle_operations` 历史行同理清理"不再执行：该账本以
   不可变快照列引用版本身份、对版本行没有外键，清理它不改变 GC 候选集，
   却会销毁审计相邻的恢复证据。开放操作继续作为 GC 判据挡板；终态历史行
   与发布历史一样按 append-only 证据保留。
5. **确认 token 语义。** dry-run 签发一次性 token（库内仅存 sha256 摘要），
   token 与"选项 + 精确候选集"的计划摘要绑定——dry-run 与 execute 之间任何
   判据/清单/选项漂移都使 token 失效（fail closed，操作者重跑 dry-run）。
   TTL 取 15 分钟：与 trust challenge 同为一次性语义，但 GC 的计划审阅
   （读清单 → 复制 token → 执行）允许比 trust 的 5 分钟更宽。
6. **若未来确实需要回收"曾上线"版本**，前提是一项独立决策：放宽
   append-only 触发器，或引入发布历史归档机制。两者都改变多节点收敛语义，
   不属于版本 GC 的范围。

## Consequences

- 架构：GC 是纯增量能力，不触碰信任模型、不可变制品语义与 CAS 指针切换；
  `Models/Extensions` 的 GC 代码以阶段 0 的腾位换入（flat 基线 94 → 92，
  五次同族合并 + 一次 GC 配对合并）。
- 运维：`disk_missing` 候选不在 GC 语义内（指向缺失制品清理）；
  `publication_history` 候选会在每次 dry-run 中持续出现——这是"被钉住"的
  如实报告，不是错误。
- 测试：`postgres_store_version_gc_integration_test.go` 覆盖判据矩阵
  （allowed + denied 双侧）、真库执行两段序（删盘成功 → 台账删；删盘失败 →
  台账保留且暂存目录恢复；FK RESTRICT 背线；锁内复核漂移跳过）、token
  一次性/过期/计划漂移拒绝、dry-run 零副作用；CLI 测试覆盖 JSON schema、
  token 绑定与幂等。

## Addendum (2026-10-17, stage 5): the admin entry reuses the plan-bound token

The admin entry (task book stage 5, shipped the same day after the original
deferral) keeps the same confirmation semantics as the CLI instead of the
weaker missing-artifact dialog-only confirmation:

- `GET /admin/extensions/version-gc/candidates` is a pure preview and never
  mints tokens; `POST .../version-gc/confirmations` mints the one-time
  plan-bound token (same 15-minute TTL, same digest binding); `POST
  .../version-gc/execute` consumes it. Without this binding a stale dialog
  could execute against a drifted plan; the in-lock rechecks stay as the
  per-item backstop.
- Guard layout is two-sided: the GET goes through the extensions read guard
  (and enters `requireExtensionsReadAuthority` + the read-guard partition
  matrix), the POSTs through the `super_admin` guard, and the authoritative
  policy is a dedicated `VersionGCAdminService` boundary that enforces
  super_admin on all three operations and binds the actor into the audit
  (`mode=api` + anchor `requested_by_user_id`).

## Evidence

- 触发器原文：`202607160027_plugin_runtime_publications.sql`
  （`reject_plugin_runtime_desired_mutation`、
  `plugin_runtime_publication_member_immutable`）。
- 集成测试：`apps/api/app/Models/Extensions/postgres_store_version_gc_integration_test.go`
  （隔离库 `SFORUM_TEST_DATABASE_URL`，12/12 绿）。
- CLI 测试：`apps/api/cmd/sforum/extension_gc_test.go`（schema 隔离 fixture）。
