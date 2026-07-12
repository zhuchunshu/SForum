-- +goose Up
-- 个人访问令牌（PAT / API tokens，F3.4）。只存哈希，明文仅创建时返回一次。
CREATE TABLE api_tokens (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  -- public_id 用于查找，非密钥；完整令牌 = sft_<public_id>_<secret>
  public_id TEXT NOT NULL UNIQUE,
  token_hash TEXT NOT NULL,
  name TEXT NOT NULL,
  -- scopes: 权限 key 数组；请求时与用户当前权限取交集
  scopes JSONB NOT NULL DEFAULT '[]'::jsonb,
  last_used_at TIMESTAMPTZ,
  expires_at TIMESTAMPTZ,
  revoked_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX api_tokens_user_created_idx ON api_tokens (user_id, created_at DESC);
CREATE INDEX api_tokens_public_id_active_idx ON api_tokens (public_id) WHERE revoked_at IS NULL;

-- +goose Down
DROP INDEX IF EXISTS api_tokens_public_id_active_idx;
DROP INDEX IF EXISTS api_tokens_user_created_idx;
DROP TABLE IF EXISTS api_tokens;
