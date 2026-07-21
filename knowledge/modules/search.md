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

## Current Status (2026-07-21)

- Core has **no** Meilisearch client; `MEILI_*` is not required.
- Default: `sforum.search-site` (builtin, cannot uninstall). Host implements
  `PostgresSiteEngine` against `search_documents` (tsvector + GIN).
- Site-search admin entry is **About only** (no settings fields); Manage opens
  plugin info.
- Optional: `extensions/optional/plugins/sforum-search-meilisearch`.
- Compose: `meilisearch` service has profile `search` only.
- Decision: `decisions/2026-07-21-search-framework-site-default.md`
  (supersedes “default no engine → 503”).

### Runtime behavior

- Public search always has a resolved provider (site search when nothing pinned).
- Topic write path enqueues index/delete for the selected engine.
- Restore defaults → clear pin → site search.

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
2. Install `sforum-search-meilisearch` (optional package)
3. Super-admin enable + trust
4. Configure host/master key; select provider in Forum settings → Search; reindex

## Document shape

`TopicSearchDoc` (Host-owned): title, plainText, excerpt, category/tag slugs,
status, pin, activity timestamps, author summary. Index UID: `sforum_topics`
(external engines). Site engine table: `search_documents`.
