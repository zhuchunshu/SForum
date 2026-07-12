# 2026-07-12 Forum policy controls must be server-enforced

## Context

Admin forum settings exposed several policy toggles that were stored and shown
in the UI but not consistently enforced by the API: mentions, author lock,
duplicate titles, edit marks, soft-delete visibility, and idle auto-lock.

## Decision

1. **mentionsEnabled=false** — do not parse mention usernames and do not fan out
   mention notifications; `@text` remains ordinary content.
2. **allowAuthorCloseReplies** — authors may lock/unlock their own topics only
   when the setting is true; moderators retain `topic.lock`.
3. **duplicateTitlePolicy** — only `block` is server-authoritative. Historical
   `warn` is accepted for compatibility but behaves like `off` (no silent
   half-implemented warn contract). Recommended default is `off`.
4. **showTopicEditMark / showCommentEditMark** — API sets `edited` on list/detail
   when enabled; themes must not invent marks without this field.
5. **softDeleteVisibility** — list SQL remains active-only for public pages;
   when deleted rows appear (or in future staff views), presentation filters
   tombstones and strips body content. Values:
   `author_and_staff | staff_only | hidden`.
6. **autoLockIdleDays** — `0` disables; `>0` is enforced by durable schedule
   `forum.auto_lock_idle` (daily maintenance job).

## Consequences

- Admin switches map to real API behavior.
- `warn` duplicate policy is intentionally non-blocking until a product contract
  for client-side warnings exists.
- Soft-delete tombstones are not leaked to strangers when visibility is
  author/staff restricted.
