# 2026-07-11 Moderation Workbenches Handoff

## Changed

- Added independent `moderation.manage` and `moderation.review` permissions.
- Added moderation settings, decisions, pending/rejected forum states, rule
  evaluation, transactional decisions, enriched queues, audit APIs, and author
  status API.
- Replaced the old admin report queue with policy management and audit history.
- Added the frontend moderator workbench and author content-review page.
- Added OpenAPI coverage, Go tests, frontend static validation, bilingual text,
  and search refresh after review decisions.

## Decisions

- `off` remains the compatibility default; the UI recommends `rules`.
- Core rules are limited to new users and explicit external links. Advanced
  detection is a plugin/filter concern.
- Admin management and daily review permissions do not imply each other.
- Pending/rejected content never enters public reads, counters, or search.

## Next

- Consider task assignment/SLA only after real moderation-team demand.
- Add plugin-provided risk signals through a documented publication filter.

## Open Questions

- None for this release.
