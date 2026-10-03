# Decision: Flat Comment Stream Is Chronological (Created-Time Floor Order)

## Status

Accepted

## Date

2026-10-03

## Context

`GET /topics/{id}/comments?view=flat` is the only comment surface the default
theme renders (`commentView` is pinned to `flat`; the builtin theme template
mounts the same `sf-topic-show-page` island). Its ordering key was
`path_key ASC, id ASC` — the hierarchical materialized path, i.e. a depth-first
pre-order of the reply tree. Consequences observed on the dev instance
(topic 403):

- A reply was inserted directly under the comment it answered. The newest
  comment was not at the bottom of the stream; older root comments appeared
  below it (`…1675, 1676, 1563, 1566`).
- Floor labels are derived from the list index
  (`commentFloorNumber(index, page/perPage)`), so any reply to an earlier
  comment re-numbered every later floor. A root comment moved from `#2` to
  `#9` purely because someone replied above it.
- The frontend already declares the intended product model: "默认主题只提供连续
  时间流；回复关系由引用块表达" (a continuous time stream with the reply
  relationship expressed by the per-row `replyTo` reference card).

Two alternatives were rejected:

- **Keep the hierarchical grouping but indent it (tree UI).** The product owner
  reported the current behaviour as wrong precisely because the reply is *not*
  at the bottom; indenting would keep that property.
- **Root-grouped chronological** (roots by time, replies under their own root by
  time). A reply to a deep comment still would not land at the bottom, so the
  reported symptom survives.

Adding an operator option for the ordering was deferred: `view=tree` already
exists as the nested consumer path, and a new admin surface is not justified
until someone asks for selectable comment order.

## Decision

1. **Flat view is a chronological stream:** `ORDER BY comments.created_at ASC,
   comments.id ASC`. New comments always append at the end (the bottom floor);
   existing floors never move because of someone else's reply.
2. **`replyTo` carries the relationship.** Every reply row keeps the
   `replyTo { id, author, excerpt, depth }` reference produced by the existing
   parent join; the client renders it as a clickable card that jumps to the
   parent (cross-page jumps reuse `GET /topics/{topicID}/comments/{commentID}/page`).
3. **One ordering key everywhere the flat position is computed:**
   - keyset cursor payload is now `{v, k: RFC3339Nano(created_at), i: id}`
     (legacy `path_key` cursors are rejected with `forum.cursor_invalid`, so a
     stale client simply reloads page 1);
   - `CountCommentsBefore(topicID, createdAt, id, includeDeleted, authorUserID)`
     compares `ROW(created_at, id)`;
   - `ResolveCommentPage` (deep-link page resolve) passes `summary.CreatedAt`;
   - profile public activity `commentPage` counts with the same row comparison.
4. **Index:** `comments_topic_created_idx (topic_id, created_at, id)`
   (migration `202610030001`, `CREATE INDEX CONCURRENTLY`). It serves both the
   paginated `ORDER BY` and the keyset seek predicate. A partial
   `WHERE status = 'active'` index was rejected because the public list's
   `status = 'active' OR (… deleted …)` predicate cannot imply the index
   predicate, so the planner could not use it.
5. **Tree view is unchanged** (`path_key` ordering, nested `children`), still
   consumed by `PageViewModels`/theme contracts that want hierarchy.
6. **Posting a reply must land on it.** Because the new comment is no longer
   adjacent to its parent, `useTopicCommentSubmission` calls the new
   `focusCreatedComment` hook after a successful publish;
   `useTopicCommentAnchor.focusCreatedComment` resolves the target page and
   navigates to `/page/N#comment-<id>`, which reuses the existing scroll +
   flash machinery. Pending-review comments do not jump (they are not in the
   list yet).

## Consequences

- Floors are stable and monotonic; the last row of the last page is always the
  newest comment, and the header "latest" anchor keeps meaning "newest".
- Reading a long thread is a timeline; answering an old comment produces a new
  bottom-floor comment whose card links back to the parent. This is the classic
  BBS/Discuz model, not the Discourse nested model.
- Deep links, profile activity links, notifications and admin surfaces that
  reuse `commentPage` stay correct because every position computation moved to
  the same key.
- The M3/M5 perf numbers in `reports/2026-07-21-perf-m3-list-comments.md` and
  `reports/2026-07-21-perf-m5-keyset.md` were measured under `path_key`
  ordering. The new index supports the same access pattern, but the flat
  read-path benchmark has not been re-run against `created_at` ordering.
- The tree/flat toggle component `SFCommentStreamControls.vue` is now fully
  unreferenced by production code (only a negative assertion in a test); it is
  left in place pending a deliberate cleanup.

## Follow-up

- Re-run the flat read-path benchmark (deep `page` + keyset) against the new
  ordering key and record it under `knowledge/reports/`.
- Delete `SFCommentStreamControls.vue` together with its i18n keys once a
  cleanup pass is scheduled.
