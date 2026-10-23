# Decision: Attachment Reference Uniqueness Scope (Slot vs Multi-value)

## Status

Accepted

## Date

2026-10-24

## Context

迁移 `202607110015_seo_attachment_reference_unique.sql` 在
`attachment_references` 上建立了全局唯一索引
`attachment_references_resource_context_unique (resource_type, resource_id, context)`。
其本意是保护**槽位型**资源——站点品牌图（site）、SEO 图（seo）、用户头像（user）、
扩展图标（extension）——一个命名槽位只能绑定一张图。

但该索引同时覆盖了 topic / comment。论坛正文附件是**多值型**资源：
`replaceForumAttachmentReferences` 逐附件插入引用行，同一 (resource, context) 下
第二行起必然违反该索引。生产表现：

- 0–1 张图发帖/评论成功，≥2 张图 100% 失败；
- 唯一约束冲突被 `fmt.Errorf` 包装后未进入 `mapForumError` 任何分支，
  冒泡为 500 `internal_error`（"服务器暂时不可用，请稍后再试。"），
  用户与运维都无法从响应判断真实原因；
- dev 库日志实证：`duplicate key value violates unique constraint
  "attachment_references_resource_context_unique"`（2026-10-24 03:52/03:57）；
- 2026-07-11（索引建立）之后所有多图发布均失败，库中不存在任何
  (resource, context) 重复行 → 无历史数据漂移，修复不需要回填。

## Decision

- 附件引用唯一性按资源语义分两类：
  - **多值型**（`topic` / `comment`）：一条内容可绑定多个附件，唯一性只由基表
    `UNIQUE (attachment_id, resource_type, resource_id, context)` 保证；
  - **槽位型**（`site` / `seo` / `user` / `extension`）：一个槽位一张图，由
    部分唯一索引 `attachment_references_slot_context_unique ... WHERE
    resource_type IN ('site','seo','user','extension')` 保证。
- 迁移 `202610260001_attachment_reference_scope_unique.sql` 删除全局索引并建立
  槽位部分索引；Down 方向重建全局唯一索引，若多值引用已存在则**刻意失败**
  （fail-closed），不给隐性回归留空间。
- 论坛侧写入（`app/Models/Forum/attachment_references_postgres.go`）配套：
  - 插入使用 `ON CONFLICT DO NOTHING`，把基表唯一性冲突吸收为幂等；
  - 行锁校验 `ORDER BY a.id` 固定加锁顺序，避免并发发布共享附件成环死锁；
  - `reference_count` 按引用行重算而非 ±1，历史漂移在下次写入时自愈；
  - 唯一约束冲突映射为具名 409 `forum.attachment_conflict`，附件失效映射为
    具名 422 `forum.attachment_unavailable`（`data.fields.content` 携带
    reason），两者都不再冒泡为不可诊断的 500。
- 修订快照读取统一按 `context = 'content'` 过滤并 `DISTINCT`，避免未来出现
  新 context 时污染附件快照。
- Restore 路径继续使用既有的 `ErrRevisionAttachmentUnavailable` 语义
  （历史快照与当前附件状态不符），与多值写入的新 422 分开，不合并。

## Consequences

- 多图发帖/评论自此可用；无数据迁移负担。
- 槽位型资源的一槽一图约束保持不变，站点品牌图/SEO 图/扩展图标上传路径
  无需改动（已核实其写入不依赖被删除的全局索引做 ON CONFLICT 推断）。
- 前端可针对 `forum.attachment_unavailable` 给出"重新上传这张图"的可操作
  提示，替代此前泛化的服务器错误。
- 若未来新增槽位型资源类型，必须同步扩展部分索引的 `IN` 列表，否则该资源
  将退化为多值语义；迁移 Down 的 fail-closed 行为会在回滚时暴露多值数据。

## References

- 根因迁移：`apps/api/database/migrations/202607110015_seo_attachment_reference_unique.sql`
- 修复迁移：`apps/api/database/migrations/202610260001_attachment_reference_scope_unique.sql`
- 写入实现：`apps/api/app/Models/Forum/attachment_references_postgres.go`
- 集成测试：`apps/api/app/Models/Forum/attachment_references_postgres_integration_test.go`
