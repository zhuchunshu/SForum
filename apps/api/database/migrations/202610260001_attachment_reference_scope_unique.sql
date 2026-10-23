-- +goose Up
-- 附件引用的唯一性分两类语义：
--   多值型（topic/comment）：一条内容可以绑定多个附件，唯一性由基表
--     UNIQUE (attachment_id, resource_type, resource_id, context) 保证。
--   槽位型（site/seo/user/extension）：一个命名槽位只能有一张图，靠
--     (resource_type, resource_id, context) 唯一索引保证。
-- 202607110015 把槽位型唯一性套在了所有资源上，导致正文第二张图插入即
-- 违反唯一约束（论坛侧表现为 500 internal_error）。这里把它收窄到槽位型。
DROP INDEX IF EXISTS attachment_references_resource_context_unique;

CREATE UNIQUE INDEX attachment_references_slot_context_unique
  ON attachment_references (resource_type, resource_id, context)
  WHERE resource_type IN ('site', 'seo', 'user', 'extension');

-- +goose Down
-- 回滚会重新引入「一帖只能一张图」的缺陷；若库中已存在多附件引用，
-- 重建全局唯一索引会失败。这是有意的 fail-closed 行为，不给隐性回归留空间。
DROP INDEX IF EXISTS attachment_references_slot_context_unique;

CREATE UNIQUE INDEX attachment_references_resource_context_unique
  ON attachment_references (resource_type, resource_id, context);
