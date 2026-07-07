-- +goose Up
-- 用户公开资料表。保持精简：不存放背景图、自定义代码、生日、手机、关注计数或游戏化字段。
CREATE TABLE user_profiles (
  user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  bio TEXT NOT NULL DEFAULT '',
  signature TEXT NOT NULL DEFAULT '',
  location TEXT NOT NULL DEFAULT '',
  website_url TEXT NOT NULL DEFAULT '',
  avatar_attachment_id BIGINT REFERENCES attachments(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- 每个用户注册后应有一条资料行；先建表，资料行在首次读取/更新时按需 upsert。
CREATE INDEX user_profiles_avatar_idx ON user_profiles (avatar_attachment_id)
  WHERE avatar_attachment_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS user_profiles_avatar_idx;
DROP TABLE IF EXISTS user_profiles;
