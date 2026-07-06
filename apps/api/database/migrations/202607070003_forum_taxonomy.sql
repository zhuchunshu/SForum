-- +goose Up
CREATE TABLE category_groups (
  id BIGSERIAL PRIMARY KEY,
  slug TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  visibility TEXT NOT NULL DEFAULT 'public' CHECK (visibility IN ('public', 'hidden')),
  position INTEGER NOT NULL DEFAULT 0,
  is_system BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE categories
  ADD COLUMN group_id BIGINT REFERENCES category_groups(id) ON DELETE RESTRICT,
  ADD COLUMN position INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN default_sort TEXT NOT NULL DEFAULT 'latest' CHECK (default_sort IN ('latest', 'hot'));

CREATE INDEX categories_group_position_idx ON categories (group_id, position, id);

CREATE TABLE tags (
  id BIGSERIAL PRIMARY KEY,
  slug TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'pending', 'disabled')),
  topic_count BIGINT NOT NULL DEFAULT 0 CHECK (topic_count >= 0),
  created_by_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
  reviewed_by_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
  reviewed_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX tags_status_name_idx ON tags (status, name, id);
CREATE INDEX tags_topic_count_idx ON tags (topic_count DESC, id DESC) WHERE status = 'active';

CREATE TABLE topic_tags (
  topic_id BIGINT NOT NULL REFERENCES topics(id) ON DELETE CASCADE,
  tag_id BIGINT NOT NULL REFERENCES tags(id) ON DELETE RESTRICT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (topic_id, tag_id)
);

CREATE INDEX topic_tags_tag_topic_idx ON topic_tags (tag_id, topic_id DESC);

INSERT INTO category_groups (slug, name, description, visibility, position, is_system)
VALUES ('default', '默认版块', '系统默认版块分组。', 'public', 0, TRUE)
ON CONFLICT (slug) DO NOTHING;

UPDATE categories
SET group_id = (SELECT id FROM category_groups WHERE slug = 'default')
WHERE group_id IS NULL;

ALTER TABLE categories
  ALTER COLUMN group_id SET NOT NULL;

INSERT INTO web_options (name, value)
VALUES
  ('forum.default_category_slug', 'general'),
  ('forum.tags.creation_mode', 'controlled'),
  ('forum.tags.public_pages', 'enabled'),
  ('forum.tags.max_per_topic', '5')
ON CONFLICT (name) DO NOTHING;

INSERT INTO permissions (key, module, description)
VALUES ('tag.manage', 'forum', 'Create, approve, disable, and manage tags.')
ON CONFLICT (key) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_key)
SELECT roles.id, 'tag.manage'
FROM roles
WHERE roles.key = 'super_admin'
ON CONFLICT DO NOTHING;

-- +goose Down
DELETE FROM role_permissions
WHERE permission_key = 'tag.manage';

DELETE FROM permissions
WHERE key = 'tag.manage';

DELETE FROM web_options
WHERE name IN (
  'forum.default_category_slug',
  'forum.tags.creation_mode',
  'forum.tags.public_pages',
  'forum.tags.max_per_topic'
);

DROP TABLE IF EXISTS topic_tags;
DROP INDEX IF EXISTS tags_topic_count_idx;
DROP INDEX IF EXISTS tags_status_name_idx;
DROP TABLE IF EXISTS tags;
DROP INDEX IF EXISTS categories_group_position_idx;
ALTER TABLE categories
  DROP COLUMN IF EXISTS default_sort,
  DROP COLUMN IF EXISTS position,
  DROP COLUMN IF EXISTS group_id;
DROP TABLE IF EXISTS category_groups;
