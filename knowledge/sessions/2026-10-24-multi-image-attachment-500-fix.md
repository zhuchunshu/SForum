# 2026-10-24 Multi-image Post 500 Fix (Attachment Reference Uniqueness Scope)

## Changed

- 根因：迁移 `202607110015` 把槽位型唯一索引
  `(resource_type, resource_id, context)` 建到了所有资源上，topic/comment 的
  第二条附件引用必然违反 `attachment_references_resource_context_unique`，
  冲突被 `fmt.Errorf` 包装后未进入 `mapForumError`，冒泡为 500
  `internal_error`（"服务器暂时不可用"）。0–1 图必成功、≥2 图必失败，
  用户视角即"带图发帖有时候报错"。dev 库日志两次 23505 实证
  （2026-10-24 03:52/03:57），库中无历史漂移、无需回填。
- 修复（另一并行会话起头，本会话收尾）：迁移
  `202610260001_attachment_reference_scope_unique.sql` 收窄为槽位型部分索引
  （site/seo/user/extension），Down 方向 fail-closed；论坛写入抽到
  `app/Models/Forum/attachment_references_postgres.go`（`ON CONFLICT DO
  NOTHING`、`ORDER BY id` 锁序、按引用行重算 reference_count、具名错误）。
- 本会话补齐：`COUNT(ref.id)`→`COUNT(ref.attachment_id)`（兼容无 id 列的
  最小化既有夹具）；夹具改为**每用例一次性独立数据库**跑完整迁移链
  （schema 隔离覆盖不了 `sforum_core_v1` 等数据库级对象，且会把套件库稳定
  视图重绑进测试 schema）；`seedCategory` 幂等复用迁移内置种子分类；
  补 UpdateTopicRecord 的 `Operation/Origin`。
- 架构棘轮：`mapForumError`/`mapReindexError`/reindex sentinels 抽到
  `app/Http/Controllers/Forum/error_mapping.go`（controller.go 1005→865）；
  附件专用 reason/sentinel 移入 `content_attachments.go`
  （types.go 1010→998）。未抬任何基线。
- 契约：`contracts/openapi/paths/forum.yaml` 四个写路径（create/update topic
  与 comment）登记 422 `forum.attachment_unavailable`（`data.fields.content`
  携带 reason）与 409 `forum.attachment_conflict`。
- 决策：`decisions/2026-10-24-attachment-reference-uniqueness-scope.md`；
  模块：`modules/attachments.md`。

## Verification

- `./scripts/test-api-isolated.sh`：`Models/Forum`、`Controllers/Forum`、
  `Models/Attachments`（含 ExtensionReference/UploadPolicy）全绿，新增
  集成测试覆盖多图创建/编辑替换/多图评论、具名 422、槽位唯一性边界、
  既有修订账本回归。
- `./scripts/test.sh --fast` 三线全 PASS（api 331s / web 84s / static 17s）；
  架构门禁通过；`validate-openapi-refs.rb` 2899 refs OK。
- 真机端到端（dev API 8080）：QA 账号上传两图 → POST /topics（editor-document，
  两 image 节点）→ **201**（修复前必 500）；DB 两条引用行、计数各 1；
  负例 PATCH 引用失效附件 → **422 `forum.attachment_unavailable`** +
  可操作中文文案 + `data.fields.content`；浏览器确认两图渲染；QA 帖子已删、
  引用行清零、计数归零（Delete 路径重算同样正确）。
- Web 端零改动：composer 走 `apiErrorMessage` 展示服务端本地化 message，
  新 reason 自动获得可操作文案。

## Next

- 并行会话若仍有未提交改动（revision_writes.go 的 context 收敛、
  postgres_store_topic_mutations.go 的旧实现删除等），提交时统一走查一遍。
- QA 用户 218（qa_multi_image）与两张 74B 测试图留在 dev 库，可忽略或在
  后台缺失制品清理/用户管理中手动处理。

## Open Questions

- 无。
