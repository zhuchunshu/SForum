-- +goose Up
CREATE TABLE extension_frontend_trust_grants (
  id BIGSERIAL PRIMARY KEY,
  -- 授权记录是安全审计历史；扩展卸载后仍保留稳定 ID 与摘要。
  extension_id TEXT NOT NULL,
  extension_version TEXT NOT NULL CHECK (extension_version <> ''),
  package_digest TEXT NOT NULL CHECK (package_digest ~ '^[0-9a-f]{64}$'),
  admin_frontend_digest TEXT NOT NULL CHECK (admin_frontend_digest ~ '^[0-9a-f]{64}$'),
  api_version INTEGER NOT NULL CHECK (api_version > 0),
  component_ids JSONB NOT NULL DEFAULT '[]'::jsonb
    CHECK (jsonb_typeof(component_ids) = 'array'),
  granted_by_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
  granted_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  revoked_at TIMESTAMPTZ,
  revoked_by_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL
);

CREATE UNIQUE INDEX extension_frontend_trust_grants_live_exact_idx
  ON extension_frontend_trust_grants (
    extension_id,
    extension_version,
    api_version,
    admin_frontend_digest
  )
  WHERE revoked_at IS NULL;

CREATE INDEX extension_frontend_trust_grants_extension_created_idx
  ON extension_frontend_trust_grants (extension_id, granted_at DESC, id DESC);

-- +goose Down
DROP TABLE IF EXISTS extension_frontend_trust_grants;
