# 卸载物理清理自动收敛（2026-10-24）

## 背景

Lifecycle V2 卸载是两段式：状态机进入 durable `succeeded` 终态后，`finalizeLifecycleUninstall` 才执行宿主物理清理（隔离资产发布 → 释放附件引用 → 清除 provider 选择 → exact-receipt 物理 purge）。任一步失败会停在「台账 succeeded、tombstone pending」中间态并返回 503 `extension.lifecycle_cleanup_failed`。原设计要求操作者「用原幂等键重试」，但前端幂等键只存内存 Map，页面刷新即丢，且失败响应只有 reason code，操作者无从判断卡在哪一步，也没有任何自动收敛机制（2026-10-18 handoff 记录过一次 missing 自锁事故）。

## 决策

1. **失败可观测**：finalize 每个失败点包装类型化 `LifecycleCleanupFailure{OperationID, Step, Retriable}`，controller 经 `errors.As` 写入 503 响应 `data.details`（operationId/failedStep/retriable）。`ErrorData.Details map[string]any` 是增量信封字段，transport 层保持领域无关；键由 OpenAPI `LifecycleCleanupFailureDetails` 约束。`errors.Join` 链与 `ErrLifecycleCleanupFinalization` 哨兵全部保留，日志与既有 errors.Is 判定不变。

2. **自动收敛优先于用户重试**：新增 River 周期任务 `extensions.finalize_uninstall_cleanup`（5 分钟 + RunOnStart，QueueMaintenance，批上限 10），扫描 `uninstall_tombstone AND status='pending'` 且台账已 succeeded 的条目，从失败点续跑 finalize。安全依据全部来自既有机制：终态校验、`samePurgeFence` 并发围栏、`FOR UPDATE` 行锁、finalized 后幂等直返、purger 按 CleanupID 幂等——不新增锁，也不跳过 receipt 校验。单条失败隔离并留待下一轮；仅数据源不可用时 job 整体失败交给 River 重试。

3. **免键恢复入口**：`POST /admin/extensions/:id/uninstall-cleanup/finalize`（super_admin，服务层 `actor.IsSuperAdmin()` 权威判定 + guard `"*"`），按扩展维度把全部 pending tombstone 依序收敛；无 pending 时幂等返回 `uninstalled=false`。「原幂等键」由此从用户必须保管的秘密退化为内部实现细节；DELETE 卸载仍要求原键（新键=新操作语义不变）。

4. **前端兜底顺序**：确认框内一键重试（原键，sessionStorage 按扩展持久化，成功才清除）→ 免键收尾按钮（键丢失后仍可用）→ 后台 sweep 自动兜底。503 文案改为「后台自动重试 + 可点击立即完成」，去掉幂等键黑话。

5. **聚焦协作对象**：`extensions.CleanupConvergenceService`（内嵌 `*Service`，共享 serviceCore 的 finalize 边界与审计器，自身无状态）承载 sweep / 免键收尾 / 徽标投影，避免继续膨胀 `Service`/`LifecycleService`/`serviceCore` 的接收器表面（三者本次全部回到基线）。pending tombstone 读取边界是 `LifecycleCleanupSweepSource`（PostgresLifecycleRepository 实现），经既有 `bindService` 注入。

## 明确不做

- 不把物理 purge 塞回状态机事务做原子化：DB 事务与文件系统操作耦合会让任何一步失败变成真正不可恢复状态。两段分离是优点。
- sweep/resume 不跳过 receipt 校验、purge fence 或 FOR UPDATE 锁。
- 收尾成功**不补发** `Extension uninstalled.` 兼容事件：首次尝试（非重放）在 finishLifecycleV2 已发过一次，重放路径补发会造成重复；完成状态以 cleanup record 的 finalized revision/updated_at 与宿主 audit（免键收尾写 `extension.uninstall` + `operation=uninstall_cleanup_finalized`）为准。

## 部署拓扑注意

sweep worker 只在 **embed worker**（API 进程内嵌，dev 默认）注册：`workerRuntimeDeps.UninstallCleanupSweeper` 由 embed 路径注入，独立 worker（`cmd/worker`）刻意为 nil——收尾依赖 quarantine/attachment/provider/finalizer 完整宿主边界，worker 侧临时重建会绕过 exact receipt 门禁（先例：AIReply/AIDerived 的 embed-only 依赖）。独立 worker 部署下自动收敛关闭，但 503 结构化重试与免键收尾端点仍然可用；周期任务在这些部署的 jobs 目录中无构造器，手动触发按 `schedule_not_found` fail closed。

## 已知折中

- 免键收尾端点以扩展为粒度而非 operationId：多个 pending tombstone 依序收敛，前端刷新后无需回传 operationId；与 sweep 共用同一查询语义。
- 徽标投影（`pendingUninstallCleanup`）在 controller 层做 best-effort 装饰，收敛边界不可用时静默降级为无徽标，不影响任何操作路径。
- 基线变更：`useAdminExtensionsManager.ts` 990→921（卸载清理抽到 `useAdminUninstallCleanup.ts`）；新增 `CleanupConvergenceService` 接收器上限 5。
