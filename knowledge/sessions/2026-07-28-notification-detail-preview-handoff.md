# 2026-07-28 Notification Detail Preview Handoff

## Changed

- Added recipient-owned `GET /api/v1/notifications/:id` with current-authorized
  topic/comment preview, foreign/missing 404 behavior, contracts, and tests.
- Added `/notifications/:notificationId` and `forum.notification.show`; list
  clicks stay on the notification surface and only `查看回复` opens the target.
- Moved the list shell to `pages/notifications/index.vue` so Nuxt renders the
  sibling detail route instead of keeping a flat parent mounted.
- Completed Page Registry catalog, ViewModel population, Host island bindings,
  built-in templates, generated catalogs, and completeness tests.
- Removed duplicate notification Host-island footers; theme and Core fallback
  paths now each own exactly one public navbar/footer.
- Expanded `AGENTS.md` with nested-route, Page Registry completion, runtime
  provider/template, and single-owner chrome requirements. Added a Go guard
  against parent route files shadowing nested pages without `<NuxtPage>`.

## Decisions

- Notification bodies are not durable snapshots. Preview reads current visible
  forum content and fails closed when the target is no longer authorized.
- Frontend login middleware is UX only; the API's recipient ownership check is
  authoritative and deliberately hides foreign rows as 404.
- The detail theme ViewModel contains shell data only. Recipient content remains
  inside the Host API/island boundary.

## Evidence

- Chrome `/notifications/58`: `forum.notification.show`, provider
  `sforum.default-theme`, `data-template="1"`, `data-host-chrome="0"`, one
  visible footer, and no new warning/error logs.
- Chrome flow: `/notifications` row -> `/notifications/58` -> explicit target
  action -> `/t/63/vibecoding#comment-279`.
- Focused Go suites, 88 Bun tests, Nuxt typecheck, OpenAPI refs, V3 catalog
  check, `git diff --check`, and architecture validation pass.

## Next

- Repeat the detail visual check at a mobile viewport when the selected Chrome
  control surface exposes viewport emulation; current Chrome only exposed the
  `pageAssets` capability.

## Open Questions

- None.
