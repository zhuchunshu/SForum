# 2026-10-11 Achievements: Grant Provisioning + Route Schema Fragment Fixes

## Changed

- **own_schema grant provisioning** (`app/Models/Extensions/activation.go`):
  `ActivationCoordinator.Start` now mints an actor-bound `DatabaseGrantAuthority`
  (via `audit.IDWriter.AppendReturningID` with action `extension.database_grant`)
  and attaches it to the runtime start context before `runtime.Start`. The
  `ProtocolStarter` (`app/Support/Extensions/protocol_database_lease.go`) reads
  this authority from context and issues the lease with `Actor` kind, so the
  registry's `resolveRuntimeLeaseGrant` auto-provisions the exact grant row.
  Previously only tests issued actor-bound leases; production always used Host
  authority → `extension database exact grant not found` for any `own_schema`
  plugin.
- **OpenAPI fragment** (`extensions/builtin/plugins/sforum-gamification/`):
  the manifest's 9 HTTP routes declared `requestSchema`/`responseSchema` as
  aggregate IDs (`sforum.gamification.route.*@1`) but the package had no
  `openapi[]` fragment. Boot's `RouteSchemaCatalog` build failed with
  `route schema is missing`, crashing the API. A JSON OpenAPI fragment
  (`openapi/gamification.openapi.json`) was generated with all 12 operations
  (9 paths × methods), inline request/response schemas, and the required
  `x-sforum-*` metadata fields. Route schema refs remain as `@1` aggregate
  IDs (required by protocol v2 runtime `protocolV2SchemaRef`).
- **Route schema contract check** (`sdk/plugin/contract.go`):
  `TestManifest` now validates that HTTP route schema references are
  resolvable: `.json` refs must have a `packageFiles(kind=schema)` entry and
  file on disk; aggregate ID refs (`@version`) require `openapi[]` fragments.
  This prevents the "CLI all-green but API boot crashes" trap.
- **Integration tests**: `gamification_database_grant_enable_integration_test.go`
  (operator-bound start provisions grant; Host start fails closed),
  `gamification_catalog_integration_test.go` (real package RouteSchemaCatalog
  build). The events fixture (`sforum-contract-events`) was fixed to include
  schema files + packageFiles declarations.
- **Audit constant**: `ActionExtensionDatabaseGrant = "extension.database_grant"`.

## Decisions

- Actor-bound grant provisioning lives in the enable layer (ActivationCoordinator),
  not the registry — mirrors the lifecycle-v2 pattern (`appendLifecycleRequestAudit`
  → AuditEventID passed to coordinator). The registry's own validation stays strict
  (`ActorUserID > 0 && AuditEventID > 0`).
- Route schema refs stay as `@1` aggregate IDs (not `.json` paths) because
  protocol v2 runtime (`protocolV2SchemaRef`) requires `id@version` format. The
  OpenAPI fragment bridges both: boot's catalog resolves from the fragment, and
  the runtime validates the `@1` format.

## Verification

- `go test ./app/Models/Extensions/ ./app/Support/Extensions/ ./app/Support/Audit/ ./app/Support/ExtensionOpenAPI/ ./sdk/plugin/` — all pass.
- `go test ./app/Support/Extensions/IntegrationTests/ -run TestGamification` — 4/4 pass (admin routes, grant provisioning, host fail-closed, observe delivery).
- `extension validate` + `extension test` — PASS (10 checks, 0 errors).
- Architecture gate — passed.
- API boot with enabled plugin (version 920, fragment manifest) — succeeds;
  routes registered (503 recovery-required instead of 404 not-found).

## Next

- **Re-enable plugin**: log into admin UI (`/control-panel/extensions`), click
  启用 on the gamification card. The enable flow will provision a grant for
  version 920 via the actor-bound path, start the runtime, and make routes
  fully functional (200/401 instead of 503).
- **Rendered Browser QA**: desktop + 390x844 on `/control-panel/gamification`
  (admin page) and `/badges/{username}` (public badge wall).
- **Permission catalog**: confirm `sforum.gamification.manage` appears in admin
  permissions UI after re-enable; approve role suggestion for `admin`.
- **Module note**: add a gamification section to `knowledge/modules/extensions.md`.
