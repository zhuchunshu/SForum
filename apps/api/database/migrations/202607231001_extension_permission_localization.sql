-- +goose Up
-- 权限授权仍由 permission key 与 extension_permission_catalog 约束；这些字段只保存
-- 扩展声明的展示文案，供 Host 按请求语言解析。
ALTER TABLE permissions
  ADD COLUMN label TEXT NOT NULL DEFAULT '',
  ADD COLUMN label_locales JSONB NOT NULL DEFAULT '{}'::jsonb
    CHECK (jsonb_typeof(label_locales) = 'object'),
  ADD COLUMN description_locales JSONB NOT NULL DEFAULT '{}'::jsonb
    CHECK (jsonb_typeof(description_locales) = 'object');

-- 旧制品只声明了字符串时也补回 label；locale map 则保留完整扩展文案。
WITH localized AS (
  SELECT catalog.permission_key,
         definition.value -> 'label' AS label_value,
         definition.value -> 'description' AS description_value
  FROM extension_permission_catalog AS catalog
  JOIN extension_versions AS version
    ON version.id = catalog.extension_version_id
  CROSS JOIN LATERAL jsonb_array_elements(
    CASE
      WHEN jsonb_typeof(version.manifest -> 'permissionDefinitions') = 'array'
        THEN version.manifest -> 'permissionDefinitions'
      ELSE '[]'::jsonb
    END
  ) AS definition(value)
  WHERE definition.value ->> 'key' = catalog.permission_key
)
UPDATE permissions AS permission
SET label = CASE jsonb_typeof(localized.label_value)
      WHEN 'string' THEN localized.label_value #>> '{}'
      WHEN 'object' THEN COALESCE(
        localized.label_value ->> 'en-US',
        localized.label_value ->> 'en',
        (SELECT value FROM jsonb_each_text(localized.label_value) ORDER BY key LIMIT 1),
        permission.key
      )
      ELSE permission.key
    END,
    description = CASE jsonb_typeof(localized.description_value)
      WHEN 'string' THEN localized.description_value #>> '{}'
      WHEN 'object' THEN COALESCE(
        localized.description_value ->> 'en-US',
        localized.description_value ->> 'en',
        (SELECT value FROM jsonb_each_text(localized.description_value) ORDER BY key LIMIT 1),
        permission.description
      )
      ELSE permission.description
    END,
    label_locales = CASE
      WHEN jsonb_typeof(localized.label_value) = 'object' THEN localized.label_value
      ELSE '{}'::jsonb
    END,
    description_locales = CASE
      WHEN jsonb_typeof(localized.description_value) = 'object' THEN localized.description_value
      ELSE '{}'::jsonb
    END
FROM localized
WHERE permission.key = localized.permission_key;

-- +goose Down
ALTER TABLE permissions
  DROP COLUMN IF EXISTS description_locales,
  DROP COLUMN IF EXISTS label_locales,
  DROP COLUMN IF EXISTS label;
