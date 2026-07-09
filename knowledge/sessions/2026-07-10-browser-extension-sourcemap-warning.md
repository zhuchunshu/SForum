# 2026-07-10 Browser Extension Sourcemap Warning

## Changed

- Confirmed repeated Nuxt dev warnings for `/content.css.map` and `/sidebar.css.map` are caused by Chrome extension-injected CSS sourcemap comments, not by SForum assets.
- `theme-proxy.mjs` now returns empty `204` responses for those two known root-level sourcemap probes before forwarding to Nuxt, preventing Vue Router "No match found" noise.
- Added a proxy regression test that verifies the two probes are not forwarded upstream.

## Decisions

- Keep the handling in the shared theme proxy instead of Nuxt pages/routes, because the requests are external browser-extension artifacts and should not enter app routing.
- Limit the ignore list to the two observed root-level paths so normal `/_nuxt/**` sourcemaps and other static assets continue through the existing path.

## Verification

- `bun test tests/themeProxy.test.ts`
- `bun run typecheck`

## Next

- Restart the currently running `bun run dev` process before expecting port `3000` to use the updated proxy code.
