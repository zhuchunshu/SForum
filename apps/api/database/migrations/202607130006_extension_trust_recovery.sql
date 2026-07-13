-- +goose Up
-- V3 P1：一次性确认挑战只持久化 token 摘要；明文 token 仅返回给发起者。
CREATE TABLE extension_trust_challenges (
  id BIGSERIAL PRIMARY KEY,
  token_hash TEXT NOT NULL UNIQUE
    CHECK (token_hash ~ '^[0-9a-f]{64}$'),
  actor_user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  extension_id TEXT NOT NULL REFERENCES extensions(id) ON DELETE CASCADE,
  extension_version TEXT NOT NULL CHECK (extension_version <> ''),
  package_digest TEXT NOT NULL CHECK (package_digest ~ '^[0-9a-f]{64}$'),
  action TEXT NOT NULL
    CHECK (action IN ('enable', 'upgrade', 'frontend_import', 'authority_change')),
  artifact_digests JSONB NOT NULL DEFAULT '{}'::jsonb
    CHECK (jsonb_typeof(artifact_digests) = 'object'),
  impact_document JSONB NOT NULL
    CHECK (jsonb_typeof(impact_document) = 'object'),
  impact_digest TEXT NOT NULL CHECK (impact_digest ~ '^[0-9a-f]{64}$'),
  expires_at TIMESTAMPTZ NOT NULL,
  consumed_at TIMESTAMPTZ,
  invalidated_at TIMESTAMPTZ,
  invalidation_reason TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CHECK (expires_at > created_at),
  CHECK (consumed_at IS NULL OR invalidated_at IS NULL)
);

CREATE INDEX extension_trust_challenges_actor_created_idx
  ON extension_trust_challenges (actor_user_id, created_at DESC, id DESC);
CREATE INDEX extension_trust_challenges_extension_live_idx
  ON extension_trust_challenges (extension_id, expires_at DESC)
  WHERE consumed_at IS NULL AND invalidated_at IS NULL;

-- grant 是安全审计历史，扩展卸载后仍保留稳定身份和精确授权文档。
CREATE TABLE extension_trust_grants (
  id BIGSERIAL PRIMARY KEY,
  extension_id TEXT NOT NULL,
  extension_version TEXT NOT NULL CHECK (extension_version <> ''),
  package_digest TEXT NOT NULL CHECK (package_digest ~ '^[0-9a-f]{64}$'),
  action TEXT NOT NULL
    CHECK (action IN ('enable', 'upgrade', 'frontend_import', 'authority_change')),
  artifact_digests JSONB NOT NULL DEFAULT '{}'::jsonb
    CHECK (jsonb_typeof(artifact_digests) = 'object'),
  impact_document JSONB NOT NULL
    CHECK (jsonb_typeof(impact_document) = 'object'),
  impact_digest TEXT NOT NULL CHECK (impact_digest ~ '^[0-9a-f]{64}$'),
  granted_by_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
  granted_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  revoked_at TIMESTAMPTZ,
  revoked_by_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
  revocation_reason TEXT NOT NULL DEFAULT ''
);

CREATE UNIQUE INDEX extension_trust_grants_live_exact_idx
  ON extension_trust_grants (
    extension_id,
    extension_version,
    package_digest,
    action,
    impact_digest
  )
  WHERE revoked_at IS NULL;
CREATE INDEX extension_trust_grants_extension_created_idx
  ON extension_trust_grants (extension_id, granted_at DESC, id DESC);

-- append-only attempt 历史允许下一次启动识别上次遗留的 starting 记录并跳过故障摘要。
CREATE TABLE extension_activation_attempts (
  id BIGSERIAL PRIMARY KEY,
  extension_id TEXT NOT NULL REFERENCES extensions(id) ON DELETE CASCADE,
  extension_version TEXT NOT NULL CHECK (extension_version <> ''),
  package_digest TEXT NOT NULL CHECK (package_digest ~ '^[0-9a-f]{64}$'),
  boot_id TEXT NOT NULL CHECK (boot_id <> ''),
  trigger TEXT NOT NULL CHECK (trigger IN ('enable', 'startup')),
  status TEXT NOT NULL
    CHECK (status IN ('starting', 'healthy', 'failed', 'skipped')),
  actor_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
  failure_reason TEXT NOT NULL DEFAULT '',
  started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  completed_at TIMESTAMPTZ,
  CHECK ((status = 'starting' AND completed_at IS NULL)
      OR (status <> 'starting' AND completed_at IS NOT NULL))
);

CREATE UNIQUE INDEX extension_activation_attempts_one_open_idx
  ON extension_activation_attempts (extension_id)
  WHERE status = 'starting';
CREATE INDEX extension_activation_attempts_extension_started_idx
  ON extension_activation_attempts (extension_id, started_at DESC, id DESC);

-- +goose Down
DROP TABLE IF EXISTS extension_activation_attempts;
DROP TABLE IF EXISTS extension_trust_grants;
DROP TABLE IF EXISTS extension_trust_challenges;
