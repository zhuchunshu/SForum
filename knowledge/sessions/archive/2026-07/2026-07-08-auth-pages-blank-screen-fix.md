# 2026-07-08 Auth Pages Blank Screen Fix

## Changed

- Made client-side app startup refresh lazy so SPA-only routes are not blocked
  by `/web-options` or `/auth/session` before the first render.
- Removed `ssr: false` from public auth routes: login, register,
  forgot-password, reset-password, and their English paths.
- Removed `ssr: false` from ordinary protected user workflows: settings,
  posting, topic editing, and their English paths.
- Explicitly disabled route cache on those protected user workflows. Topic
  editing uses routeRules dynamic segments (`/t/:topicID/:topicSlug/edit`) so
  it overrides the broader public `/t/**` SWR rule.
- Added a global `requiresAuth` route middleware for ordinary user pages. It
  redirects missing users to the locale-aware login page, including when the
  auth API is temporarily unavailable and no cached user is present.
- Added regression tests for non-blocking client startup and auth route SSR.
- Added regression tests that protected user workflows are not SPA shells and
  do not render an auth-unavailable 503 shell.

## Decisions

- Keep public auth pages server-rendered so the browser receives useful form
  HTML before the Nuxt client bundle finishes.
- Keep ordinary protected user pages server-rendered so unauthenticated users
  get an immediate login redirect instead of waiting on client-only route
  mounting.
- Keep admin and component-preview routes as SPA-only. Admin uses its dedicated
  guard and can show a temporary unavailable error for auth-service failures.

## Verification

- `bun test tests/appStartup.test.ts tests/authRouteRendering.test.ts tests/protectedRouteRendering.test.ts tests/useApiClient.test.ts`
- `bun run typecheck`
- `bun run build`
- Browser QA on `http://127.0.0.1:3000/login` and `/register` confirmed the
  rendered DOM contains the auth forms instead of an empty `#__nuxt` shell.
- Browser QA on `http://127.0.0.1:3000/topics/new` confirmed an unauthenticated
  visit lands on `/login` with visible form content instead of an empty shell.
- HTTP checks confirmed `/topics/new` and `/settings/profile` return `302
  Location: /login`; `/topic/new` is not a real route and returns 404.
- HTTP checks confirmed `/t/:topicID/:topicSlug/edit` and the English
  equivalent return locale-aware login redirects without `s-maxage` caching.

## Next

- The dev proxy still reports a Vite HMR websocket warning through port 3000;
  it did not block rendering in this session.

## Open Questions

- None.
