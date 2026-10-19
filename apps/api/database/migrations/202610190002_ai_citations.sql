-- +goose Up
-- +sforum OnlineSafe
-- B4：AI 产物的证据链。每条引用指向一个真实存在且对访客可见的站内楼层，
-- 由生成侧在发布之前校验、发布之后写入。
--
-- 为什么不进 ai_executions：那张表声明「不含用户内容」。excerpt 是被引楼层的
-- 正文片段，属于用户内容，只能进这张新表（任务书硬约束 1 与全局陷阱 8）。
--
-- source_comment_id 是任务书表结构之外的第四列（见任务书「偏离登记」D6）：
-- 渲染要求是「在 AI 产出评论下显示依据」，而 run_id 无法承担这条边——一次 run
-- 产出一条评论是当前事实，不是契约（重试与将来的批量用途会变成 1:N）。该列是
-- 后端内部列，不进入对前端的 JSON 契约。
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '1min';

CREATE TABLE ai_citations (
  id BIGSERIAL PRIMARY KEY,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  run_id TEXT NOT NULL DEFAULT '',
  topic_id BIGINT NOT NULL DEFAULT 0,
  comment_id BIGINT NOT NULL DEFAULT 0,
  excerpt TEXT NOT NULL DEFAULT '',
  -- AI 产出的那条评论。发布成功之后才写，因此它恒指向一条已存在的公开评论；
  -- 评论随主题级联删除时依据一并消失（ON DELETE CASCADE），不留孤儿行。
  --
  -- 刻意可空且不带 DEFAULT 0：外键列给一个 0 默认值会让任何省略该列的 INSERT
  -- 直接违反外键（comments 里没有 id=0 的行）。旧行语义由 NULL 表达「未知」，
  -- 与 0 区分开——0 在读取侧会被当作「没有产出评论」而静默漏掉。
  source_comment_id BIGINT
    REFERENCES comments(id) ON DELETE CASCADE
);

-- 回放与归因：按 run 取这次回复引用过什么。
CREATE INDEX ai_citations_run_idx ON ai_citations (run_id, created_at DESC);
-- 反向审计：某个楼层被多少条 AI 回复当作依据（楼层被隐藏时的影响面）。
CREATE INDEX ai_citations_target_idx ON ai_citations (topic_id, comment_id);
-- 读取主路径：按 AI 产出评论取依据，顺序即写入顺序。
CREATE INDEX ai_citations_source_idx ON ai_citations (source_comment_id, id);

-- +goose Down
SET LOCAL lock_timeout = '5s';

-- 先删索引再删表：中途失败时 Down 仍可重放。
DROP INDEX IF EXISTS ai_citations_source_idx;
DROP INDEX IF EXISTS ai_citations_target_idx;
DROP INDEX IF EXISTS ai_citations_run_idx;
DROP TABLE IF EXISTS ai_citations;
