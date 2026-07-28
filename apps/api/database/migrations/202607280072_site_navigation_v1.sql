-- +goose Up
-- Configurable public navigation keeps its document independent from the
-- original SiteChrome CRUD tables. The latter remains an API-LTS surface.
CREATE TABLE site_navigation_state (
    id SMALLINT PRIMARY KEY CHECK (id = 1),
    revision BIGINT NOT NULL DEFAULT 1 CHECK (revision >= 1),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO site_navigation_state (id, revision) VALUES (1, 1)
ON CONFLICT (id) DO NOTHING;

CREATE TABLE site_navigation_definitions (
    source_key VARCHAR(160) PRIMARY KEY,
    source_kind VARCHAR(24) NOT NULL CHECK (source_kind IN ('core', 'operator', 'extension', 'dynamic')),
    link_kind VARCHAR(32) NOT NULL CHECK (link_kind IN ('coreRoute', 'internalLink', 'externalLink', 'extensionHostLink', 'extensionRoute', 'dynamicBlock')),
    label_zh_cn VARCHAR(80) NOT NULL DEFAULT '',
    label_en_us VARCHAR(80) NOT NULL DEFAULT '',
    href VARCHAR(500) NOT NULL DEFAULT '',
    icon VARCHAR(120) NOT NULL DEFAULT '',
    open_in_new_tab BOOLEAN NOT NULL DEFAULT FALSE,
    extension_id VARCHAR(120) NOT NULL DEFAULT '',
    contribution_id VARCHAR(120) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK ((source_kind <> 'extension') OR extension_id <> '')
);

CREATE TABLE site_navigation_placements (
    source_key VARCHAR(160) NOT NULL REFERENCES site_navigation_definitions(source_key) ON DELETE CASCADE,
    location VARCHAR(64) NOT NULL CHECK (location IN ('public.topbar.primary', 'public.sidebar.primary', 'public.mobile.primary', 'public.footer.primary')),
    position INTEGER NOT NULL CHECK (position BETWEEN -100000 AND 100000),
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    visibility VARCHAR(24) NOT NULL DEFAULT 'public' CHECK (visibility IN ('public', 'anonymous', 'authenticated', 'permission')),
    permission VARCHAR(120) NOT NULL DEFAULT '',
    label_zh_cn VARCHAR(80) NOT NULL DEFAULT '',
    label_en_us VARCHAR(80) NOT NULL DEFAULT '',
    icon VARCHAR(120) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (source_key, location),
    CHECK ((visibility = 'permission' AND permission <> '') OR (visibility <> 'permission' AND permission = ''))
);

CREATE INDEX site_navigation_placements_location_order_idx
    ON site_navigation_placements (location, position, source_key);
CREATE INDEX site_navigation_definitions_extension_idx
    ON site_navigation_definitions (extension_id, contribution_id)
    WHERE source_kind = 'extension';

-- M2 owns snapshot creation/retention. Creating the additive table now makes
-- the durable document schema stable without mutating the legacy table.
CREATE TABLE site_navigation_snapshots (
    id BIGSERIAL PRIMARY KEY,
    revision BIGINT NOT NULL CHECK (revision >= 1),
    operation VARCHAR(64) NOT NULL,
    reason VARCHAR(240) NOT NULL DEFAULT '',
    affected_locations JSONB NOT NULL DEFAULT '[]'::jsonb,
    document JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX site_navigation_snapshots_revision_idx ON site_navigation_snapshots (revision DESC, id DESC);

-- Preserve existing topbar order, labels, enabled state, and target. Exact
-- historical defaults become Core placements; all other rows receive a stable,
-- portable key derived from their content rather than exposing a database id.
INSERT INTO site_navigation_definitions (
    source_key, source_kind, link_kind, label_zh_cn, label_en_us, href, open_in_new_tab
)
SELECT
    CASE
        WHEN href = '/' AND open_in_new_tab = FALSE THEN 'core.home'
        WHEN href = '/categories' AND open_in_new_tab = FALSE THEN 'core.categories'
        WHEN href = '/tags' AND open_in_new_tab = FALSE THEN 'core.tags'
        ELSE 'operator.migrated.' || substr(md5(label_zh_cn || chr(31) || label_en_us || chr(31) || href || chr(31) || id::text), 1, 32)
    END,
    CASE WHEN href IN ('/', '/categories', '/tags') AND open_in_new_tab = FALSE THEN 'core' ELSE 'operator' END,
    CASE
        WHEN href IN ('/', '/categories', '/tags') AND open_in_new_tab = FALSE THEN 'coreRoute'
        WHEN href ~ '^https?://' THEN 'externalLink'
        ELSE 'internalLink'
    END,
    CASE WHEN href IN ('/', '/categories', '/tags') AND open_in_new_tab = FALSE THEN '' ELSE label_zh_cn END,
    CASE WHEN href IN ('/', '/categories', '/tags') AND open_in_new_tab = FALSE THEN '' ELSE label_en_us END,
    CASE WHEN href IN ('/', '/categories', '/tags') AND open_in_new_tab = FALSE THEN '' ELSE href END,
    CASE WHEN href IN ('/', '/categories', '/tags') AND open_in_new_tab = FALSE THEN FALSE ELSE open_in_new_tab END
FROM site_nav_items
ON CONFLICT (source_key) DO NOTHING;

INSERT INTO site_navigation_placements (
    source_key, location, position, enabled, visibility, label_zh_cn, label_en_us
)
SELECT
    CASE
        WHEN href = '/' AND open_in_new_tab = FALSE THEN 'core.home'
        WHEN href = '/categories' AND open_in_new_tab = FALSE THEN 'core.categories'
        WHEN href = '/tags' AND open_in_new_tab = FALSE THEN 'core.tags'
        ELSE 'operator.migrated.' || substr(md5(label_zh_cn || chr(31) || label_en_us || chr(31) || href || chr(31) || id::text), 1, 32)
    END,
    'public.topbar.primary', position, enabled, 'public',
    CASE WHEN href IN ('/', '/categories', '/tags') AND open_in_new_tab = FALSE THEN label_zh_cn ELSE '' END,
    CASE WHEN href IN ('/', '/categories', '/tags') AND open_in_new_tab = FALSE THEN label_en_us ELSE '' END
FROM site_nav_items
ON CONFLICT (source_key, location) DO NOTHING;

-- +goose Down
DROP TABLE IF EXISTS site_navigation_snapshots;
DROP TABLE IF EXISTS site_navigation_placements;
DROP TABLE IF EXISTS site_navigation_definitions;
DROP TABLE IF EXISTS site_navigation_state;
