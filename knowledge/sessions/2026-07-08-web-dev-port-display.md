# 2026-07-08 Web Dev Port Display

## Changed

- Fixed misleading frontend dev startup output. `bun run dev` now loads the
  repository root `.env` through `bun --env-file=../../.env`, so the dev
  supervisor can use `PORT` or `WEB_PORT` for its public proxy port.
- `dev-theme-runtime.mjs` now prints the supervisor public URL and suppresses
  Nuxt child-process `Local`/`Network` address lines. Those Nuxt lines point at
  internal random `PORT=0` ports used only for blue-green switching health
  checks.
- Added startup display helpers in `theme-proxy.mjs` and regression tests for
  public URL formatting, internal Nuxt address-line detection, and the dev
  script's root-env loading behavior.

## Decisions

- Keep the existing blue-green dev architecture: supervisor listens on the
  public port, inner Nuxt dev processes listen on temporary ports. The fix is
  to display only the public proxy URL, not to make inner Nuxt own the public
  port again.
- Prefer `PORT` when explicitly set; otherwise use `.env` `WEB_PORT`, falling
  back to `3000`.

## Verification

- `bun test tests/devRuntimeStartup.test.ts tests/themeProxy.test.ts` in
  `apps/web` passed: 17 pass / 0 fail.

## Next

- Manual smoke when convenient: `cd apps/web && bun run dev` should show
  `[sforum-dev-runtime] public URL: http://127.0.0.1:<WEB_PORT>/` and should no
  longer show Nuxt's internal random Local/Network port as the access URL.
