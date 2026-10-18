# 2026-10-18 Core Attachment Extension API Handoff

## Changed

计划 `plans/2026-10-18-core-attachment-extension-api.md` 的 Core 侧全部落地。

**新增包 `apps/api/app/Models/Attachments/ExtensionReference`**（package
`extensionreference`）——扩展拥有的附件引用的唯一写入口：

- `types.go`：存储约定（`resource_type='extension'`、`resource_id=0`、
  `context="<extID>:<pluginContext>"`）、MIME 白名单、`Policy` 解析、
  `URLPath`、扩展 id / context 校验、扩展级 advisory 锁键。
- `store.go`：`PrepareClaim` / `ApplyClaim` / `Release` / `Count` / `List` /
  `ReleaseExtensionAttachmentReferences`。全部要求调用方提供 `pgx.Tx`。
- `store_postgres_integration_test.go`：10 个真实 PostgreSQL 用例，含并发
  「同 context、不同附件」的 `reference_count == 引用行数` 断言。
- `access_postgres_integration_test.go`（父包）：claim → `ListReferenceAccess` /
  `LoadReadGuardSubject` 真的带出 `content_type` → 匿名读放行 → URL 仍走代理。

**Host Command 目录**（`apps/api/app/Support/HostAPI/v2_command_attachment_extensions.go`）：

- `sforum.attachments.extension.claim@1` / `.release@1` / `.list@1`，全部
  `ActorMode=delegated` + `RequiredPermissions=[attachment.upload]`。
- claim 的 `expectedRevision` 走请求信封（`GetExpectedRevision` /
  `CommittedRevision`），与 `sforum.attachments.status.set@1` 一致。
- 输出整数一律编码为十进制字符串（`attachmentId`/`width`/`sizeBytes`/
  `referenceCount`/`total`），避免 protobuf `Struct` 的 double 精度损失。
- 在 `NewPostgresProtocolV2CommandRuntime` 注册，配置新增
  `AttachmentExtensions`（缺失即 boot 失败，不静默省略命令）。

**公开读谓词拆分**（`apps/api/app/Models/Attachments/access.go`）：

- `isAnonymousReadableReference`（站点资产 + 扩展图片，MIME 读时重查）与
  `isPermanentURLReference`（仅站点资产）分开；扩展引用只进前者。
- `ReferenceAccess` 新增 `ContentType`，`ListReferenceAccess` 与
  `LoadReadGuardSubject` 的 SQL 多选 `a.content_type`。

**卸载自动释放**：`Models/Extensions` 新增窄接口
`ExtensionAttachmentReferenceReaper` + `WithExtensionAttachmentReferenceReaper`
选项，在 `finalizeLifecycleUninstall` 里于物理 purge 之前调用；bootstrap
注入 `extensionreference.Store`。禁用不释放。

**新 option**：`attachment.extension_reference_limit`（默认 500，1..10000），
注册在 `Models/Options/attachment_options.go` + `service_normalize.go` +
`contracts/openapi/schemas/options.yaml` 的两个枚举。

**装配**：`bootstrap/host_command_runtime.go` 新增
`protocolV2AttachmentExtensionRuntime` 适配器；API 与 worker 两条路径都传入
同一形态的依赖。**文档**：`docs/extensions/host-api-v2.md`「Extension
attachment commands」+ `modules/attachments.md`「Extension-Owned Assets」。

## Decisions

- **实现放进子包而不是 `Models/Attachments` 主包**：`Support/HostAPI` 需要
  引用这些类型，而 `attachments` 的同包测试会 import `Support/Extensions`
  → `Support/HostAPI`，直接放主包会形成 test-build 循环依赖。子包不反向
  依赖主包（URL 常量自带），因此两个方向都干净。
- **claim 的 `expectedRevision` 必填**：与 `attachments.status.set` 一致，
  并发保护靠信封而不是输入体。
- **`Released` 计数按引用行而不是按附件**：同一附件允许被同扩展多个 context
  共享，「删了 2 行 / 影响 1 个附件」是两个数字，插件语义需要前者。
- **context 字符集放宽到允许 `:`**：任务书给的字符集 `[a-z0-9._-]` 与其自身
  示例 `oauth2-client:sfo_client_xxx` 冲突。允许 `:` 是安全的：扩展 id 不含
  `:`，所有命名空间判定都是对已知前缀 `extID:` 的等值比较，从不按键内容切分。
- **`updated_at` 只在计数真正变化时推进**：重复认领同一附件同一 context
  不应让插件手里的修订号无谓失效。

## Verification

- `go test ./...`（apps/api）129 包全通过。
- 卸载收口三个用例：释放发生在 purge 之前、释放失败则卸载失败、禁用不释放。
- `DATABASE_URL=… go test ./app/Models/Attachments/ExtensionReference/`：10 个
  集成用例通过（真实 PostgreSQL）。
- `ruby scripts/validate-openapi-refs.rb`：2862 refs / 61 files OK。
- `node tests/validate-architecture-boundaries.mjs`：与改动前**同样的 8 条**
  既有失败；唯一新增越线是 `Models/Extensions/service.go` 1001 → 1003 行
  （新增注入字段 + 注释），其余 7 条改动前就存在。

## Next

- 插件侧（`sforum-oauth2-provider`，不在本仓库）：manifest 加
  `database.grants: ["host_commands"]`、client-write 走 claim/release/list、
  两处上传控件 + `Idempotency-Key`、旧宿主可读降级提示。
- 若要让 `Models/Extensions/service.go` 回到 1000 行以内：把 `ServiceOption`
  构造器块抽到同包新文件（需要同时把 `Models/Extensions` 的 93/92 文件基线
  降下来，否则文件数上限会先挡住）。
- 扩展引用目前没有对应的 admin 治理视图（谁认领了哪些图）；按需再开。

## Open Questions

- `attachment.extension_reference_limit` 是否需要出现在附件设置的运营 UI？
  目前只能通过通用 web-options 接口调整，默认值对新手是安全的。
