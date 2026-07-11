-- +goose Up
-- Wave 2：前台壳结构化目录（导航 / 友情链接 / 公告横幅）。
-- 标量品牌与法律正文仍走 web_options；列表类资源用专用表，避免 JSON 选项膨胀。

CREATE TABLE site_nav_items (
  id BIGSERIAL PRIMARY KEY,
  label_zh_cn TEXT NOT NULL DEFAULT '',
  label_en_us TEXT NOT NULL DEFAULT '',
  href TEXT NOT NULL DEFAULT '',
  open_in_new_tab BOOLEAN NOT NULL DEFAULT FALSE,
  position INTEGER NOT NULL DEFAULT 0,
  enabled BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX site_nav_items_position_idx ON site_nav_items (position, id);

CREATE TABLE site_friend_links (
  id BIGSERIAL PRIMARY KEY,
  name TEXT NOT NULL DEFAULT '',
  url TEXT NOT NULL DEFAULT '',
  description TEXT NOT NULL DEFAULT '',
  logo_url TEXT NOT NULL DEFAULT '',
  position INTEGER NOT NULL DEFAULT 0,
  enabled BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX site_friend_links_position_idx ON site_friend_links (position, id);

CREATE TABLE site_announcements (
  id BIGSERIAL PRIMARY KEY,
  title_zh_cn TEXT NOT NULL DEFAULT '',
  title_en_us TEXT NOT NULL DEFAULT '',
  body_zh_cn TEXT NOT NULL DEFAULT '',
  body_en_us TEXT NOT NULL DEFAULT '',
  style TEXT NOT NULL DEFAULT 'info'
    CHECK (style IN ('info', 'success', 'warning', 'danger')),
  href TEXT NOT NULL DEFAULT '',
  dismissible BOOLEAN NOT NULL DEFAULT TRUE,
  position INTEGER NOT NULL DEFAULT 0,
  enabled BOOLEAN NOT NULL DEFAULT TRUE,
  starts_at TIMESTAMPTZ,
  ends_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX site_announcements_active_idx
  ON site_announcements (enabled, position, id)
  WHERE enabled = TRUE;

-- 推荐默认导航：首页 / 分类 / 标签 / 搜索（可运营再改）。
INSERT INTO site_nav_items (label_zh_cn, label_en_us, href, open_in_new_tab, position, enabled)
VALUES
  ('首页', 'Home', '/', FALSE, 0, TRUE),
  ('分类', 'Categories', '/categories', FALSE, 10, TRUE),
  ('标签', 'Tags', '/tags', FALSE, 20, TRUE),
  ('搜索', 'Search', '/search', FALSE, 30, TRUE);

-- +goose Down
DROP TABLE IF EXISTS site_announcements;
DROP TABLE IF EXISTS site_friend_links;
DROP TABLE IF EXISTS site_nav_items;
