-- +goose Up
CREATE TABLE categories (
  id BIGSERIAL PRIMARY KEY,
  slug TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  visibility TEXT NOT NULL DEFAULT 'public' CHECK (visibility IN ('public', 'hidden')),
  is_system BOOLEAN NOT NULL DEFAULT FALSE,
  topic_count BIGINT NOT NULL DEFAULT 0 CHECK (topic_count >= 0),
  comment_count BIGINT NOT NULL DEFAULT 0 CHECK (comment_count >= 0),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE posts (
  id BIGSERIAL PRIMARY KEY,
  raw_content TEXT NOT NULL,
  html_content TEXT NOT NULL,
  plain_text TEXT NOT NULL,
  excerpt TEXT NOT NULL DEFAULT '',
  source_format TEXT NOT NULL DEFAULT 'markdown' CHECK (source_format IN ('markdown', 'html', 'json')),
  editor_type TEXT NOT NULL DEFAULT 'markdown',
  editor_version TEXT NOT NULL DEFAULT '',
  render_version TEXT NOT NULL DEFAULT '',
  content_hash TEXT NOT NULL,
  created_by_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
  updated_by_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX posts_created_by_idx ON posts (created_by_user_id, created_at DESC);
CREATE INDEX posts_content_hash_idx ON posts (content_hash);

CREATE TABLE topics (
  id BIGSERIAL PRIMARY KEY,
  category_id BIGINT NOT NULL REFERENCES categories(id) ON DELETE RESTRICT,
  author_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
  content_id BIGINT NOT NULL REFERENCES posts(id) ON DELETE RESTRICT,
  title TEXT NOT NULL,
  slug TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'locked', 'hidden', 'deleted')),
  is_pinned BOOLEAN NOT NULL DEFAULT FALSE,
  pinned_at TIMESTAMPTZ,
  locked_at TIMESTAMPTZ,
  comment_count BIGINT NOT NULL DEFAULT 0 CHECK (comment_count >= 0),
  view_count BIGINT NOT NULL DEFAULT 0 CHECK (view_count >= 0),
  last_activity_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  deleted_at TIMESTAMPTZ,
  UNIQUE (id, slug)
);

CREATE INDEX topics_category_activity_idx ON topics (category_id, is_pinned DESC, last_activity_at DESC, id DESC)
  WHERE status IN ('active', 'locked');
CREATE INDEX topics_author_created_idx ON topics (author_user_id, created_at DESC);
CREATE INDEX topics_slug_idx ON topics (slug);

CREATE TABLE comments (
  id BIGSERIAL PRIMARY KEY,
  topic_id BIGINT NOT NULL REFERENCES topics(id) ON DELETE CASCADE,
  content_id BIGINT NOT NULL REFERENCES posts(id) ON DELETE RESTRICT,
  author_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
  parent_comment_id BIGINT REFERENCES comments(id) ON DELETE SET NULL,
  root_comment_id BIGINT REFERENCES comments(id) ON DELETE SET NULL,
  path_key TEXT NOT NULL DEFAULT '',
  depth INTEGER NOT NULL DEFAULT 0 CHECK (depth >= 0),
  reply_count BIGINT NOT NULL DEFAULT 0 CHECK (reply_count >= 0),
  status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'hidden', 'deleted')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  deleted_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX comments_topic_path_idx ON comments (topic_id, path_key);
CREATE INDEX comments_topic_root_idx ON comments (topic_id, root_comment_id, path_key)
  WHERE status = 'active';
CREATE INDEX comments_parent_idx ON comments (parent_comment_id, path_key)
  WHERE status = 'active';
CREATE INDEX comments_author_created_idx ON comments (author_user_id, created_at DESC);

CREATE TABLE post_revisions (
  id BIGSERIAL PRIMARY KEY,
  post_id BIGINT NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
  edited_by_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
  raw_content TEXT NOT NULL,
  html_content TEXT NOT NULL,
  plain_text TEXT NOT NULL,
  excerpt TEXT NOT NULL DEFAULT '',
  source_format TEXT NOT NULL,
  editor_type TEXT NOT NULL,
  editor_version TEXT NOT NULL DEFAULT '',
  render_version TEXT NOT NULL DEFAULT '',
  content_hash TEXT NOT NULL,
  reason TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX post_revisions_post_created_idx ON post_revisions (post_id, created_at DESC);

INSERT INTO categories (slug, name, description, visibility, is_system)
VALUES ('general', '综合讨论', '默认综合讨论版块。', 'public', TRUE)
ON CONFLICT (slug) DO NOTHING;

-- +goose Down
DROP TABLE IF EXISTS post_revisions;
DROP INDEX IF EXISTS comments_author_created_idx;
DROP INDEX IF EXISTS comments_parent_idx;
DROP INDEX IF EXISTS comments_topic_root_idx;
DROP INDEX IF EXISTS comments_topic_path_idx;
DROP TABLE IF EXISTS comments;
DROP INDEX IF EXISTS topics_slug_idx;
DROP INDEX IF EXISTS topics_author_created_idx;
DROP INDEX IF EXISTS topics_category_activity_idx;
DROP TABLE IF EXISTS topics;
DROP INDEX IF EXISTS posts_content_hash_idx;
DROP INDEX IF EXISTS posts_created_by_idx;
DROP TABLE IF EXISTS posts;
DROP TABLE IF EXISTS categories;
