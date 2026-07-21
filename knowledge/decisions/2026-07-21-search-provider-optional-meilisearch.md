# 2026-07-21 Search Provider: Optional Meilisearch (Not Built-in)

## Status

Accepted for **Meilisearch optional / not built-in**.  
Product default for “no engine → 503” is **superseded** by
`2026-07-21-search-framework-site-default.md` (protected site search default).

## Context

Meilisearch was wired into core (client in `Support/Search`, Compose dependency,
readiness component, production secret validation). The process typically uses
hundreds of MB of RAM even when the product feature flag is off. The platform
already declares `search.provider` as a Host catalog slot.

Operators asked to extract Meilisearch so the default stack does not run it, and
explicitly: **it must not be a built-in plugin**.

## Decision

1. **Core owns** document schema (`TopicSearchDoc`), ACL filters contract, River
   index/delete/reindex jobs, public `/search` API, and the `search.provider`
   slot. Core **does not** embed Meilisearch or require `MEILI_*` env vars.
2. **Default**: no search engine selected → search returns unavailable (503),
   index enqueue is a no-op, no Meilisearch process in default Compose/dev.
3. **Engine transport** lives in an **optional** package:
   `extensions/optional/plugins/sforum-search-meilisearch` (id
   `sforum.search-meilisearch`). Not under `extensions/builtin`, not
   `SyncBuiltins`.
4. Compose service `meilisearch` uses **profile `search`** only.
5. When exactly one enabled plugin declares `search.provider`, Host uses it
   automatically; operators may also pin via `mail_provider_selection` row
   `slot = 'search.provider'`.

## Consequences

- Default memory footprint drops by not running Meilisearch.
- Full-text search is opt-in: install plugin, start Meili (`--profile search`),
  configure host/key, enable + trust, reindex if needed.
- Alternative engines can implement the same known-slot operations:
  `probe` / `ensure` / `index` / `delete` / `search`.
