# 2026-10-24 Session Handoff

## Changed

- 卸载物理清理自动收敛落地（决策：`knowledge/decisions/2026-10-24-uninstall-cleanup-convergence.md`）：
  - 503 `extension.lifecycle_cleanup_failed` 携带结构化 `data.details`（operationId/failedStep/retriable）；finalize 每个失败点有类型化 `LifecycleCleanupFailure`，`ErrorData` 新增 `Details` 信封字段（OpenAPI 同步）。
  - River 周期任务 `extensions.finalize_uninstall_cleanup`（5 分钟 + RunOnStart，批 10）自动重放 pending tombstone；仅 embed worker 注册（`workerRuntimeDeps.UninstallCleanupSweeper`），独立 worker 刻意关闭。
  - 免键收尾端点 `POST /admin/extensions/:id/uninstall-cleanup/finalize`（super_admin，guard `"*"`，幂等），路由 ID `core.route.extensions.finalize_uninstall_cleanup`（catalog 已再生成）。
  - 前端：卸载幂等键按扩展落 sessionStorage（`useAdminUninstallCleanup.ts`，从 manager 抽出的聚焦协作对象）；确认框显示失败步骤 + 一键重试 + 免键「立即完成清理」；扩展列表/主题列表/总览的 `pendingUninstallCleanup` 徽标。
  - 503 文案改为「后台自动重试 + 立即完成」，不再要求用户理解幂等键。
- 架构基线随收敛调整：`Service`/`LifecycleService`/`serviceCore` 接收器全部回到基线（新增聚焦协作对象 `CleanupConvergenceService`，上限 5）；`useAdminExtensionsManager.ts` 990→921；`Models/Extensions` 平铺目录 92 文件封顶（新代码并入既有 lifecycle 文件）。

## Decisions

- 两段式卸载保留，不把物理 purge 塞回状态机事务；自动收敛的安全全部来自既有幂等机制（终态校验/purge fence/FOR UPDATE/finalized 直返/CleanupID 幂等），零新增锁。
- 收尾成功不补发 `Extension uninstalled.` 兼容事件（首次尝试已发，重放补发会重复）；免键收尾写宿主 audit（action `extension.uninstall` + `operation=uninstall_cleanup_finalized`）。

## Next

- 观察期内验证真实失败路径：在 dev 插件上制造 purge 失败（只读挂载/权限），确认 503 details → 一键重试 → sweep 收敛 → 徽标消失的全链路渲染表现（本次以单测/契约测试与静态验证为主，未做浏览器级失败注入）。
- 遗留缺陷（独立任务）：`BuildAndDigestDevPackage` stage 幂等只看 DB 不看磁盘的 missing 自愈；CLI `recovery lifecycle-close` 与 API 的 operation 终态判定统一（2026-10-18 handoff 的 operation 703 案例）。

## Open Questions

- 独立 worker 部署（生产分离拓扑）是否需要把 sweep 依赖轻量化下沉到 worker 进程；当前决策是 embed-only + 免键端点兜底，若生产广泛采用分离拓扑再复议。
