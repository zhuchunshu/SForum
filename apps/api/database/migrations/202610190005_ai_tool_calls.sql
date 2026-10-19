-- +goose Up
-- +sforum OnlineSafe
-- AI 产物的一次工具调用记录（W2 · 可见性面）。
--
-- 为什么不进 ai_executions：那张表声明「不含用户内容」（见 Support/AI 的
-- ExecutionRecord 注释）。arguments 是模型输出、result_digest 是工具结果片段，
-- 两者都是用户内容，只能进这张新表。与 ai_citations 同一条理由。
--
-- 为什么需要 seq：ToolTraceEntry.Step 是「第几次模型调用」，同一条 assistant
-- 消息里的并行工具调用共享同一个 step。用它做幂等键会把并行调用压成一行，
-- 所以幂等键是 (comment_id, seq)，seq 是 trace 切片下标。
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '1min';

CREATE TABLE ai_tool_calls (
  id BIGSERIAL PRIMARY KEY,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  -- 与 ai_executions(resource_type='ai.run', resource_id) 同一个 run id。
  run_id TEXT NOT NULL DEFAULT '',
  topic_id BIGINT NOT NULL DEFAULT 0,
  -- AI 产出的那条评论。发布成功之后才写，因此它恒指向一条已存在的评论；
  -- 评论随主题级联删除时记录一并消失（ON DELETE CASCADE），不留孤儿行。
  comment_id BIGINT NOT NULL
    REFERENCES comments(id) ON DELETE CASCADE,
  -- trace 切片下标（同一次 run 内从 0 开始，含被拒绝的调用）。
  seq INT NOT NULL,
  -- 第几次模型调用（RunResult.ModelCalls）。只用于 admin 回放，不参与幂等。
  step INT NOT NULL DEFAULT 0,
  tool TEXT NOT NULL,
  -- 写入侧已按站点脱敏规则处理并截断（MaxToolTraceArgumentsBytes = 1 KiB）。
  arguments TEXT NOT NULL DEFAULT '',
  arguments_bytes INT NOT NULL DEFAULT 0,
  result_bytes INT NOT NULL DEFAULT 0,
  result_digest TEXT NOT NULL DEFAULT '',
  is_error BOOLEAN NOT NULL DEFAULT false,
  -- 空串表示模型自己拿到了结果；非空是五种结局之一（见 Support/AI/tool_trace.go）。
  -- 公开读会过滤掉拒绝类结局，admin 读保留全部。
  failure TEXT NOT NULL DEFAULT '',
  latency_ms INT NOT NULL DEFAULT 0,
  CONSTRAINT ai_tool_calls_comment_seq_key UNIQUE (comment_id, seq)
);

-- 读取主路径：按 AI 产出评论取工具调用，顺序即写入顺序。
CREATE INDEX ai_tool_calls_comment_idx ON ai_tool_calls (comment_id, seq);
-- 回放与归因：按 run 取这次回复调过哪些工具（admin 时间线）。
CREATE INDEX ai_tool_calls_run_idx ON ai_tool_calls (run_id, created_at DESC);

-- +goose Down
SET LOCAL lock_timeout = '5s';

-- 先删索引再删表：中途失败时 Down 仍可重放。
DROP INDEX IF EXISTS ai_tool_calls_run_idx;
DROP INDEX IF EXISTS ai_tool_calls_comment_idx;
DROP TABLE IF EXISTS ai_tool_calls;
