# 2026-07-21 Session Handoff — Million-scale M2

## Changed

- **View count (D3 / Iteration A WS1, single implementation):**
  - `forum.TopicViewRecorder` + Redis/Memory counter (`view_counter.go`)
  - Controller `GET` detail id/slug → `RecordTopicView` after public resolve
  - River `forum.flush_view_counts` (45s) + `PostgresStore.ApplyViewCountDeltas`
  - OpenAPI documents GET side-effect; no `POST /view`
- **hot_score:** migration `202607210047_topics_hot_score.sql`, list `sort=hot` column+indexes, comment/moderation/seed maintain score
- **FE:** `SFTopicShowPage` documents single detail GET per navigation
- **Perf:** `knowledge/reports/2026-07-21-perf-m2-view-hot.md`
- Builtin plugin digests refreshed for content-policy/smtp/storage-fs (local binary drift)

## Decisions

- Product view path owned by Iteration A WS1; M2 only couples load proof + hot_score (no dual design).
- Display remains PG `view_count` (stale until flush); no live Redis delta merge in v1.
- Flush interval 45s (within 30–60s plan band).

## Next

- **M3** ListComments bounds + cache (D2 tree descendant cap default 50).
- Optional: search reindex batch on view flush; theme island allowlist for `sf-my-home-page`.

## Open Questions

- Exact hot_score time-decay formula later (M2 ships non-decay `comment*5+view`).
