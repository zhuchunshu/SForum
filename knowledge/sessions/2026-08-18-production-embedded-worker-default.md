# 2026-08-18 Production Embedded Worker Default

## Changed

- Production now defaults to `EMBED_WORKER_IN_API=true`; the generated and
  example production configurations agree with the Go default.
- Normal Compose keeps the standalone Worker in the optional `split-worker`
  profile. `deploy.sh` selects required images, identity checks, services, and
  health expectations from the configured mode.
- API and Worker Compose environments now receive their database-pool, queue
  concurrency, and shutdown controls instead of silently using loader defaults.
- Release smoke verifies that no standalone Worker starts and that Redis
  receives the embedded Worker heartbeat.
- Blue/green slots continue to use explicit standalone Workers for drainable
  queue handoff.

## Decisions

- Recommended single-node production optimizes for the lower fixed memory cost
  of a shared API/Worker runtime. Split mode remains supported for isolation,
  independent scaling, and blue/green operations.

## Verification

- `./scripts/test.sh`: release/deploy/docs/architecture and all Go tests passed;
  stopped only at the required compatibility-farm PostgreSQL cell because this
  environment has no `DATABASE_URL` / `SFORUM_TEST_DATABASE_URL`.
- The complete post-farm gate passed separately: Protobuf and SDK drift,
  OpenAPI refs, production trust/WebSocket contracts, Nuxt typecheck, focused
  Web tests, admin/identity/SEO/moderation/theme/Page Registry/catalog checks.
- Production Compose render tests cover both default embedded mode and the
  `EMBED_WORKER_IN_API=false` split profile. Blue/green tests prove slot APIs
  force split mode.
- `go test ./config ./bootstrap`, shell syntax checks, ShellCheck, and
  `git diff --check` passed.

## Next

- Capture API, plugin, PostgreSQL connection, and total RSS/PSS before and after
  the next immutable release deployment to quantify the production saving.

## Open Questions

- None.
