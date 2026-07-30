# 2026-07-31 Navbar Notification Preview

## Changed

- The public Navbar bell now opens an accessible notification preview with
  all, reply, and mention tabs instead of navigating directly to the inbox.
- Every tab requests `limit=3` and defensively caps the rendered list at the
  latest three API-ordered rows. The footer explains that complete history is
  available in the notification center.
- The preview includes detail excerpts, loading/error/empty states, unread and
  mark-all-read behavior, click-outside/Escape dismissal, a desktop popover,
  and a body-teleported mobile bottom sheet.

## Decisions

- Existing notification APIs, recipient authorization, SSE reconciliation,
  presentation helpers, and full inbox/detail routes remain authoritative; no
  backend or OpenAPI change was needed.
- The preview belongs to the notification domain and keeps `SFNavbar` focused
  on public chrome composition.

## Verification

- `bun run typecheck`
- `bun test tests/notifications/notificationsPage.test.ts` (23 pass)
- `node tests/validate-architecture-boundaries.mjs`
- Desktop rendered QA passed at `1280x720`; the user completed final interaction
  verification and reported no issues.

## Next

- None for this scope.

## Open Questions

- None.
