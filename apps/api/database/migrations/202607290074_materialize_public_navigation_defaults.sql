-- +goose Up
-- Public rendering previously synthesized missing recommended placements while
-- the admin editor exposed only stored rows. Materialize the code-owned Core
-- defaults once so both surfaces read the same revisioned document. Existing
-- placements win, preserving operator visibility, ordering, and overrides.
INSERT INTO site_navigation_definitions (
    source_key, source_kind, link_kind, label_zh_cn, label_en_us, href, icon
)
VALUES
    ('core.home', 'core', 'coreRoute', '首页', 'Home', '/', 'i-lucide-layout-list'),
    ('core.categories', 'core', 'coreRoute', '分类', 'Categories', '/categories', 'i-lucide-layout-grid'),
    ('core.tags', 'core', 'coreRoute', '标签', 'Tags', '/tags', 'i-lucide-tags'),
    ('core.dynamic.categories', 'dynamic', 'dynamicBlock', '分类', 'Categories', '', 'i-lucide-folders'),
    ('core.terms', 'core', 'coreRoute', '服务条款', 'Terms', '/terms', 'i-lucide-file-text'),
    ('core.privacy', 'core', 'coreRoute', '隐私政策', 'Privacy', '/privacy', 'i-lucide-shield-check'),
    ('core.guidelines', 'core', 'coreRoute', '社区指南', 'Guidelines', '/guidelines', 'i-lucide-book-open')
ON CONFLICT (source_key) DO UPDATE SET
    source_kind = EXCLUDED.source_kind,
    link_kind = EXCLUDED.link_kind,
    label_zh_cn = EXCLUDED.label_zh_cn,
    label_en_us = EXCLUDED.label_en_us,
    href = EXCLUDED.href,
    icon = EXCLUDED.icon,
    open_in_new_tab = FALSE,
    extension_id = '',
    contribution_id = '',
    updated_at = now();

INSERT INTO site_navigation_placements (
    source_key, location, position, enabled, visibility
)
VALUES
    ('core.home', 'public.topbar.primary', 10, TRUE, 'public'),
    ('core.categories', 'public.topbar.primary', 20, TRUE, 'public'),
    ('core.tags', 'public.topbar.primary', 30, TRUE, 'public'),
    ('core.home', 'public.sidebar.primary', 10, TRUE, 'public'),
    ('core.categories', 'public.sidebar.primary', 20, TRUE, 'public'),
    ('core.tags', 'public.sidebar.primary', 30, TRUE, 'public'),
    ('core.dynamic.categories', 'public.sidebar.primary', 40, TRUE, 'public'),
    ('core.home', 'public.mobile.primary', 10, TRUE, 'public'),
    ('core.categories', 'public.mobile.primary', 20, TRUE, 'public'),
    ('core.tags', 'public.mobile.primary', 30, TRUE, 'public')
ON CONFLICT (source_key, location) DO NOTHING;

UPDATE site_navigation_state
SET revision = revision + 1,
    updated_at = now()
WHERE id = 1;

INSERT INTO web_options (name, value)
VALUES ('site.public_surface_revision', '2')
ON CONFLICT (name) DO UPDATE SET value = (
    CASE
        WHEN trim(web_options.value) ~ '^[0-9]+$'
            THEN greatest(web_options.value::numeric, 1) + 1
        ELSE 2
    END
)::text;

-- +goose Down
-- Deliberately no-op. These rows become operator-managed navigation state after
-- migration; deleting them on rollback would discard subsequent configuration.
