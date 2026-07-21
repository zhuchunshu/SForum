# 2026-07-21 Session Handoff — Million-scale M6

## Changed

- **CachedStore topics gen sharding:** `forum:gen:topics:global` /
  `cat:{slug}` / `tag:{slug}`; list cache key embeds scope gen (cat+tag dual
  filter uses both).
- **Write invalidation:** Create/Update/Delete topic, Create/Delete comment,
  lifecycle action bump only global + affected cat/tag scopes. Scope resolved
  from detail cache or inner `GetTopic` fallback.
- **COUNT audit:** public ListTopics / flat comments still no hot-path
  full-table COUNT; tree root COUNT only on ListComments miss; author COUNTs
  write-path only. Comments in `postgres_store_ops.go`.
- **Tests:** cat A write keeps cat B list cache hit; tag-x vs tag-y; comment
  write scoped by topic category.
- **Perf:** `knowledge/reports/2026-07-21-perf-m6-cache-sharding.md`
  (multi-cat warm p99 ~9–19 ms; unit scoped invalidation).

## Decisions

- Home always invalidates on public topic-list-affecting writes (activity feed).
- No composite topic-page cache (M4 still holds).
- No M7 read-replica code; next is doc-only.

## Next

- **M7** optional: when to add read replica (metrics threshold) — documentation
  only unless product demands.

## Open Questions

- None for M6 product defaults.
