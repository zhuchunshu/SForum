-- +goose Up
-- 密码重置令牌表。令牌以哈希形式存储，单次使用，带过期时间。
CREATE TABLE password_reset_tokens (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  token_hash TEXT NOT NULL UNIQUE,
  expires_at TIMESTAMPTZ NOT NULL,
  consumed_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  request_ip_hash TEXT NOT NULL DEFAULT ''
);

CREATE INDEX password_reset_tokens_user_idx ON password_reset_tokens (user_id, created_at DESC);
CREATE INDEX password_reset_tokens_expires_idx ON password_reset_tokens (expires_at)
  WHERE consumed_at IS NULL;

-- 邮件运行时选项默认值。
INSERT INTO web_options (name, value)
VALUES
  ('mail.provider', 'dev_log'),
  ('mail.from_address', 'noreply@example.com'),
  ('mail.from_name', 'SForum'),
  ('mail.smtp.host', ''),
  ('mail.smtp.port', '587'),
  ('mail.smtp.username', ''),
  ('mail.smtp.password', ''),
  ('mail.smtp.encryption', 'starttls')
ON CONFLICT (name) DO NOTHING;

-- +goose Down
DELETE FROM web_options
WHERE name IN (
  'mail.provider',
  'mail.from_address',
  'mail.from_name',
  'mail.smtp.host',
  'mail.smtp.port',
  'mail.smtp.username',
  'mail.smtp.password',
  'mail.smtp.encryption'
);

DROP INDEX IF EXISTS password_reset_tokens_expires_idx;
DROP INDEX IF EXISTS password_reset_tokens_user_idx;
DROP TABLE IF EXISTS password_reset_tokens;
