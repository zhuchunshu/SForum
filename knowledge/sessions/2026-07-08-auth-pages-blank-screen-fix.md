# 2026-07-08 Auth Pages Blank Screen Fix

## Changed

- Made client-side app startup refresh lazy so SPA-only routes are not blocked
  by `/web-options` or `/auth/session` before the first render.
- Removed `ssr: false` from public auth routes: login, register,
  forgot-password, reset-password, and their English paths.
- Added regression tests for non-blocking client startup and auth route SSR.

## Decisions

- Keep public auth pages server-rendered so the browser receives useful form
  HTML before the Nuxt client bundle finishes.
- Keep settings, posting, editing, admin, and component-preview routes as SPA
  because they depend more heavily on authenticated client state and rich
  interactions.

## Verification

- `bun test tests/appStartup.test.ts tests/authRouteRendering.test.ts tests/useApiClient.test.ts`
- `bun run typecheck`
- `bun run build`
- Browser QA on `http://127.0.0.1:3000/login` and `/register` confirmed the
  rendered DOM contains the auth forms instead of an empty `#__nuxt` shell.

## Next

- The dev proxy still reports a Vite HMR websocket warning through port 3000;
  it did not block rendering in this session.

## Open Questions

- None.
