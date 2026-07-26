# 2026-07-27 Topic Contributors Handoff

## Changed

- Topic detail side card **参与者** renamed to **贡献者** (en: Contributors).
- Public read model on `TopicDetail`:
  - `contributors` — unique author + edit/restore actors, author first, max 5
  - `contributorCount` — full unique count for `+N`
- Public endpoint `GET /topics/{topicID}/contribution-timeline` (guest-read same
  as topic detail): newest-first header events only (actor, operation, origin,
  changedFields, committedAt, redacted). No reason, raw source, preview, or
  restorable fields. Staff editors fully exposed by default.
- Frontend: `SFAvatarGroup`, `SFTopicContributorsModal`, side-card wiring,
  i18n zh-CN/en-US, `useForumApi.listTopicContributionTimeline`.
- **Fix (same day):** register
  `core.route.forum.topic_contribution_timeline` in V3 route catalog +
  `core.guard.forum.read` partition (Fiber-only registration was 404 via
  dispatcher). Bump topic detail cache prefix to `forum:topic:v2:` so old
  cached details without `contributors` are not reused.

## Decisions

- Label is **贡献者**, not discussion participants (reply authors stay out).
- Privacy toggle for staff actor visibility is deferred; default is full expose.
- Full revision source/preview/restore remains behind `topic.revision.view_any`.

## Next

- Optional: operator setting to hide staff identities on the public timeline.
- Optional: authorized users deep-link from modal into staff revision detail.
- Optional: discussion-participant stack as a separate surface later.

## Open Questions

- None for V1 of this surface.
