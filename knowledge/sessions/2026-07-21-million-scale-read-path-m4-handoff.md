# 2026-07-21 Session Handoff — Million-scale M4

## Changed

- **GetTopic profile:** documented slug `topics_slug_idx` UNIQUE path; revisions
  stay `EXISTS` only; no heavy re-query rewrite needed at measured cost
- **CachedStore:** dual-write id + slug detail keys; `forum:topic-id-slug:{id}`
  reverse map so comment writes invalidate by-slug detail (comment_count)
- **No composite** `forum:topic-page:{id}:{page}` — evidence warm by-slug under
  plan p99 budget after M1+M3 caches
- **FE:** `SFTopicShowPage` `Promise.all` topic+comments when URL has id; pure
  slug awaits same `topicAsync` (D3, no second detail GET / no onMounted refetch)
- **SWR:** `/t/**` + `/en/t/**` `swr: 60`; `server/middleware/topic-page-cache.ts`
  no-store for `sforum_session` or `?edit=`
- **Perf:** `knowledge/reports/2026-07-21-perf-m4-topic-detail.md`
- **Tests:** CachedStore dual-write/invalidate; FE D3+parallel static checks;
  protected route SWR middleware expectations

## Decisions

- Skip anonymous composite topic+comments cache until a future miss is measured.
- Anonymous HTML SWR for topic pages is safe only with session/edit middleware
  gate (Nitro keys by URL, not user).

## Next

- **M5** keyset pagination + total UX.
- Optional: cold tree payload field trim (M3 residual).

## Open Questions

- None for M4 product defaults.
