# sforum.search-site

Protected built-in **PostgreSQL full-text site search** engine.

Default `search.provider`. The Host owns the provider-neutral search ledger,
indexing jobs, and reconciliation; this plugin declares the protected
PostgreSQL FTS engine as the runtime provider.

## Ownership boundary

| Concern | Owner |
| --- | --- |
| Provider-neutral search ledger, indexing jobs, 15-minute reconciliation | **Host** |
| PostgreSQL `search_documents` + tsvector queries (protected engine) | **This plugin** |

## Features

- **Not uninstallable** (protected builtin);
- zero external processes: no Meilisearch required;
- hot paths short-circuit inside the Host process (no plugin RPC for read
  queries);
- switching to an optional Meili engine later, and "Restore Default" to return
  to this engine, are supported by the Host provider selection;
- **no configuration fields**: the admin entry is an About page showing the
  plugin description and package identity.

## When to use something else

Very large sites needing stronger tokenization/relevance can install the
optional `sforum-search-meilisearch` plugin and select it as
`search.provider`, then rebuild the index. See
[搜索 / Search](../../../../docs/zh-CN/usage/search.md) for operator guidance.
