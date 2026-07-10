# Server-Authoritative Forum Pagination Defaults

## Decision

Public topic/search pagination and public comment pagination use independent
runtime defaults stored in `web_options`. Both recommended values are 20 and
the supported range is 1-100.

The API resolves the configured value only when `perPage` is omitted or not
positive. A positive caller value remains an explicit override, capped at 100.
The existing page-200 deep-pagination guard remains unchanged.

## Why

Resolving defaults in the API gives the built-in theme, plugins, and third-party
clients identical behavior. A frontend-only option would leave API consumers
on a different default and duplicate fallback logic. The existing Options
service already provides caching, typed validation, permissions, and reset
semantics, so no new library or storage mechanism is needed.

## Boundaries

- `forum.pagination.topics_per_page` controls PostgreSQL topic lists and
  Meilisearch topic results.
- `forum.pagination.comments_per_page` controls public topic comments.
- Admin tables, account sessions, attachments, extension logs, and batch jobs
  retain their own page sizes.
- Updating either option requires `settings.manage`; no new permission is
  introduced.
