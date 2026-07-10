# 2026-07-11 Auth Return Navigation Session Handoff

## Changed

- Added validated frontend auth return resolution with explicit redirect,
  same-origin client referrer, and localized-home fallback precedence.
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

## Next

- Require future tracked theme auth pages to declare `middleware: 'guest'` and
  use `returnFromAuth()` after updating frontend session state.

## Open Questions

- None for this change.
