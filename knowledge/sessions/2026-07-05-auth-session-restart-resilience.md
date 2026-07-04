# 2026-07-05 Session Handoff

## Changed

- Updated `useAuthSession()` so only 401 or `auth.required` clears the current
  user. API restart, timeout, or gateway-style failures now mark auth as
  temporarily unavailable and preserve existing frontend state.
- App startup now attempts to restore `/auth/session` alongside web options, so
  a valid `sforum_session` cookie repopulates frontend auth state after reload.
- Admin route middleware now shows a temporary auth-service unavailable error
  instead of redirecting to login when session refresh cannot reach the API.
- Added localized auth-service unavailable messages and Bun tests for
  unauthenticated-versus-transient auth refresh errors.

## Decisions

- Keep Redis-backed browser sessions. The issue was frontend transient-failure
  handling, not a need to switch to JWT.
- Treat confirmed unauthenticated responses and service availability failures as
  separate states in the frontend auth store.

## Next

- Add CSRF protection for cookie-authenticated unsafe requests.
- Consider a small user-facing retry affordance on temporary auth-service
  unavailable pages if the admin shell gets a dedicated error layout.

## Open Questions

- None.
