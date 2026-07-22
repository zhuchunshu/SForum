# 2026-07-21 Session Handoff — Million-scale M1 complete

## Changed

- **M1 ListTopics cold path shipped:**
  - `apps/api/app/Models/Forum/list_topics.go`: page CTE (IDs first) + hydrate
    with `left(posts.plain_text, N)` only for page rows; no list ILIKE; D1
    totals (cat/tag `topic_count`, home sum + approximate, multi min approx).
  - Migration `202607210046_topics_public_activity_idx.sql` for home default
    sort; category path EXPLAIN uses `topics_category_activity_idx` LIMIT 20.
  - `TopicList.TotalApproximate`; CachedStore key includes sort; page1 TTL 45s.
  - OpenAPI TopicList + listTopics description; FE `formatForumTopicListTotal`
    + i18n「约 {n}」only when approximate.
  - Unit tests: D1 mode classify, no ILIKE, order columns, page SQL shapes.
  - Report: `knowledge/reports/2026-07-21-perf-m1-list-topics.md`

## Decisions

- D1–D4 unchanged (still law).
- Home total = sum of public category counters (not reltuples); still marked
  approximate for UI「约」.
- Multi-filter total = min(cat, tag) counters + approximate (no COUNT(*)).

## Next

- **M2** view count (Iteration A WS1 / D3) + `hot_score` popular sort.
- Optional: re-capture k6 LIGHT=1 when k6 binary is available (M1 used
  sequential curl + 5-worker concurrent probe).

## Open Questions

- None for D1–D4. Residual: pin+keyset (M5); hot formula weights (M2).
