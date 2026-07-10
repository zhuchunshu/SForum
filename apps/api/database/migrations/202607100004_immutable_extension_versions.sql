-- +goose Up
ALTER TABLE extension_versions
  ADD COLUMN package_digest TEXT NOT NULL DEFAULT '';

ALTER TABLE extension_versions
  DROP CONSTRAINT extension_versions_extension_id_version_key;

ALTER TABLE extension_versions
  ADD CONSTRAINT extension_versions_extension_id_version_package_digest_key
  UNIQUE (extension_id, version, package_digest);

-- +goose Down
-- 回滚旧模型前，每个扩展版本只保留一条记录：当前活动版本优先，否则保留最新安装记录。
WITH ranked_versions AS (
  SELECT extension_versions.id,
    row_number() OVER (
      PARTITION BY extension_versions.extension_id, extension_versions.version
      ORDER BY
        (extensions.active_version_id = extension_versions.id) DESC,
        extension_versions.installed_at DESC,
        extension_versions.id DESC
    ) AS row_number
  FROM extension_versions
  JOIN extensions ON extensions.id = extension_versions.extension_id
)
DELETE FROM extension_versions
USING ranked_versions
WHERE extension_versions.id = ranked_versions.id
  AND ranked_versions.row_number > 1;

ALTER TABLE extension_versions
  DROP CONSTRAINT extension_versions_extension_id_version_package_digest_key;

ALTER TABLE extension_versions
  DROP COLUMN package_digest;

ALTER TABLE extension_versions
  ADD CONSTRAINT extension_versions_extension_id_version_key
  UNIQUE (extension_id, version);
