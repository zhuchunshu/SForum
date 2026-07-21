# 2026-07-21 Session Handoff — Million-scale M3

## Changed

- **D2 tree bound:** option `forum.comments.tree_descendants_per_root` (1–100,
  default 50, restore defaults); store per-root `ROW_NUMBER` cap;
  `Comment.hasMoreChildren`; OpenAPI + admin comments tab
- **FE:** `SFComment`「加载更多回复」→ `listCommentReplies` merge into local tree
- **Totals:** flat public total from `topics.comment_count`; tree root COUNT
  (payload cached with list)
- **CachedStore:** `ListComments` short TTL + `forum:gen:comments:{topicID}`;
  skip when `IncludeDeleted` / author scope; invalidate on create/update/delete
- **Perf:** `knowledge/reports/2026-07-21-perf-m3-list-comments.md`
- **Tests:** `list_comments_bounds_test.go`, CachedStore comment hit/miss

## Decisions

- ListCommentReplies not cached in v1 (rarer load-more path).
- No mobile-only lower API cap (D2).
- Viewer-scoped soft-delete comment lists never share Redis cache.

## Next

- **M4** topic detail assembly / optional composite cache.
- Optional: cold tree payload field trim if still over budget after warm cache.

## Open Questions

- None for D2/M3 product defaults.
