# 2026-10-17 Session Handoff — 扩展版本垃圾回收（Version GC）

Task book: `plans/archive/2026-10/2026-10-17-extension-version-gc.md`
（完成，原 `ready`，已归档）。阶段 0→1→2→3→4→**5→6 全部交付**——阶段 5
（管理端入口）在任务书归档后由操作者指示补齐，同日完成。

## Changed

- **阶段 0（架构腾位，硬前置）**：`Models/Extensions` flat 目录 96 → 92，
  基线 94 → **92**（棘轮精确匹配，必须下调）。五次同族合并：
  ① `artifact_health.go` + `missing_artifact_cleanup.go` 的输入归一化助手
  → 新 `artifact_retention.go`（GC 与缺失制品清理共用的制品健康/保留判据）；
  ② `frontend_store.go` → `frontend_types.go`；③ `staged_version.go` →
  `types.go`；④ `plugin_runtime_lifecycle_fence_postgres.go` →
  `lifecycle_repository.go`；⑤ `plugin_runtime_trust_revocation.go` →
  `plugin_runtime_publication_transition.go`。随后 GC 落地时第 6 次合并
  （`theme_publication.go` → `theme_publication_postgres.go`）与
  `postgres_store_version_gc.go` 配对，净文件数保持 92。**注意**：起点是
  96 而非上一会话的 94——并行工作流的 `dev_trust_session*.go`（未提交）
  已占两个名额；本轮腾位把它一并吸收，未触碰那些文件。
- **阶段 1+3（store 层，`postgres_store_version_gc.go`）**：
  `ListVersionGCCandidates`（判据 SQL：非 active/staged + 无开放 op + 无
  发布引用 + 磁盘在；保留窗口在 Go 侧纯函数 `versionGCBlockedReason` 判定）、
  `CreateVersionGCConfirmation`（一次性 token，库内只存 sha256 摘要，绑定
  "选项+精确候选集"计划摘要，TTL 15 分钟）、`ExecuteVersionGCSweep`（原子
  消费 token → 逐条：测体积 → **先删盘**（改名暂存）→ 单事务锁行复核全部
  判据 → 删行 + `extension_version_gc_removals` 锚点 + `extension.version_gc`
  审计 → 提交后删暂存）。事务失败时暂存目录恢复原路径（绝不制造"行在、
  盘缺"）；暂存删除失败以 warning 报告残留路径。
- **阶段 2（决策记录）**：
  `decisions/2026-10-17-version-gc-publication-restrict-cooperation.md`。
  核心修正：任务书方案 A"删除过期发布修订"与 `202607160027` 的 append-only
  触发器（`plugin_runtime_publication_immutable` / member 同款，任何
  UPDATE/DELETE RAISE）直接冲突——**以运行时行为为准，发布历史永不删除**，
  被引用版本行永久钉住（`blocked=publication_history`）；"lifecycle 历史
  行同理清理"条款取消（快照列不钉版本行，清理零收益还毁证据）。
- **阶段 4（CLI，`cmd/sforum/extension_gc.go`）**：`sforum extension gc`
  注册进 `newExtensionCommand()`。默认 dry-run（清单 + 保留/阻塞原因 +
  一次性 token；`--json` schema 稳定）；`--confirm-token` 执行（token 与
  计划摘要绑定，判据/清单/选项漂移即失效）；`--retain`/`--retain-days`/
  `--extension-id`/`--json`/`--database-url`；失败条目完整打印后非零退出。
- **迁移**：`202610170002_extension_version_gc.sql`（`extension_version_gc_
  removals` 锚点表，对齐 `extension_missing_artifact_removals` 模式；
  `extension_gc_confirmations` 确认表，token 摘要唯一）。
- **文档**：`docs/extensions/version-gc.md`（canonical）+
  `docs/zh-CN/extensions/version-gc.md`（全译）+ `docs/en-US/extensions/
  version-gc.md`（指针页，保持平行）；双语 CLI 文档新增 `extension gc`
  表行与章节。`Support/Audit` 新增 `ActionExtensionVersionGC`。

## Decisions

- 见决策记录（上）。补两条实现层取舍：① token TTL 15 分钟（同 trust
  challenge 一次性语义，但 dry-run→execute 的计划审阅更宽）；② 单次
  清单/执行上限 1000 候选（超大批次用 `--extension-id` 分片，防一次事务
  吞掉整个磁盘）。
- 测试隔离：全部真库测试走 `./scripts/test-api-isolated.sh` 一次性容器
  （全新库 + 全量迁移，含并行工作流的 `202610170001` 与本任务的
  `202610170002`），dev 库零污染。

## Evidence

- 隔离库 `go test ./app/Models/Extensions/ -run TestVersionGC -v`：
  **12/12 绿**——纯函数判据矩阵（allowed+denied 双侧 ×7）、真库判据矩阵
  （active/staged 挡、open op 挡、发布引用挡、窗口挡、可回收、
  `--extension-id` 过滤）、dry-run 零副作用、执行两段序（删盘成功→台账删+
  锚点+审计）、token 语义（未知/漂移/重放/过期拒绝）、锁内复核漂移
  （私有不变量直测）、FK RESTRICT 背线（SQL 层拒绝删被钉行）、删盘失败
  →台账保留（chmod 0500 模拟，root 跳过）、盘缺候选不属 GC、
  暂存目录恢复。
- 隔离库 `go test ./cmd/sforum/ -run TestExtensionGC -v`：**4/4 绿**
  （dry-run 清单+token+零副作用、执行消费 token 一次性+幂等、token 绑定
  精确计划、flag 契约互斥/负参拒绝）。
- **实机 dry-run（dev 库，零删除）**：225 条历史行 → 139 可回收 /
  63 `publication_history` / 13 `within_retention` / 10 `disk_missing`——
  判据在真实数据上的分布与决策记录的"诚实回收面"结论一致。
- `go build ./...` 绿；`node tests/validate-architecture-boundaries.mjs`
  对 `Models/Extensions` 绿（92/92，基线已下调）；全量门禁仍剩既有红
  `Support/Pages/registry.go 1002/999`（本工作区既有状态，按指示未修）。
- **全量 `go test ./...`（隔离库 `-p 1`）**：本任务触及的包全绿——
  `Models/Extensions ok 64.1s`（含全部真库集成测试）、`cmd/sforum ok 10.2s`、
  `Support/Audit ok`，无 panic / 无 data race。两个残留失败均**非本任务引入**：
  ① `Support/Localization TestAPIErrorCodesHaveLocalizedMessages`——并行
  dev-trust 工作流（提交 `a7cf7048b`）新增的四个错误码
  （`extension.dev_trust_*`）缺双语文案，失败清单里没有任何 GC 相关码；
  ② `bootstrap` 两个 lifecycle e2e 在全量顺序下 flake（"invalid manifest"
  preflight），单独复跑两次均全绿（2.4s），属顺序/状态敏感既有 flake。
- `node tests/validate-docs.mjs`：本任务新增文档全部通过；剩余 2 条失败
  来自同一并行工作流（`POST /admin/extensions/:id/dev-trust-refresh` 未进
  OpenAPI/路由目录），非本任务引入。

## Stage 5（管理端入口，操作者指示补齐）

- **API 三入口**（`app/Http/Controllers/Extensions/version_gc.go`，静态段注册在
  `:id` 路由之前）：`GET /admin/extensions/version-gc/candidates`（只读预览，
  绝不铸造 token）、`POST .../version-gc/confirmations`（签发计划绑定的一次性
  token，等价 CLI dry-run）、`POST .../version-gc/execute`（消费 token 执行；
  存在失败条目仍返回 200 + 完整逐条结果）。
- **权限模型（两侧纵深）**：GET 走 `core.guard.extensions.read` 粗门
  （进 `requireExtensionsReadAuthority` 清单 + 读守卫分区测试），POST 走
  `super_admin` 守卫；服务层 `VersionGCAdminService`（`postgres_store_version_gc.go`
  内，不新增 flat 文件）对三个入口统一强制 super_admin（硬约束 4 的权威策略）
  并把 actor 绑进审计（`mode=api` + 锚点 `requested_by_user_id`，users FK）。
- **前端**：`SFAdminVersionGcDialog.vue`（`components/admin/` 域路径）三阶段
  对话框（计划→确认→结果，几何照抄缺失制品清理对话框）；工具栏 super_admin
  常驻入口按钮；`utils/admin/adminVersionGc.ts` 独立类型/助手文件
  （adminExtensions.ts 已在 1000 行预算边缘，按域拆分）；i18n 双语键。
- **目录与契约**：路由计数 367→370（`registry_test.go`）；
  `tests/validate-v3-p0-catalogs.mjs` 的 routes/UI tripwire 从陈旧的
  363/321 修正到 370/322（routes 在我之前就已漂移）；UI 身份
  `core.component.admin.sfadmin_version_gc_dialog` 入台账；OpenAPI 三操作
  + 8 个 schema，`validate-openapi-refs.rb` 2849 refs 全过。
- **测试**：store 层管理边界 allowed/denied（plugin.manage 403 语义、
  denied 零副作用、actor 绑定断言、CLI/API token 共用一次性语义）——隔离库
  全绿；Http 读守卫分区测试收录新路由；Localization 门禁绿。
- **HTTP 双侧实机取证（dev 实例）**：super_admin 会话 GET 200（139 可回收）、
  POST confirmations 200（token + planDigest）；无 cookie 401；member 会话
  GET/POST 均 403。
- **渲染态 Browser QA（真实 Chrome CDP）**：桌面（1356x828）计划/确认两阶段
  渲染正确（`/tmp/gc-qa-desktop-{plan,confirm}.png`）；**390x844 dpr3** 计划
  +确认阶段、无横向溢出、0 裁切控件、0 console error
  （`/tmp/gc-qa-390-confirmation.png`）。**dev 实例上不执行真实删除**（执行
  语义由隔离库测试覆盖）；QA 账号 `gc-qa-bot`（super_admin）/`gc-qa-member`
  留在 dev 库作 QA 夹具，运维可随时删除。

## Next

- **真实执行事故闭环（2026-10-17 09:11，操作者在 dev 实例触发）**：操作者经
  管理端执行了一次真实 GC——服务端 139/139 全部成功（锚点 + api 审计各 139、
  requested_by_user_id=1、释放 4,329,878,400 字节 ≈ 4.03 GiB；抽查
  acme.iconprobe 两个历史目录与台账行均消失，active/rank1 保留），但 UI 误报
  "扩展操作失败"——根因是 Go 端空结果切片序列化为 JSON `null`（违反 OpenAPI
  数组契约），前端 `result.failed.length` 对 null 抛 TypeError 被 catch 误判。
  **双侧已修**：后端 `executeVersionGCSweep` 显式初始化三个结果数组为空切片
  （真实空 sweep 的线上 JSON 契约测试钉住 `[]`），前端对 removed/skipped/
  failed/warnings 做 null 归一（防旧后端/缓存）。剩余候选 86 条全部为阻塞态
  （10 disk_missing / 63 publication_history / 13 within_retention）。
- 带外 CLI / 管理端执行一轮真实 GC 的实机演练已由上述操作者执行完成（本条
  替代原"留给运维"项）；dry-run、confirmations 与执行语义均已实测。
- 并行 dev-trust 工作流的共享门禁欠账（上一条所述三处红均非本任务引入，
  归其会话收口）：`route_catalog` 的 UI 身份条目缺 `owners`（生成器当前对
  所有人生成失败）、其读守卫分区测试 map 条目先于目录落地（Http 分区测试红）、
  `dev-trust-refresh` 仍未进 OpenAPI/路由目录（validate-docs 2 失败）。
  本会话已代修其 Localization 错误码文案缺口（4 码 × 双语，纯增量）。

## Open Questions

- 无。
