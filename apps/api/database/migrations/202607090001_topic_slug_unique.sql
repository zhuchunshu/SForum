-- 帖子 slug 全局唯一化：支持 "纯 slug" URL 模式 (seo.topic_url_mode = slug)。
--
-- 背景：topics.slug 原仅有 UNIQUE(id, slug)（同一帖子改标题产生历史 slug 时
-- 仍保持唯一），以及非唯一索引 topics_slug_idx。纯 slug 模式要求 slug 全局唯一，
-- 否则 GET /topics/by-slug/:slug 会命中多行。
--
-- 本迁移：
--   1. 先对现有重复 slug 去重（每组保留最小 id，其余追加 "-<id>" 后缀，
--      "-<id>" 因 id 全局唯一故新 slug 互不冲突）；
--   2. 再将 topics_slug_idx 升级为 UNIQUE INDEX。
-- 注意：id_slug / id 模式不依赖此约束，但全局唯一 slug 对它们也无害。

-- +goose Up
-- 去重：对每个重复 slug 组，保留最小 id，其余行追加 "-<id>"。
UPDATE topics t
SET slug = t.slug || '-' || t.id::text,
    updated_at = now()
WHERE t.id <> (
    SELECT MIN(id) FROM topics sub WHERE sub.slug = t.slug
);

-- 升级为全局唯一索引。先删旧的非唯一索引，再建唯一索引。
DROP INDEX IF EXISTS topics_slug_idx;
CREATE UNIQUE INDEX topics_slug_idx ON topics (slug);

-- +goose Down
-- 回退：唯一索引恢复为普通索引。无法还原去重时追加的后缀（数据变更不可逆），
-- 仅恢复结构。
DROP INDEX IF EXISTS topics_slug_idx;
CREATE INDEX topics_slug_idx ON topics (slug);
