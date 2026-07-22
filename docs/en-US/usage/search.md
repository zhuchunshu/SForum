# Search

[← Usage](./README.md)

## Default: site search (recommended)

| Item | Detail |
| --- | --- |
| Package | Protected built-in `sforum.search-site` |
| Engine | Host PostgreSQL FTS (`search_documents`) |
| Dependencies | **No** Meilisearch process or `MEILI_*` required |
| Uninstall | Not uninstallable—search always has a provider |

Restoring search defaults returns to site search.

## Optional: Meilisearch plugin

1. Start the service:

   ```sh
   docker compose --profile search up -d meilisearch
   ```

2. Clone the independent plugin repository, set its collection root, and restart API:

   ```env
   EXTERNAL_EXTENSION_ROOTS=/absolute/path/to/sforum-plugins
   ```

   The package lives at `sforum-plugins/plugins/sforum-search-meilisearch` and
   is discovered as an inert immutable snapshot; it is not enabled automatically.
3. Super-admin enable + **trust**  
4. Configure host/master key; select `search.provider`  
5. Reindex  

Dev port example: `http://127.0.0.1:17700`.

## Operator notes

- Topic writes enqueue index/delete for the selected engine  
- Reindex after switching engines  
- Public results still respect guest-read and ACL  

Decision: `knowledge/decisions/2026-07-21-search-framework-site-default.md`.
