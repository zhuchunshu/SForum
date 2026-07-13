-- +goose Up
ALTER TABLE extension_versions
  ADD COLUMN admin_frontend_digest TEXT NOT NULL DEFAULT ''
  CONSTRAINT extension_versions_admin_frontend_digest_check
  CHECK (admin_frontend_digest = '' OR admin_frontend_digest ~ '^[0-9a-f]{64}$');

-- Existing trusted Vue packages retain their exact package digest as a legacy
-- frontend digest. New installs compute the narrower digest from admin inputs.
UPDATE extension_versions
SET admin_frontend_digest = package_digest
WHERE manifest->'frontend'->'admin' IS NOT NULL
  AND package_digest ~ '^[0-9a-f]{64}$';

ALTER TABLE extension_frontend_trust_grants
  ADD COLUMN admin_frontend_digest TEXT;
UPDATE extension_frontend_trust_grants
SET admin_frontend_digest = package_digest;
ALTER TABLE extension_frontend_trust_grants
  ALTER COLUMN admin_frontend_digest SET NOT NULL,
  ADD CONSTRAINT extension_frontend_trust_grants_admin_digest_check
  CHECK (admin_frontend_digest ~ '^[0-9a-f]{64}$');

DROP INDEX extension_frontend_trust_grants_live_exact_idx;
CREATE UNIQUE INDEX extension_frontend_trust_grants_live_exact_idx
  ON extension_frontend_trust_grants (extension_id, extension_version, api_version, admin_frontend_digest)
  WHERE revoked_at IS NULL;

ALTER TABLE web_release_extensions
  ADD COLUMN admin_frontend_digest TEXT;
UPDATE web_release_extensions
SET admin_frontend_digest = package_digest;
ALTER TABLE web_release_extensions
  ALTER COLUMN admin_frontend_digest SET NOT NULL,
  ADD CONSTRAINT web_release_extensions_admin_digest_check
  CHECK (admin_frontend_digest ~ '^[0-9a-f]{64}$');

CREATE INDEX web_release_extensions_frontend_idx
  ON web_release_extensions (extension_id, extension_version, admin_frontend_digest);

-- +goose Down
DROP INDEX IF EXISTS web_release_extensions_frontend_idx;
ALTER TABLE web_release_extensions DROP COLUMN admin_frontend_digest;

DROP INDEX extension_frontend_trust_grants_live_exact_idx;
CREATE UNIQUE INDEX extension_frontend_trust_grants_live_exact_idx
  ON extension_frontend_trust_grants (extension_id, extension_version, package_digest)
  WHERE revoked_at IS NULL;
ALTER TABLE extension_frontend_trust_grants DROP COLUMN admin_frontend_digest;
ALTER TABLE extension_versions DROP COLUMN admin_frontend_digest;
