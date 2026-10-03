# Decision: Public List Recommended Default Sort Is `active`

## Status

Accepted

## Date

2026-10-03

## Context

`GET /topics` accepts `sort=latest|active|hot`; an empty `sort` resolves to the
site option `forum.list.default_sort`. The shipped recommended default was
`latest` (newest created first), and the homepage control exposed it as one of
two options while the rest of the ordering was done client-side.

Three things surfaced while reworking the homepage sort control:

- There is no index supporting created-time ordering on the public feed.
  Migrations provide `topics_public_activity_idx (is_pinned, last_activity_at,
  id)`, `topics_category_activity_idx`, `topics_public_hot_idx`,
  `topics_category_hot_idx`, and `topics_author_created_idx`; the public feed has
  no `(is_pinned, created_at, id)` index. `sort=latest` therefore cannot use an
  index for the homepage feed, while `sort=active` matches an existing one.
- Forum reading convention is "latest" = most recent activity (a thread with a
  new reply moves to the top); "newest created" is the secondary view.
- The public control now distinguishes the two explicitly: 最新 (`active`) and
  最新帖子 (`latest`).

## Decision

The recommended default for `forum.list.default_sort` is `active` (last
activity), defined in:

- `apps/api/app/Models/Forum/content_limits.go` (forum settings defaults)
- `apps/api/app/Models/Options/community_policy_options.go` (public option
  recommended defaults, used by admin "restore recommended")
- `apps/web/app/utils/forum/forumTaxonomy.ts` (web forum settings defaults)

The option stays operator-configurable (`latest|active|hot`) and the homepage
control always highlights the effective sort, so a site that pins `latest` or
`hot` still renders a truthful control instead of a hard-coded "latest".

## Consequences

- Fresh installs and "restore recommended defaults" now order the feed by last
  activity; an operator who wants creation order selects 最新帖子 / `latest`.
- Existing installations that already stored an explicit value are unaffected
  until the operator changes it (options are stored, not derived).
- `sort=latest` on the homepage remains supported but is not index-backed; if it
  becomes a hot path, add `(is_pinned, created_at DESC, id DESC)` rather than
  reordering in the client.
