# 2026-07-22 Forum Content Revisions V1 M2 Handoff

## Changed

- Added `topic.revision.view_any` and `post.revision.view_any` to Go seed
  constants, seed catalog, permission migration `202607220053`, built-in
  moderator template, frontend permission constants/templates, and zh-CN/en-US
  permission labels.
- Added Forum revision summary/detail read models plus keyset cursor support.
  Topic/comment revision lists return headers only; detail returns raw source
  only after the matching history permission and renders a safe preview on
  demand without persisting derived HTML.
- Added read-only admin topic/comment content list/detail models. Admin content
  reads require `admin.access` plus matching edit-any or history-view
  permission and use id/status/author/date/category/title-prefix/topic filters,
  not body substring scans.
- Added topic/comment revision routes, admin content read routes, controller
  handlers, error mapping for `forum.revision_not_found` and
  `forum.revision_redacted`, modular OpenAPI path/schema refs, and focused
  service/controller/PostgreSQL tests.

## Decisions

- M2 did not change edit write semantics, `expectedRevision`, restore,
  redaction, admin UI, diff UI, audit writes, privacy deletion, or plugin
  boundaries.
- History authorization is service-authoritative. Unauthorized callers are
  denied before store lookup, preserving hidden-target non-enumeration for
  actors without history permissions.

## Verification

- `GOCACHE=/private/tmp/sforum-gocache go test ./app/Models/Identity ./database/migrations ./app/Models/Forum ./app/Http/Controllers/Forum`
- `set -a; . ../../.env; set +a; GOCACHE=/private/tmp/sforum-gocache go test ./app/Models/Forum -run 'TestRevision(ReadModels|Ledger).*Postgres' -count=1`
- `ruby scripts/validate-openapi-refs.rb`
- `node tests/validate-identity-ui.js`

## Next

- Start M3 only: add `expectedRevision`/`reason` inputs, mandatory CAS, accepted
  edit snapshots, no-op detection, reason rules, comment update hooks/events,
  transaction-aware audit for cross-author edits, and first-party edit token
  submission.
- Keep M4+ deferred: restore, attachment restore validation, redaction, admin
  UI, diff UI, browser QA, and final rollout/perf closure remain later
  milestones.

## Open Questions

- None blocking M3.
