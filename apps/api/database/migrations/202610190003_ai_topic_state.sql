-- +goose Up
-- +sforum OnlineSafe
-- B5：话题级会话状态。同一话题的多次回复共享「已查证事实」，减少重复查询与
-- 自我矛盾。
--
-- 它与聊天历史的根本区别在 document 的内容：只存通过 B4 校验的引用事实
-- （楼层 id + 从库里取到的引文），不存模型自由文本。模型的说法一旦进持久记忆，
-- 会在后续每一轮被当前提引用，而 B4 只校验「引用存在」，不校验「引用支持这个
-- 结论」（任务书 B5 易错点 1）。
--
-- revision 是乐观并发令牌，不是版本历史：写入用 UPDATE ... WHERE revision=$n，
-- 影响行数为 0 时放弃本次写入（记忆连续性 · 层 2 第 1 条）。后写者不一定是更
-- 正确的那个，因此这里不存在「后写覆盖先写」的策略。
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '1min';

CREATE TABLE ai_topic_state (
  topic_id BIGINT NOT NULL REFERENCES topics(id) ON DELETE CASCADE,
  bot_user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  document JSONB NOT NULL,
  revision BIGINT NOT NULL DEFAULT 1 CHECK (revision >= 1),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (topic_id, bot_user_id)
);

-- 过期清扫与运维排查：按更新时间倒序找陈旧状态。主键已覆盖「按话题 + 机器人
-- 读取」这一条主路径，因此不额外建 (topic_id) 索引。
CREATE INDEX ai_topic_state_updated_idx ON ai_topic_state (updated_at DESC);

-- +goose Down
SET LOCAL lock_timeout = '5s';

DROP INDEX IF EXISTS ai_topic_state_updated_idx;
DROP TABLE IF EXISTS ai_topic_state;
