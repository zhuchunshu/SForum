-- +goose Up
-- +sforum OnlineSafe
-- 回复结果：一条触发最后有没有变成一条评论，以及为什么没有。
--
-- 为什么单开一张表而不是往 ai_executions 里加行：那张表的语义是「一次模型调用」
-- （provider/profile/token/延迟/闸门），而一次回复可能包含多次调用、也可能一次
-- 都没调用就被闸门或空回复挡住。把两种事实塞进同一张表，会让「调用成功」继续
-- 冒充「回复已发布」——那正是这次要修的故障。
--
-- 表里没有提示词与响应正文：定位只需要「哪条触发、结果、原因」。正文可能包含
-- 用户内容，detail 只放稳定短语（停止原因、失败码、调用次数）。
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '1min';

CREATE TABLE ai_reply_outcomes (
  id                  BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
  topic_id            BIGINT NOT NULL,
  parent_comment_id   BIGINT NOT NULL DEFAULT 0,
  trigger_comment_id  BIGINT NOT NULL DEFAULT 0,
  bot_user_id         BIGINT NOT NULL DEFAULT 0,
  run_id              TEXT NOT NULL DEFAULT '',
  comment_id          BIGINT NOT NULL DEFAULT 0,
  outcome             TEXT NOT NULL,
  reason              TEXT NOT NULL DEFAULT '',
  detail              TEXT NOT NULL DEFAULT '',
  model_calls         INTEGER NOT NULL DEFAULT 0,
  tool_calls          INTEGER NOT NULL DEFAULT 0,
  input_tokens        INTEGER NOT NULL DEFAULT 0,
  output_tokens       INTEGER NOT NULL DEFAULT 0,
  CONSTRAINT ai_reply_outcomes_outcome_check CHECK (
    outcome IN ('posted', 'empty_reply', 'tool_loop_stuck', 'unavailable', 'publish_failed', 'failed')
  )
);

-- 控制台按时间倒序读最近 N 条。
CREATE INDEX ai_reply_outcomes_created_idx
  ON ai_reply_outcomes (created_at DESC, id DESC);

-- 幂等判据：同一条触发只应有一条 posted。
CREATE INDEX ai_reply_outcomes_trigger_idx
  ON ai_reply_outcomes (topic_id, trigger_comment_id, outcome)
  WHERE outcome = 'posted';

-- 结果行跳回 run 回放。
CREATE INDEX ai_reply_outcomes_run_idx
  ON ai_reply_outcomes (run_id)
  WHERE run_id <> '';

-- +goose Down
SET LOCAL lock_timeout = '5s';

DROP INDEX IF EXISTS ai_reply_outcomes_run_idx;
DROP INDEX IF EXISTS ai_reply_outcomes_trigger_idx;
DROP INDEX IF EXISTS ai_reply_outcomes_created_idx;
DROP TABLE IF EXISTS ai_reply_outcomes;
