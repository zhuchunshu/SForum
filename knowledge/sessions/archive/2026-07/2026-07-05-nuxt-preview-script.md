# 2026-07-05 Nuxt Preview Script

## Changed

- Changed `apps/web` production preview from `nuxt preview --host 0.0.0.0` to
  `HOST=0.0.0.0 bun --env-file=../../.env .output/server/index.mjs`.
- Documented that local preview should run the generated Nitro server directly
  after `bun run build`, while loading the repository root `.env` so the local
  Nuxt proxy reaches the same API target as development.

## Findings

- The installed `nuxi preview` help lists `ROOTDIR`, `--cwd`, `--logLevel`,
  `--envName`, `--extends`, `--port`, and `--dotenv`, but no `--host`.
- Running `nuxt preview --host 0.0.0.0` makes the CLI interpret `0.0.0.0` as
  `ROOTDIR`, so it searches for `.output/nitro.json` under
  `apps/web/0.0.0.0/.output/`.
- The generated Nitro entry point exists at `.output/server/index.mjs`, matching
  the existing Docker production command.
- Without loading the root `.env`, local production preview serves the homepage
  but the Nuxt `/api/v1/*` proxy falls back to `http://api:8080/api/v1`, which
  is only valid inside the Compose production network.

## Next

- Continue using `bun run build` before `bun run preview` when testing the
  production web server locally.

## Verification

- `bun ./node_modules/.bin/nuxt preview --help` in `apps/web`.
- Existing `.output/server/index.mjs` and `.output/nitro.json` were present from
  the latest successful build.
- `PORT=39126 bun run preview` started successfully outside the sandbox and
  served the homepage, `/api/v1/web-options`, and `/api/v1/health` with HTTP
  200.
