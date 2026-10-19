-- +goose Up
-- +sforum OnlineSafe
-- B3：AI 派生内容（话题蒸馏 forum.digest、长楼导览 forum.tour）的产物表。
--
-- 为什么需要这张表：任务书要求产物「由运营者预览后手动发布，或直接落成草稿，
-- 默认不自动发布」，而 SForum 没有服务端草稿实体——comments.status 受 CHECK
-- 约束限定为 active/hidden/deleted/pending/rejected，前端的「草稿」是浏览器
-- localStorage（sforum.topic-composer.draft.v1），纯客户端。见任务书 D15。
--
-- 为什么不用 status='pending' 的评论当草稿（被否决的方案）：pending 是审核专用，
-- 审核台以 WHERE status='pending' 列出待审并发送待审通知，AI 草稿混进去之后
-- 审核员无法与垃圾内容区分；更严重的是那会把 SubmitDecision 变成 AI 发布器，
-- 与硬约束 2 「SubmitDecision 语义不变」直接冲突。
--
-- 为什么 content 不写进 ai_executions：那张表声明「不含用户内容」。派生产物是
-- 模型文本，属于用户内容，只能进这张表（任务书硬约束 1）。
--
-- 与本表的关系边界：本表不写 comments / topics。手动触发的运营动作不该在公共
-- 内容里留下痕迹，也不该被审核、通知、索引、修订任何一条已有链路看见。
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '1min';

CREATE TABLE ai_derived_artifacts (
  id BIGSERIAL PRIMARY KEY,
  -- 用途 id：只允许登记表里的派生用途。写约束而不是注释，是因为这张表的
  -- purpose 会出现在运维查询与将来的控制台列表里，一个拼错的 id 会变成一条
  -- 永远匹配不上任何用途声明的孤儿产物。
  purpose TEXT NOT NULL CHECK (purpose IN ('forum.digest', 'forum.tour')),
  -- 派生对象。不带外键：话题被删除后产物仍是一份可读的历史草稿，
  -- 而 ON DELETE CASCADE 会让「运营者刚看过的那份摘要」随话题一起静默消失。
  topic_id BIGINT NOT NULL CHECK (topic_id > 0),
  -- 产出这条草稿的 run id。单次补全没有 run（见 completer.go 的口径），
  -- 因此允许空串：它表达「这次没有可聚合的 run」，不是「忘了写」。
  run_id TEXT NOT NULL DEFAULT '',
  prompt_version TEXT NOT NULL,
  -- 本包只会写 draft。published / discarded 先写进约束，是为了让「这张表能表达
  -- 什么」在迁移里就说清楚，而不是等第一次发布时再 ALTER。
  status TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'published', 'discarded')),
  content TEXT NOT NULL CHECK (content <> ''),
  -- 触发这次派生的运营者。SET NULL 而不是 CASCADE：账号删除不该抹掉审计归属以外
  -- 的产物本身，而产物仍属于站点。
  created_by_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- 控制台/CLI 的主查询：按用途列最近的草稿。
CREATE INDEX ai_derived_artifacts_purpose_idx
  ON ai_derived_artifacts (purpose, created_at DESC, id DESC);
-- 按话题反查：运营者看着一栋楼想知道「之前给它生成过什么」。
CREATE INDEX ai_derived_artifacts_topic_idx
  ON ai_derived_artifacts (topic_id, created_at DESC, id DESC);
-- 待处置队列：只看还没被发布/丢弃的草稿。
CREATE INDEX ai_derived_artifacts_status_idx
  ON ai_derived_artifacts (status, created_at DESC);

-- +goose Down
SET LOCAL lock_timeout = '5s';

-- 先删索引再删表：中途失败时 Down 仍可重放。
DROP INDEX IF EXISTS ai_derived_artifacts_status_idx;
DROP INDEX IF EXISTS ai_derived_artifacts_topic_idx;
DROP INDEX IF EXISTS ai_derived_artifacts_purpose_idx;
DROP TABLE IF EXISTS ai_derived_artifacts;
