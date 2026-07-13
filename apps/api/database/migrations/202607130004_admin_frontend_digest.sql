-- +goose Up
ALTER TABLE extension_versions
  ADD COLUMN IF NOT EXISTS admin_frontend_digest TEXT NOT NULL DEFAULT '';

-- +goose StatementBegin
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint
    WHERE conname = 'extension_versions_admin_frontend_digest_check'
  ) THEN
    ALTER TABLE extension_versions
      ADD CONSTRAINT extension_versions_admin_frontend_digest_check
      CHECK (admin_frontend_digest = '' OR admin_frontend_digest ~ '^[0-9a-f]{64}$');
  END IF;
END $$;
-- +goose StatementEnd

-- 兼容已经跑过开发期旧迁移、但尚未应用本迁移的本地数据库。
ALTER TABLE extension_frontend_trust_grants
  ADD COLUMN IF NOT EXISTS admin_frontend_digest TEXT;
UPDATE extension_frontend_trust_grants
SET admin_frontend_digest = package_digest
WHERE admin_frontend_digest IS NULL;
ALTER TABLE extension_frontend_trust_grants
  ALTER COLUMN admin_frontend_digest SET NOT NULL;

DROP INDEX IF EXISTS extension_frontend_trust_grants_live_exact_idx;
CREATE UNIQUE INDEX extension_frontend_trust_grants_live_exact_idx
  ON extension_frontend_trust_grants (
    extension_id,
    extension_version,
    api_version,
    admin_frontend_digest
  )
  WHERE revoked_at IS NULL;

-- +goose Down
DROP INDEX IF EXISTS extension_frontend_trust_grants_live_exact_idx;
CREATE UNIQUE INDEX extension_frontend_trust_grants_live_exact_idx
  ON extension_frontend_trust_grants (extension_id, extension_version, package_digest)
  WHERE revoked_at IS NULL;
ALTER TABLE extension_frontend_trust_grants DROP COLUMN IF EXISTS admin_frontend_digest;
ALTER TABLE extension_versions DROP COLUMN IF EXISTS admin_frontend_digest;
