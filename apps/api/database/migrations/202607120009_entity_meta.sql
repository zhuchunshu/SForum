-- +goose Up
-- F4.4：宿主拥有的实体自定义字段定义与稀疏值表。
-- 插件不得对核心表做任意 ALTER；字段定义由运营/API 管理。

CREATE TABLE entity_field_definitions (
  id BIGSERIAL PRIMARY KEY,
  -- 稳定 key：小写字母数字与点/下划线，全局唯一。
  field_key TEXT NOT NULL,
  -- user | topic（v1）
  entity_type TEXT NOT NULL,
  -- string | text | number | boolean
  value_type TEXT NOT NULL,
  -- public | owner | admin
  visibility TEXT NOT NULL DEFAULT 'public',
  label_zh_cn TEXT NOT NULL DEFAULT '',
  label_en_us TEXT NOT NULL DEFAULT '',
  description_zh_cn TEXT NOT NULL DEFAULT '',
  description_en_us TEXT NOT NULL DEFAULT '',
  -- 可选：声明来源扩展（仅元数据，不授予写权）
  owner_extension_id TEXT NOT NULL DEFAULT '',
  required BOOLEAN NOT NULL DEFAULT FALSE,
  enabled BOOLEAN NOT NULL DEFAULT TRUE,
  sort_order INT NOT NULL DEFAULT 100,
  -- 可选约束 JSON：{"maxLength":200,"min":0,"max":100}
  constraints JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT entity_field_definitions_key_unique UNIQUE (field_key),
  CONSTRAINT entity_field_definitions_entity_check CHECK (entity_type IN ('user', 'topic')),
  CONSTRAINT entity_field_definitions_value_check CHECK (value_type IN ('string', 'text', 'number', 'boolean')),
  CONSTRAINT entity_field_definitions_visibility_check CHECK (visibility IN ('public', 'owner', 'admin'))
);

CREATE INDEX entity_field_definitions_entity_enabled_idx
  ON entity_field_definitions (entity_type, enabled, sort_order, field_key);

CREATE TABLE entity_meta_values (
  entity_type TEXT NOT NULL,
  entity_id BIGINT NOT NULL,
  field_key TEXT NOT NULL REFERENCES entity_field_definitions (field_key) ON DELETE CASCADE ON UPDATE CASCADE,
  -- 校验后的规范化字符串；boolean 存 "true"/"false"，number 存十进制文本。
  value_text TEXT NOT NULL DEFAULT '',
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_by_user_id BIGINT REFERENCES users (id) ON DELETE SET NULL,
  PRIMARY KEY (entity_type, entity_id, field_key),
  CONSTRAINT entity_meta_values_entity_check CHECK (entity_type IN ('user', 'topic'))
);

CREATE INDEX entity_meta_values_entity_idx ON entity_meta_values (entity_type, entity_id);
CREATE INDEX entity_meta_values_field_idx ON entity_meta_values (field_key);

-- 定义管理权限（F4.4）。
INSERT INTO permissions (key, module, description) VALUES
  ('entity_meta.manage', 'admin', 'Manage entity custom field definitions and admin-visible meta values.')
ON CONFLICT (key) DO UPDATE SET module = EXCLUDED.module, description = EXCLUDED.description;

INSERT INTO role_permissions (role_id, permission_key)
SELECT roles.id, 'entity_meta.manage'
FROM roles
WHERE roles.key = 'super_admin'
ON CONFLICT DO NOTHING;

-- 持有 settings.manage 的角色获得 entity_meta.manage（升级不收权）。
INSERT INTO role_permissions (role_id, permission_key)
SELECT DISTINCT rp.role_id, 'entity_meta.manage'
FROM role_permissions rp
WHERE rp.permission_key = 'settings.manage'
ON CONFLICT DO NOTHING;

-- +goose Down
DELETE FROM role_permissions WHERE permission_key = 'entity_meta.manage';
DELETE FROM permissions WHERE key = 'entity_meta.manage';
DROP INDEX IF EXISTS entity_meta_values_field_idx;
DROP INDEX IF EXISTS entity_meta_values_entity_idx;
DROP TABLE IF EXISTS entity_meta_values;
DROP INDEX IF EXISTS entity_field_definitions_entity_enabled_idx;
DROP TABLE IF EXISTS entity_field_definitions;
