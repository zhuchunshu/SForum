# 2026-08-19 API-Owned Worker Handoff

## Changed

- Removed `EmbedWorkerInAPI` and the `EMBED_WORKER_IN_API` runtime/configuration
  surface; the API always starts the embedded River Worker outside Safe Mode or
  host recovery.
- Removed standalone Worker services from development, production, release,
  and blue/green Compose files.
- Updated normal and blue/green deploy flows to pull only API, migrator, and
  Web images and to remove legacy `worker`, `worker-blue`, and `worker-green`
  containers by exact Compose labels, including installations whose project
  name is configured only in `.env.production`.
- Updated PostgreSQL restore handling and V3 trust topology validation so they
  manage only the API and never restart a retained legacy Worker.
- Removed the public Worker image and server-bundle binary from release
  workflows, release notes, anonymous-pull gates, and asset validation.
- Retired `scripts/worker-dev.sh` as an explicit compatibility error and
  updated bilingual operator/developer documentation.

## Decisions

- API, background jobs, SettingsLifecycle, SecretStore, and the extension
  runtime are one process ownership boundary. Split Worker deployment is no
  longer supported.
- The standalone binary remains internal compatibility/test scaffolding and is
  not exposed by Compose or deploy tooling.

## Verification

- `go test ./...`
- release/deploy/configure, PostgreSQL restore safety, production Compose, and
  blue/green topology/state tests
- architecture, documentation, OpenAPI, Protobuf, V3 trust/catalog, extension,
  WebSocket, and development Worker validators
- V3 P12 compatibility farm with both required cells passing
- Nuxt typecheck and the eight focused Web unit suites (38 tests)
- `git diff --check` and affected Shell syntax checks

`./scripts/test.sh` passed all checks before its compatibility-farm step. The
farm requires an explicit database URL in this local environment, so it and all
subsequent gates were run separately; injecting that URL into the entire script
enables unrelated shared-database integration tests and is not equivalent to
the normal local gate.

## Next

- Publish and deploy a new immutable release, then confirm the old standalone
  Worker container and duplicate plugin subprocess set are gone.
- Send a fresh admin test email; historical failed deliveries remain terminal.

## Open Questions

- None.
