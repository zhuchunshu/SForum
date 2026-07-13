-- +goose Up
-- SForum 尚未发布：移除已退役的运行时前端构建/发布数据面，只保留 digest 信任授权。

DELETE FROM web_options
WHERE name IN (
  'web_release.typecheck_fail',
  'web_release.typecheck_mode',
  'themes.layer_activation_enabled'
);

DELETE FROM user_permission_overrides
WHERE permission_key = 'extension.release.manage';
DELETE FROM role_permissions
WHERE permission_key = 'extension.release.manage';
DELETE FROM permissions
WHERE key = 'extension.release.manage';

ALTER TABLE IF EXISTS extension_theme_releases
  DROP COLUMN IF EXISTS web_release_id;
DROP TABLE IF EXISTS extension_theme_releases;
DROP TABLE IF EXISTS web_release_events;
DROP TABLE IF EXISTS web_release_extension_effects;
DROP TABLE IF EXISTS web_release_extensions;
DROP TABLE IF EXISTS web_releases;
DROP SEQUENCE IF EXISTS web_release_generation_seq;

-- +goose StatementBegin
DO $$
DECLARE
  constraint_name TEXT;
BEGIN
  FOR constraint_name IN
    SELECT con.conname
    FROM pg_constraint con
    JOIN pg_class rel ON rel.oid = con.conrelid
    WHERE rel.relname = 'extension_frontend_trust_grants'
      AND con.contype = 'c'
      AND pg_get_constraintdef(con.oid) LIKE '%revocation_requested_at%'
  LOOP
    EXECUTE format('ALTER TABLE extension_frontend_trust_grants DROP CONSTRAINT %I', constraint_name);
  END LOOP;
END $$;
-- +goose StatementEnd

ALTER TABLE extension_frontend_trust_grants
  DROP COLUMN IF EXISTS contribution_points,
  DROP COLUMN IF EXISTS revocation_requested_at,
  DROP COLUMN IF EXISTS revocation_requested_by_user_id;

-- +goose Down
-- 旧发布表承载的是已删除的运行时构建架构，Down 不复活该执行面。
