# 2026-07-06 Session Handoff

## Changed

- Added explicit plugin event and extension-point v1.
- Added `app/Support/Events` with host event definitions, envelopes, results,
  no-op publisher, and rejection errors.
- Extended extension manifests with `events`; legacy `hooks` remain compatible.
- Added event delivery tracking through `extension_event_deliveries` and admin
  APIs for event definitions and delivery attempts.
- Extended plugin RPC event payloads with kind, delivery id, correlation id,
  timeout, patch fields, and patch responses.
- Wired `user.registered`, `topic.before_create`, `topic.created`,
  `comment.created`, and `attachment.uploaded` into service boundaries.
- Updated the admin Event Log page to show event definitions, listener delivery
  attempts, and lifecycle audit logs separately.

## Decisions

- Plugins cannot override core routes or monkey-patch arbitrary behavior.
- Replacement behavior must use explicit filter events or future Provider Slots.
- `topic.before_create` is the first synchronous filter event and only allows
  patches to `title`, `categorySlug`, and `content`.
- Delivery attempts are separate from lifecycle audit events.
- River job args and worker plumbing exist for async delivery; runtime falls
  back to inline delivery when no dispatcher is configured.

## Next

- Wire a production River dispatcher/worker path once River migrations and
  worker registration are operationally enabled.
- Add owner-module Provider Slot selection UIs before allowing storage,
  verification, or search replacement in production paths.
- Add plugin author documentation with manifest `events` examples.

## Open Questions

- Should event delivery payloads be retained for debugging, or should delivery
  records intentionally remain metadata-only for privacy?
- Which Provider Slot should be implemented first: search, sanitizer, or a low
  risk content enrichment slot?
