# 让扩展复用 Core 附件（claim / release）- Task Book

Status: **landed** - 2026-10-18 实现完成（Core 侧全部交付：三条 Host 命令 + 公开读谓词
拆分 + 卸载自动释放 + 新 option；插件侧接线在 `sforum-oauth2-provider` 仓库）

Date: 2026-10-18
Last updated: 2026-10-18（实现完成）

- 修订：2026-10-18 评审后补充——并发防护（扩展级 advisory 锁 + 计数重算）、公开读谓词拆分、release 作用域与共享语义、`expectedRevision` 走请求信封、只读 list 命令、卸载清理注入接口、context 前缀等值匹配、option 名更正
- 需求来源：`sforum-oauth2-provider` 的应用图标需要「上传图片」而不是「填 URL」
- 影响仓库：SForum（Core，`apps/api`）；插件侧只做调用与 UI 改造

---

## 1. 结论摘要

Core 的附件系统已经具备上传、对象存储、出图、压缩、变体、引用计数与孤儿回收。
插件唯一缺的能力是**「把某个附件登记成扩展拥有的公开资产」**，也就是往
`attachment_references` 里写一行并维持 `reference_count`。

本提案新增三条 Host Command（不改 proto、不改 SDK 生成物）：

| 命令 | 作用 |
| --- | --- |
| `sforum.attachments.extension.claim@1` | 认领：校验归属与类型，强制公开，写引用并重算计数 |
| `sforum.attachments.extension.release@1` | 释放：删引用并重算计数，让孤儿回收正常生效 |
| `sforum.attachments.extension.list@1` | 只读：列出本扩展全部存活引用，供对账与前端回显 |

外加一处**安全白名单**改动：把 `isSitePublicReference` 拆成「匿名可读」与「永久 URL」两个谓词，
扩展分支只加入匿名可读判定。不拆分，图标对其他用户一律 403；不拆谓词，
已认领附件会在管理列表里暴露存储直链（见 §3.4）。

---

## 2. 现状（代码证据）

| 事实 | 位置 |
| --- | --- |
| 上传端点 | `apps/api/app/Http/Controllers/Attachments/routes.go` → `POST /attachments` |
| 出图端点 | 同上 → `GET /attachments/:publicId/content`、`/variants/:variant/content` |
| 孤儿回收条件 | `Models/Attachments/postgres_store.go` → `ListCleanupCandidates`：`reference_count = 0` 且早于 cutoff（active 附件按 `created_at` 判定） |
| 回收定时 | `docs/extensions/catalogs/schedules.md`：`attachments.cleanup_orphans`，周期 `1d`，默认启用 |
| 保留期默认值 | `Models/Options/types.go`：`attachment.cleanup_orphan_after_days = 30` |
| 引用表结构 | `database/migrations/202607050004_attachments.sql`：`UNIQUE(attachment_id, resource_type, resource_id, context)`；`202607110015_seo_attachment_reference_unique.sql` 另有 `UNIQUE(resource_type, resource_id, context)` —— 同一 context 全局至多一行引用 |
| 上传默认可见性 | `Models/Attachments/service.go`：`attachment.default_visibility`，默认 `public`（运营可改私有，claim 的强制公开在该场景下真实生效） |
| 现有引用写入方 | forum / avatar（`Profile/postgres_store.go`）/ seo / site（`Options/postgres_store.go`），全部在 Core 内部 |
| 公开读白名单 | `Models/Attachments/access.go` → `isSitePublicReference`（仅 `seo`、`user:avatar`、`site:logo|favicon|apple-touch-icon|brand|chrome`），同时服务读授权与 URL 装饰两个调用点 |
| 命令通道 | `Support/HostAPI/v2_command_attachments.go`：已有 `sforum.attachments.status.set@1`（修订号走请求信封 `GetExpectedRevision` / `CommittedRevision`） |
| 命令注册点 | `Support/HostAPI/v2_command_runtime_postgres.go`：definitions 列表，数据驱动 |
| 命令级并发 | `Support/HostAPI/v2_command_postgres.go`：引擎 advisory 锁的键含 `IdempotencyKey`，只串行化同一请求的重试，**不**串行化「同 context、不同附件」的并发提交 |
| 委派签发 | `Support/HostAPI/v2_command_delegation_bundle.go`：按 `RequiredPermissions` 过滤，执行时事务内重载 actor 权限 |
| 插件侧调用 | `sdk/plugin/v2/host.go` → `Host.DelegatedCommandRequest` + `Commands.Execute`；旧宿主返回 `ErrHostActorDelegationUnavailable` |
| `host_commands` 前提 | `docs/extensions/host-api-v2.md`「Host Command prerequisite」；`database_grants.go` canonical 集合含 `host_commands` |

关键结论：**不需要新的 RPC、proto 字段或代码生成**。命令目录是数据驱动的，加一条
definition 即可被委派与审计。

---

## 3. 设计

### 3.1 claim 命令

```json
{
  "id": "sforum.attachments.extension.claim",
  "version": "1",
  "inputSchema": "sforum.attachments.extension.claim.input@1",
  "outputSchema": "sforum.attachments.extension.claim.result@1"
}
```

**输入**

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `attachmentPublicId` | string，必填 | 浏览器上传后拿到的 `publicId` |
| `context` | string，必填 | 扩展内唯一键，`[a-z0-9._-]{1,64}`；例如 `oauth2-client:sfo_client_xxx` |

并发保护不进输入体：`expectedRevision` 走请求信封（`request.GetExpectedRevision()`），
结果经 `CommittedRevision` 返回——与 `sforum.attachments.status.set@1` 的既有约定一致。

**输出**

| 字段 | 说明 |
| --- | --- |
| `publicId` / `attachmentId` | 附件标识 |
| `url` | 相对地址 `/media/attachments/<publicId>`（宿主域名无关，换域名不烂） |
| `contentType` / `width` / `height` / `sizeBytes` / `sha256` | 供前端做缓存键与展示 |
| `referenceCount` | 认领后（重算所得）的引用数 |

**效果（单事务）**

0. **扩展级 advisory 锁**：`pg_advisory_xact_lock(hashtextextended('sforum:attachment-extension@1:' || <extID>, 0))`。
   以 Host 证实的扩展 ID 为粒度，串行化本扩展全部 claim/release 变更（先例：`v2_command_notifications.go` 按 extID 加锁）。
   没有这一步，两个不同 `Idempotency-Key` 的并发提交（同 context、不同附件）会在唯一索引上互等，
   `ON CONFLICT DO NOTHING` 吞掉插入后仍盲目 `+1`，产生「计数 >0 却无引用行」的幽灵附件——永不回收，
   且破坏 `reference_count == 引用行数` 不变式。命令引擎自带的 advisory 键含 `IdempotencyKey`，防不住这个场景。
1. `SELECT … FROM attachments WHERE public_id = $1 AND status = 'active' FOR UPDATE`
2. **授权**：`owner_user_id = actor.id` 或 actor 持有 `attachment.manage`
3. **校验**：`content_type ∈ {image/png, image/jpeg, image/webp}`；`size_bytes ≤ 站点上限与 512KB 的较小值`
4. **拒绝二次归属变更**：若该附件已有非 `extension` 类型的引用（论坛/头像/seo/site），直接拒绝，
   避免把正在别处使用的文件改成公开资产
5. 若 `visibility != 'public'` → 置为 `public`。上传默认可见性由 `attachment.default_visibility`
   决定（默认 `public`），运营改私有时此步真实生效
6. 若同 context 已有旧引用：`DELETE` 旧行。受 `(resource_type, resource_id, context)` 全表唯一索引约束，
   同一 context 全局至多一行——**「换图」必须删旧插新**，不能并存
7. `INSERT INTO attachment_references(attachment_id, resource_type, resource_id, context, created_by_user_id)
   VALUES (…, 'extension', 0, '<extID>:<context>', actor.id) ON CONFLICT DO NOTHING`。
   第 0 步的锁就位后该冲突不应发生；若发生视为锁失效，返回冲突错误并记审计，不静默继续
8. **计数重算**：对本事务涉及的每个附件执行
   `UPDATE attachments SET reference_count = (SELECT count(*) FROM attachment_references WHERE attachment_id = attachments.id), updated_at = now()`
   替代盲目 ±1。重算是自愈的：即使历史数据已有漂移，下一次 claim/release 会把它拉回不变式
9. 输出 URL 与元数据；`url` 固定为相对地址 `/media/attachments/<publicId>`，**不复用 `decorateURL`**，
   避免把存储 PublicURL 泄漏进插件数据

**命名空间规则（安全关键）**：`<extID>` 必须取自 Host 已证实的运行时身份
`request.GetContext().GetExtension().GetExtensionId()`，**绝不取输入**。否则 A 插件可以
覆盖或释放 B 插件的引用。

**前缀匹配规则**：存储键 `<extID>:<context>` 的所有前缀判定（容量上限、release、卸载清理）用前缀等值
`LEFT(context, length(extID) + 1) = extID || ':'`，不用 `LIKE`——extID 含 `_`/`%` 时 LIKE 会通配误切。
extID 字符集沿用 manifest 校验。

**容量上限**：单扩展存活引用数超过 `attachment_extension_reference_limit`（新 option，默认 500）
时拒绝，防止插件把站点当图床。计数查询同样使用前缀等值匹配。

**范围说明**：v1 仅图片（png/jpeg/webp、≤512KB）。命令 ID 是泛化的 `extension.claim`，
后续放宽 MIME/体积约束属于 `@2` 版本，不向后兼容地塞进 v1。

### 3.2 release 命令

输入 `{context}` 或 `{attachmentPublicId}`，两者都必须限定 `resource_type='extension'`
且前缀等值本扩展 `<extID>:`：

- `{context}`：删除 `context = '<extID>:<context>'` 的引用行（至多一行）。
- `{attachmentPublicId}`：删除**本扩展**指向该附件的全部引用行。同一附件允许被同扩展多个
  context 共享（owner 校验通过即可），因此可能删除多行。

之后按 claim 第 8 步重算受影响附件的 `reference_count`。幂等：无匹配行时返回成功（`released = 0`）。
作用域若不限定 extID 前缀，A 扩展就能删掉 B 扩展的引用——这是安全边界，不是实现细节。

### 3.3 list 命令

`sforum.attachments.extension.list@1`：只读，输入 `{}`（可选分页），输出本扩展（`<extID>:` 前缀）
全部存活引用：`context / publicId / url / contentType / width / height / sizeBytes / sha256 / createdAt`。

- 用途：插件对账自检、前端回显「当前图标」（插件业务表字段丢失时可自愈）。
- 符合数据面四个 sanctioned surface 中的 query 语义，不引入特殊 RPC。
- 只返回本扩展数据，无越权面；`RequiredPermissions` 与 claim/release 一致为 `attachment.upload`。

### 3.4 公开读白名单（拆分谓词）

`access.go` 当前用同一个 `isSitePublicReference(ref AttachmentReference)` 服务两个调用点：

- `AuthorizeReadGuardSubject` —— 匿名可读判定
- `shouldProxyAuthorizedURL` —— URL 装饰（决定返回存储永久 URL 还是代理路径）

扩展分支**只加入前者**。实现：

- `isAnonymousReadableReference(ref ReferenceAccess) bool`：现有 `seo` / `user:avatar` / `site:*` 分支
  ＋ 新扩展分支（`resource_type='extension'` 且 `content_type ∈ {image/png, image/jpeg, image/webp}`；
  content_type 读时重查，fail-closed）
- `isPermanentURLReference`（原函数收窄）：仅 site/avatar/seo。扩展引用**不加入**——
  admin 列表与附件 API 对已认领附件继续返回稳定的 `/media/attachments/<publicId>` 代理路径，
  不暴露存储 PublicURL

当前 `ReferenceAccess` 不含 `content_type`，因此：

- `ListReferenceAccess` 的 SQL 多选一列 `a.content_type`
- `ReferenceAccess` 增加 `ContentType string`
- 两个调用点改传 `ReferenceAccess`（它们本来就来自它）

**为什么必须改**：不改的话，`AuthorizeReadGuardSubject` 对「非站点资产、非论坛内容」的
引用一律返回 `ErrPermissionDenied`。结果是：确认页、公开应用列表、任何其他用户看到的
图标都会 403，且 `<img>` 静默失败、回退成生成图标——故障表现是「图标没生效」，
极难排查。

**为什么不能借用现有类型**：`site` 的公开判定是 context 白名单
（`logo|favicon|apple-touch-icon|brand|chrome`），借用会被 `Options` 的品牌图槽位语义污染，
且多应用会互相覆盖。

### 3.5 卸载清理

挂载点不变：`Models/Extensions/service_lifecycle_v2.go` 的 `finalizeLifecycleUninstall`
是含 forced/recovery 在内的卸载唯一收口。否则插件被删掉后，附件永远留着引用、永远不回收。

实现方式：`app/Models/Extensions` 是 receiver 计数受架构基线约束的 legacy 包，**不在其上新增方法**。
定义窄接口（如 `ExtensionAttachmentReferenceReaper.ReleaseExtensionAttachmentReferences(ctx, extID string) error`），
由 `bootstrap` 装配注入；实现内部：前缀等值找出本扩展全部引用行 → 删除 →
按 claim 第 8 步同一 helper 重算计数。

禁用（disable）不释放，只有卸载释放；禁用期间图标保持可读是预期行为。

---

## 4. 授权矩阵

| 调用者 | claim | release | list |
| --- | --- | --- | --- |
| 附件所有者（上传者本人） | 允许 | 允许（仅本扩展 context） | 允许 |
| 持 `attachment.manage` 的用户 | 允许（任意附件） | 允许（仅本扩展 context） | 允许 |
| 其他登录用户 | 拒绝 | 拒绝 | 拒绝（无 `attachment.upload` 委派） |
| 匿名 | 不可达（命令需要 actor） | 不可达 | 不可达 |

**为什么不统一要求 `attachment.manage`**：插件的前台允许持 `create_clients` 权限的普通用户
自助创建应用。这类用户通常**没有** `attachment.manage`（那是管理全站附件库的权限）。
统一要求会导致普通用户点保存即被拒，自助建应用功能直接废掉。按「附件归属」判定既符合
最小权限，也符合直觉：你能上传，就能把自己的上传用作图标。

**委派签发条件**：三条命令 `RequiredPermissions` 均设为 `attachment.upload`（能上传才可能认领），
命令体内再做归属判定。这样命令委派不会发给完全没有上传能力的主体。

---

## 5. 生命周期

```
浏览器 POST /api/attachments            → 附件 active，reference_count = 0
插件 claim(context)                     → 引用写入，visibility=public，计数重算
插件换图 claim(同 context)              → 旧引用删除、新引用插入（唯一索引保证单行），计数重算
插件删应用 release(context/publicId)    → 删本扩展引用行，计数重算
30 天后 cleanup_orphans                 → 无引用的附件连同对象存储被删
扩展卸载                                → Core 自动释放该扩展全部引用并重算计数
```

注意：`ListCleanupCandidates` 用的是 `created_at <= cutoff`，所以一个**从未被引用**的
附件在认领前就可能已经超过 30 天——认领不会刷新 `created_at`，但只要有引用就不会被选中。
真正需要担心的是「上传后从未认领」的中间态：由前端流程保证（上传成功即提交保存），
失败则由既有回收机制兜底，这属于 Core 现有语义，无需新增逻辑。

release / 卸载只删引用行，附件本体仍要等 `attachment.cleanup_orphan_after_days`（默认 30 天）
才被回收——**回收期本身就是误删宽限期**，期内重新 claim 即可恢复，不另设宽限。

---

## 6. 威胁模型

| 威胁 | 缓解 |
| --- | --- |
| 插件认领他人的私有附件并转公开 | 归属校验；`attachment.manage` 才能越权 |
| 跨扩展覆盖/释放引用 | context 前缀取自 Host 证实的 extension id |
| 并发 claim 同 context 不同附件 → 幽灵计数 | 扩展级 advisory 锁 + 按引用行重算计数（§3.1 第 0/8 步） |
| 前缀清理误删其他扩展引用 | 前缀等值匹配（非 LIKE），extID 取自 Host 证实身份 |
| 借认领把论坛在用文件改成公开资产 | 已有非 extension 引用时拒绝 |
| SVG / HTML 借公开读形成同源 XSS | MIME 白名单仅 png/jpeg/webp；出图仍带 `nosniff` |
| 插件把站点当图床 | 单扩展引用上限 + 单文件体积上限 |
| 匿名读白名单被泛化 | 白名单分支同时校验 resource_type 与 content_type |
| 已认领附件在管理列表暴露存储直链 | 谓词拆分：扩展分支只进匿名可读判定，不进 URL 装饰（§3.4） |
| 扩展卸载后引用悬挂 | 卸载路径自动释放 |

---

## 7. 兼容与版本

- **proto / SDK 不变**：命令目录是数据驱动，`host_commands` 已是合法 grant（
  `database_grants.go` 的 canonical 集合里就有），manifest schema 无需改动。
- **调用方要求**：插件 manifest `database.grants` 加 `host_commands`；路由请求必须带
  `Idempotency-Key`（否则 `attachInvocationActorDelegation` 不签发委派），且运行时
  `hostCommands` 为真（由 `HasDatabaseGrant(…, host_commands)` 决定）。并发保护使用
  请求信封 `expectedRevision`，重试时携带上次响应的 `revision`。
- **旧宿主降级**：旧 Core 没有该命令 → 委派里没有对应条目 →
  `DelegatedCommandRequest` 返回 `ErrHostActorDelegationUnavailable`。插件应把三条命令都
  映射成可读错误（「当前宿主版本不支持图标上传，请升级 SForum 或改用图标 URL」），
  **不要静默退回 base64 内联**。list 不可用时前端回显退化为插件业务表里存的 `publicId`，不阻塞主流程。
- 插件 `sforumVersion` 需要提到含该命令的版本。

---

## 8. 测试计划

**Core 单测**
- 授权矩阵四种调用者
- MIME 白名单（svg/html 被拒）
- 体积上限、容量上限
- visibility 私有 → 公开
- 同 context 换图：旧引用删除、新引用插入、计数重算正确
- 已有论坛/头像引用的附件被拒
- 跨扩展 context 隔离（A 不能 release B）
- **并发：同 context、不同附件、不同 Idempotency-Key 并发 claim → 无幽灵计数（`reference_count == 引用行数`）**
- release：同附件被同扩展多个 context 共享 → 逐个 release 计数正确；release(publicId) 不触碰其他扩展的引用
- extID 含 `_`/`%` 时前缀等值匹配不错切
- expectedRevision 走信封：修订冲突返回 `ERROR_CODE_CONFLICT`
- 谓词拆分回归：已认领附件在 admin 列表/附件 API 仍返回 `/media/attachments/<publicId>` 代理路径
- 幂等：重复 claim 同附件同 context 不重复 +1

**Core 集成**
- 浏览器上传 → claim → 匿名 `GET /api/attachments/:id/content` 200
- release → 把保留期设为 0 跑一次 cleanup → 对象存储与元数据被删
- 论坛/头像/seo/site 的读权限行为零变化（回归）

**端到端（插件）**
- 管理端与前台自助页：上传 → 保存 → 列表/确认页显示
- 换图 → URL 变化（内容寻址）
- list 回显：重启/对账后前端展示与引用一致
- 删应用 → release → 30 天后回收
- 旧宿主：清晰错误提示

---

## 9. 被否的备选

| 方案 | 否决原因 |
| --- | --- |
| base64 内联进 client-write | 业务写接口装二进制；与 Core 附件体系并行两套；用户已明确不要 |
| 插件自建 `bytea` 存储 | 与 Core 能力重复；多一套出图路由与缓存策略 |
| 插件私有目录（`PluginFiles`） | 私有目录无公开 URL，每次出图要 RPC 读 |
| 复用 `site` 引用类型 | 与品牌图槽位语义冲突，多应用互相覆盖 |
| 把站点保留期调大/关回收 | 全站副作用，为了几个图标改运维策略 |
| `ON CONFLICT DO NOTHING` + 盲目 ±1 计数 | 并发下产生幽灵计数（计数 >0 却无引用行），破坏 `reference_count` 不变式（见 §3.1 第 0/8 步） |
| 扩展分支直接加进现有 `isSitePublicReference` 单谓词 | 会同时改变 URL 装饰行为，管理列表暴露存储直链（见 §3.4） |

---

## 10. 评审决议（2026-10-18）

1. 是否允许认领已被论坛/头像引用的附件？——**禁止**；拒绝范围明确包含 seo 与 site（§3.1 第 4 步）。
2. 单扩展引用上限默认 500 是否合适？——**合适**；按存活引用数计，做成可配置 option。
3. 站点关闭上传后，已认领的图标是否仍可读？——**可读**；读路径从不检查 `UploadEnabled`，无回归风险。
4. 卸载自动释放是否需要回收宽限期？——**不需要**；删引用后仍有 `attachment.cleanup_orphan_after_days`（默认 30 天）兜底，期内重新 claim 即恢复（§5）。
5. 是否需要只读 list 命令？——**是**，纳入本次范围（§3.3）。

---

## 11. 工作量

| 部分 | 估算 |
| --- | --- |
| Core：claim / release / list 命令 + 注册（含 advisory 锁与计数重算） | 320–380 行（含测试） |
| Core：公开读谓词拆分 + `ReferenceAccess.ContentType` | 50–70 行 |
| Core：卸载自动释放（注入窄接口） | 60 行 |
| Core：新 option 与默认值 | 30 行 |
| 文档：`docs/extensions` 命令目录 + attachments 模块 Extension Surface Matrix | ~20 行 |
| 插件：manifest、client-write、claim/release/list 接线 | ~220 行 |
| 前端两处：上传控件 + Idempotency-Key | ~120 行 × 2 |

约 2–2.5 个工作日（含验证），其中 Core 白名单改动需要单独的安全评审。

**Core 侧已 landed（2026-10-18）。** 插件侧任务书已移交到插件仓库：
`sforum-oauth2-provider/docs/plans/2026-10-18-icon-upload-claim.md`
（配套提示词 `…-prompt.md`）。
