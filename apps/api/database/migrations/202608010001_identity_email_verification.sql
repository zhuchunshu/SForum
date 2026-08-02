-- +goose Up
-- +sforum OnlineSafe
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '5min';

ALTER TABLE users ADD COLUMN email_verified_at TIMESTAMPTZ;

-- Existing accounts predate verification state and must not be locked out when
-- an operator enables the policy after upgrading.
UPDATE users SET email_verified_at = created_at WHERE email_verified_at IS NULL;

CREATE TABLE email_verification_tokens (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  email TEXT NOT NULL,
  token_hash TEXT NOT NULL UNIQUE,
  expires_at TIMESTAMPTZ NOT NULL,
  consumed_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  request_ip_hash TEXT NOT NULL DEFAULT ''
);

CREATE INDEX email_verification_tokens_user_idx
  ON email_verification_tokens (user_id, created_at DESC);
CREATE INDEX email_verification_tokens_expires_idx
  ON email_verification_tokens (expires_at)
  WHERE consumed_at IS NULL;

-- +goose Down
DROP INDEX IF EXISTS email_verification_tokens_expires_idx;
DROP INDEX IF EXISTS email_verification_tokens_user_idx;
DROP TABLE IF EXISTS email_verification_tokens;
ALTER TABLE users DROP COLUMN IF EXISTS email_verified_at;
