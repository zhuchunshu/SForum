# 2026-07-11 Auth Return Navigation Session Handoff

## Changed

- Added validated frontend auth return resolution with explicit redirect,
  optional usable same-origin browser referrer, and localized-home fallback
  precedence.
- Protected global and admin middleware now preserve `to.fullPath` when sending
  guests to login.
- Added host `guest` middleware for auth pages. It refreshes unknown session
  state, continues for guests or unavailable auth, and returns authenticated
  visitors before page setup.
- The tracked default-theme login/register pages opt into `guest`, update the
  frontend user after successful authentication, and use replace-style return
  navigation.

## Decisions

- Accept only local absolute return paths; reject external, protocol-relative,
  malformed, and login/register destinations to prevent open redirects and
  auth loops.
- Keep authenticated entry behavior in host middleware so themes inherit it by
  opting in, rather than duplicating session refresh and redirect logic.
- Keep development-theme auth files ignored/untracked. This delivery covers the
  tracked default-theme pages and does not stage development-theme files.
- This is frontend navigation only. No API endpoint, permission, or backend
  authorization boundary changed.
- Explicit `redirect` is the reliable protected-route restoration mechanism.
  The referrer fallback is best-effort because Nuxt SPA client navigation does
  not guarantee that `document.referrer` follows route changes.

## Implementation Entrypoints

- `apps/web/app/utils/authReturn.ts` validates and resolves return candidates.
- `apps/web/app/composables/useAuthReturnNavigation.ts` adapts route query,
  optional browser referrer, localized fallback, and replace navigation.
- `apps/web/app/middleware/guest.ts` handles authenticated auth-page entry.
- `apps/web/app/middleware/auth.global.ts` and
  `apps/web/app/middleware/admin.ts` preserve protected destinations.
- `extensions/builtin/themes/sforum-default/layer/app/pages/login.vue` and
  `register.vue` opt into `guest` and use post-auth return navigation.

## Verification

- Focused frontend auth suite: 26 passed, 0 failed.
- `bun run typecheck`: exited 0.
- Full-worktree `git diff --check`: blocked only by unrelated trailing
  whitespace in `apps/api/app/Models/Extensions/service_test.go`; the auth
  return documentation diff passed its scoped whitespace check.

## Next

- Require future tracked theme auth pages to declare `middleware: 'guest'` and
  use `returnFromAuth()` after updating frontend session state.

## Open Questions

- None for this change.
