# 2026-07-21 Search Framework: Protected Site Search Default

## Status

Accepted. **Supersedes** the product default in
`2026-07-21-search-provider-optional-meilisearch.md` (that decision remains
correct that Meilisearch is **not** built into core and is optional).

## Context

Meilisearch was removed from core so default stacks need no Meili process.
Leaving **no** search engine made public keyword search return 503 out of the
box, which fails beginner-friendly and open-source framework defaults.

## Decision

1. **Core owns a search framework** (`app/Support/Search`): document schema
   (`TopicSearchDoc`), ACL/public-only contract, River index/delete/reindex,
   public `/search` API, and slot `search.provider`. Core embeds **no**
   Meilisearch SDK and does not require `MEILI_*`.
2. **Default engine** is protected built-in **site search**
   (`sforum.search-site` under `extensions/builtin/plugins/sforum-search-site`).
   Implementation: PostgreSQL `search_documents` + `tsvector`/GIN, in-process
   Host short-circuit (no RPC on the hot path).
3. **Cannot uninstall** the site-search extension (SourceBuiltin / IsSystem /
   !IsDeletable path already rejects uninstall).
4. **Restore defaults** for `search.provider` clears explicit pin and resolves
   to site search.
5. **Meilisearch** remains an **optional**, non-builtin package maintained in
   the independent `sforum-plugins` repository and discovered through
   `EXTERNAL_EXTENSION_ROOTS`.
6. **v1 dual-write**: only the **selected** engine receives index/delete jobs.
   Switching engines requires admin reindex for the newly selected engine.
7. **Engine failure**: selected external engine unavailable → search errors
   with unavailable semantics (no automatic silent failover to site search).

## Consequences

- Fresh install with PostgreSQL only: keyword search works after indexing
  (writes enqueue index jobs against site engine).
- Operators may select Meili (or another `search.provider` plugin) after install
  + trust; then reindex.
- Memory default stack no longer needs Meilisearch.
