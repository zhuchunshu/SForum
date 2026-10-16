# 孤儿生命周期操作回收（orphan lifecycle operation reclamation）

**Status**: completed (2026-10-17)
**规划日期**: 2026-10-17
**触发事故**: 生产/开发库 API 全接口 503 `service.recovery_required` / `plugin_runtime_convergence_failed`

## 1. 背景与问题

### 1.1 事故现象

API 启动时所有接口返回 503：

```json
{"code":503,"message":"service.recovery_required","data":{"code":"plugin_runtime_convergence_failed","component":"plugin_runtime","message":"plugin runtime did not converge before the startup deadline; an extension lifecycle operation is still open","publicationRevision":188,...}}
```

### 1.2 根因（已用 SQL 核实）

表 `extension_lifecycle_operations` 存在一条 `completed_at IS NULL` 的开放操作：

```
id=115, extension_id=sforum.oauth2-provider, operation=disable, state=draining,
current_step_id=lifecycle.disable.03.host.draining, started_at=2026-10-17 02:38:03
```

对应 `extension_lifecycle_steps` 中步骤 395（`lifecycle.disable.03.host.draining`, `host.gate`）停在 `status=running`，其 `lease_expires_at=2026-10-17 02:38:33` 早已过期（lease 时长 30s，见 `apps/api/app/Models/Extensions/lifecycle_coordinator_lease.go:13`）。即：**执行该操作的进程在 draining 中途崩溃/被杀，durable 行永远无人推进到终态**。

### 1.3 栅栏机制（为什么一行卡死能挡住整个 API）

- `apps/api/app/Models/Extensions/plugin_runtime_coordinator.go:265-273`：`reconcileOnce` 每轮先调 `ListOpenLifecycleOperations(ctx, 1)`，只要有任意一行 `completed_at IS NULL` 就返回 `errPluginRuntimeLifecyclePending`，拒绝 apply。
- `apps/api/app/Models/Extensions/plugin_runtime_lifecycle_fence_postgres.go` → `PostgresLifecycleRepository.ListOpenOperations`（`lifecycle_repository_operations.go:160`）：`WHERE completed_at IS NULL`。
- `apps/api/bootstrap/plugin_runtime_coordinator.go:24-29`：genesis ensure 最多等 30s，首轮收敛就绪也最多 30s；超时进入 recovery-only 降级启动 → `/api/v1/ready` 503 + `recovery_required`。

### 1.4 现有的正常恢复路径为什么救不了它

`LifecycleService.RecoverLifecycleOperation`（`apps/api/app/Models/Extensions/service_lifecycle_recovery.go:44-47`）要求 `CompletedAt != nil` 且终态为 failed/cancelled——鸡生蛋问题：操作永远等不到完成，也就永远不满足恢复入口。

## 2. 目标

1. **代码兜底（主要交付）**：启动协调器在等待首轮收敛期间，自动回收"所有 step lease 均已过期"的孤儿 lifecycle operation，使其不再阻塞栅栏；回收必须 fail-closed、可审计、CAS 安全。
2. **带外 CLI（次要交付）**：`sforum extension recovery` 增加 `lifecycle-close` 子命令，允许运维显式把一条开放操作推进到 `cancelled` 终态（审计落库），用于代码兜底覆盖不到的紧急场景。
3. 不改栅栏语义：仍有活跃 lease 的操作必须继续阻塞收敛（进程可能只是慢，不是死）。

## 3. 非目标

- 不改 `RecoverLifecycleOperation` 的 HTTP 管理端路径语义。
- 不引入新的表；只复用 `extension_lifecycle_operations` / `extension_lifecycle_steps` / `audit_events`。
- 不做 admin UI（后续如需，另立任务）。
- 不重试/续跑被回收的操作；回收后的重试仍走现有 `RecoverLifecycleOperation`（终态为 failed/cancelled 后它就满足入口条件了，这正是设计的衔接点）。

## 4. 现状关键事实（实现前必读）

- Lease 策略：`lifecycleCoordinatorLeaseDuration = 30s`，`lifecycleCoordinatorLeaseHeartbeatInterval = 10s`（`lifecycle_coordinator_lease.go:12-15`）。step 行有 `lease_owner_token` / `lease_expires_at` / `lease_revision`。
- 租约获取 SQL 已存在过期回收语义：`lifecycle_repository_step_leases.go:48` `AND (lease_owner_token = '' OR lease_expires_at <= statement_timestamp())`。
- 操作终态常量：`LifecycleTerminalSucceeded/Failed/Cancelled/Skipped`（`lifecycle_repository.go:33-36`）。
- 关闭操作的标准写法参考 `lifecycle_coordinator.go:401-435` `reconcilePendingTerminal` + `CompleteOperation(CompleteLifecycleOperationInput{...})`（CAS：`ExpectedRevision` + `ExpectedState`）。
- 现有带外 CLI 先例：`apps/api/cmd/sforum/recovery.go`（只连 PostgreSQL、不启动插件代码；审计写 `extension.cli_recovery`）；命令注册在 `apps/api/cmd/sforum/validate.go:30-33` `newExtensionCommand()`。
- 启动等待路径：`bootstrap/plugin_runtime_coordinator.go` 的 `ensurePluginRuntimeCoordinatorGenesis`（30s 有界重试 `ErrLifecycleOperationInProgress`）与首轮就绪等待（30s）。回收逻辑应在**每次 genesis 重试前**以及 **coordinator 首轮 reconcile 检测到 pending 时**各尝试一次。
- 进程崩溃窗口：op 115 的 checkpoint 是 `lifecycle.disable.02.disable:execute.done`，即插件本体 disable 已完成，只剩 host draining 善后——回收此类操作不会留下半启用状态。但实现**不能**假设这一点；回收只是关闭 durable 行，运行时残留由后续正常收敛清理。

## 5. 实现方案

### 5.1 仓库层：`PostgresLifecycleRepository` 新增孤儿回收方法

文件：`apps/api/app/Models/Extensions/lifecycle_repository_orphan.go`（新文件，勿塞进已有的 500 行+ 文件）。

```go
// ReclaimOrphanedLifecycleOperations 回收满足以下全部条件的开放操作：
//   completed_at IS NULL
//   AND 不存在任何 status='running' 且 lease_expires_at > now() 的 step
//       （没有活 lease = 没有活进程在推进它）
// 对每条命中的操作，在单事务内：
//   1. SELECT ... FOR UPDATE 锁住 operation 行，二次确认仍开放
//   2. 把所有 status='running'/'planned' 的 step 置为 'failed'，
//      清空 lease 字段，error_code='lifecycle.orphaned'
//   3. UPDATE operation SET state='failed', terminal_result='cancelled',
//      error_code='lifecycle.orphaned', completed_at=now(), revision=revision+1
//      （用 revision CAS；冲突则跳过该条，不算错误）
//   4. INSERT audit_events(action='extension.lifecycle_orphan_reclaim', metadata=...)
// 返回回收的 operation id 列表。
func (r *PostgresLifecycleRepository) ReclaimOrphanedLifecycleOperations(ctx context.Context, limit int) ([]int64, error)
```

要点：

- `limit` 默认 16，上限 64。
- 候选筛选用 `NOT EXISTS (SELECT 1 FROM extension_lifecycle_steps s WHERE s.operation_id = o.id AND s.status = 'running' AND s.lease_expires_at > statement_timestamp())`。
- step 也要覆盖"从未 claim 过 lease 的 planned/running 行"（lease 字段为空）：这类行同样阻塞，且进程显然没活着（活着就会 claim）。但要求 operation 的 `updated_at < now() - interval '45 seconds'`（一个 lease 周期 + 余量）作为死亡宽限，避免刚启动还没来得及 claim 的竞态。
- 新增错误/审计常量就近放 `lifecycle_repository.go` 常量区下方或新文件内；审计 action 字符串用 `extension.lifecycle_orphan_reclaim`（同时在 `apps/api/app/Support/Audit/audit.go` 加常量 `ActionExtensionLifecycleOrphanReclaim`）。
- `terminal_result` 用 `cancelled` 而不是 `failed`：语义是"宿主确认放弃"，且 `RecoverLifecycleOperation` 两个终态都接受。

### 5.2 启动路径接入（bootstrap）

文件：`apps/api/bootstrap/plugin_runtime_coordinator.go`

- 在 `ensurePluginRuntimeCoordinatorGenesis` 的每次重试循环里、`EnsureInitialPluginRuntimePublication` 返回 `ErrLifecycleOperationInProgress` 后，先调一次 `ReclaimOrphanedLifecycleOperations`（通过 `PostgresStore` 暴露的薄封装，同 `ListOpenLifecycleOperations` 的模式），再 sleep 重试。回收失败只记 warn，不改变重试行为。
- 不改变 30s 总预算；回收是循环内的一个快操作（一次事务）。
- 接口隔离：定义窄接口 `pluginRuntimeLifecycleOrphanReclaimer interface { ReclaimOrphanedLifecycleOperations(context.Context, int) ([]int64, error) }`，对 `config.Ensurer` 做可选类型断言（同 `pluginRuntimeLifecycleFenceRepository` 的可选断言模式，`plugin_runtime_coordinator.go:265`）。测试 double 不实现该接口时行为与现状完全一致。

### 5.3 Coordinator 运行时接入（Models 层）

文件：`apps/api/app/Models/Extensions/plugin_runtime_coordinator.go` `reconcileOnce`

- 在 fence 检测到 `len(open) != 0` 后、返回 `errPluginRuntimeLifecyclePending` 前，如果 repository 同时实现 orphan-reclaim 接口，先尝试回收一次；回收后重新 `ListOpenLifecycleOperations`，仍非空才 pending。
- 注意：运行时路径的回收频率受 poll 间隔（5s）自然限流，无需额外节流。

### 5.4 带外 CLI：`sforum extension recovery lifecycle-close`

新文件：`apps/api/cmd/sforum/recovery_lifecycle_close.go`，注册进 `newExtensionCommand()`（`validate.go`）。

```
sforum extension recovery lifecycle-close --operation-id 115 --reason "process died mid-disable; closing to unblock startup" [--database-url ...] [--json]
```

行为：

- 只连 PostgreSQL（复用 `withRecoveryRepository` / 新建窄连接助手，参考 `recovery.go` 既有模式）。
- 参数校验：`--operation-id` 必填 >0；`--reason` 必填非空（审计要求）。
- 单事务：`SELECT ... FOR UPDATE` 锁 operation 行 → 必须 `completed_at IS NULL` 否则报"已关闭"错误 → 检查是否仍有活跃 lease（有则拒绝并提示先等 lease 过期或加 `--force`）→ 关闭 open steps → 关闭 operation（`terminal_result='cancelled'`，error_code=`lifecycle.cli_closed`）→ 写 `audit_events(action='extension.cli_recovery', metadata={mode:'lifecycle_close', operationId, extensionId, reason})`。
- `--force`：跳过活跃 lease 检查（运维明确负责）。
- `--json` 输出被关闭的操作摘要。
- 成功后提示重启 API。

### 5.5 测试（必须全部新增，不改旧断言语义）

仓库层（Postgres 集成测试，`lifecycle_repository_orphan_integration_test.go`，模式参考 `lifecycle_repository_integration_test.go`）：

1. 构造开放 operation + running step + 未过期 lease → 不回收。
2. lease 过期 → 回收成功；operation 终态 cancelled、step failed、审计行存在、revision+1。
3. 开放 operation 无任何 step 且 `updated_at` 新于 45s → 不回收；旧于 45s → 回收。
4. 已 completed 的 operation → 不回收。
5. 并发回收同一 operation 两次 → 第二次 CAS 落空但不出错，幂等。

Coordinator 单测（`plugin_runtime_coordinator_run_test.go` 新增用例，参考既有 `TestPluginRuntimeCoordinatorDefersFullSetUntilLifecycleOperationCloses`）：

6. fence 返回 open op、reclaim 接口清掉它 → 当轮即收敛，OnReady 触发。
7. reclaim 后仍 open → pending 上报，不收敛。

Bootstrap 单测（`apps/api/bootstrap/` 对应测试文件）：

8. genesis 返回 `ErrLifecycleOperationInProgress`、注入的 reclaimer 清掉 → ensure 成功，未耗尽 30s。

CLI（`apps/api/cmd/sforum/recovery_lifecycle_close_test.go` + 集成测试参考 `recovery_integration_test.go`）：

9. 关闭一条开放 operation，校验终态/审计/JSON 输出。
10. 活跃 lease 且无 `--force` → 拒绝。
11. 已关闭 operation → 明确报错。

### 5.6 验证清单（DoD）

- `cd apps/api && go build ./... && go test ./app/Models/Extensions/... ./bootstrap/... ./cmd/sforum/...` 全绿。
- 手工复现验证（重要）：
  1. 向当前 dev 库插入一条模拟孤儿（或直接复用现有 op 115 做 CLI 路径验证）。
  2. `./scripts/api-dev.sh` 重启 API → 启动日志出现 orphan reclaim 记录 → `/api/v1/ready` 返回 ready=true → `/api/v1/health` 不再含 `recovery_required`。
  3. `SELECT * FROM extension_lifecycle_operations WHERE completed_at IS NULL` 为空。
- `node tests/validate-architecture-boundaries.mjs` 通过。
- 更新 `knowledge/modules/` 中扩展生命周期相关模块笔记 + 新增 `knowledge/sessions/2026-10-17-lifecycle-orphan-reclaim.md` 短交接，`knowledge/index.md` Latest Handoff 指向它。

### 5.7 约束与红线

- 遵循 AGENTS.md：新代码进领域子目录/新文件；`extensionsruntime`、`Models/Extensions` 的 Manager/Service receiver 数是非增长基线——本任务**不**给 `Service`/`Manager` 加方法，只给 `PostgresLifecycleRepository` 加 1 个方法、给 `PostgresStore` 加 1 个薄封装。
- 回收绝不触碰仍有未过期 lease 的操作（fail-closed 方向反过来就破坏栅栏语义）。
- 不改 `ListOpenOperations` 语义、不改栅栏、不改 genesis 30s 预算。
- 注释用中文（项目惯例），只写非显然意图。
- 网络命令前先 `export https_proxy=http://127.0.0.1:7897 http_proxy=http://127.0.0.1:7897 all_proxy=socks5://127.0.0.1:7897`。

## 6. 立即止血（实施代码前的运维动作）

当前生产/开发库中的 op 115 可以先用 5.4 的 CLI 关闭；CLI 未落地前，可手工执行（已由规划者确认 checkpoint 显示插件本体 disable 已完成）：

```sql
UPDATE extension_lifecycle_steps SET status='failed',
  error_code='lifecycle.orphaned', error_reason='lifecycle.orphaned',
  error_message='process died while draining; closed manually',
  lease_owner_token='', lease_expires_at=NULL, lease_heartbeat_at=NULL,
  completed_at=statement_timestamp(), updated_at=statement_timestamp()
WHERE operation_id=115 AND status='running';

UPDATE extension_lifecycle_operations SET state='failed', terminal_result='cancelled',
  error_code='lifecycle.orphaned', error_reason='lifecycle.orphaned',
  error_message='process died mid-disable; closed manually to unblock startup fence',
  completed_at=statement_timestamp(), updated_at=statement_timestamp(), revision=revision+1
WHERE id=115 AND completed_at IS NULL;
```

然后重启 API（air 热加载即可）验证 ready 恢复。
