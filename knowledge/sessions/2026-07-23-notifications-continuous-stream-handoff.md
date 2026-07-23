# 2026-07-23 Notifications Continuous Stream Handoff

## Changed

- Reworked the default-theme `/notifications` body island into a full-width
  three-column continuous stream aligned to
  `tmp/demos/notification-inbox-directions-20260723/01-continuous-stream.html`.
- Kept the page component below the project file-size warning by moving its
  private scoped styles to `SFNotificationsPage.css` and referencing them from
  the SFC.
- Added presentation helpers for API-backed notification mapping, loaded-only
  filtering, date grouping, unread counting, and conservative target links.
- Updated `useNotifications` for `admin_test`, `beforeId` pagination, and
  read-all unread sync.
- Added bilingual copy and focused Bun tests for mapping, pagination,
  filtering, Page Registry shell, drawer state, optimistic read actions, and
  failure copy.

## Decisions

- Filters stay client-side over the loaded notification page because the API
  does not expose type filters.
- The right rail uses `/notifications/unread-count` for authoritative global
  unread count; type bars/counts are loaded-list summaries only.
- Unreliable targets stay disabled rather than fabricating routes.

## Verification

- Passed `cd apps/web && bun test tests/notificationsPage.test.ts`.
- Passed `cd apps/web && bun run typecheck`.
- `git diff --check` passed.
- Browser plugin and Playwright/Nuxt rendered QA were attempted but blocked by
  local browser/dev-server environment issues: Browser CDP `Page.navigate`
  timed out on localhost, Nuxt dev hit `EMFILE: too many open files`, and
  later dev instances listened without returning HTTP bytes.

## Next

- Re-run rendered browser QA once the local Nuxt dev/browser environment is
  healthy: desktop, medium, mobile, dark mode, mobile drawers, filter/select,
  mark single read, mark all read, load more, and target navigation.

## Open Questions

- None for the data contract; interaction QA only awaits a stable local
  rendered environment.
