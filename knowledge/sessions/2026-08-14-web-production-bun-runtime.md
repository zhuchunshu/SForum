# 2026-08-14 Web Production Bun Runtime Handoff

## Changed

- Web production runtime migrated from Node to Bun:
  - `apps/web/Dockerfile` `prod` stage now `FROM oven/bun:1.3.14-alpine`
    (same fixed version as deps/build), removed `apk add --no-cache nodejs`,
    kept the `sforum` non-root user and `production-entrypoint.sh`, and now
    runs `CMD ["bun", ".output/server/index.mjs"]`.
  - `apps/web/package.json` `start` is now `bun .output/server/index.mjs`.
  - `apps/web/nuxt.config.ts` sets `nitro.preset: 'bun'` — required because the
    default `node-server` output cannot start under Bun (see Decisions).
  - `tests/validate-theme-runtime.js` static assertion updated to expect the
    `bun` CMD.

## Decisions

- Run the Nitro output under Bun via the `bun` preset rather than pinning the
  `node-server` output. Root cause: `node-server` externalizes `srvx`
  (via `ipx`/`h3`) with only the `node` adapter, but `srvx`'s `exports` map
  lists a `bun` condition before `node`, so Bun resolves a tree-shaken
  `dist/adapters/bun.mjs` and crashes with `Cannot find package 'srvx'`.
  The `bun` preset bundles the bun adapter and serves with `Bun.serve()`.
- See `decisions/2026-08-14-web-production-bun-runtime.md`.

## Verification

- `docker build --target prod -f apps/web/Dockerfile -t sforum-web:bun .`: pass.
  Local build initially OOM'd because Docker Desktop was capped at 3 GB; it was
  raised to 12 GB to complete the Nitro bundling step (environmental, not a
  code change).
- Image inspection: `bun --version` 1.3.14; runs as `sforum` (uid 100); no
  `nodejs` package via `apk info`; `node` is Bun's fallback symlink (not Node).
- Runtime: `docker run` served `/health` 200, process `bun
  .output/server/index.mjs` as PID 1 under `sforum`, log `Listening on
  http://0.0.0.0:3000/...`, and `production-entrypoint.sh` injected
  `NUXT_PUBLIC_I18N_BASE_URL`/`NUXT_PUBLIC_SITE_URL`/`NUXT_SITE_URL` from
  `APP_URL` (with `http://127.0.0.1:3000` default when unset) — no regression.
- `node tests/validate-architecture-boundaries.mjs`: pass.
- `node tests/validate-theme-runtime.js`: pass.
- `cd apps/web && bun test`: 883 pass / 0 fail.
- `cd apps/web && bun run typecheck`: pass.

## Next

- Exercise an actual `@nuxt/image` optimization request under Bun once the API
  and a source image are available (sharp native addon is the main unverified
  surface).
- Restore Docker Desktop memory from 12 GB to the operator's preferred value;
  note that a full web image rebuild needs roughly 8-12 GB in the Docker VM.

## Open Questions

- None.
