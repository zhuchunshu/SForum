# Extension Version Garbage Collection (Version GC)

- Status: **completed (2026-10-17)**——阶段 0/1/2/3/4/5/6 全部交付并取证（阶段 5 管理端入口原计划拆分，同日由操作者指示补齐）。已归档：`plans/archive/2026-10/`。冲突修正记录见 `knowledge/decisions/2026-10-17-version-gc-publication-restrict-cooperation.md`（方案 A 的发布修订删除与 append-only 触发器冲突，以运行时行为为准；附录记录管理端入口复用计划绑定 token 的语义）
- Created: 2026-10-17
- Scope: `apps/api`（Models/Extensions、cmd/sforum、migrations）＋ 可选管理端（`apps/web`、contracts）
- Preconditions: 无依赖工作流；阶段 0 是硬前置
- Related: `knowledge/modules/extensions.md`、`extensions/README.md`、
  `sessions/2026-10-16-extension-authoring-closeout-handoff.md`（悬挂版本行背景）

## 0. Problem Statement

扩展采用「不可变制品 + 台账指针」模型：每个版本以
`storage/extensions/<id>/<version>/<digest>/` 的只读目录存在，
`extension_versions` 行（`(extension_id, version, package_digest)` 唯一）记录
`package_path`，`extensions.active_version_id` / `staged_version_id` 是指针。
今天**旧版本行与磁盘目录从不删除**（连 `DiscardStagedVersion` 都只清指针，
`postgres_store_staged_version.go` 注释明示 never deletes）。磁盘与台账无限增长，
且 dev 实例已出现「行在、盘缺」的悬挂版本行（`artifact_missing`）。

产品决定：**不改为 WordPress 式单版本覆盖模型**（会摧毁精确制品信任锚点、
回滚能力与多节点摘要收敛），改为提供显式、可审计、fail-closed 的版本 GC。

## 1. Non-Goals

- 不改变信任模型、不可变制品语义、CAS 指针切换。
- 不做后台自动/定时回收（删除不可逆，保持「一切删除可审计」；未来自动化需独立决策）。
- 不处理「行在、盘缺」的历史悬挂行——那是既有缺失制品清理
  （`missing_artifact_cleanup.go`、`POST /admin/extensions/missing-artifacts/uninstall`）
  的职责；GC 与它判据互斥，dry-run 中只提示 `disk_missing` 并指向该流程。
- 不引入 `pruned` 之类的第三种版本状态（已否决，见阶段 2）。

## 2. Hard Constraints（先读，违者返工）

1. **架构基线**：`apps/api/app/Models/Extensions` 是 flat-directory 非增长包
   （基线 94 个非测试文件，当前 96——新增**非测试**文件必须先抽取腾位）；
   `Support/Extensions` 上限 148。`receiverMethodCaps`：
   `Service` ≤70、`LifecycleService` ≤29，不得增长；不够余量就用独立类型。
   每次落地后跑 `node tests/validate-architecture-boundaries.mjs`，
   基线只许下调/持平。任何文件 ≤1000 行（超 500 行时留意内聚）。
2. **数据库防线**：`plugin_runtime_publication_members` 对
   `extension_versions(id, extension_id, version, package_digest)` 是复合 FK
   **ON DELETE RESTRICT**（迁移 `202607160027`）——被发布历史引用的版本行
   数据库层就删不掉，这是设计输入而非障碍，见阶段 2。
3. **删除顺序铁律**：先删磁盘目录、成功后才删台账行；磁盘删除失败必须
   中止该条并报告，**绝不**产生新的 `artifact_missing` 悬挂态。FK RESTRICT
   是最后一道 fail-closed 防线。
4. **权限**：CLI 与 API 入口均 super_admin；一切动作写审计（复用
   `extension_missing_artifact_removals` 的审计锚点模式，
   见 `postgres_store_missing_artifacts.go`）。
5. **测试纪律**：每个新行为有 allowed + denied 双侧测试；store 判据走真
   PostgreSQL 集成测试（参考 `postgres_store_staged_version_integration_test.go`
   的隔离库模式，勿污染 dev 库——见 `sessions/2026-10-12-plugin-runtime-recovery-and-test-isolation.md`）。
6. **知识库工作流**：开工前读 `knowledge/index.md` + `knowledge/modules/extensions.md`；
   收尾时更新模块笔记、hot handoff、决策记录，并把本计划 Status 改为对应状态。

## 3. Phases

### 阶段 0：架构腾位（硬前置）

从 `Models/Extensions` 抽取一个相关职责合并成新文件，腾出文件名额。建议：
把 `artifact_health.go` 的制品健康判定 + `missing_artifact_cleanup.go` 的输入
归一化助手抽到 `artifact_retention.go`（GC 与缺失制品清理共用），净文件数
不增或下降；或另选等效抽取。基线下调。门禁绿后才进入阶段 1。

### 阶段 1：回收判据查询（store 层）

新文件 `postgres_store_version_gc.go`（依赖阶段 0）。纯 SQL 输出候选清单。
一个版本行**可回收**当且仅当全部满足：

- 不是该 extension 的 `active_version_id`，也不是 `staged_version_id`；
- 无**开放状态**的 `extension_lifecycle_operations` 行以其
  `(extension_id, extension_version, package_digest)` 为目标
  （状态集见迁移 `202607140001`；终态行不阻塞——终态历史行的清理属阶段 2）；
- 不被任何 `plugin_runtime_publication_members` 行引用（阶段 2 协同后为真）；
- 超出保留窗口（阶段 3 策略）；
- 磁盘目录存在（缺失行走缺失制品清理，不在 GC 语义内）。

返回 `RetiredVersionCandidate{ExtensionID, VersionID, Version, PackageDigest,
PackagePath, InstalledAt, BlockedReason?}`，CLI 预览与管理端复用同一查询。
参考 SQL：迁移 `202607100004` goose Down 的 ranked_versions 写法。

测试：判据矩阵（active 挡 / staged 挡 / 开放 op 挡 / 发布引用挡 / 窗口内挡 /
全过可回收）+ 真库集成测试。

### 阶段 2：发布台账协同（含已拍板决策）

曾上线版本的版本行被 append-only 发布历史钉住。**采纳方案 A**：
GC 同轮清理**过期发布修订**——仅当 publication 修订 `created_at` 超过保留
窗口、且其全部成员版本均非 active/staged 时，按 members → publications 顺序
删除；每个 extension 的**当前拓扑修订（最新一条）永不删**。
`extension_lifecycle_operations` 历史行同理：仅清终态 + 超窗行，开放行绝不碰。
（方案 B「`pruned` 标记保留行只删盘」已否决：制造第三种制品状态，
`artifactState` 消费方全要改。）

交付物含一条决策记录：
`knowledge/decisions/2026-10-17-version-gc-publication-restrict-cooperation.md`。

### 阶段 3：保留策略与执行

- 默认策略：每 extension 保留 **active + 1 个最近历史版本**；
  `--retain N`、`--retain-days D` 可调；built-in/protected 插件同规则。
- 执行序（两段）：
  1. 逐条锁定候选（复用 `CleanupMissingArtifacts` 的 CAS 行锁模式）并**在锁内
     复核**判据（防 TOCTOU 与并发 enable/upgrade/rollback）；
  2. 先删磁盘目录（移到临时名再删，失败中止该条报告原因）；成功后删台账：
     过期 publication members → publications → 版本行；任一步 FK RESTRICT
     失败即整条回滚并报告；
  3. 全程审计事件。
- 幂等：重复执行安全；`--dry-run` 零副作用。

### 阶段 4：CLI 入口

新文件 `apps/api/cmd/sforum/extension_gc.go`（该目录无 flat 限制），
命令注册方式照抄 `recovery_lifecycle_close.go`：

- `sforum extension gc --dry-run`（**默认**）：打印清单 + 保留/阻塞原因；
  `disk_missing` 行提示走缺失制品清理；`--json` 输出 schema 稳定。
- `sforum extension gc --confirm-token <token>`：执行。token 由 dry-run 输出
  （防误粘贴，一次性语义，同 trust challenge）。
- `--retain N` / `--retain-days D` / `--extension-id <id>` / `--json`。
- production 可用（运维要在生产回收），但必须 `--confirm-token`。

测试：dry-run 零副作用、无 token 拒绝、token 复用/过期拒绝、json schema。

### 阶段 5：管理端入口（可选第二批）

复用缺失制品清理整套模式：

- `GET /api/v1/admin/extensions/version-gc/candidates` +
  `POST .../version-gc/execute`；进四张审核矩阵（身份台账 /
  `requireExtensionsReadAuthority` 清单 / 读守卫分区测试 / 目录路由计数）。
- 前端 `SFAdminVersionGcDialog.vue`（`apps/web/app/components/admin/` 域路径），
  确认几何照抄 `SFAdminMissingArtifactsCleanupDialog.vue`；错误码与文案进
  `messages.go` 双语。
- OpenAPI：`contracts/openapi/paths/extensions.yaml` + schemas；跑
  `ruby scripts/validate-openapi-refs.rb`。
- 渲染态 Browser QA（桌面 + 390x844）后才算完成。

若时间/范围受限，阶段 5 允许拆成独立后续任务书，CLI 先行。

### 阶段 6：文档与知识库收尾

- `docs/extensions/` 运维文档（判据、保留策略、与缺失制品清理边界）+
  `docs/{zh-CN,en-US}` 手册同步。
- 决策记录（阶段 2 一条；若默认保留值与 `active + 1` 有出入再补一条）。
- `knowledge/modules/extensions.md`、hot handoff、`knowledge/index.md` 指针、
  本计划 Status 更新。

## 4. Acceptance Criteria

1. 判据矩阵测试全绿（每条判据 allowed + denied）。
2. 真库集成测试：删盘成功 → 台账删；删盘失败 → 台账保留且错误可读；
   FK RESTRICT 漏网引用 → 删除回滚。
3. CLI dry-run 实测零副作用；execute 幂等实测；token 一次性语义实测。
4. `go test ./...` + `node tests/validate-architecture-boundaries.mjs` 绿；
   阶段 5 做了则 OpenAPI 引用校验与目录计数门禁绿。
5. 审计事件可在事件台账查到每次 GC 的 actor 与逐条结果。
6. 知识库收尾完成（模块笔记、handoff、决策记录、计划状态）。

## 5. Risks / Notes

- 最大工作量风险在阶段 0（flat 腾位）与阶段 2（发布台账协同），预留时间。
- 多节点：GC 单节点执行即可（共享存储 + 共享库）；判据里的开放 op /
  当前发布修订查询天然覆盖所有节点；节点本地 lease 只指向 active 版本。
- `Models/Extensions` 测试文件已 98 个，新测试尽量并进同族 `_test.go`。
- 网络命令前设代理（AGENTS.md Commands 节），本任务预期不需要网络。
