# Search Module

## Purpose

Owns full-text forum search contracts and synchronization between PostgreSQL and
the selected `search.provider` engine.

## Provider Slot

Host catalog slot: **`search.provider`**.

| Layer | Responsibility |
| --- | --- |
| Core | Document schema, public ACL filter contract, enqueue index/delete, reindex orchestration, `/search` API, default site engine |
| Plugin | Engine transport only (store/query documents), except site search which is Host short-circuited |
| Default | **Protected built-in site search** (`sforum.search-site`) — PostgreSQL FTS |

## Current Status (2026-07-22)

- Core has **no** Meilisearch client; `MEILI_*` is not required.
- Default: `sforum.search-site` (builtin, cannot uninstall). Host implements
  `PostgresSiteEngine` against `search_documents`: built-in `simple` FTS,
  Unicode Han unigram/bigram `cjk_tsv`, and `pg_trgm` title/excerpt fuzzy search,
  all backed by GIN indexes.
- Search metadata is indexed separately in `metadata_tsv`: author username /
  display name, category name / slug, tag slugs, and topic slug. Chinese
  metadata is also included in `cjk_tsv` during indexing.
- Site-search admin entry is **About only** (no settings fields); Manage opens
  plugin info.
- Optional: independent package
  `sforum-plugins/plugins/sforum-search-meilisearch`, discovered through
  `EXTERNAL_EXTENSION_ROOTS`.
- Compose: `meilisearch` service has profile `search` only.
- Decision: `decisions/2026-07-21-search-framework-site-default.md`
  (supersedes “default no engine → 503”).

### Search regression remediation (closed 2026-07-23)

Task book:
`../plans/2026-07-22-current-head-regression-remediation.md`.

- Invalid `pg_catalog` regconfig use was restored to built-in `simple` and
  verified against real PostgreSQL.
- Ghost validation now checks only the requested engine page, preserves stable
  engine totals, and never borrows adjacent pages.
- HTTP search uses one request-scoped Forum hydration batch and retains engine
  ordering without a second summary/tag query.
- The later CJK/fuzzy decision adds default Chinese n-grams and `pg_trgm` while
  preserving optional Meilisearch for dictionary segmentation, synonyms, and
  more aggressive typo tolerance.
- Search-focused, fresh-database ownership, OpenAPI, typecheck, UI unit, API,
  and G0 gate checks pass. The parent current-HEAD regression plan is closed;
  later 404 work owns only selected-theme resource-not-found behavior.

### Runtime behavior

- Public search always has a resolved provider (site search when nothing pinned).
- Topic write path enqueues index/delete for the selected engine.
- Restore defaults → clear pin → site search.
- Site-search ranking is relevance first, then pinned/activity/topic ID for a
  deterministic order. It supports English lexical search, Chinese title/body
  n-grams, literal infix matching, and conservative typo tolerance.
- Upgrades that add or change CJK derivation must run the normal admin reindex;
  generated title/excerpt trigram text is populated by the migration itself.
- Incremental index/delete jobs deduplicate only while a matching job is still
  active; completed jobs never block later edits. An explicit full reindex does
  not use River uniqueness, so legacy completed jobs cannot turn a rebuild into
  a false no-op. Worker engine failures remain retryable instead of being marked
  completed.

### Admin UI

- **Forum settings → Search tab** (`/admin/forum/settings?tab=search`):
  list/select/reset `search.provider`, reindex shortcut.
- APIs (permission `search.manage`):
  - `GET /admin/forum/search/providers`
  - `PUT /admin/forum/search/provider`
  - `POST /admin/forum/search/provider/reset`
  - existing reindex endpoints under `/admin/forum/search/reindex*`
- Standalone `/admin/search` remains the reindex progress/history page.

### Enabling Meilisearch

1. `docker compose --profile search up -d meilisearch`
2. Configure the independent `sforum-plugins` collection through
   `EXTERNAL_EXTENSION_ROOTS`; restart API/worker to snapshot the package
3. Super-admin enable + trust
4. Configure host/master key; select provider in Forum settings → Search; reindex

## Document shape

`TopicSearchDoc` (Host-owned): title, plainText, excerpt, category/tag slugs,
status, pin, activity timestamps, author summary. Index UID: `sforum_topics`
(external engines). Site engine table: `search_documents`. Decision:
`../decisions/2026-07-22-default-site-search-cjk-fuzzy.md`.
