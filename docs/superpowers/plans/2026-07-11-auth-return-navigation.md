# Auth Return Navigation Implementation Plan

**Goal:** Return authenticated visitors and newly authenticated guests to a
validated internal destination instead of a fixed admin or forum home page.

**Final architecture:** A pure utility validates return paths. A Nuxt composable
resolves explicit route query, same-origin client referrer, and localized home,
then navigates with history replacement. Protected-route middleware preserves
`to.fullPath`. The host `guest` middleware owns authenticated auth-page entry
before page setup, and tracked default-theme pages opt into it.

## Task 1: Safe Return Path Resolver

- [x] Add `apps/web/app/utils/authReturn.ts` and focused unit tests.
- [x] Accept local absolute paths while preserving query and fragment.
- [x] Reject external URLs, protocol-relative paths, malformed encodings,
  non-string values, and localized or unlocalized login/register paths.
- [x] Resolve explicit redirect, referrer, then safe fallback.

## Task 2: Nuxt Return Navigation Adapter

- [x] Add `useAuthReturnNavigation` around the pure resolver.
- [x] Read `route.query.redirect` and capture only same-origin browser
  referrers on the client.
- [x] Use `localePath('/')` as fallback and `navigateTo(..., { replace: true })`
  so Back does not reopen an auth form.

## Task 3: Preserve Protected Destinations

- [x] Update global auth and admin middleware to carry `to.fullPath` in the
  localized login route's `redirect` query.
- [x] Cover both middleware paths with focused tests.

## Task 4: Host Guest Middleware And Tracked Theme Integration

- [x] Add host `guest` middleware and tests for unknown-session refresh,
  guest/unavailable continuation, and authenticated return navigation.
- [x] Make the tracked default-theme login/register pages opt into
  `middleware: 'guest'`.
- [x] After successful login/registration, call `setUser` and then
  `returnFromAuth`; remove fixed admin/home routing.
- [x] Keep development-theme files outside the tracked delivery. Ignored or
  untracked development themes inherit host entry handling only when their auth
  pages opt into `guest` middleware.

Quality correction: the original plan placed authenticated entry checks inside
both bundled themes. The final implementation centralizes that behavior in the
host `guest` middleware before page setup. Only the tracked default-theme pages
are part of this change; ignored development-theme files must not be staged.

## Task 5: Knowledge And End-To-End Verification

- [x] Update the identity module note, design, and session handoff with the
  final middleware ownership and tracked-file boundary.
- [x] Record that this frontend-only change adds no API or permission boundary.
- [x] Run the complete focused test set, Nuxt typecheck, and whitespace check.

```bash
cd apps/web && bun test tests/guestMiddleware.test.ts tests/authReturn.test.ts tests/authRouteRendering.test.ts tests/protectedRouteRendering.test.ts tests/adminRouteRendering.test.ts
cd apps/web && bun run typecheck
git diff --check
```

- [x] Stage only these documentation files and commit as
  `docs: record auth return navigation`:

```text
knowledge/modules/identity.md
knowledge/sessions/2026-07-11-auth-return-navigation.md
docs/superpowers/specs/2026-07-11-auth-return-navigation-design.md
docs/superpowers/plans/2026-07-11-auth-return-navigation.md
```
