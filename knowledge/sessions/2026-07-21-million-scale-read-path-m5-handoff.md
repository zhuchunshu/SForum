# 2026-07-21 Session Handoff — Million-scale M5

## Changed

- **OpenAPI:** `GET /topics` + `GET /topics/{id}/comments` accept `after`; responses
  require `hasMore`; optional `nextCursor`. Cursor **wins over** `page`.
- **ListTopics keyset:** opaque cursor `(sort, pin, sortKey, id)`; seek-friendly
  SQL so depth does not walk OFFSET-equivalent prefixes; pins first dimension.
- **ListComments flat keyset:** `path_key` + `id`; tree keeps page + `hasMore`.
- **CachedStore:** cache keys include `after`.
- **FE home:** infinite scroll uses `nextCursor` → `after`; end detection via
  `hasMore`; taxonomy「约」unchanged (D1).
- **Perf:** `tests/perf/deep_scroll.js` +
  `knowledge/reports/2026-07-21-perf-m5-keyset.md` (100-step cursor p99 ~19 ms).

## Decisions

- No composite topic-page cache (M4 still holds).
- Cursor payload is signed-shape base64url JSON (public sort keys only), not
  Query Registry HMAC (that codec is plugin-offset oriented).
- Seek form `col <= k AND (col < k OR id < id)` required for Index Cond; plain
  OR prefix filtered like OFFSET.

## Next

- **M6** cache sharding by category/tag gen (not started).
- Optional: category/tag pages infinite scroll (home only in M5).

## Open Questions

- None for M5 product defaults.
