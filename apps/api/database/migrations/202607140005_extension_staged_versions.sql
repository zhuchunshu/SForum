-- +goose Up
-- V3 P4：静态上传只登记候选制品。现有 active_version_id 继续代表当前生效制品，
-- 生命周期协调器成功提交升级后才允许切换它。
ALTER TABLE extensions
  ADD COLUMN staged_version_id BIGINT;

ALTER TABLE extension_versions
  ADD CONSTRAINT extension_versions_extension_id_id_key
  UNIQUE (extension_id, id);

ALTER TABLE extensions
  ADD CONSTRAINT extensions_staged_version_fk
  FOREIGN KEY (id, staged_version_id)
  REFERENCES extension_versions(extension_id, id)
  ON DELETE SET NULL (staged_version_id);

CREATE INDEX extensions_staged_version_idx
  ON extensions (staged_version_id)
  WHERE staged_version_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS extensions_staged_version_idx;

ALTER TABLE extensions
  DROP CONSTRAINT IF EXISTS extensions_staged_version_fk;

ALTER TABLE extensions
  DROP COLUMN IF EXISTS staged_version_id;

ALTER TABLE extension_versions
  DROP CONSTRAINT IF EXISTS extension_versions_extension_id_id_key;
