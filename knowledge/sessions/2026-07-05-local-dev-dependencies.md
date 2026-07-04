# 2026-07-05 Local Dev Dependencies

## Changed

- Changed `./scripts/dev.sh` to start only PostgreSQL, Redis, Meilisearch, and
  Mailpit for local development.
- Added loopback-only development ports for dependency services so host-run
  Nuxt and Air can connect.
- Updated Air and Nuxt dev commands to load the repository root `.env`.
- Updated local development docs and defaults for host-process frontend/API
  work.

## Decisions

- Frontend and API hot reload now run locally with `bun run dev` and `air`.
- `scripts/dev.sh` runs Goose migrations after dependencies are ready unless
  `--no-migrate` is passed.
- Production Compose keeps internal services private and publishes only web.

## Next

- Run `./scripts/dev.sh`, then start `cd apps/web && bun run dev` and
  `cd apps/api && air` manually.
- Use Mailpit at `http://127.0.0.1:18025` when email flows are added.

## Open Questions

- Whether a helper script should be added later for starting the optional
  worker locally when real job handlers are wired.
