-- +goose Up
-- 机器人账号与人类账号共用 users 表：它们占用真实用户名额、计入在线人数、可被
-- @、可被举报。用独立的影子表模拟会让这些既有路径到处特判，因此只加一个类型
-- 列，语义差异由该列驱动。
ALTER TABLE users ADD COLUMN kind TEXT NOT NULL DEFAULT 'human';
ALTER TABLE users ADD CONSTRAINT users_kind_check CHECK (kind IN ('human', 'bot'));

-- 站点上的机器人数量很少：部分索引只覆盖它们，不影响人类用户的索引体积。
CREATE INDEX users_bot_kind_idx ON users (kind) WHERE kind = 'bot';

-- +goose Down
DROP INDEX IF EXISTS users_bot_kind_idx;
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_kind_check;
ALTER TABLE users DROP COLUMN IF EXISTS kind;
