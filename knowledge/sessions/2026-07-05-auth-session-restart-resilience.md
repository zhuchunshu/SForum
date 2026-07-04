# 2026-07-05 Session Handoff

## Changed

- Updated `useAuthSession()` so only 401 or `auth.required` clears the current
  user. API restart, timeout, or gateway-style failures now mark auth as
  temporarily unavailable and preserve existing frontend state.
- App startup now attempts to restore `/auth/session` alongside web options, so
  a valid `sforum_session` cookie repopulates frontend auth state after reload.
- Admin route middleware now shows a temporary auth-service unavailable error
  instead of redirecting to login when session refresh cannot reach the API.
- The admin route middleware intentionally avoids `useI18n()` so post-login
  navigation cannot be blocked by route-middleware composable context issues.
- Added Bun tests for unauthenticated-versus-transient auth refresh errors and
  login-page success navigation.

## Decisions

- Keep Redis-backed browser sessions. The issue was frontend transient-failure
  handling, not a need to switch to JWT.
- Treat confirmed unauthenticated responses and service availability failures as
  separate states in the frontend auth store.
- Do not call `useI18n()` directly in admin route middleware. The regression
  symptom was successful login/register responses with no visible error and no
  navigation; the fix was to keep middleware context-light and verify the
  success navigation path.

## Next

- Add CSRF protection for cookie-authenticated unsafe requests.
- Consider a small user-facing retry affordance on temporary auth-service
  unavailable pages if the admin shell gets a dedicated error layout.

## Open Questions

- None.
