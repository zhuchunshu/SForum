-- +goose Up
-- 清理早期表单校验缺失造成的空白自定义用户组，避免前端无法通过空路径段删除。
DELETE FROM user_roles
WHERE role_id IN (
  SELECT id
  FROM roles
  WHERE is_system = FALSE
    AND is_deletable = TRUE
    AND (btrim(key) = '' OR btrim(alias) = '')
);

DELETE FROM roles
WHERE is_system = FALSE
  AND is_deletable = TRUE
  AND (btrim(key) = '' OR btrim(alias) = '');

ALTER TABLE roles
  ADD CONSTRAINT roles_key_not_blank CHECK (btrim(key) <> ''),
  ADD CONSTRAINT roles_alias_not_blank CHECK (btrim(alias) <> '');

-- +goose Down
ALTER TABLE roles
  DROP CONSTRAINT IF EXISTS roles_alias_not_blank,
  DROP CONSTRAINT IF EXISTS roles_key_not_blank;
